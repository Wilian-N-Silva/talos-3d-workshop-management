package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	applicationfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/files"
	applicationsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/settings"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
	domainenergy "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/energy"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
	domaininventory "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
	domainjobs "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	domainlabor "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/labor"
	domainmaintenance "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	domainprinters "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
	httpplatform "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/platform/http"
)

type readinessStub struct {
	err error
}

var testMetadata = httpplatform.MetaResponse{
	APIVersion:            httpplatform.APIVersion,
	ServerVersion:         "test",
	WorkshopName:          "Test Workshop",
	MinimumDesktopVersion: "0.0.0",
}

var testSetupService = setupServiceStub{needsSetup: true}
var testLoginService = loginServiceStub{}
var testAuthenticationService = authenticationServiceStub{}
var testSessionManagementService = sessionManagementServiceStub{}
var testWorkshopSettingsService = workshopSettingsServiceStub{}
var testWorkshopLogoService = workshopLogoServiceStub{}
var testFileTransferService = fileTransferServiceStub{}
var testCatalogItemService = catalogItemServiceStub{}
var testCatalogDesignService = catalogDesignServiceStub{}
var testFilamentInventoryService = filamentInventoryServiceStub{}
var testSupplyInventoryService = supplyInventoryServiceStub{}
var testCatalogBOMService = catalogBOMServiceStub{}
var testPrinterService = printerServiceStub{}
var testJobService = jobServiceStub{}
var testJobMaterialUsageService = jobMaterialUsageServiceStub{}
var testEnergyService = energyServiceStub{}
var testLaborService = laborServiceStub{}
var testMaintenanceService = maintenanceServiceStub{}

func (stub readinessStub) Check(context.Context) error {
	return stub.err
}

type setupServiceStub struct {
	needsSetup bool
}

type loginServiceStub struct{}

type authenticationServiceStub struct{}

type sessionManagementServiceStub struct{}

type workshopSettingsServiceStub struct{}

type workshopLogoServiceStub struct{}
type fileTransferServiceStub struct{}
type catalogItemServiceStub struct{}
type catalogDesignServiceStub struct{}
type filamentInventoryServiceStub struct{}
type supplyInventoryServiceStub struct{}
type catalogBOMServiceStub struct{}
type printerServiceStub struct{}
type jobServiceStub struct{}
type jobMaterialUsageServiceStub struct{}
type energyServiceStub struct{}
type laborServiceStub struct{}
type maintenanceServiceStub struct{}

func (loginServiceStub) Login(
	context.Context,
	applicationauth.LoginInput,
) (applicationauth.LoginResult, error) {
	return applicationauth.LoginResult{}, nil
}

func (authenticationServiceStub) Authenticate(
	context.Context,
	string,
) (applicationauth.AuthenticationResult, error) {
	return applicationauth.AuthenticationResult{
		User: domainauth.User{
			ID:     "11111111-1111-4111-8111-111111111111",
			Status: domainauth.UserStatusActive,
			Role:   domainauth.RoleViewer,
		},
		Session: domainauth.Session{
			ID:        "33333333-3333-4333-8333-333333333333",
			DeviceID:  "44444444-4444-4444-8444-444444444444",
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}, nil
}

func (sessionManagementServiceStub) List(
	context.Context,
	applicationauth.SessionActor,
	string,
) ([]domainauth.SessionDetails, error) {
	return []domainauth.SessionDetails{}, nil
}

func (sessionManagementServiceStub) Revoke(
	context.Context,
	applicationauth.SessionActor,
	string,
) (domainauth.Session, error) {
	return domainauth.Session{}, nil
}

func (workshopSettingsServiceStub) Get(context.Context) (domainsettings.WorkshopSettings, error) {
	return domainsettings.WorkshopSettings{
		WorkshopName:    testMetadata.WorkshopName,
		DefaultLocale:   "pt-BR",
		DefaultCurrency: "BRL",
		DisplayTimezone: "America/Sao_Paulo",
		DefaultTheme:    domainsettings.ThemeSystem,
	}, nil
}

func (workshopSettingsServiceStub) Update(
	context.Context,
	domainsettings.Values,
) (domainsettings.WorkshopSettings, error) {
	return domainsettings.WorkshopSettings{}, nil
}

func (workshopLogoServiceStub) Upload(
	context.Context,
	applicationsettings.LogoUpload,
) (applicationsettings.LogoUploadResult, error) {
	return applicationsettings.LogoUploadResult{}, nil
}

func (workshopLogoServiceStub) OpenCurrent(context.Context) (applicationsettings.LogoDownload, error) {
	return applicationsettings.LogoDownload{}, domainfiles.ErrFileNotFound
}

func (stub setupServiceStub) NeedsSetup(context.Context) (bool, error) {
	return stub.needsSetup, nil
}

func (setupServiceStub) CreateAdmin(
	context.Context,
	applicationauth.CreateAdminInput,
) (domainauth.User, error) {
	return domainauth.User{}, nil
}

func TestHandlerRegistersLiveness(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{err: errors.New("dependencies unavailable")}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRegistersReadiness(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRegistersMeta(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.MetaPath, nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var got httpplatform.MetaResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if got != testMetadata {
		t.Fatalf("metadata = %#v, want %#v", got, testMetadata)
	}
}

func TestHandlerRegistersSetupStatus(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.SetupPrefix+httpplatform.SetupStatusPath, nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var got struct {
		NeedsSetup bool `json:"needs_setup"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode setup status: %v", err)
	}
	if !got.NeedsSetup {
		t.Fatal("needs_setup = false, want true")
	}
}

func TestHandlerRejectsUnknownProductRoutes(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/unknown", nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID is empty")
	}
}

func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	handler := newTestHandler(t, readinessStub{})
	go func() {
		result <- run(ctx, listener, log.New(io.Discard, "", 0), handler)
	}()

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + listener.Addr().String() + "/health/live")
	if err != nil {
		cancel()
		t.Fatalf("request liveness: %v", err)
	}
	_ = response.Body.Close()

	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after context cancellation")
	}
}

func TestHandlerRegistersLogin(t *testing.T) {
	body := `{"email_or_username":"owner","password":"a long owner passphrase","device":{"display_name":"PC","os":"Windows","app_version":"dev"}}`
	request := httptest.NewRequest(http.MethodPost, httpplatform.APIV1Prefix+httpplatform.LoginPath, strings.NewReader(body))
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRegistersSessionManagement(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.SessionsPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersWorkshopSettings(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.WorkshopSettingsPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersWorkshopLogo(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.WorkshopLogoDownloadPath, nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

func (fileTransferServiceStub) UploadFile(context.Context, applicationfiles.Upload) (applicationfiles.UploadResult, error) {
	return applicationfiles.UploadResult{}, nil
}

func (fileTransferServiceStub) OpenFile(context.Context, string) (applicationfiles.Download, error) {
	return applicationfiles.Download{}, domainfiles.ErrFileNotFound
}

func (catalogItemServiceStub) Create(context.Context, domaincatalog.Values) (domaincatalog.Item, error) {
	return domaincatalog.Item{}, nil
}

func (catalogItemServiceStub) Get(context.Context, string) (domaincatalog.Item, error) {
	return domaincatalog.Item{}, domaincatalog.ErrItemNotFound
}

func (catalogItemServiceStub) List(_ context.Context, filter domaincatalog.ListFilter) (domaincatalog.Page, error) {
	if filter.Limit == 0 {
		filter.Limit = applicationcatalog.DefaultListLimit
	}
	return domaincatalog.Page{Items: []domaincatalog.Item{}, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (catalogItemServiceStub) Update(context.Context, string, domaincatalog.Values) (domaincatalog.Item, error) {
	return domaincatalog.Item{}, nil
}

func (catalogItemServiceStub) Delete(context.Context, string) error {
	return nil
}

func (catalogDesignServiceStub) CreatePart(context.Context, string, domaincatalog.PartValues) (domaincatalog.Part, error) {
	return domaincatalog.Part{}, nil
}
func (catalogDesignServiceStub) GetPart(context.Context, string) (domaincatalog.Part, error) {
	return domaincatalog.Part{}, domaincatalog.ErrPartNotFound
}
func (catalogDesignServiceStub) ListParts(context.Context, string) ([]domaincatalog.Part, error) {
	return []domaincatalog.Part{}, nil
}
func (catalogDesignServiceStub) UpdatePart(context.Context, string, domaincatalog.PartValues) (domaincatalog.Part, error) {
	return domaincatalog.Part{}, nil
}
func (catalogDesignServiceStub) DeletePart(context.Context, string) error { return nil }
func (catalogDesignServiceStub) CreateVersion(context.Context, string, string, domaincatalog.DesignVersionValues) (domaincatalog.DesignVersion, error) {
	return domaincatalog.DesignVersion{}, nil
}
func (catalogDesignServiceStub) ListVersions(context.Context, string) ([]domaincatalog.DesignVersion, error) {
	return []domaincatalog.DesignVersion{}, nil
}
func (catalogDesignServiceStub) AttachFile(context.Context, string, string, string, domaincatalog.DesignFileRole) (domaincatalog.DesignFile, error) {
	return domaincatalog.DesignFile{}, nil
}

func (filamentInventoryServiceStub) CreateMaterial(context.Context, domaininventory.MaterialValues) (domaininventory.Material, error) {
	return domaininventory.Material{}, nil
}
func (filamentInventoryServiceStub) GetMaterial(context.Context, string) (domaininventory.Material, error) {
	return domaininventory.Material{}, domaininventory.ErrMaterialNotFound
}
func (filamentInventoryServiceStub) ListMaterials(context.Context) ([]domaininventory.Material, error) {
	return []domaininventory.Material{}, nil
}
func (filamentInventoryServiceStub) UpdateMaterial(context.Context, string, domaininventory.MaterialValues) (domaininventory.Material, error) {
	return domaininventory.Material{}, nil
}
func (filamentInventoryServiceStub) DeleteMaterial(context.Context, string) error { return nil }
func (filamentInventoryServiceStub) CreateSpool(context.Context, domaininventory.SpoolValues) (domaininventory.Spool, error) {
	return domaininventory.Spool{}, nil
}
func (filamentInventoryServiceStub) GetSpool(context.Context, string) (domaininventory.Spool, error) {
	return domaininventory.Spool{}, domaininventory.ErrSpoolNotFound
}
func (filamentInventoryServiceStub) ListSpools(context.Context) ([]domaininventory.Spool, error) {
	return []domaininventory.Spool{}, nil
}
func (filamentInventoryServiceStub) UpdateSpool(context.Context, string, domaininventory.SpoolValues) (domaininventory.Spool, error) {
	return domaininventory.Spool{}, nil
}
func (filamentInventoryServiceStub) DeleteSpool(context.Context, string) error { return nil }
func (filamentInventoryServiceStub) RecordMeasurement(context.Context, string, string, domaininventory.MeasurementValues) (domaininventory.SpoolMeasurement, error) {
	return domaininventory.SpoolMeasurement{}, nil
}
func (filamentInventoryServiceStub) ListMeasurements(context.Context, string) ([]domaininventory.SpoolMeasurement, error) {
	return []domaininventory.SpoolMeasurement{}, nil
}

func (supplyInventoryServiceStub) CreateSupply(context.Context, domaininventory.SupplyValues) (domaininventory.Supply, error) {
	return domaininventory.Supply{}, nil
}
func (supplyInventoryServiceStub) GetSupply(context.Context, string) (domaininventory.Supply, error) {
	return domaininventory.Supply{}, nil
}
func (supplyInventoryServiceStub) ListSupplies(context.Context) ([]domaininventory.Supply, error) {
	return []domaininventory.Supply{}, nil
}
func (supplyInventoryServiceStub) UpdateSupply(context.Context, string, domaininventory.SupplyValues) (domaininventory.Supply, error) {
	return domaininventory.Supply{}, nil
}
func (supplyInventoryServiceStub) DeleteSupply(context.Context, string) error { return nil }
func (supplyInventoryServiceStub) RecordMovement(context.Context, string, string, domaininventory.SupplyMovementValues) (domaininventory.SupplyMovement, error) {
	return domaininventory.SupplyMovement{}, nil
}
func (supplyInventoryServiceStub) ListMovements(context.Context, string) ([]domaininventory.SupplyMovement, error) {
	return []domaininventory.SupplyMovement{}, nil
}
func (supplyInventoryServiceStub) ListLowInventory(context.Context, string) (domaininventory.LowInventory, error) {
	return domaininventory.LowInventory{Spools: []domaininventory.Spool{}, Supplies: []domaininventory.Supply{}}, nil
}
func (catalogBOMServiceStub) Create(context.Context, string, domaincatalog.BOMValues) (domaincatalog.BOMItem, error) {
	return domaincatalog.BOMItem{}, nil
}
func (catalogBOMServiceStub) Get(context.Context, string, string) (domaincatalog.BOMItem, error) {
	return domaincatalog.BOMItem{}, nil
}
func (catalogBOMServiceStub) Preview(context.Context, string) (domaincatalog.BOMPreview, error) {
	return domaincatalog.BOMPreview{Items: []domaincatalog.BOMPreviewLine{}, ExactTotalReplacementCostCents: "0"}, nil
}
func (catalogBOMServiceStub) Update(context.Context, string, string, domaincatalog.BOMValues) (domaincatalog.BOMItem, error) {
	return domaincatalog.BOMItem{}, nil
}
func (catalogBOMServiceStub) Delete(context.Context, string, string) error { return nil }
func (printerServiceStub) Create(context.Context, domainprinters.Values) (domainprinters.Printer, error) {
	return domainprinters.Printer{}, nil
}
func (printerServiceStub) Get(context.Context, string) (domainprinters.Printer, error) {
	return domainprinters.Printer{}, nil
}
func (printerServiceStub) List(context.Context) ([]domainprinters.Printer, error) {
	return []domainprinters.Printer{}, nil
}
func (printerServiceStub) Update(context.Context, string, domainprinters.Values) (domainprinters.Printer, error) {
	return domainprinters.Printer{}, nil
}
func (printerServiceStub) Delete(context.Context, string) error { return nil }
func (jobServiceStub) Create(context.Context, domainjobs.Values, domainjobs.Actor) (domainjobs.Job, error) {
	return domainjobs.Job{}, nil
}
func (jobServiceStub) Get(context.Context, string) (domainjobs.Job, error) {
	return domainjobs.Job{}, nil
}
func (jobServiceStub) List(context.Context) ([]domainjobs.Job, error) { return []domainjobs.Job{}, nil }
func (jobServiceStub) Update(context.Context, string, domainjobs.Values) (domainjobs.Job, error) {
	return domainjobs.Job{}, nil
}
func (jobServiceStub) Transition(context.Context, string, domainjobs.TransitionValues, domainjobs.Actor) (domainjobs.Job, error) {
	return domainjobs.Job{}, nil
}
func (jobServiceStub) Review(context.Context, string, domainjobs.ReviewValues, domainjobs.Actor) (domainjobs.Job, error) {
	return domainjobs.Job{}, nil
}
func (jobServiceStub) Delete(context.Context, string) error { return nil }
func (jobServiceStub) ListEvents(context.Context, string) ([]domainjobs.Event, error) {
	return []domainjobs.Event{}, nil
}

func (jobMaterialUsageServiceStub) Create(context.Context, string, domainjobs.MaterialUsageValues) (domainjobs.MaterialUsage, error) {
	return domainjobs.MaterialUsage{}, nil
}
func (jobMaterialUsageServiceStub) List(context.Context, string) (domainjobs.MaterialUsageSummary, error) {
	return domainjobs.MaterialUsageSummary{Items: []domainjobs.MaterialUsage{}, TotalPlannedGrams: "0", TotalActualGrams: "0"}, nil
}
func (jobMaterialUsageServiceStub) Update(context.Context, string, string, domainjobs.MaterialUsageValues) (domainjobs.MaterialUsage, error) {
	return domainjobs.MaterialUsage{}, nil
}
func (jobMaterialUsageServiceStub) Delete(context.Context, string, string) error { return nil }
func (energyServiceStub) Create(context.Context, string, string, domainenergy.Values) (domainenergy.Measurement, error) {
	return domainenergy.Measurement{}, nil
}
func (energyServiceStub) List(context.Context, string) ([]domainenergy.Measurement, error) {
	return []domainenergy.Measurement{}, nil
}
func (laborServiceStub) CreateRate(context.Context, domainlabor.RateValues) (domainlabor.Rate, error) {
	return domainlabor.Rate{}, nil
}
func (laborServiceStub) ListRates(context.Context) ([]domainlabor.Rate, error) {
	return []domainlabor.Rate{}, nil
}
func (laborServiceStub) UpdateRate(context.Context, string, domainlabor.RateValues) (domainlabor.Rate, error) {
	return domainlabor.Rate{}, nil
}
func (laborServiceStub) CreateEntry(context.Context, string, string, domainlabor.EntryValues) (domainlabor.Entry, error) {
	return domainlabor.Entry{}, nil
}
func (laborServiceStub) ListEntries(context.Context, string) (domainlabor.Summary, error) {
	return domainlabor.Summary{Items: []domainlabor.Entry{}, MinutesByActivity: map[domainlabor.ActivityType]int64{}}, nil
}
func (maintenanceServiceStub) Create(context.Context, string, string, domainmaintenance.Values) (domainmaintenance.Event, error) {
	return domainmaintenance.Event{}, nil
}
func (maintenanceServiceStub) List(context.Context, string) ([]domainmaintenance.Event, error) {
	return []domainmaintenance.Event{}, nil
}

func TestHandlerRegistersFiles(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.FilesPath+"/11111111-1111-4111-8111-111111111111", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

func TestHandlerRegistersCatalogItems(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.CatalogItemsPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersFilamentInventory(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.SpoolsPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersSupplyInventory(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.SuppliesPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersCatalogBOM(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+"/catalog/items/item-id/bom", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersPrinters(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.PrintersPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersJobs(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.JobsPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersJobMaterialUsage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.JobsPath+"/job-id/materials", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersEnergy(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.JobsPath+"/job-id/energy", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func newTestHandler(t *testing.T, readiness httpplatform.ReadinessChecker) http.Handler {
	t.Helper()
	limiter, err := httpplatform.NewLoginRateLimiter(100, time.Minute)
	if err != nil {
		t.Fatalf("NewLoginRateLimiter() error = %v", err)
	}
	return newHandler(
		readiness,
		testMetadata,
		testSetupService,
		testLoginService,
		limiter,
		testAuthenticationService,
		testSessionManagementService,
		testWorkshopSettingsService,
		testWorkshopLogoService,
		applicationsettings.DefaultMaximumLogoBytes,
		testFileTransferService,
		1024,
		testCatalogItemService,
		testCatalogDesignService,
		testFilamentInventoryService,
		testSupplyInventoryService,
		testCatalogBOMService,
		testPrinterService,
		testJobService,
		testJobMaterialUsageService,
		testEnergyService,
		testLaborService,
		testMaintenanceService,
	)
}
