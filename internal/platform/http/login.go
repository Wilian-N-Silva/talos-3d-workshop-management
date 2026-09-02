package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const (
	LoginPath      = "/auth/login"
	loginBodyLimit = 64 * 1024
)

// LoginService authenticates credentials and issues an opaque session.
type LoginService interface {
	Login(context.Context, applicationauth.LoginInput) (applicationauth.LoginResult, error)
}

type loginRequest struct {
	EmailOrUsername string             `json:"email_or_username"`
	Password        string             `json:"password"`
	Device          loginDeviceRequest `json:"device"`
}

type loginDeviceRequest struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"display_name"`
	OS          string `json:"os"`
	AppVersion  string `json:"app_version"`
}

type loginResponse struct {
	Token     string              `json:"token"`
	ExpiresAt time.Time           `json:"expires_at"`
	User      loginUserResponse   `json:"user"`
	Device    loginDeviceResponse `json:"device"`
}

type loginUserResponse struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	EmailOrUsername string                  `json:"email_or_username"`
	Status          domainauth.UserStatus   `json:"status"`
	Role            domainauth.Role         `json:"role"`
	Permissions     []domainauth.Permission `json:"permissions"`
	LastLoginAt     *time.Time              `json:"last_login_at"`
}

type loginDeviceResponse struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	OS          string    `json:"os"`
	AppVersion  string    `json:"app_version"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// RegisterLogin registers the unauthenticated login endpoint on API v1.
func RegisterLogin(router *APIV1Router, service LoginService, limiter *LoginRateLimiter) {
	router.HandleFunc(http.MethodPost, LoginPath, func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		allowed, retryAfter := limiter.Allow(loginPeer(request))
		if !allowed {
			response.Header().Set("Retry-After", strconv.FormatInt(retryAfterSeconds(retryAfter), 10))
			WriteError(response, http.StatusTooManyRequests, "rate_limited", "Too many login attempts", nil)
			return
		}

		var body loginRequest
		if err := decodeLoginRequest(response, request, &body); err != nil {
			WriteError(response, http.StatusBadRequest, "invalid_request", "Invalid request", nil)
			return
		}

		result, err := service.Login(request.Context(), applicationauth.LoginInput{
			EmailOrUsername: body.EmailOrUsername,
			Password:        body.Password,
			Device: applicationauth.LoginDeviceInput{
				ID:          body.Device.ID,
				DisplayName: body.Device.DisplayName,
				OS:          body.Device.OS,
				AppVersion:  body.Device.AppVersion,
			},
		})
		switch {
		case errors.Is(err, applicationauth.ErrInvalidCredentials):
			WriteError(response, http.StatusUnauthorized, "invalid_credentials", "Invalid credentials", nil)
			return
		case errors.Is(err, applicationauth.ErrInvalidLoginDevice):
			WriteError(response, http.StatusBadRequest, "invalid_request", "Invalid request", map[string]any{"field": "device"})
			return
		case err != nil:
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}

		writeJSON(response, http.StatusOK, loginResponse{
			Token:     result.Session.Token,
			ExpiresAt: result.Session.Session.ExpiresAt,
			User: loginUserResponse{
				ID:              result.User.ID,
				Name:            result.User.Name,
				EmailOrUsername: result.User.EmailOrUsername,
				Status:          result.User.Status,
				Role:            result.User.Role,
				Permissions:     domainauth.PermissionsForRole(result.User.Role),
				LastLoginAt:     result.User.LastLoginAt,
			},
			Device: loginDeviceResponse{
				ID:          result.Device.ID,
				DisplayName: result.Device.DisplayName,
				OS:          result.Device.OS,
				AppVersion:  result.Device.AppVersion,
				LastSeenAt:  result.Device.LastSeenAt,
			},
		})
	})
}

func decodeLoginRequest(response http.ResponseWriter, request *http.Request, destination any) error {
	request.Body = http.MaxBytesReader(response, request.Body, loginBodyLimit)
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

func loginPeer(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return request.RemoteAddr
}

func retryAfterSeconds(duration time.Duration) int64 {
	seconds := int64((duration + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}
