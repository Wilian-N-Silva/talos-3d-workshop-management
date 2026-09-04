// Package desktopapp owns the Wails application boundary.
package desktopapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/apiclient"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/buildinfo"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/credentials"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/serverconnection"
)

type connectionStore interface {
	Load() (serverconnection.Configuration, error)
	Save(string) (serverconnection.Configuration, error)
}

type remoteClient interface {
	CheckConnection(context.Context) (apiclient.ConnectionResult, error)
	Login(context.Context, apiclient.LoginInput) (apiclient.LoginResult, error)
	FetchBranding(context.Context) (apiclient.Branding, error)
	GetWorkshopSettings(context.Context, string) (apiclient.WorkshopSettings, error)
	ListCatalogItems(context.Context, string) (apiclient.CatalogPage, error)
	CreateCatalogItem(context.Context, string, apiclient.CatalogItemInput) (apiclient.CatalogItem, error)
	UpdateCatalogItem(context.Context, string, string, apiclient.CatalogItemInput) (apiclient.CatalogItem, error)
}

type connectionClientFactory func(string, string) (remoteClient, error)

type sessionStore interface {
	Save(string, credentials.Session) error
	Load(string) (credentials.Session, error)
	Delete(string) error
}

// ConnectionTestResult is the safe server metadata exposed to React.
type ConnectionTestResult struct {
	ServerBaseURL         string `json:"server_base_url"`
	DesktopVersion        string `json:"desktop_version"`
	APIVersion            string `json:"api_version"`
	ServerVersion         string `json:"server_version"`
	WorkshopName          string `json:"workshop_name"`
	MinimumDesktopVersion string `json:"minimum_desktop_version"`
	Compatible            bool   `json:"compatible"`
	CompatibilityIssue    string `json:"compatibility_issue"`
}

// AuthenticationState is safe to expose to React and deliberately omits the token.
type AuthenticationState struct {
	Authenticated   bool     `json:"authenticated"`
	UserID          string   `json:"user_id,omitempty"`
	UserName        string   `json:"user_name,omitempty"`
	EmailOrUsername string   `json:"email_or_username,omitempty"`
	ExpiresAt       string   `json:"expires_at,omitempty"`
	Role            string   `json:"role,omitempty"`
	Permissions     []string `json:"permissions"`
}

// ShellContext contains safe user, permission, theme, and branding data.
type ShellContext struct {
	Authentication AuthenticationState `json:"authentication"`
	WorkshopName   string              `json:"workshop_name,omitempty"`
	LogoDataURL    string              `json:"logo_data_url,omitempty"`
	DefaultTheme   string              `json:"default_theme,omitempty"`
}

// App owns native desktop lifecycle and server connection state.
type App struct {
	ctxMu          sync.RWMutex
	ctx            context.Context
	store          connectionStore
	sessions       sessionStore
	desktopVersion string
	newClient      connectionClientFactory
}

// New creates the desktop application with user-scoped connection storage.
func New() (*App, error) {
	store, err := serverconnection.NewDefaultStore()
	if err != nil {
		return nil, err
	}
	return newApp(store, credentials.NewStore(), buildinfo.DesktopVersion, func(baseURL, desktopVersion string) (remoteClient, error) {
		return apiclient.New(baseURL, desktopVersion)
	}), nil
}

func newApp(store connectionStore, sessions sessionStore, desktopVersion string, factory connectionClientFactory) *App {
	return &App{store: store, sessions: sessions, desktopVersion: desktopVersion, newClient: factory}
}

// Login authenticates through native Go and moves the returned token directly
// into Windows Credential Manager without exposing it to the WebView.
func (a *App) Login(emailOrUsername, password string) (AuthenticationState, error) {
	configuration, err := a.store.Load()
	if err != nil {
		return AuthenticationState{}, fmt.Errorf("load server connection: %w", err)
	}
	if strings.TrimSpace(configuration.ServerBaseURL) == "" {
		return AuthenticationState{}, errors.New("configure the workshop server before login")
	}
	client, err := a.newClient(configuration.ServerBaseURL, a.desktopVersion)
	if err != nil {
		return AuthenticationState{}, fmt.Errorf("create API client: %w", err)
	}
	displayName, err := os.Hostname()
	if err != nil || strings.TrimSpace(displayName) == "" {
		displayName = "Windows desktop"
	}
	result, err := client.Login(a.applicationContext(), apiclient.LoginInput{
		EmailOrUsername: emailOrUsername,
		Password:        password,
		Device: apiclient.LoginDevice{
			DisplayName: displayName,
			OS:          runtime.GOOS,
			AppVersion:  a.desktopVersion,
		},
	})
	if err != nil {
		return AuthenticationState{}, err
	}
	session := credentials.Session{
		Token:           result.Token,
		ExpiresAt:       result.ExpiresAt.UTC(),
		UserID:          result.User.ID,
		UserName:        result.User.Name,
		EmailOrUsername: result.User.EmailOrUsername,
		Role:            result.User.Role,
		Permissions:     append([]string(nil), result.User.Permissions...),
		DeviceID:        result.Device.ID,
	}
	if err := a.sessions.Save(configuration.ServerBaseURL, session); err != nil {
		return AuthenticationState{}, fmt.Errorf("secure session: %w", err)
	}
	return authenticationState(session), nil
}

// GetWorkshopBranding returns public branding without exposing server HTTP to React.
func (a *App) GetWorkshopBranding() (apiclient.Branding, error) {
	configuration, err := a.store.Load()
	if err != nil {
		return apiclient.Branding{}, fmt.Errorf("load server connection: %w", err)
	}
	if strings.TrimSpace(configuration.ServerBaseURL) == "" {
		return apiclient.Branding{}, nil
	}
	client, err := a.newClient(configuration.ServerBaseURL, a.desktopVersion)
	if err != nil {
		return apiclient.Branding{}, fmt.Errorf("create API client: %w", err)
	}
	return client.FetchBranding(a.applicationContext())
}

// LoadShell validates the restored bearer session against an authenticated
// server endpoint. A 401 clears the unusable local credential; a 403 remains a
// visible authorization error and never bypasses the server decision.
func (a *App) LoadShell() (ShellContext, error) {
	configuration, err := a.store.Load()
	if err != nil {
		return ShellContext{}, fmt.Errorf("load server connection: %w", err)
	}
	if strings.TrimSpace(configuration.ServerBaseURL) == "" {
		return ShellContext{}, nil
	}
	session, err := a.sessions.Load(configuration.ServerBaseURL)
	if errors.Is(err, credentials.ErrNotFound) {
		return ShellContext{}, nil
	}
	if err != nil {
		return ShellContext{}, fmt.Errorf("restore secure session: %w", err)
	}
	client, err := a.newClient(configuration.ServerBaseURL, a.desktopVersion)
	if err != nil {
		return ShellContext{}, fmt.Errorf("create API client: %w", err)
	}
	settings, err := client.GetWorkshopSettings(a.applicationContext(), session.Token)
	if err != nil {
		var clientError *apiclient.ClientError
		if errors.As(err, &clientError) && clientError.StatusCode == 401 {
			if deleteErr := a.sessions.Delete(configuration.ServerBaseURL); deleteErr != nil {
				return ShellContext{}, fmt.Errorf("remove rejected session: %w", deleteErr)
			}
			return ShellContext{}, nil
		}
		return ShellContext{}, err
	}
	branding, err := client.FetchBranding(a.applicationContext())
	if err != nil {
		branding = apiclient.Branding{WorkshopName: settings.WorkshopName}
	}
	return ShellContext{
		Authentication: authenticationState(session),
		WorkshopName:   settings.WorkshopName,
		LogoDataURL:    branding.LogoDataURL,
		DefaultTheme:   settings.DefaultTheme,
	}, nil
}

// GetAuthenticationState restores a non-expired secure session at startup.
func (a *App) GetAuthenticationState() (AuthenticationState, error) {
	configuration, err := a.store.Load()
	if err != nil {
		return AuthenticationState{}, fmt.Errorf("load server connection: %w", err)
	}
	if strings.TrimSpace(configuration.ServerBaseURL) == "" {
		return AuthenticationState{}, nil
	}
	session, err := a.sessions.Load(configuration.ServerBaseURL)
	if errors.Is(err, credentials.ErrNotFound) {
		return AuthenticationState{}, nil
	}
	if err != nil {
		return AuthenticationState{}, fmt.Errorf("restore secure session: %w", err)
	}
	return authenticationState(session), nil
}

// Logout removes the local bearer credential. Server-side revocation remains
// available through session management and is integrated by a later UI package.
func (a *App) Logout() error {
	configuration, err := a.store.Load()
	if err != nil {
		return fmt.Errorf("load server connection: %w", err)
	}
	if strings.TrimSpace(configuration.ServerBaseURL) == "" {
		return nil
	}
	return a.sessions.Delete(configuration.ServerBaseURL)
}

// ListCatalogItems returns the bounded catalog page without exposing the bearer token.
func (a *App) ListCatalogItems() (apiclient.CatalogPage, error) {
	client, session, baseURL, err := a.authenticatedClient()
	if err != nil {
		return apiclient.CatalogPage{}, err
	}
	page, err := client.ListCatalogItems(a.applicationContext(), session.Token)
	if err != nil {
		return apiclient.CatalogPage{}, a.handleAuthenticatedError(baseURL, err)
	}
	return page, nil
}

// CreateCatalogItem creates one item through the native authenticated client.
func (a *App) CreateCatalogItem(input apiclient.CatalogItemInput) (apiclient.CatalogItem, error) {
	client, session, baseURL, err := a.authenticatedClient()
	if err != nil {
		return apiclient.CatalogItem{}, err
	}
	item, err := client.CreateCatalogItem(a.applicationContext(), session.Token, input)
	if err != nil {
		return apiclient.CatalogItem{}, a.handleAuthenticatedError(baseURL, err)
	}
	return item, nil
}

// UpdateCatalogItem replaces editable item fields through the native authenticated client.
func (a *App) UpdateCatalogItem(id string, input apiclient.CatalogItemInput) (apiclient.CatalogItem, error) {
	client, session, baseURL, err := a.authenticatedClient()
	if err != nil {
		return apiclient.CatalogItem{}, err
	}
	item, err := client.UpdateCatalogItem(a.applicationContext(), session.Token, id, input)
	if err != nil {
		return apiclient.CatalogItem{}, a.handleAuthenticatedError(baseURL, err)
	}
	return item, nil
}

func (a *App) authenticatedClient() (remoteClient, credentials.Session, string, error) {
	configuration, err := a.store.Load()
	if err != nil {
		return nil, credentials.Session{}, "", fmt.Errorf("load server connection: %w", err)
	}
	if strings.TrimSpace(configuration.ServerBaseURL) == "" {
		return nil, credentials.Session{}, "", errors.New("configure the workshop server before continuing")
	}
	session, err := a.sessions.Load(configuration.ServerBaseURL)
	if errors.Is(err, credentials.ErrNotFound) {
		return nil, credentials.Session{}, "", errors.New("authentication required")
	}
	if err != nil {
		return nil, credentials.Session{}, "", fmt.Errorf("restore secure session: %w", err)
	}
	client, err := a.newClient(configuration.ServerBaseURL, a.desktopVersion)
	if err != nil {
		return nil, credentials.Session{}, "", fmt.Errorf("create API client: %w", err)
	}
	return client, session, configuration.ServerBaseURL, nil
}

func (a *App) handleAuthenticatedError(baseURL string, err error) error {
	var clientError *apiclient.ClientError
	if errors.As(err, &clientError) && clientError.StatusCode == 401 {
		if deleteErr := a.sessions.Delete(baseURL); deleteErr != nil {
			return fmt.Errorf("remove rejected session: %w", deleteErr)
		}
	}
	return err
}

func authenticationState(session credentials.Session) AuthenticationState {
	return AuthenticationState{
		Authenticated:   true,
		UserID:          session.UserID,
		UserName:        session.UserName,
		EmailOrUsername: session.EmailOrUsername,
		ExpiresAt:       session.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Role:            session.Role,
		Permissions:     append([]string(nil), session.Permissions...),
	}
}

// Startup stores the Wails lifecycle context for native operations.
func (a *App) Startup(ctx context.Context) {
	a.ctxMu.Lock()
	defer a.ctxMu.Unlock()
	a.ctx = ctx
}

// GetServerConnection returns the local non-secret endpoint configuration.
func (a *App) GetServerConnection() (serverconnection.Configuration, error) {
	configuration, err := a.store.Load()
	if err != nil {
		return serverconnection.Configuration{}, fmt.Errorf("load server connection: %w", err)
	}
	return configuration, nil
}

// SaveServerConnection validates and stores the endpoint without requiring the server to be online.
func (a *App) SaveServerConnection(baseURL string) (serverconnection.Configuration, error) {
	configuration, err := a.store.Save(baseURL)
	if err != nil {
		return serverconnection.Configuration{}, fmt.Errorf("save server connection: %w", err)
	}
	return configuration, nil
}

// TestServerConnection calls the public compatibility endpoint through the native API client.
func (a *App) TestServerConnection(baseURL string) (ConnectionTestResult, error) {
	normalized, err := serverconnection.NormalizeBaseURL(baseURL)
	if err != nil {
		return ConnectionTestResult{}, err
	}
	client, err := a.newClient(normalized, a.desktopVersion)
	if err != nil {
		return ConnectionTestResult{}, fmt.Errorf("create API client: %w", err)
	}
	remote, err := client.CheckConnection(a.applicationContext())
	if err != nil {
		return ConnectionTestResult{}, err
	}
	return ConnectionTestResult{
		ServerBaseURL:         normalized,
		DesktopVersion:        a.desktopVersion,
		APIVersion:            remote.APIVersion,
		ServerVersion:         remote.ServerVersion,
		WorkshopName:          remote.WorkshopName,
		MinimumDesktopVersion: remote.MinimumDesktopVersion,
		Compatible:            remote.Compatible,
		CompatibilityIssue:    remote.CompatibilityIssue,
	}, nil
}

func (a *App) applicationContext() context.Context {
	a.ctxMu.RLock()
	defer a.ctxMu.RUnlock()
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}
