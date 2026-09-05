package desktopapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/apiclient"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/credentials"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/serverconnection"
)

func TestAppLoadsAndSavesServerConnection(t *testing.T) {
	store := &connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://saved.local"}}
	app := newApp(store, &sessionStoreStub{}, "1.2.0", nil)

	loaded, err := app.GetServerConnection()
	if err != nil || loaded.ServerBaseURL != "http://saved.local" {
		t.Fatalf("GetServerConnection() = %#v, %v", loaded, err)
	}
	saved, err := app.SaveServerConnection(" https://new.local/ ")
	if err != nil || saved.ServerBaseURL != "https://new.local" || store.saved != " https://new.local/ " {
		t.Fatalf("SaveServerConnection() = %#v, %v; input %q", saved, err, store.saved)
	}
}

func TestAppTestsConnectionThroughNativeClient(t *testing.T) {
	checker := &connectionCheckerStub{result: apiclient.ConnectionResult{
		Meta: apiclient.Meta{
			APIVersion:            "v1",
			ServerVersion:         "1.4.0",
			WorkshopName:          "Prototype Lab",
			MinimumDesktopVersion: "1.1.0",
		},
		Compatible: true,
	}}
	var factoryBaseURL, factoryDesktopVersion string
	app := newApp(&connectionStoreStub{}, &sessionStoreStub{}, "1.2.0", func(baseURL, desktopVersion string) (remoteClient, error) {
		factoryBaseURL = baseURL
		factoryDesktopVersion = desktopVersion
		return checker, nil
	})
	app.Startup(context.Background())

	result, err := app.TestServerConnection(" http://workshop.local:8080/ ")
	if err != nil {
		t.Fatalf("TestServerConnection() error = %v", err)
	}
	if factoryBaseURL != "http://workshop.local:8080" || factoryDesktopVersion != "1.2.0" ||
		result.ServerBaseURL != factoryBaseURL || result.WorkshopName != "Prototype Lab" || !result.Compatible {
		t.Fatalf("TestServerConnection() = %#v, factory = %q/%q", result, factoryBaseURL, factoryDesktopVersion)
	}
}

func TestAppRejectsInvalidConnectionBeforeClientCreation(t *testing.T) {
	called := false
	app := newApp(&connectionStoreStub{}, &sessionStoreStub{}, "1.2.0", func(string, string) (remoteClient, error) {
		called = true
		return &connectionCheckerStub{}, nil
	})
	if _, err := app.TestServerConnection("postgres://database.local/talos"); !errors.Is(err, serverconnection.ErrInvalidBaseURL) {
		t.Fatalf("TestServerConnection() error = %v", err)
	}
	if called {
		t.Fatal("client factory called for invalid base URL")
	}
}

func TestAppLoginStoresTokenWithoutReturningItAndRestoresState(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	remote := &connectionCheckerStub{loginResult: apiclient.LoginResult{
		Token: "opaque-token", ExpiresAt: expiresAt,
		User:   apiclient.LoginUser{ID: "user-1", Name: "Workshop Owner", EmailOrUsername: "owner", Role: "owner", Permissions: []string{"settings.manage"}},
		Device: apiclient.LoginDevice{ID: "device-1"},
	}}
	secureStore := &sessionStoreStub{}
	app := newApp(
		&connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://workshop.local"}},
		secureStore,
		"1.2.0",
		func(string, string) (remoteClient, error) { return remote, nil },
	)
	state, err := app.Login("owner", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if !state.Authenticated || state.UserName != "Workshop Owner" || secureStore.saved.Token != "opaque-token" {
		t.Fatalf("Login() state = %#v, stored = %#v", state, secureStore.saved)
	}
	if remote.loginInput.Password != "correct horse battery staple" || remote.loginInput.Device.OS != "windows" {
		t.Fatalf("native login input = %#v", remote.loginInput)
	}
	secureStore.loaded = secureStore.saved
	restored, err := app.GetAuthenticationState()
	if err != nil || !restored.Authenticated || restored.UserID != "user-1" {
		t.Fatalf("GetAuthenticationState() = %#v, %v", restored, err)
	}
	if err := app.Logout(); err != nil || secureStore.deleted != "http://workshop.local" {
		t.Fatalf("Logout() error = %v, deleted = %q", err, secureStore.deleted)
	}
}

func TestAppLoadsPermissionAwareBrandedShell(t *testing.T) {
	session := credentials.Session{Token: "secure-token", ExpiresAt: time.Now().UTC().Add(time.Hour), UserID: "user-1", UserName: "Owner", EmailOrUsername: "owner", Role: "owner", Permissions: []string{"settings.manage"}, DeviceID: "device-1"}
	remote := &connectionCheckerStub{
		settings: apiclient.WorkshopSettings{WorkshopName: "Talos Lab", DefaultTheme: "system"},
		branding: apiclient.Branding{WorkshopName: "Talos Lab", LogoDataURL: "data:image/png;base64,cG5n"},
	}
	app := newApp(
		&connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://workshop.local"}},
		&sessionStoreStub{loaded: session},
		"1.2.0",
		func(string, string) (remoteClient, error) { return remote, nil },
	)
	context, err := app.LoadShell()
	if err != nil || !context.Authentication.Authenticated || context.Authentication.Permissions[0] != "settings.manage" ||
		context.WorkshopName != "Talos Lab" || context.DefaultTheme != "system" || context.LogoDataURL == "" {
		t.Fatalf("LoadShell() = %#v, %v", context, err)
	}
	if remote.settingsToken != "secure-token" {
		t.Fatalf("settings bearer token = %q", remote.settingsToken)
	}
}

func TestAppHandlesUnauthorizedAndForbiddenServerResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantError  bool
		wantDelete bool
	}{
		{name: "unauthenticated", statusCode: 401, wantDelete: true},
		{name: "forbidden", statusCode: 403, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			secureStore := &sessionStoreStub{loaded: credentials.Session{Token: "token"}}
			remote := &connectionCheckerStub{settingsError: &apiclient.ClientError{Kind: apiclient.ErrorAPI, StatusCode: test.statusCode, Code: "authorization_error", Message: "denied"}}
			app := newApp(&connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://workshop.local"}}, secureStore, "1.2.0", func(string, string) (remoteClient, error) { return remote, nil })
			context, err := app.LoadShell()
			if (err != nil) != test.wantError || context.Authentication.Authenticated || (secureStore.deleted != "") != test.wantDelete {
				t.Fatalf("LoadShell() = %#v, %v; deleted = %q", context, err, secureStore.deleted)
			}
		})
	}
}

func TestCatalogOperationsKeepBearerTokenInNativeLayer(t *testing.T) {
	page := apiclient.CatalogPage{Items: []apiclient.CatalogItem{{ID: "item-id", Name: "Cube"}}}
	remote := &connectionCheckerStub{catalogPage: page, catalogItem: apiclient.CatalogItem{ID: "item-id", Name: "Cube"}}
	secureStore := &sessionStoreStub{loaded: credentials.Session{Token: "native-only-token"}}
	app := newApp(
		&connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://workshop.local"}},
		secureStore, "1.2.0", func(string, string) (remoteClient, error) { return remote, nil },
	)
	listed, err := app.ListCatalogItems()
	if err != nil || len(listed.Items) != 1 || remote.catalogToken != "native-only-token" {
		t.Fatalf("ListCatalogItems() = %#v, %v; token = %q", listed, err, remote.catalogToken)
	}
	input := apiclient.CatalogItemInput{Name: "Cube", Purpose: "test", Status: "active"}
	if _, err := app.CreateCatalogItem(input); err != nil || remote.catalogInput.Name != "Cube" {
		t.Fatalf("CreateCatalogItem() error = %v, input = %#v", err, remote.catalogInput)
	}
	if _, err := app.UpdateCatalogItem("item-id", input); err != nil || remote.catalogID != "item-id" {
		t.Fatalf("UpdateCatalogItem() error = %v, id = %q", err, remote.catalogID)
	}
	if _, err := app.ListCatalogParts("item-id"); err != nil || remote.designToken != "native-only-token" || remote.designItemID != "item-id" {
		t.Fatalf("ListCatalogParts() error = %v, token/item = %q/%q", err, remote.designToken, remote.designItemID)
	}
	if _, err := app.CreateDesignVersion("part-id", apiclient.DesignVersionInput{Version: "v1"}); err != nil || remote.designPartID != "part-id" {
		t.Fatalf("CreateDesignVersion() error = %v, part = %q", err, remote.designPartID)
	}
	if _, err := app.AttachDesignFile("version-id", "file-id", "print"); err != nil || remote.designVersionID != "version-id" || remote.designFileID != "file-id" {
		t.Fatalf("AttachDesignFile() error = %v, version/file = %q/%q", err, remote.designVersionID, remote.designFileID)
	}
}

func TestCatalogUnauthorizedClearsRejectedSession(t *testing.T) {
	remote := &connectionCheckerStub{catalogError: &apiclient.ClientError{
		Kind: apiclient.ErrorAPI, StatusCode: 401, Code: "unauthenticated", Message: "Authentication required",
	}}
	secureStore := &sessionStoreStub{loaded: credentials.Session{Token: "expired-token"}}
	app := newApp(
		&connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://workshop.local"}},
		secureStore, "1.2.0", func(string, string) (remoteClient, error) { return remote, nil },
	)
	if _, err := app.ListCatalogItems(); err == nil || secureStore.deleted != "http://workshop.local" {
		t.Fatalf("ListCatalogItems() error = %v; deleted = %q", err, secureStore.deleted)
	}
}

type connectionStoreStub struct {
	loaded    serverconnection.Configuration
	loadError error
	saved     string
	saveError error
}

func (stub *connectionStoreStub) Load() (serverconnection.Configuration, error) {
	return stub.loaded, stub.loadError
}

func (stub *connectionStoreStub) Save(value string) (serverconnection.Configuration, error) {
	stub.saved = value
	if stub.saveError != nil {
		return serverconnection.Configuration{}, stub.saveError
	}
	normalized, err := serverconnection.NormalizeBaseURL(value)
	return serverconnection.Configuration{ServerBaseURL: normalized}, err
}

type connectionCheckerStub struct {
	result          apiclient.ConnectionResult
	err             error
	loginResult     apiclient.LoginResult
	loginError      error
	loginInput      apiclient.LoginInput
	branding        apiclient.Branding
	brandingErr     error
	settings        apiclient.WorkshopSettings
	settingsError   error
	settingsToken   string
	catalogPage     apiclient.CatalogPage
	catalogItem     apiclient.CatalogItem
	catalogError    error
	catalogToken    string
	catalogInput    apiclient.CatalogItemInput
	catalogID       string
	parts           []apiclient.CatalogPart
	part            apiclient.CatalogPart
	versions        []apiclient.DesignVersion
	version         apiclient.DesignVersion
	designFile      apiclient.DesignFile
	designToken     string
	designItemID    string
	designPartID    string
	designVersionID string
	designFileID    string
}

func (stub *connectionCheckerStub) Login(_ context.Context, input apiclient.LoginInput) (apiclient.LoginResult, error) {
	stub.loginInput = input
	return stub.loginResult, stub.loginError
}

func (stub *connectionCheckerStub) FetchBranding(context.Context) (apiclient.Branding, error) {
	return stub.branding, stub.brandingErr
}

func (stub *connectionCheckerStub) GetWorkshopSettings(_ context.Context, token string) (apiclient.WorkshopSettings, error) {
	stub.settingsToken = token
	return stub.settings, stub.settingsError
}

func (stub *connectionCheckerStub) ListCatalogItems(_ context.Context, token string) (apiclient.CatalogPage, error) {
	stub.catalogToken = token
	return stub.catalogPage, stub.catalogError
}

func (stub *connectionCheckerStub) CreateCatalogItem(_ context.Context, token string, input apiclient.CatalogItemInput) (apiclient.CatalogItem, error) {
	stub.catalogToken, stub.catalogInput = token, input
	return stub.catalogItem, stub.catalogError
}

func (stub *connectionCheckerStub) UpdateCatalogItem(_ context.Context, token, id string, input apiclient.CatalogItemInput) (apiclient.CatalogItem, error) {
	stub.catalogToken, stub.catalogID, stub.catalogInput = token, id, input
	return stub.catalogItem, stub.catalogError
}

func (stub *connectionCheckerStub) ListCatalogParts(_ context.Context, token, itemID string) ([]apiclient.CatalogPart, error) {
	stub.designToken, stub.designItemID = token, itemID
	return stub.parts, stub.catalogError
}
func (stub *connectionCheckerStub) CreateCatalogPart(_ context.Context, token, itemID string, _ apiclient.CatalogPartInput) (apiclient.CatalogPart, error) {
	stub.designToken, stub.designItemID = token, itemID
	return stub.part, stub.catalogError
}
func (stub *connectionCheckerStub) ListDesignVersions(_ context.Context, token, partID string) ([]apiclient.DesignVersion, error) {
	stub.designToken, stub.designPartID = token, partID
	return stub.versions, stub.catalogError
}
func (stub *connectionCheckerStub) CreateDesignVersion(_ context.Context, token, partID string, _ apiclient.DesignVersionInput) (apiclient.DesignVersion, error) {
	stub.designToken, stub.designPartID = token, partID
	return stub.version, stub.catalogError
}
func (stub *connectionCheckerStub) AttachDesignFile(_ context.Context, token, versionID, fileID, _ string) (apiclient.DesignFile, error) {
	stub.designToken, stub.designVersionID, stub.designFileID = token, versionID, fileID
	return stub.designFile, stub.catalogError
}

type sessionStoreStub struct {
	saved     credentials.Session
	loaded    credentials.Session
	loadError error
	deleted   string
}

func (stub *sessionStoreStub) Save(_ string, session credentials.Session) error {
	stub.saved = session
	return nil
}

func (stub *sessionStoreStub) Load(string) (credentials.Session, error) {
	return stub.loaded, stub.loadError
}

func (stub *sessionStoreStub) Delete(baseURL string) error {
	stub.deleted = baseURL
	return nil
}

func (stub *connectionCheckerStub) CheckConnection(context.Context) (apiclient.ConnectionResult, error) {
	return stub.result, stub.err
}
