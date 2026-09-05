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
	if _, err := app.GetCatalogBOM("item-id"); err != nil || remote.bomCatalogID != "item-id" {
		t.Fatalf("GetCatalogBOM() error = %v, catalog = %q", err, remote.bomCatalogID)
	}
	bomInput := apiclient.CatalogBOMInput{SupplyID: "supply-id", QuantityPerUnit: "2", WastePercent: "5"}
	if _, err := app.CreateCatalogBOMItem("item-id", bomInput); err != nil || remote.bomInput.SupplyID != "supply-id" {
		t.Fatalf("CreateCatalogBOMItem() error = %v, input = %#v", err, remote.bomInput)
	}
	if _, err := app.UpdateCatalogBOMItem("item-id", "bom-id", bomInput); err != nil || remote.bomItemID != "bom-id" {
		t.Fatalf("UpdateCatalogBOMItem() error = %v, item = %q", err, remote.bomItemID)
	}
	if err := app.DeleteCatalogBOMItem("item-id", "bom-id"); err != nil || remote.bomItemID != "bom-id" {
		t.Fatalf("DeleteCatalogBOMItem() error = %v, item = %q", err, remote.bomItemID)
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

func TestInventoryOperationsKeepBearerTokenNative(t *testing.T) {
	remote := &connectionCheckerStub{spools: []apiclient.Spool{{ID: "spool-id"}}, supplies: []apiclient.Supply{{ID: "supply-id"}}}
	app := newApp(&connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://workshop.local"}}, &sessionStoreStub{loaded: credentials.Session{Token: "native-inventory-token"}}, "1.2.0", func(string, string) (remoteClient, error) { return remote, nil })
	if _, err := app.ListSpools(); err != nil || remote.inventoryToken != "native-inventory-token" {
		t.Fatalf("ListSpools() error=%v token=%q", err, remote.inventoryToken)
	}
	if _, err := app.ListSpoolMeasurements("spool-id"); err != nil || remote.inventorySpoolID != "spool-id" {
		t.Fatalf("ListSpoolMeasurements() error=%v spool=%q", err, remote.inventorySpoolID)
	}
	if _, err := app.RecordSpoolMeasurement("spool-id", apiclient.MeasurementInput{GrossWeightG: "845.5"}); err != nil || remote.inventorySpoolID != "spool-id" {
		t.Fatalf("RecordSpoolMeasurement() error=%v spool=%q", err, remote.inventorySpoolID)
	}
	if _, err := app.ListSupplies(); err != nil || remote.inventoryToken != "native-inventory-token" {
		t.Fatalf("ListSupplies() error=%v token=%q", err, remote.inventoryToken)
	}
	if _, err := app.CreateSupply(apiclient.SupplyInput{Name: "NFC"}); err != nil || remote.supplyInput.Name != "NFC" {
		t.Fatalf("CreateSupply() error=%v input=%#v", err, remote.supplyInput)
	}
	if _, err := app.RecordSupplyMovement("supply-id", apiclient.SupplyMovementInput{Quantity: "10"}); err != nil || remote.inventorySupplyID != "supply-id" {
		t.Fatalf("RecordSupplyMovement() error=%v supply=%q", err, remote.inventorySupplyID)
	}
	if _, err := app.ListLowInventory("75"); err != nil || remote.lowThreshold != "75" {
		t.Fatalf("ListLowInventory() error=%v threshold=%q", err, remote.lowThreshold)
	}
}

func TestJobMaterialUsageOperationsKeepBearerTokenNative(t *testing.T) {
	remote := &connectionCheckerStub{jobs: []apiclient.Job{{ID: "job-id"}}, jobUsage: apiclient.JobMaterialUsageSummary{Items: []apiclient.JobMaterialUsage{}, TotalPlannedGrams: "0", TotalActualGrams: "0"}}
	app := newApp(&connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://workshop.local"}}, &sessionStoreStub{loaded: credentials.Session{Token: "native-jobs-token"}}, "1.2.0", func(string, string) (remoteClient, error) { return remote, nil })
	if _, err := app.ListJobs(); err != nil || remote.jobToken != "native-jobs-token" {
		t.Fatalf("ListJobs() error=%v token=%q", err, remote.jobToken)
	}
	if _, err := app.ListJobMaterialUsage("job-id"); err != nil || remote.jobID != "job-id" {
		t.Fatalf("ListJobMaterialUsage() error=%v job=%q", err, remote.jobID)
	}
	input := apiclient.JobMaterialUsageInput{SpoolID: "spool-id", PlannedGrams: "8", MeasurementSource: "slicer"}
	if _, err := app.CreateJobMaterialUsage("job-id", input); err != nil || remote.jobUsageInput.PlannedGrams != "8" {
		t.Fatalf("CreateJobMaterialUsage() error=%v input=%#v", err, remote.jobUsageInput)
	}
	if _, err := app.UpdateJobMaterialUsage("job-id", "usage-id", input); err != nil || remote.jobUsageID != "usage-id" {
		t.Fatalf("UpdateJobMaterialUsage() error=%v usage=%q", err, remote.jobUsageID)
	}
	if err := app.DeleteJobMaterialUsage("job-id", "usage-id"); err != nil || remote.jobUsageID != "usage-id" {
		t.Fatalf("DeleteJobMaterialUsage() error=%v usage=%q", err, remote.jobUsageID)
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
	result            apiclient.ConnectionResult
	err               error
	loginResult       apiclient.LoginResult
	loginError        error
	loginInput        apiclient.LoginInput
	branding          apiclient.Branding
	brandingErr       error
	settings          apiclient.WorkshopSettings
	settingsError     error
	settingsToken     string
	catalogPage       apiclient.CatalogPage
	catalogItem       apiclient.CatalogItem
	catalogError      error
	catalogToken      string
	catalogInput      apiclient.CatalogItemInput
	catalogID         string
	parts             []apiclient.CatalogPart
	part              apiclient.CatalogPart
	versions          []apiclient.DesignVersion
	version           apiclient.DesignVersion
	designFile        apiclient.DesignFile
	designToken       string
	designItemID      string
	designPartID      string
	designVersionID   string
	designFileID      string
	materials         []apiclient.Material
	spools            []apiclient.Spool
	measurements      []apiclient.SpoolMeasurement
	inventoryToken    string
	inventorySpoolID  string
	supplies          []apiclient.Supply
	supplyInput       apiclient.SupplyInput
	inventorySupplyID string
	lowThreshold      string
	bom               apiclient.CatalogBOMPreview
	bomCatalogID      string
	bomItemID         string
	bomInput          apiclient.CatalogBOMInput
	jobs              []apiclient.Job
	jobUsage          apiclient.JobMaterialUsageSummary
	jobToken          string
	jobID             string
	jobUsageID        string
	jobUsageInput     apiclient.JobMaterialUsageInput
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
func (stub *connectionCheckerStub) GetCatalogBOM(_ context.Context, token, itemID string) (apiclient.CatalogBOMPreview, error) {
	stub.catalogToken, stub.bomCatalogID = token, itemID
	return stub.bom, stub.catalogError
}
func (stub *connectionCheckerStub) CreateCatalogBOMItem(_ context.Context, token, itemID string, input apiclient.CatalogBOMInput) (apiclient.CatalogBOMItem, error) {
	stub.catalogToken, stub.bomCatalogID, stub.bomInput = token, itemID, input
	return apiclient.CatalogBOMItem{SupplyID: input.SupplyID}, stub.catalogError
}
func (stub *connectionCheckerStub) UpdateCatalogBOMItem(_ context.Context, token, itemID, bomItemID string, input apiclient.CatalogBOMInput) (apiclient.CatalogBOMItem, error) {
	stub.catalogToken, stub.bomCatalogID, stub.bomItemID, stub.bomInput = token, itemID, bomItemID, input
	return apiclient.CatalogBOMItem{ID: bomItemID}, stub.catalogError
}
func (stub *connectionCheckerStub) DeleteCatalogBOMItem(_ context.Context, token, itemID, bomItemID string) error {
	stub.catalogToken, stub.bomCatalogID, stub.bomItemID = token, itemID, bomItemID
	return stub.catalogError
}

func (stub *connectionCheckerStub) ListMaterials(_ context.Context, token string) ([]apiclient.Material, error) {
	stub.inventoryToken = token
	return stub.materials, stub.catalogError
}
func (stub *connectionCheckerStub) ListSpools(_ context.Context, token string) ([]apiclient.Spool, error) {
	stub.inventoryToken = token
	return stub.spools, stub.catalogError
}
func (stub *connectionCheckerStub) ListSpoolMeasurements(_ context.Context, token, spoolID string) ([]apiclient.SpoolMeasurement, error) {
	stub.inventoryToken, stub.inventorySpoolID = token, spoolID
	return stub.measurements, stub.catalogError
}
func (stub *connectionCheckerStub) RecordSpoolMeasurement(_ context.Context, token, spoolID string, _ apiclient.MeasurementInput) (apiclient.SpoolMeasurement, error) {
	stub.inventoryToken, stub.inventorySpoolID = token, spoolID
	return apiclient.SpoolMeasurement{}, stub.catalogError
}
func (stub *connectionCheckerStub) ListSupplies(_ context.Context, token string) ([]apiclient.Supply, error) {
	stub.inventoryToken = token
	return stub.supplies, stub.catalogError
}
func (stub *connectionCheckerStub) CreateSupply(_ context.Context, token string, input apiclient.SupplyInput) (apiclient.Supply, error) {
	stub.inventoryToken, stub.supplyInput = token, input
	return apiclient.Supply{Name: input.Name}, stub.catalogError
}
func (stub *connectionCheckerStub) ListSupplyMovements(_ context.Context, token, supplyID string) ([]apiclient.SupplyMovement, error) {
	stub.inventoryToken, stub.inventorySupplyID = token, supplyID
	return []apiclient.SupplyMovement{}, stub.catalogError
}
func (stub *connectionCheckerStub) RecordSupplyMovement(_ context.Context, token, supplyID string, _ apiclient.SupplyMovementInput) (apiclient.SupplyMovement, error) {
	stub.inventoryToken, stub.inventorySupplyID = token, supplyID
	return apiclient.SupplyMovement{}, stub.catalogError
}
func (stub *connectionCheckerStub) ListLowInventory(_ context.Context, token, threshold string) (apiclient.LowInventory, error) {
	stub.inventoryToken, stub.lowThreshold = token, threshold
	return apiclient.LowInventory{Spools: []apiclient.Spool{}, Supplies: []apiclient.Supply{}}, stub.catalogError
}

func (stub *connectionCheckerStub) ListJobs(_ context.Context, token string) ([]apiclient.Job, error) {
	stub.jobToken = token
	return stub.jobs, stub.catalogError
}
func (stub *connectionCheckerStub) ListJobMaterialUsage(_ context.Context, token, jobID string) (apiclient.JobMaterialUsageSummary, error) {
	stub.jobToken, stub.jobID = token, jobID
	return stub.jobUsage, stub.catalogError
}
func (stub *connectionCheckerStub) CreateJobMaterialUsage(_ context.Context, token, jobID string, input apiclient.JobMaterialUsageInput) (apiclient.JobMaterialUsage, error) {
	stub.jobToken, stub.jobID, stub.jobUsageInput = token, jobID, input
	return apiclient.JobMaterialUsage{SpoolID: input.SpoolID}, stub.catalogError
}
func (stub *connectionCheckerStub) UpdateJobMaterialUsage(_ context.Context, token, jobID, usageID string, input apiclient.JobMaterialUsageInput) (apiclient.JobMaterialUsage, error) {
	stub.jobToken, stub.jobID, stub.jobUsageID, stub.jobUsageInput = token, jobID, usageID, input
	return apiclient.JobMaterialUsage{ID: usageID}, stub.catalogError
}
func (stub *connectionCheckerStub) DeleteJobMaterialUsage(_ context.Context, token, jobID, usageID string) error {
	stub.jobToken, stub.jobID, stub.jobUsageID = token, jobID, usageID
	return stub.catalogError
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

func (s *connectionCheckerStub) ListLaborRates(context.Context, string) ([]apiclient.LaborRate, error) {
	return []apiclient.LaborRate{}, nil
}
func (s *connectionCheckerStub) SaveLaborRate(_ context.Context, token, id string, input apiclient.LaborRateInput) (apiclient.LaborRate, error) {
	s.jobToken = token
	return apiclient.LaborRate{ID: id, CostHourlyRateCents: input.CostHourlyRateCents}, nil
}
func (s *connectionCheckerStub) SuggestLaborRate(_ context.Context, token string, input apiclient.LaborAssumptions) (apiclient.LaborSuggestion, error) {
	s.jobToken = token
	return apiclient.LaborSuggestion{InternalHourlyCostCents: input.TargetMonthlyCompensationCents}, nil
}
func TestLaborOperationsKeepExactMoneyAndBearerNative(t *testing.T) {
	remote := &connectionCheckerStub{}
	app := newApp(&connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://workshop.local"}}, &sessionStoreStub{loaded: credentials.Session{Token: "native-labor-token"}}, "1.2.0", func(string, string) (remoteClient, error) { return remote, nil })
	suggestion, err := app.SuggestLaborRate(apiclient.LaborAssumptions{TargetMonthlyCompensationCents: "9223372036854775807"})
	if err != nil || suggestion.InternalHourlyCostCents != "9223372036854775807" || remote.jobToken != "native-labor-token" {
		t.Fatal(suggestion, err)
	}
	rate, err := app.SaveLaborRate("rate-id", apiclient.LaborRateInput{CostHourlyRateCents: "2918"})
	if err != nil || rate.CostHourlyRateCents != "2918" || rate.ID != "rate-id" || remote.jobToken != "native-labor-token" {
		t.Fatal(rate, err)
	}
}
