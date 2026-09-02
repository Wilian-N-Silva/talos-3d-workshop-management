package apiclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientChecksTypedMetadataAndCompatibility(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/talos/api/v1/meta" || request.Header.Get("Accept") != "application/json" {
			t.Fatalf("request = %s %s, Accept %q", request.Method, request.URL.Path, request.Header.Get("Accept"))
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"api_version":"v1","server_version":"1.4.0","workshop_name":"Prototype Lab","logo_url":null,"minimum_desktop_version":"1.2.0"}`))
	}))
	defer server.Close()

	client, err := New(server.URL+"/talos/", "1.3.0")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result, err := client.CheckConnection(context.Background())
	if err != nil {
		t.Fatalf("CheckConnection() error = %v", err)
	}
	if !result.Compatible || result.CompatibilityIssue != "" || result.WorkshopName != "Prototype Lab" ||
		result.ServerVersion != "1.4.0" || result.MinimumDesktopVersion != "1.2.0" {
		t.Fatalf("CheckConnection() = %#v", result)
	}
}

func TestClientDetectsCompatibilityFailures(t *testing.T) {
	tests := []struct {
		name           string
		apiVersion     string
		minimumVersion string
		wantIssue      string
	}{
		{name: "API version", apiVersion: "v2", minimumVersion: "1.0.0", wantIssue: "api_version_mismatch"},
		{name: "desktop update", apiVersion: "v1", minimumVersion: "1.2.0", wantIssue: "desktop_update_required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				_, _ = response.Write([]byte(`{"api_version":"` + test.apiVersion + `","server_version":"1.0.0","workshop_name":"Lab","minimum_desktop_version":"` + test.minimumVersion + `"}`))
			}))
			defer server.Close()
			client, err := New(server.URL, "1.1.0")
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			result, err := client.CheckConnection(context.Background())
			if err != nil || result.Compatible || result.CompatibilityIssue != test.wantIssue {
				t.Fatalf("CheckConnection() = %#v, %v", result, err)
			}
		})
	}
}

func TestClientMapsAPIErrorEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)
		_, _ = response.Write([]byte(`{"error":{"code":"not_ready","message":"Server is not ready","details":{}}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, "1.0.0")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.CheckConnection(context.Background())
	var clientError *ClientError
	if !errors.As(err, &clientError) || clientError.Kind != ErrorAPI || clientError.StatusCode != http.StatusServiceUnavailable ||
		clientError.Code != "not_ready" || clientError.Message != "Server is not ready" {
		t.Fatalf("CheckConnection() error = %#v", err)
	}
}

func TestClientMapsInvalidMetadataAndTimeout(t *testing.T) {
	t.Run("network", func(t *testing.T) {
		client, err := newClient(
			"http://workshop.local",
			"1.0.0",
			time.Second,
			httpDoerFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") }),
		)
		if err != nil {
			t.Fatalf("newClient() error = %v", err)
		}
		_, err = client.CheckConnection(context.Background())
		assertClientErrorKind(t, err, ErrorNetwork)
	})

	t.Run("invalid metadata", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			_, _ = response.Write([]byte(`{"api_version":"v1"}`))
		}))
		defer server.Close()
		client, _ := New(server.URL, "1.0.0")
		_, err := client.CheckConnection(context.Background())
		assertClientErrorKind(t, err, ErrorInvalidResponse)
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			<-request.Context().Done()
		}))
		defer server.Close()
		client, err := newClient(server.URL, "1.0.0", 20*time.Millisecond, http.DefaultClient)
		if err != nil {
			t.Fatalf("newClient() error = %v", err)
		}
		_, err = client.CheckConnection(context.Background())
		assertClientErrorKind(t, err, ErrorTimeout)
	})
}

func TestClientRejectsInvalidConfiguration(t *testing.T) {
	if _, err := New("postgres://database.local/talos", "1.0.0"); err == nil {
		t.Fatal("New() invalid base URL error = nil")
	}
	if _, err := New("http://workshop.local", "dev"); err == nil {
		t.Fatal("New() invalid desktop version error = nil")
	}
}

func TestSemanticVersionComparison(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{left: "1.0.0", right: "1.0.0", want: 0},
		{left: "1.1.0", right: "1.0.9", want: 1},
		{left: "1.0.0-beta.2", right: "1.0.0-beta.11", want: -1},
		{left: "1.0.0", right: "1.0.0-rc.1", want: 1},
	}
	for _, test := range tests {
		left, leftErr := parseSemanticVersion(test.left)
		right, rightErr := parseSemanticVersion(test.right)
		if leftErr != nil || rightErr != nil {
			t.Fatalf("parse versions %q/%q = %v/%v", test.left, test.right, leftErr, rightErr)
		}
		if got := compareSemanticVersions(left, right); got != test.want {
			t.Fatalf("compareSemanticVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

type httpDoerFunc func(*http.Request) (*http.Response, error)

func (do httpDoerFunc) Do(request *http.Request) (*http.Response, error) {
	return do(request)
}

func assertClientErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var clientError *ClientError
	if !errors.As(err, &clientError) || clientError.Kind != want {
		t.Fatalf("error = %#v, want kind %q", err, want)
	}
}
