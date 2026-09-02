package desktopapp

import (
	"context"
	"errors"
	"testing"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/apiclient"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/serverconnection"
)

func TestAppLoadsAndSavesServerConnection(t *testing.T) {
	store := &connectionStoreStub{loaded: serverconnection.Configuration{ServerBaseURL: "http://saved.local"}}
	app := newApp(store, "1.2.0", nil)

	loaded, err := app.GetServerConnection()
	if err != nil || loaded.ServerBaseURL != "http://saved.local" {
		t.Fatalf("GetServerConnection() = %#v, %v", loaded, err)
	}
	saved, err := app.SaveServerConnection(" https://new.local/ ")
	if err != nil || saved.ServerBaseURL != "https://new.local" || store.saved != " https://new.local/ " {
		t.Fatalf("SaveServerConnection() = %#v, %v; input %q", saved, err, store.saved)
	}
}

func TestAppTestsConnectionThroughNativeClient(t *testing.T) {
	checker := &connectionCheckerStub{result: apiclient.ConnectionResult{
		Meta: apiclient.Meta{
			APIVersion:            "v1",
			ServerVersion:         "1.4.0",
			WorkshopName:          "Prototype Lab",
			MinimumDesktopVersion: "1.1.0",
		},
		Compatible: true,
	}}
	var factoryBaseURL, factoryDesktopVersion string
	app := newApp(&connectionStoreStub{}, "1.2.0", func(baseURL, desktopVersion string) (connectionChecker, error) {
		factoryBaseURL = baseURL
		factoryDesktopVersion = desktopVersion
		return checker, nil
	})
	app.Startup(context.Background())

	result, err := app.TestServerConnection(" http://workshop.local:8080/ ")
	if err != nil {
		t.Fatalf("TestServerConnection() error = %v", err)
	}
	if factoryBaseURL != "http://workshop.local:8080" || factoryDesktopVersion != "1.2.0" ||
		result.ServerBaseURL != factoryBaseURL || result.WorkshopName != "Prototype Lab" || !result.Compatible {
		t.Fatalf("TestServerConnection() = %#v, factory = %q/%q", result, factoryBaseURL, factoryDesktopVersion)
	}
}

func TestAppRejectsInvalidConnectionBeforeClientCreation(t *testing.T) {
	called := false
	app := newApp(&connectionStoreStub{}, "1.2.0", func(string, string) (connectionChecker, error) {
		called = true
		return &connectionCheckerStub{}, nil
	})
	if _, err := app.TestServerConnection("postgres://database.local/talos"); !errors.Is(err, serverconnection.ErrInvalidBaseURL) {
		t.Fatalf("TestServerConnection() error = %v", err)
	}
	if called {
		t.Fatal("client factory called for invalid base URL")
	}
}

type connectionStoreStub struct {
	loaded    serverconnection.Configuration
	loadError error
	saved     string
	saveError error
}

func (stub *connectionStoreStub) Load() (serverconnection.Configuration, error) {
	return stub.loaded, stub.loadError
}

func (stub *connectionStoreStub) Save(value string) (serverconnection.Configuration, error) {
	stub.saved = value
	if stub.saveError != nil {
		return serverconnection.Configuration{}, stub.saveError
	}
	normalized, err := serverconnection.NormalizeBaseURL(value)
	return serverconnection.Configuration{ServerBaseURL: normalized}, err
}

type connectionCheckerStub struct {
	result apiclient.ConnectionResult
	err    error
}

func (stub *connectionCheckerStub) CheckConnection(context.Context) (apiclient.ConnectionResult, error) {
	return stub.result, stub.err
}
