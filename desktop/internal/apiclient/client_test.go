package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientLogsInThroughNativeHTTPClient(t *testing.T) {
	expiresAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/v1/auth/login" ||
			request.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("request = %s %s content-type %q", request.Method, request.URL.Path, request.Header.Get("Content-Type"))
		}
		var input LoginInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if input.EmailOrUsername != "owner" || input.Password != "correct horse battery staple" || input.Device.DisplayName != "Workshop PC" {
			t.Fatalf("login input = %#v", input)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"token":"opaque-token","expires_at":"` + expiresAt.Format(time.RFC3339) + `","user":{"id":"user-1","name":"Owner","email_or_username":"owner","status":"active","role":"owner","permissions":["settings.manage"]},"device":{"id":"device-1","display_name":"Workshop PC","os":"Windows","app_version":"1.0.0"}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, "1.0.0")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result, err := client.Login(context.Background(), LoginInput{EmailOrUsername: "owner", Password: "correct horse battery staple", Device: LoginDevice{DisplayName: "Workshop PC", OS: "Windows", AppVersion: "1.0.0"}})
	if err != nil || result.Token != "opaque-token" || result.User.Name != "Owner" || result.Device.ID != "device-1" || !result.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("Login() = %#v, %v", result, err)
	}
}

func TestClientMapsLoginErrorAndRejectsIncompleteResponse(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantKind ErrorKind
	}{
		{name: "invalid credentials", status: http.StatusUnauthorized, body: `{"error":{"code":"invalid_credentials","message":"Invalid credentials"}}`, wantKind: ErrorAPI},
		{name: "incomplete success", status: http.StatusOK, body: `{"token":"secret"}`, wantKind: ErrorInvalidResponse},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(test.status)
				_, _ = response.Write([]byte(test.body))
			}))
			defer server.Close()
			client, _ := New(server.URL, "1.0.0")
			_, err := client.Login(context.Background(), LoginInput{})
			assertClientErrorKind(t, err, test.wantKind)
		})
	}
}

func TestClientFetchesPublicBrandingAndValidatedLogo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/meta":
			_, _ = response.Write([]byte(`{"api_version":"v1","server_version":"1.0.0","workshop_name":"Talos Lab","logo_url":"/api/v1/meta/logo","minimum_desktop_version":"0.0.0"}`))
		case "/api/v1/meta/logo":
			response.Header().Set("Content-Type", "image/png")
			_, _ = response.Write([]byte("png-data"))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	branding, err := client.FetchBranding(context.Background())
	if err != nil || branding.WorkshopName != "Talos Lab" || branding.LogoDataURL != "data:image/png;base64,cG5nLWRhdGE=" {
		t.Fatalf("FetchBranding() = %#v, %v", branding, err)
	}
}

func TestClientRejectsCrossOriginBrandingLogo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"api_version":"v1","server_version":"1.0.0","workshop_name":"Talos Lab","logo_url":"https://untrusted.example/logo.png","minimum_desktop_version":"0.0.0"}`))
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	_, err := client.FetchBranding(context.Background())
	assertClientErrorKind(t, err, ErrorInvalidResponse)
}

func TestClientSendsBearerForSettingsAndMapsAuthorizationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secure-token" {
			response.WriteHeader(http.StatusUnauthorized)
			_, _ = response.Write([]byte(`{"error":{"code":"unauthenticated","message":"Authentication required"}}`))
			return
		}
		_, _ = response.Write([]byte(`{"workshop_name":"Talos Lab","logo_url":null,"default_locale":"pt-BR","default_currency":"BRL","display_timezone":"America/Sao_Paulo","default_theme":"system"}`))
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	settings, err := client.GetWorkshopSettings(context.Background(), "secure-token")
	if err != nil || settings.WorkshopName != "Talos Lab" || settings.DefaultTheme != "system" {
		t.Fatalf("GetWorkshopSettings() = %#v, %v", settings, err)
	}
	_, err = client.GetWorkshopSettings(context.Background(), "wrong-token")
	var clientError *ClientError
	if !errors.As(err, &clientError) || clientError.StatusCode != http.StatusUnauthorized || clientError.Code != "unauthenticated" {
		t.Fatalf("GetWorkshopSettings() unauthorized error = %#v", err)
	}
}

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
