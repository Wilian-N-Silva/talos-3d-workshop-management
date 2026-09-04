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
		User:   apiclient.LoginUser{ID: "user-1", Name: "Workshop Owner", EmailOrUsername: "owner"},
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
	result      apiclient.ConnectionResult
	err         error
	loginResult apiclient.LoginResult
	loginError  error
	loginInput  apiclient.LoginInput
}

func (stub *connectionCheckerStub) Login(_ context.Context, input apiclient.LoginInput) (apiclient.LoginResult, error) {
	stub.loginInput = input
	return stub.loginResult, stub.loginError
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
