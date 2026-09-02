package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	applicationsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/settings"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

const (
	WorkshopSettingsPath = "/settings"
	settingsBodyLimit    = 64 * 1024
)

// WorkshopSettingsReader returns the current durable workshop configuration.
type WorkshopSettingsReader interface {
	Get(context.Context) (domainsettings.WorkshopSettings, error)
}

// WorkshopSettingsService reads and updates workshop configuration.
type WorkshopSettingsService interface {
	WorkshopSettingsReader
	Update(context.Context, domainsettings.Values) (domainsettings.WorkshopSettings, error)
}

type workshopSettingsRequest struct {
	WorkshopName    string               `json:"workshop_name"`
	DefaultLocale   string               `json:"default_locale"`
	DefaultCurrency string               `json:"default_currency"`
	DisplayTimezone string               `json:"display_timezone"`
	DefaultTheme    domainsettings.Theme `json:"default_theme"`
}

type workshopSettingsResponse struct {
	WorkshopName    string               `json:"workshop_name"`
	LogoFileID      *string              `json:"logo_file_id"`
	LogoURL         *string              `json:"logo_url"`
	DefaultLocale   string               `json:"default_locale"`
	DefaultCurrency string               `json:"default_currency"`
	DisplayTimezone string               `json:"display_timezone"`
	DefaultTheme    domainsettings.Theme `json:"default_theme"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// RegisterWorkshopSettings registers authenticated read and settings.manage update routes.
func RegisterWorkshopSettings(
	router *APIV1Router,
	authentication BearerAuthenticationService,
	service WorkshopSettingsService,
) {
	router.Handle(http.MethodGet, WorkshopSettingsPath, AuthenticationMiddleware(authentication)(http.HandlerFunc(
		func(response http.ResponseWriter, request *http.Request) {
			response.Header().Set("Cache-Control", "no-store")
			settings, err := service.Get(request.Context())
			if err != nil {
				WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
				return
			}
			writeJSON(response, http.StatusOK, newWorkshopSettingsResponse(settings))
		},
	)))

	router.Handle(http.MethodPut, WorkshopSettingsPath, RequirePermission(
		authentication,
		domainauth.PermissionSettingsManage,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		var body workshopSettingsRequest
		if err := decodeWorkshopSettingsRequest(response, request, &body); err != nil {
			WriteError(response, http.StatusBadRequest, "invalid_request", "Invalid request", nil)
			return
		}
		settings, err := service.Update(request.Context(), domainsettings.Values{
			WorkshopName:    body.WorkshopName,
			DefaultLocale:   body.DefaultLocale,
			DefaultCurrency: body.DefaultCurrency,
			DisplayTimezone: body.DisplayTimezone,
			DefaultTheme:    body.DefaultTheme,
		})
		switch {
		case errors.Is(err, applicationsettings.ErrInvalidWorkshopSettings):
			WriteError(response, http.StatusBadRequest, "invalid_settings", "Invalid workshop settings", nil)
			return
		case err != nil:
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		writeJSON(response, http.StatusOK, newWorkshopSettingsResponse(settings))
	})))
}

func newWorkshopSettingsResponse(settings domainsettings.WorkshopSettings) workshopSettingsResponse {
	return workshopSettingsResponse{
		WorkshopName:    settings.WorkshopName,
		LogoFileID:      settings.LogoFileID,
		LogoURL:         workshopLogoURL(settings.LogoFileID),
		DefaultLocale:   settings.DefaultLocale,
		DefaultCurrency: settings.DefaultCurrency,
		DisplayTimezone: settings.DisplayTimezone,
		DefaultTheme:    settings.DefaultTheme,
		UpdatedAt:       settings.UpdatedAt,
	}
}

func workshopLogoURL(logoFileID *string) *string {
	if logoFileID == nil {
		return nil
	}
	value := APIV1Prefix + WorkshopLogoDownloadPath
	return &value
}

func decodeWorkshopSettingsRequest(
	response http.ResponseWriter,
	request *http.Request,
	destination any,
) error {
	request.Body = http.MaxBytesReader(response, request.Body, settingsBodyLimit)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}
