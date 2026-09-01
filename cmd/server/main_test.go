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
	"testing"
	"time"

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

func (stub readinessStub) Check(context.Context) error {
	return stub.err
}

func TestHandlerRegistersLiveness(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	newHandler(readinessStub{err: errors.New("dependencies unavailable")}, testMetadata).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRegistersReadiness(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()

	newHandler(readinessStub{}, testMetadata).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRegistersMeta(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.MetaPath, nil)
	response := httptest.NewRecorder()

	newHandler(readinessStub{}, testMetadata).ServeHTTP(response, request)

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

func TestHandlerHasNoProductRoutes(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	response := httptest.NewRecorder()

	newHandler(readinessStub{}, testMetadata).ServeHTTP(response, request)

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
	go func() {
		result <- run(ctx, listener, log.New(io.Discard, "", 0), newHandler(readinessStub{}, testMetadata))
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
