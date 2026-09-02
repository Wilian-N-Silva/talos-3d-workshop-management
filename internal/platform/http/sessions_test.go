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
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const (
	httpTestUserID    = "11111111-1111-4111-8111-111111111111"
	httpTestSessionID = "33333333-3333-4333-8333-333333333333"
	httpTestDeviceID  = "44444444-4444-4444-8444-444444444444"
)

func TestSessionListReturnsSafeAuditMetadata(t *testing.T) {
	now := time.Date(2026, time.September, 2, 14, 0, 0, 0, time.UTC)
	authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleViewer)
	service := &sessionManagementServiceStub{listResult: []domainauth.SessionDetails{{
		Session: domainauth.Session{
			ID:        httpTestSessionID,
			UserID:    httpTestUserID,
			DeviceID:  httpTestDeviceID,
			TokenHash: []byte("must-not-be-returned"),
			CreatedAt: now.Add(-time.Hour),
			ExpiresAt: now.Add(24 * time.Hour),
		},
		Device: domainauth.ClientDevice{
			ID:          httpTestDeviceID,
			DisplayName: "Workshop PC",
			OS:          "Windows 11",
			AppVersion:  "1.0.0",
			LastSeenAt:  now,
		},
	}}}

	response := serveSessionRequest(t, http.MethodGet, APIV1Prefix+SessionsPath, authentication, service)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if strings.Contains(response.Body.String(), "must-not-be-returned") || strings.Contains(response.Body.String(), "token_hash") {
		t.Fatal("session response exposed token hash material")
	}
	var got sessionListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Sessions) != 1 || !got.Sessions[0].Current || got.Sessions[0].Device.DisplayName != "Workshop PC" {
		t.Fatalf("sessions = %#v", got.Sessions)
	}
	if service.actor.UserID != httpTestUserID || service.actor.Role != domainauth.RoleViewer || service.targetUserID != "" {
		t.Fatalf("service input = %#v, %q", service.actor, service.targetUserID)
	}
}

func TestSessionListPassesExplicitAdminTarget(t *testing.T) {
	targetUserID := "22222222-2222-4222-8222-222222222222"
	authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleOwner)
	service := &sessionManagementServiceStub{}
	response := serveSessionRequest(t, http.MethodGet, APIV1Prefix+SessionsPath+"?user_id="+targetUserID, authentication, service)

	if response.Code != http.StatusOK || service.targetUserID != targetUserID || service.actor.Role != domainauth.RoleOwner {
		t.Fatalf("status = %d, target = %q, actor = %#v", response.Code, service.targetUserID, service.actor)
	}
}

func TestSessionRevokeUsesRouteSessionAndReturnsNoContent(t *testing.T) {
	authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleViewer)
	service := &sessionManagementServiceStub{}
	response := serveSessionRequest(t, http.MethodPost, APIV1Prefix+SessionsPath+"/"+httpTestSessionID+"/revoke", authentication, service)

	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	if service.revokeID != httpTestSessionID || service.actor.UserID != httpTestUserID {
		t.Fatalf("service input = %q, %#v", service.revokeID, service.actor)
	}
}

func TestSessionEndpointsRequireAuthentication(t *testing.T) {
	authentication := &bearerAuthenticationServiceStub{err: applicationauth.ErrUnauthenticated}
	service := &sessionManagementServiceStub{}
	response := serveSessionRequest(t, http.MethodGet, APIV1Prefix+SessionsPath, authentication, service)

	assertUnauthenticatedResponse(t, response)
	if service.listCalls != 0 {
		t.Fatalf("list calls = %d, want 0", service.listCalls)
	}
}

func TestSessionEndpointsMapManagementErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid", err: applicationauth.ErrInvalidSessionManagementInput, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "forbidden", err: applicationauth.ErrSessionAccessDenied, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "missing", err: domainauth.ErrSessionNotFound, wantStatus: http.StatusNotFound, wantCode: "session_not_found"},
		{name: "internal", err: errors.New("database details"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleViewer)
			service := &sessionManagementServiceStub{revokeError: test.err}
			response := serveSessionRequest(t, http.MethodPost, APIV1Prefix+SessionsPath+"/"+httpTestSessionID+"/revoke", authentication, service)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			var envelope ErrorEnvelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if envelope.Error.Code != test.wantCode || strings.Contains(response.Body.String(), "database details") {
				t.Fatalf("error response = %s", response.Body.String())
			}
		})
	}
}

func authenticatedSessionStub(userID, sessionID string, role domainauth.Role) *bearerAuthenticationServiceStub {
	return &bearerAuthenticationServiceStub{result: applicationauth.AuthenticationResult{
		User: domainauth.User{ID: userID, Status: domainauth.UserStatusActive, Role: role},
		Session: domainauth.Session{
			ID:        sessionID,
			UserID:    userID,
			DeviceID:  httpTestDeviceID,
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}}
}

func serveSessionRequest(
	t *testing.T,
	method string,
	path string,
	authentication BearerAuthenticationService,
	sessions SessionManagementService,
) *httptest.ResponseRecorder {
	t.Helper()
	router := NewAPIV1Router()
	RegisterSessionManagement(router, authentication, sessions)
	mux := http.NewServeMux()
	RegisterAPIV1(mux, router)
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

type sessionManagementServiceStub struct {
	listResult   []domainauth.SessionDetails
	listError    error
	actor        applicationauth.SessionActor
	targetUserID string
	listCalls    int
	revokeResult domainauth.Session
	revokeError  error
	revokeID     string
	revokeCalls  int
}

func (stub *sessionManagementServiceStub) List(
	_ context.Context,
	actor applicationauth.SessionActor,
	targetUserID string,
) ([]domainauth.SessionDetails, error) {
	stub.listCalls++
	stub.actor = actor
	stub.targetUserID = targetUserID
	return stub.listResult, stub.listError
}

func (stub *sessionManagementServiceStub) Revoke(
	_ context.Context,
	actor applicationauth.SessionActor,
	sessionID string,
) (domainauth.Session, error) {
	stub.revokeCalls++
	stub.actor = actor
	stub.revokeID = sessionID
	return stub.revokeResult, stub.revokeError
}
