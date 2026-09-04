// Package apiclient contains the typed native client for the workshop API.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/serverconnection"
)

const (
	ExpectedAPIVersion  = "v1"
	defaultTimeout      = 8 * time.Second
	maximumResponseSize = 1024 * 1024
)

// ErrorKind groups failures into stable categories for the desktop application layer.
type ErrorKind string

const (
	ErrorNetwork         ErrorKind = "network"
	ErrorTimeout         ErrorKind = "timeout"
	ErrorAPI             ErrorKind = "api"
	ErrorInvalidResponse ErrorKind = "invalid_response"
)

// ClientError is the common error mapping for remote API operations.
type ClientError struct {
	Kind       ErrorKind
	StatusCode int
	Code       string
	Message    string
	cause      error
}

func (clientError *ClientError) Error() string {
	if clientError.Message != "" {
		return clientError.Message
	}
	return string(clientError.Kind)
}

func (clientError *ClientError) Unwrap() error {
	return clientError.cause
}

// Meta is the typed unauthenticated API compatibility contract.
type Meta struct {
	APIVersion            string  `json:"api_version"`
	ServerVersion         string  `json:"server_version"`
	WorkshopName          string  `json:"workshop_name"`
	LogoURL               *string `json:"logo_url"`
	MinimumDesktopVersion string  `json:"minimum_desktop_version"`
}

// ConnectionResult reports server identity and desktop compatibility.
type ConnectionResult struct {
	Meta
	Compatible         bool   `json:"compatible"`
	CompatibilityIssue string `json:"compatibility_issue"`
}

// LoginInput contains credentials and local installation audit metadata.
type LoginInput struct {
	EmailOrUsername string      `json:"email_or_username"`
	Password        string      `json:"password"`
	Device          LoginDevice `json:"device"`
}

// LoginDevice identifies the Windows installation to the server.
type LoginDevice struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"display_name"`
	OS          string `json:"os"`
	AppVersion  string `json:"app_version"`
}

// LoginResult is the typed server response. Token remains native-only.
type LoginResult struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      LoginUser   `json:"user"`
	Device    LoginDevice `json:"device"`
}

// LoginUser is safe identity metadata returned after authentication.
type LoginUser struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	EmailOrUsername string   `json:"email_or_username"`
	Status          string   `json:"status"`
	Role            string   `json:"role"`
	Permissions     []string `json:"permissions"`
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client owns the server origin, timeout, and remote request behavior.
type Client struct {
	baseURL        string
	desktopVersion semanticVersion
	timeout        time.Duration
	httpClient     httpDoer
}

// New creates the production API client. Business endpoints are intentionally absent.
func New(baseURL, desktopVersion string) (*Client, error) {
	return newClient(baseURL, desktopVersion, defaultTimeout, http.DefaultClient)
}

func newClient(baseURL, desktopVersion string, timeout time.Duration, httpClient httpDoer) (*Client, error) {
	normalized, err := serverconnection.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	version, err := parseSemanticVersion(desktopVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid desktop version: %w", err)
	}
	if timeout <= 0 || httpClient == nil {
		return nil, errors.New("invalid API client configuration")
	}
	return &Client{baseURL: normalized, desktopVersion: version, timeout: timeout, httpClient: httpClient}, nil
}

// CheckConnection calls only the public metadata endpoint and evaluates compatibility.
func (client *Client) CheckConnection(ctx context.Context) (ConnectionResult, error) {
	requestContext, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(
		requestContext,
		http.MethodGet,
		client.baseURL+"/api/v1/meta",
		nil,
	)
	if err != nil {
		return ConnectionResult{}, invalidResponseError("Unable to create server request", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		if errors.Is(requestContext.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return ConnectionResult{}, &ClientError{Kind: ErrorTimeout, Message: "Server connection timed out", cause: err}
		}
		return ConnectionResult{}, &ClientError{Kind: ErrorNetwork, Message: "Unable to connect to server", cause: err}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseSize+1))
	if err != nil {
		return ConnectionResult{}, invalidResponseError("Unable to read server response", err)
	}
	if len(body) > maximumResponseSize {
		return ConnectionResult{}, invalidResponseError("Server response is too large", nil)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ConnectionResult{}, mapAPIError(response.StatusCode, body)
	}

	var metadata Meta
	if err := json.Unmarshal(body, &metadata); err != nil {
		return ConnectionResult{}, invalidResponseError("Server returned invalid metadata", err)
	}
	if strings.TrimSpace(metadata.APIVersion) == "" || strings.TrimSpace(metadata.ServerVersion) == "" ||
		strings.TrimSpace(metadata.WorkshopName) == "" || strings.TrimSpace(metadata.MinimumDesktopVersion) == "" {
		return ConnectionResult{}, invalidResponseError("Server metadata is incomplete", nil)
	}
	minimumVersion, err := parseSemanticVersion(metadata.MinimumDesktopVersion)
	if err != nil {
		return ConnectionResult{}, invalidResponseError("Server returned an invalid minimum desktop version", err)
	}

	result := ConnectionResult{Meta: metadata, Compatible: true}
	if metadata.APIVersion != ExpectedAPIVersion {
		result.Compatible = false
		result.CompatibilityIssue = "api_version_mismatch"
	} else if compareSemanticVersions(client.desktopVersion, minimumVersion) < 0 {
		result.Compatible = false
		result.CompatibilityIssue = "desktop_update_required"
	}
	return result, nil
}

// Login authenticates through the native client. The caller must immediately
// move the returned token into secure storage and never expose it to React.
func (client *Client) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return LoginResult{}, invalidResponseError("Unable to encode login request", err)
	}
	defer clear(body)
	requestContext, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		client.baseURL+"/api/v1/auth/login",
		bytes.NewReader(body),
	)
	if err != nil {
		return LoginResult{}, invalidResponseError("Unable to create login request", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		if errors.Is(requestContext.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return LoginResult{}, &ClientError{Kind: ErrorTimeout, Message: "Login timed out", cause: err}
		}
		return LoginResult{}, &ClientError{Kind: ErrorNetwork, Message: "Unable to connect to server", cause: err}
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseSize+1))
	if err != nil {
		return LoginResult{}, invalidResponseError("Unable to read login response", err)
	}
	defer clear(responseBody)
	if len(responseBody) > maximumResponseSize {
		return LoginResult{}, invalidResponseError("Server response is too large", nil)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return LoginResult{}, mapAPIError(response.StatusCode, responseBody)
	}
	var result LoginResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return LoginResult{}, invalidResponseError("Server returned an invalid login response", err)
	}
	if strings.TrimSpace(result.Token) == "" || result.ExpiresAt.IsZero() ||
		strings.TrimSpace(result.User.ID) == "" || strings.TrimSpace(result.User.Name) == "" ||
		strings.TrimSpace(result.User.EmailOrUsername) == "" || strings.TrimSpace(result.Device.ID) == "" {
		return LoginResult{}, invalidResponseError("Server login response is incomplete", nil)
	}
	return result, nil
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func mapAPIError(statusCode int, body []byte) error {
	mapped := &ClientError{
		Kind:       ErrorAPI,
		StatusCode: statusCode,
		Code:       "http_error",
		Message:    "Server rejected the request",
	}
	var envelope errorEnvelope
	if json.Unmarshal(body, &envelope) == nil && envelope.Error.Code != "" && envelope.Error.Message != "" {
		mapped.Code = envelope.Error.Code
		mapped.Message = envelope.Error.Message
	}
	return mapped
}

func invalidResponseError(message string, cause error) error {
	return &ClientError{Kind: ErrorInvalidResponse, Message: message, cause: cause}
}

type semanticVersion struct {
	core       [3]uint64
	prerelease []string
}

func parseSemanticVersion(raw string) (semanticVersion, error) {
	value := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	if buildIndex := strings.IndexByte(value, '+'); buildIndex >= 0 {
		if buildIndex == len(value)-1 {
			return semanticVersion{}, errors.New("empty build metadata")
		}
		value = value[:buildIndex]
	}
	var prerelease []string
	if prereleaseIndex := strings.IndexByte(value, '-'); prereleaseIndex >= 0 {
		if prereleaseIndex == len(value)-1 {
			return semanticVersion{}, errors.New("empty prerelease")
		}
		prerelease = strings.Split(value[prereleaseIndex+1:], ".")
		value = value[:prereleaseIndex]
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return semanticVersion{}, errors.New("version must contain major, minor, and patch")
	}
	version := semanticVersion{prerelease: prerelease}
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return semanticVersion{}, errors.New("invalid numeric version component")
		}
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return semanticVersion{}, errors.New("invalid numeric version component")
		}
		version.core[index] = parsed
	}
	for _, identifier := range prerelease {
		if identifier == "" {
			return semanticVersion{}, errors.New("invalid prerelease")
		}
		for _, character := range identifier {
			if !(character >= '0' && character <= '9') && !(character >= 'A' && character <= 'Z') &&
				!(character >= 'a' && character <= 'z') && character != '-' {
				return semanticVersion{}, errors.New("invalid prerelease")
			}
		}
	}
	return version, nil
}

func compareSemanticVersions(left, right semanticVersion) int {
	for index := range left.core {
		if left.core[index] < right.core[index] {
			return -1
		}
		if left.core[index] > right.core[index] {
			return 1
		}
	}
	if len(left.prerelease) == 0 && len(right.prerelease) == 0 {
		return 0
	}
	if len(left.prerelease) == 0 {
		return 1
	}
	if len(right.prerelease) == 0 {
		return -1
	}
	for index := 0; index < len(left.prerelease) && index < len(right.prerelease); index++ {
		comparison := comparePrereleaseIdentifiers(left.prerelease[index], right.prerelease[index])
		if comparison != 0 {
			return comparison
		}
	}
	if len(left.prerelease) < len(right.prerelease) {
		return -1
	}
	if len(left.prerelease) > len(right.prerelease) {
		return 1
	}
	return 0
}

func comparePrereleaseIdentifiers(left, right string) int {
	leftNumber, leftErr := strconv.ParseUint(left, 10, 64)
	rightNumber, rightErr := strconv.ParseUint(right, 10, 64)
	switch {
	case leftErr == nil && rightErr == nil:
		if leftNumber < rightNumber {
			return -1
		}
		if leftNumber > rightNumber {
			return 1
		}
		return 0
	case leftErr == nil:
		return -1
	case rightErr == nil:
		return 1
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
