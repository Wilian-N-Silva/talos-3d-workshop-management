package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	applicationsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/settings"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

func TestWorkshopSettingsReadAllowsAuthenticatedUser(t *testing.T) {
	updatedAt := time.Date(2026, time.September, 2, 14, 0, 0, 0, time.UTC)
	logoFileID := "logo-file-id"
	service := &workshopSettingsServiceStub{result: domainsettings.WorkshopSettings{
		WorkshopName:    "Prototype Lab",
		LogoFileID:      &logoFileID,
		DefaultLocale:   "pt-BR",
		DefaultCurrency: "BRL",
		DisplayTimezone: "America/Sao_Paulo",
		DefaultTheme:    domainsettings.ThemeSystem,
		UpdatedAt:       updatedAt,
	}}
	authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleViewer)
	response := serveWorkshopSettingsRequest(t, http.MethodGet, APIV1Prefix+WorkshopSettingsPath, "", authentication, service)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if service.getCalls != 1 || service.updateCalls != 0 {
		t.Fatalf("service calls = get %d, update %d", service.getCalls, service.updateCalls)
	}
	var got workshopSettingsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.WorkshopName != "Prototype Lab" || got.DefaultTheme != domainsettings.ThemeSystem || !got.UpdatedAt.Equal(updatedAt) ||
		got.LogoURL == nil || *got.LogoURL != APIV1Prefix+WorkshopLogoDownloadPath {
		t.Fatalf("settings response = %#v", got)
	}
}

func TestWorkshopSettingsUpdateRequiresSettingsManage(t *testing.T) {
	body := `{"workshop_name":"Design Studio","default_locale":"en-US","default_currency":"USD","display_timezone":"UTC","default_theme":"dark"}`

	t.Run("owner", func(t *testing.T) {
		service := &workshopSettingsServiceStub{result: domainsettings.WorkshopSettings{WorkshopName: "Design Studio"}}
		authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleOwner)
		response := serveWorkshopSettingsRequest(t, http.MethodPut, APIV1Prefix+WorkshopSettingsPath, body, authentication, service)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
		}
		if service.updateCalls != 1 || service.updateInput.DefaultTheme != domainsettings.ThemeDark || service.updateInput.DisplayTimezone != "UTC" {
			t.Fatalf("update input = %#v, calls = %d", service.updateInput, service.updateCalls)
		}
	})

	t.Run("viewer", func(t *testing.T) {
		service := &workshopSettingsServiceStub{}
		authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleViewer)
		response := serveWorkshopSettingsRequest(t, http.MethodPut, APIV1Prefix+WorkshopSettingsPath, body, authentication, service)
		assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
		if service.updateCalls != 0 {
			t.Fatalf("update calls = %d, want 0", service.updateCalls)
		}
	})
}

func TestWorkshopSettingsEndpointsRequireAuthentication(t *testing.T) {
	authentication := &bearerAuthenticationServiceStub{err: applicationauth.ErrUnauthenticated}
	service := &workshopSettingsServiceStub{}
	response := serveWorkshopSettingsRequest(t, http.MethodGet, APIV1Prefix+WorkshopSettingsPath, "", authentication, service)
	assertUnauthenticatedResponse(t, response)
	if service.getCalls != 0 {
		t.Fatalf("get calls = %d, want 0", service.getCalls)
	}
}

func TestWorkshopSettingsUpdateRejectsMalformedAndInvalidValues(t *testing.T) {
	authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleOwner)

	t.Run("malformed", func(t *testing.T) {
		service := &workshopSettingsServiceStub{}
		response := serveWorkshopSettingsRequest(t, http.MethodPut, APIV1Prefix+WorkshopSettingsPath, `{"unknown":true}`, authentication, service)
		assertAPIError(t, response, http.StatusBadRequest, "invalid_request", "Invalid request")
		if service.updateCalls != 0 {
			t.Fatalf("update calls = %d, want 0", service.updateCalls)
		}
	})

	t.Run("invalid values", func(t *testing.T) {
		service := &workshopSettingsServiceStub{updateError: applicationsettings.ErrInvalidWorkshopSettings}
		body := `{"workshop_name":"","default_locale":"pt-BR","default_currency":"BRL","display_timezone":"UTC","default_theme":"system"}`
		response := serveWorkshopSettingsRequest(t, http.MethodPut, APIV1Prefix+WorkshopSettingsPath, body, authentication, service)
		assertAPIError(t, response, http.StatusBadRequest, "invalid_settings", "Invalid workshop settings")
	})
}

func TestWorkshopSettingsDoesNotExposeDependencyErrors(t *testing.T) {
	authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleViewer)
	service := &workshopSettingsServiceStub{getError: errors.New("database details")}
	response := serveWorkshopSettingsRequest(t, http.MethodGet, APIV1Prefix+WorkshopSettingsPath, "", authentication, service)
	assertAPIError(t, response, http.StatusInternalServerError, "internal_error", "Internal server error")
	if strings.Contains(response.Body.String(), "database details") {
		t.Fatal("response exposed dependency error")
	}
}

func TestWorkshopSettingsUpdateIsReflectedByMeta(t *testing.T) {
	service := &workshopSettingsServiceStub{result: domainsettings.WorkshopSettings{WorkshopName: "Old Name"}}
	authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleOwner)
	router := NewAPIV1Router()
	RegisterWorkshopSettings(router, authentication, service)
	RegisterMeta(router, MetaResponse{ServerVersion: "test"}, service)
	mux := http.NewServeMux()
	RegisterAPIV1(mux, router)

	body := `{"workshop_name":"New Name","default_locale":"pt-BR","default_currency":"BRL","display_timezone":"America/Sao_Paulo","default_theme":"system"}`
	updateRequest := httptest.NewRequest(http.MethodPut, APIV1Prefix+WorkshopSettingsPath, strings.NewReader(body))
	updateRequest.Header.Set("Authorization", "Bearer test-token")
	updateResponse := httptest.NewRecorder()
	mux.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	metaResponse := httptest.NewRecorder()
	mux.ServeHTTP(metaResponse, httptest.NewRequest(http.MethodGet, APIV1Prefix+MetaPath, nil))
	var metadata MetaResponse
	if err := json.Unmarshal(metaResponse.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if metadata.WorkshopName != "New Name" {
		t.Fatalf("meta workshop name = %q, want New Name", metadata.WorkshopName)
	}
}

func serveWorkshopSettingsRequest(
	t *testing.T,
	method string,
	path string,
	body string,
	authentication BearerAuthenticationService,
	service WorkshopSettingsService,
) *httptest.ResponseRecorder {
	t.Helper()
	router := NewAPIV1Router()
	RegisterWorkshopSettings(router, authentication, service)
	mux := http.NewServeMux()
	RegisterAPIV1(mux, router)
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

type workshopSettingsServiceStub struct {
	result      domainsettings.WorkshopSettings
	getError    error
	getCalls    int
	updateInput domainsettings.Values
	updateError error
	updateCalls int
}

func (stub *workshopSettingsServiceStub) Get(context.Context) (domainsettings.WorkshopSettings, error) {
	stub.getCalls++
	return stub.result, stub.getError
}

func (stub *workshopSettingsServiceStub) Update(
	_ context.Context,
	input domainsettings.Values,
) (domainsettings.WorkshopSettings, error) {
	stub.updateCalls++
	stub.updateInput = input
	if stub.updateError == nil {
		stub.result.WorkshopName = input.WorkshopName
		stub.result.DefaultLocale = input.DefaultLocale
		stub.result.DefaultCurrency = input.DefaultCurrency
		stub.result.DisplayTimezone = input.DisplayTimezone
		stub.result.DefaultTheme = input.DefaultTheme
	}
	return stub.result, stub.updateError
}
