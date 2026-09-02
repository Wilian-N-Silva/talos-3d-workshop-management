package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"testing"

	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

func TestMetaReturnsConfiguredMetadata(t *testing.T) {
	metadata := MetaResponse{
		APIVersion:            APIVersion,
		ServerVersion:         "1.2.3",
		WorkshopName:          "stale process default",
		MinimumDesktopVersion: "1.1.0",
	}
	want := metadata
	want.WorkshopName = "Prototype Lab"
	logoFileID := "logo-file-id"
	logoURL := APIV1Prefix + WorkshopLogoDownloadPath
	want.LogoURL = &logoURL
	router := NewAPIV1Router()
	RegisterMeta(router, metadata, &workshopSettingsServiceStub{result: domainsettings.WorkshopSettings{
		WorkshopName: want.WorkshopName,
		LogoFileID:   &logoFileID,
	}})

	response := serveAPIRequest(t, router, http.MethodGet, APIV1Prefix+MetaPath, "meta-request")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := response.Header().Get(RequestIDHeader); got != "meta-request" {
		t.Fatalf("X-Request-ID = %q, want meta-request", got)
	}

	var got MetaResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata = %#v, want %#v", got, want)
	}
}

func TestMetaRejectsUnsupportedMethods(t *testing.T) {
	router := NewAPIV1Router()
	RegisterMeta(router, MetaResponse{}, &workshopSettingsServiceStub{})

	response := serveAPIRequest(t, router, http.MethodPost, APIV1Prefix+MetaPath, "")

	assertAPIError(t, response, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
}

func TestMetaHandlesWorkshopSettingsFailure(t *testing.T) {
	router := NewAPIV1Router()
	RegisterMeta(router, MetaResponse{}, &workshopSettingsServiceStub{getError: errors.New("database details")})

	response := serveAPIRequest(t, router, http.MethodGet, APIV1Prefix+MetaPath, "")
	assertAPIError(t, response, http.StatusInternalServerError, "internal_error", "Internal server error")
}

func TestMetaHandlesConcurrentRequestsWithoutSharedMutation(t *testing.T) {
	router := NewAPIV1Router()
	RegisterMeta(router, MetaResponse{ServerVersion: "test"}, workshopSettingsReaderFunc(
		func(context.Context) (domainsettings.WorkshopSettings, error) {
			return domainsettings.WorkshopSettings{WorkshopName: "Concurrent Lab"}, nil
		},
	))

	const requests = 16
	var wait sync.WaitGroup
	errorsFound := make(chan error, requests)
	for range requests {
		wait.Add(1)
		go func() {
			defer wait.Done()
			response := serveAPIRequest(t, router, http.MethodGet, APIV1Prefix+MetaPath, "")
			if response.Code != http.StatusOK {
				errorsFound <- errors.New("unexpected response status")
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
}

type workshopSettingsReaderFunc func(context.Context) (domainsettings.WorkshopSettings, error)

func (function workshopSettingsReaderFunc) Get(ctx context.Context) (domainsettings.WorkshopSettings, error) {
	return function(ctx)
}
