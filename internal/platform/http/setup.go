package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const (
	SetupPrefix     = "/api/setup"
	SetupStatusPath = "/status"
	SetupAdminPath  = "/admin"
	setupBodyLimit  = 64 * 1024
)

// SetupService is the unauthenticated first-owner application boundary.
type SetupService interface {
	NeedsSetup(context.Context) (bool, error)
	CreateAdmin(context.Context, applicationauth.CreateAdminInput) (domainauth.User, error)
}

type setupStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
}

type createAdminRequest struct {
	Name            string `json:"name"`
	EmailOrUsername string `json:"email_or_username"`
	Password        string `json:"password"`
}

type createdAdminResponse struct {
	User createdAdmin `json:"user"`
}

type createdAdmin struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	EmailOrUsername string                  `json:"email_or_username"`
	Status          domainauth.UserStatus   `json:"status"`
	Role            domainauth.Role         `json:"role"`
	Permissions     []domainauth.Permission `json:"permissions"`
	CreatedAt       time.Time               `json:"created_at"`
}

// RegisterSetup mounts the bootstrap API at the unversioned PRD-defined path.
func RegisterSetup(mux *http.ServeMux, service SetupService) {
	registerSetup(mux, service, generateRequestID)
}

func registerSetup(mux *http.ServeMux, service SetupService, generator requestIDGenerator) {
	router := newJSONRouter()
	router.Get(SetupStatusPath, func(response http.ResponseWriter, request *http.Request) {
		needsSetup, err := service.NeedsSetup(request.Context())
		if err != nil {
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		writeJSON(response, http.StatusOK, setupStatusResponse{NeedsSetup: needsSetup})
	})
	router.Post(SetupAdminPath, func(response http.ResponseWriter, request *http.Request) {
		var body createAdminRequest
		if err := decodeSetupRequest(response, request, &body); err != nil {
			WriteError(response, http.StatusBadRequest, "invalid_request", "Invalid request", nil)
			return
		}

		user, err := service.CreateAdmin(request.Context(), applicationauth.CreateAdminInput{
			Name:            body.Name,
			EmailOrUsername: body.EmailOrUsername,
			Password:        body.Password,
		})
		switch {
		case errors.Is(err, applicationauth.ErrSetupClosed):
			WriteError(response, http.StatusConflict, "setup_closed", "Setup is no longer available", nil)
			return
		case errors.Is(err, applicationauth.ErrInvalidName):
			writeInvalidSetupField(response, "name", nil)
			return
		case errors.Is(err, applicationauth.ErrInvalidLogin):
			writeInvalidSetupField(response, "email_or_username", nil)
			return
		case errors.Is(err, applicationauth.ErrInvalidPassword):
			writeInvalidSetupField(response, "password", map[string]any{
				"minimum_length": applicationauth.MinimumPasswordLength,
				"maximum_length": applicationauth.MaximumPasswordLength,
			})
			return
		case err != nil:
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}

		writeJSON(response, http.StatusCreated, createdAdminResponse{User: createdAdmin{
			ID:              user.ID,
			Name:            user.Name,
			EmailOrUsername: user.EmailOrUsername,
			Status:          user.Status,
			Role:            user.Role,
			Permissions:     domainauth.PermissionsForRole(user.Role),
			CreatedAt:       user.CreatedAt,
		}})
	})

	handler := http.StripPrefix(SetupPrefix, requestIDMiddleware(generator)(router))
	mux.Handle(SetupPrefix, handler)
	mux.Handle(SetupPrefix+"/", handler)
}

func decodeSetupRequest(response http.ResponseWriter, request *http.Request, destination any) error {
	request.Body = http.MaxBytesReader(response, request.Body, setupBodyLimit)
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

func writeInvalidSetupField(response http.ResponseWriter, field string, extra map[string]any) {
	details := map[string]any{"field": field}
	for key, value := range extra {
		details[key] = value
	}
	WriteError(response, http.StatusBadRequest, "invalid_request", "Invalid request", details)
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}
