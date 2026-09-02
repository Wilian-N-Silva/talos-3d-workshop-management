// Package desktopapp owns the Wails application boundary.
package desktopapp

import (
	"context"
	"fmt"
	"sync"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/apiclient"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/buildinfo"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/serverconnection"
)

type connectionStore interface {
	Load() (serverconnection.Configuration, error)
	Save(string) (serverconnection.Configuration, error)
}

type connectionChecker interface {
	CheckConnection(context.Context) (apiclient.ConnectionResult, error)
}

type connectionClientFactory func(string, string) (connectionChecker, error)

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

// App owns native desktop lifecycle and server connection state.
type App struct {
	ctxMu          sync.RWMutex
	ctx            context.Context
	store          connectionStore
	desktopVersion string
	newClient      connectionClientFactory
}

// New creates the desktop application with user-scoped connection storage.
func New() (*App, error) {
	store, err := serverconnection.NewDefaultStore()
	if err != nil {
		return nil, err
	}
	return newApp(store, buildinfo.DesktopVersion, func(baseURL, desktopVersion string) (connectionChecker, error) {
		return apiclient.New(baseURL, desktopVersion)
	}), nil
}

func newApp(store connectionStore, desktopVersion string, factory connectionClientFactory) *App {
	return &App{store: store, desktopVersion: desktopVersion, newClient: factory}
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
