package postgres

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestLoginFlowAgainstPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TALOS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TALOS_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	database, err := Open(ctx, testConfig(databaseURL))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate authentication tables: %v", err)
	}

	passwords, err := applicationauth.NewPasswordService(applicationauth.DefaultPasswordParameters())
	if err != nil {
		t.Fatalf("NewPasswordService() error = %v", err)
	}
	password := []byte(validLoginPasswordForIntegration)
	passwordHash, err := passwords.Hash(password)
	clear(password)
	if err != nil {
		t.Fatalf("hash login password: %v", err)
	}
	dummyPassword := []byte("invalid login timing equalization")
	dummyHash, err := passwords.Hash(dummyPassword)
	clear(dummyPassword)
	if err != nil {
		t.Fatalf("hash dummy login password: %v", err)
	}

	users := NewUserRepository(database)
	createdUser, err := users.Create(ctx, domainauth.CreateUserParams{
		Name:            "Workshop Owner",
		EmailOrUsername: "owner@example.com",
		PasswordHash:    passwordHash,
		Status:          domainauth.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create login user: %v", err)
	}
	devices := NewClientDeviceRepository(database)
	service, err := applicationauth.NewLoginService(
		users,
		devices,
		applicationauth.NewSessionService(NewSessionRepository(database)),
		passwords,
		dummyHash,
		24*time.Hour,
	)
	if err != nil {
		t.Fatalf("NewLoginService() error = %v", err)
	}

	result, err := service.Login(ctx, applicationauth.LoginInput{
		EmailOrUsername: "OWNER@EXAMPLE.COM",
		Password:        validLoginPasswordForIntegration,
		Device: applicationauth.LoginDeviceInput{
			DisplayName: "Workshop PC",
			OS:          "Windows 11",
			AppVersion:  "1.0.0",
		},
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.ID != createdUser.ID || result.User.LastLoginAt == nil {
		t.Fatalf("logged-in user = %#v", result.User)
	}
	if result.Device.ID == "" || result.Session.Token == "" || result.Session.Session.ID == "" {
		t.Fatalf("login result = %#v", result)
	}
	if result.Session.Session.UserID != createdUser.ID || result.Session.Session.DeviceID != result.Device.ID {
		t.Fatalf("session relationships = %#v", result.Session.Session)
	}

	var storedHash []byte
	if err := database.QueryRowContext(ctx, "SELECT token_hash FROM sessions WHERE id = $1", result.Session.Session.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read login session hash: %v", err)
	}
	if !bytes.Equal(storedHash, applicationauth.HashSessionToken(result.Session.Token)) || bytes.Equal(storedHash, []byte(result.Session.Token)) {
		t.Fatal("login session did not persist exactly the token hash")
	}

	second, err := service.Login(ctx, applicationauth.LoginInput{
		EmailOrUsername: "owner@example.com",
		Password:        validLoginPasswordForIntegration,
		Device: applicationauth.LoginDeviceInput{
			ID:          result.Device.ID,
			DisplayName: "Workshop Desktop",
			OS:          "Windows 11 Pro",
			AppVersion:  "1.1.0",
		},
	})
	if err != nil {
		t.Fatalf("second Login() error = %v", err)
	}
	if second.Device.ID != result.Device.ID || second.Device.AppVersion != "1.1.0" {
		t.Fatalf("reused device = %#v", second.Device)
	}
	assertTableCount(t, ctx, database, "client_devices", 1)
	assertTableCount(t, ctx, database, "sessions", 2)

	invalidInput := applicationauth.LoginInput{
		EmailOrUsername: "owner@example.com",
		Password:        "this is the wrong password",
		Device: applicationauth.LoginDeviceInput{
			ID:          result.Device.ID,
			DisplayName: "Workshop Desktop",
			OS:          "Windows 11 Pro",
			AppVersion:  "1.1.0",
		},
	}
	if _, err := service.Login(ctx, invalidInput); !errors.Is(err, applicationauth.ErrInvalidCredentials) {
		t.Fatalf("invalid Login() error = %v, want ErrInvalidCredentials", err)
	}
	assertTableCount(t, ctx, database, "sessions", 2)
}

const validLoginPasswordForIntegration = "correct horse battery staple"

func assertTableCount(t *testing.T, ctx context.Context, database queryRower, table string, want int64) {
	t.Helper()
	queries := map[string]string{
		"client_devices": "SELECT count(*) FROM client_devices",
		"sessions":       "SELECT count(*) FROM sessions",
	}
	query, ok := queries[table]
	if !ok {
		t.Fatalf("unsupported table %q", table)
	}
	var got int64
	if err := database.QueryRowContext(ctx, query).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
