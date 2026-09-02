package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const validLoginPassword = "correct horse battery staple"

func TestLoginServiceAuthenticatesAndCreatesSession(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.FixedZone("test", -3*60*60))
	users := &loginUserRepositoryStub{found: activeLoginUser(), updated: activeLoginUser()}
	devices := &loginDeviceRepositoryStub{created: domainauth.ClientDevice{ID: "device-id"}}
	sessions := &sessionCreatorStub{issued: IssuedSession{Token: "opaque-token"}}
	passwords := &passwordVerifierStub{matches: true}
	service := newTestLoginService(t, users, devices, sessions, passwords, now)

	result, err := service.Login(context.Background(), LoginInput{
		EmailOrUsername: "  OWNER@EXAMPLE.COM  ",
		Password:        validLoginPassword,
		Device: LoginDeviceInput{
			DisplayName: "  Workshop PC  ",
			OS:          "  Windows 11  ",
			AppVersion:  "  1.0.0  ",
		},
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if users.lookup != "OWNER@EXAMPLE.COM" || users.updateID != "user-id" {
		t.Fatalf("user repository inputs = lookup %q, update %q", users.lookup, users.updateID)
	}
	if passwords.calls != 1 || passwords.hashes[0] != "$argon2id$user-hash" {
		t.Fatalf("password verification = %d calls with %#v", passwords.calls, passwords.hashes)
	}
	if devices.createCalls != 1 || devices.updateCalls != 0 {
		t.Fatalf("device calls = create %d, update %d", devices.createCalls, devices.updateCalls)
	}
	if devices.params.DisplayName != "Workshop PC" || devices.params.OS != "Windows 11" || devices.params.AppVersion != "1.0.0" {
		t.Fatalf("normalized device params = %#v", devices.params)
	}
	wantLoginAt := now.UTC()
	if !users.updatedAt.Equal(wantLoginAt) {
		t.Fatalf("last login = %s, want %s", users.updatedAt, wantLoginAt)
	}
	if sessions.calls != 1 || sessions.input.UserID != "user-id" || sessions.input.DeviceID != "device-id" {
		t.Fatalf("session input = %#v with %d calls", sessions.input, sessions.calls)
	}
	if !sessions.input.ExpiresAt.Equal(wantLoginAt.Add(24 * time.Hour)) {
		t.Fatalf("session expiry = %s, want %s", sessions.input.ExpiresAt, wantLoginAt.Add(24*time.Hour))
	}
	if result.Session.Token != "opaque-token" || result.Device.ID != "device-id" || result.User.ID != "user-id" {
		t.Fatalf("login result = %#v", result)
	}
}

func TestLoginServiceRefreshesExistingDevice(t *testing.T) {
	now := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.UTC)
	users := &loginUserRepositoryStub{found: activeLoginUser(), updated: activeLoginUser()}
	devices := &loginDeviceRepositoryStub{updated: domainauth.ClientDevice{ID: "123e4567-e89b-42d3-a456-426614174000"}}
	service := newTestLoginService(t, users, devices, &sessionCreatorStub{}, &passwordVerifierStub{matches: true}, now)

	_, err := service.Login(context.Background(), LoginInput{
		EmailOrUsername: "owner@example.com",
		Password:        validLoginPassword,
		Device: LoginDeviceInput{
			ID:          "123e4567-e89b-42d3-a456-426614174000",
			DisplayName: "Workshop PC",
			OS:          "Windows 11",
			AppVersion:  "1.1.0",
		},
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if devices.createCalls != 0 || devices.updateCalls != 1 {
		t.Fatalf("device calls = create %d, update %d", devices.createCalls, devices.updateCalls)
	}
	if devices.updateID != "123e4567-e89b-42d3-a456-426614174000" || !devices.observedAt.Equal(now) {
		t.Fatalf("device refresh = ID %q at %s", devices.updateID, devices.observedAt)
	}
}

func TestLoginServiceReturnsSameErrorForInvalidCredentials(t *testing.T) {
	tests := []struct {
		name      string
		input     LoginInput
		user      domainauth.User
		findError error
		matches   bool
		wantHash  string
	}{
		{name: "unknown user", input: validLoginInput(), findError: domainauth.ErrUserNotFound, wantHash: "$argon2id$dummy-hash"},
		{name: "wrong password", input: validLoginInput(), user: activeLoginUser(), wantHash: "$argon2id$user-hash"},
		{name: "disabled user", input: validLoginInput(), user: disabledLoginUser(), matches: true, wantHash: "$argon2id$user-hash"},
		{name: "malformed identifier", input: LoginInput{EmailOrUsername: "", Password: validLoginPassword, Device: validLoginInput().Device}, wantHash: "$argon2id$dummy-hash"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := &loginUserRepositoryStub{found: test.user, findError: test.findError}
			devices := &loginDeviceRepositoryStub{}
			sessions := &sessionCreatorStub{}
			passwords := &passwordVerifierStub{matches: test.matches}
			service := newTestLoginService(t, users, devices, sessions, passwords, time.Now())

			_, err := service.Login(context.Background(), test.input)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
			}
			if passwords.calls != 1 || passwords.hashes[0] != test.wantHash {
				t.Fatalf("password verification = %d calls with %#v, want hash %q", passwords.calls, passwords.hashes, test.wantHash)
			}
			if devices.createCalls != 0 || devices.updateCalls != 0 || users.updateCalls != 0 || sessions.calls != 0 {
				t.Fatal("invalid credentials reached successful-login dependencies")
			}
		})
	}
}

func TestLoginServiceRejectsInvalidDeviceBeforeCredentialLookup(t *testing.T) {
	input := validLoginInput()
	input.Device.ID = "not-a-uuid"
	users := &loginUserRepositoryStub{}
	passwords := &passwordVerifierStub{}
	service := newTestLoginService(t, users, &loginDeviceRepositoryStub{}, &sessionCreatorStub{}, passwords, time.Now())

	_, err := service.Login(context.Background(), input)
	if !errors.Is(err, ErrInvalidLoginDevice) {
		t.Fatalf("Login() error = %v, want ErrInvalidLoginDevice", err)
	}
	if users.findCalls != 0 || passwords.calls != 0 {
		t.Fatal("invalid device reached credential dependencies")
	}
}

func TestLoginServiceMapsMissingExistingDevice(t *testing.T) {
	input := validLoginInput()
	input.Device.ID = "123e4567-e89b-42d3-a456-426614174000"
	users := &loginUserRepositoryStub{found: activeLoginUser()}
	devices := &loginDeviceRepositoryStub{updateError: domainauth.ErrClientDeviceNotFound}
	service := newTestLoginService(t, users, devices, &sessionCreatorStub{}, &passwordVerifierStub{matches: true}, time.Now())

	_, err := service.Login(context.Background(), input)
	if !errors.Is(err, ErrInvalidLoginDevice) {
		t.Fatalf("Login() error = %v, want ErrInvalidLoginDevice", err)
	}
}

func TestNewLoginServiceRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewLoginService(nil, nil, nil, nil, "", time.Hour); !errors.Is(err, ErrInvalidLoginConfiguration) {
		t.Fatalf("empty dummy hash error = %v", err)
	}
	if _, err := NewLoginService(nil, nil, nil, nil, "hash", 0); !errors.Is(err, ErrInvalidLoginConfiguration) {
		t.Fatalf("zero lifetime error = %v", err)
	}
}

func newTestLoginService(
	t *testing.T,
	users LoginUserRepository,
	devices LoginDeviceRepository,
	sessions SessionCreator,
	passwords PasswordVerifier,
	now time.Time,
) *LoginService {
	t.Helper()
	service, err := newLoginService(users, devices, sessions, passwords, "$argon2id$dummy-hash", 24*time.Hour, func() time.Time { return now })
	if err != nil {
		t.Fatalf("newLoginService() error = %v", err)
	}
	return service
}

func validLoginInput() LoginInput {
	return LoginInput{
		EmailOrUsername: "owner@example.com",
		Password:        validLoginPassword,
		Device: LoginDeviceInput{
			DisplayName: "Workshop PC",
			OS:          "Windows 11",
			AppVersion:  "1.0.0",
		},
	}
}

func activeLoginUser() domainauth.User {
	return domainauth.User{
		ID:              "user-id",
		Name:            "Owner",
		EmailOrUsername: "owner@example.com",
		PasswordHash:    "$argon2id$user-hash",
		Status:          domainauth.UserStatusActive,
	}
}

func disabledLoginUser() domainauth.User {
	user := activeLoginUser()
	user.Status = domainauth.UserStatusDisabled
	return user
}

type loginUserRepositoryStub struct {
	found       domainauth.User
	findError   error
	lookup      string
	findCalls   int
	updated     domainauth.User
	updateError error
	updateID    string
	updatedAt   time.Time
	updateCalls int
}

func (stub *loginUserRepositoryStub) FindByEmailOrUsername(_ context.Context, login string) (domainauth.User, error) {
	stub.findCalls++
	stub.lookup = login
	return stub.found, stub.findError
}

func (stub *loginUserRepositoryStub) UpdateLastLogin(_ context.Context, id string, at time.Time) (domainauth.User, error) {
	stub.updateCalls++
	stub.updateID = id
	stub.updatedAt = at
	if stub.updateError != nil {
		return domainauth.User{}, stub.updateError
	}
	if stub.updated.ID == "" {
		stub.updated = stub.found
	}
	return stub.updated, nil
}

type loginDeviceRepositoryStub struct {
	created     domainauth.ClientDevice
	createError error
	updated     domainauth.ClientDevice
	updateError error
	params      domainauth.CreateClientDeviceParams
	updateID    string
	observedAt  time.Time
	createCalls int
	updateCalls int
}

func (stub *loginDeviceRepositoryStub) Create(_ context.Context, params domainauth.CreateClientDeviceParams) (domainauth.ClientDevice, error) {
	stub.createCalls++
	stub.params = params
	return stub.created, stub.createError
}

func (stub *loginDeviceRepositoryStub) UpdateForLogin(
	_ context.Context,
	id string,
	params domainauth.CreateClientDeviceParams,
	at time.Time,
) (domainauth.ClientDevice, error) {
	stub.updateCalls++
	stub.updateID = id
	stub.params = params
	stub.observedAt = at
	return stub.updated, stub.updateError
}

type passwordVerifierStub struct {
	matches bool
	err     error
	hashes  []string
	calls   int
}

func (stub *passwordVerifierStub) Verify(_ []byte, hash string) (bool, error) {
	stub.calls++
	stub.hashes = append(stub.hashes, hash)
	return stub.matches, stub.err
}

type sessionCreatorStub struct {
	issued IssuedSession
	err    error
	input  CreateSessionInput
	calls  int
}

func (stub *sessionCreatorStub) Create(_ context.Context, input CreateSessionInput) (IssuedSession, error) {
	stub.calls++
	stub.input = input
	return stub.issued, stub.err
}
