package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestBootstrapServiceReportsRepositoryStatus(t *testing.T) {
	repository := &bootstrapRepositoryStub{needsSetup: true}
	service := NewBootstrapService(repository, &passwordHasherStub{})

	needsSetup, err := service.NeedsSetup(context.Background())
	if err != nil {
		t.Fatalf("NeedsSetup() error = %v", err)
	}
	if !needsSetup || repository.statusCalls != 1 {
		t.Fatalf("NeedsSetup() = %t with %d calls, want true with one call", needsSetup, repository.statusCalls)
	}
}

func TestBootstrapServiceCreatesHashedActiveOwner(t *testing.T) {
	repository := &bootstrapRepositoryStub{
		needsSetup:  true,
		createdUser: domainauth.User{ID: "owner-id", Status: domainauth.UserStatusActive},
	}
	passwords := &passwordHasherStub{hash: "$argon2id$test-hash"}
	service := NewBootstrapService(repository, passwords)

	created, err := service.CreateAdmin(context.Background(), CreateAdminInput{
		Name:            "  Workshop Owner  ",
		EmailOrUsername: "  owner@example.com  ",
		Password:        "a long owner passphrase",
	})
	if err != nil {
		t.Fatalf("CreateAdmin() error = %v", err)
	}
	if created.ID != "owner-id" || created.Status != domainauth.UserStatusActive {
		t.Fatalf("created user = %#v", created)
	}
	if passwords.calls != 1 || !passwords.receivedNonEmpty {
		t.Fatalf("password hasher calls = %d, received nonempty = %t", passwords.calls, passwords.receivedNonEmpty)
	}
	if repository.createCalls != 1 {
		t.Fatalf("repository create calls = %d, want 1", repository.createCalls)
	}
	if repository.params.Name != "Workshop Owner" || repository.params.EmailOrUsername != "owner@example.com" {
		t.Fatalf("normalized identity fields = %#v", repository.params)
	}
	if repository.params.PasswordHash != "$argon2id$test-hash" || repository.params.Status != domainauth.UserStatusActive || repository.params.Role != domainauth.RoleOwner {
		t.Fatalf("persisted authentication fields = %#v", repository.params)
	}
}

func TestBootstrapServiceRejectsInvalidInputBeforeHashing(t *testing.T) {
	tests := []struct {
		name      string
		input     CreateAdminInput
		wantError error
	}{
		{
			name:      "blank name",
			input:     CreateAdminInput{EmailOrUsername: "owner", Password: "a long owner passphrase"},
			wantError: ErrInvalidName,
		},
		{
			name:      "blank login",
			input:     CreateAdminInput{Name: "Owner", Password: "a long owner passphrase"},
			wantError: ErrInvalidLogin,
		},
		{
			name:      "control character in login",
			input:     CreateAdminInput{Name: "Owner", EmailOrUsername: "owner\x00name", Password: "a long owner passphrase"},
			wantError: ErrInvalidLogin,
		},
		{
			name:      "short password",
			input:     CreateAdminInput{Name: "Owner", EmailOrUsername: "owner", Password: "too short"},
			wantError: ErrInvalidPassword,
		},
		{
			name:      "oversized password",
			input:     CreateAdminInput{Name: "Owner", EmailOrUsername: "owner", Password: strings.Repeat("a", MaximumPasswordLength+1)},
			wantError: ErrInvalidPassword,
		},
		{
			name:      "control character in password",
			input:     CreateAdminInput{Name: "Owner", EmailOrUsername: "owner", Password: "long owner pass\nword"},
			wantError: ErrInvalidPassword,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &bootstrapRepositoryStub{needsSetup: true}
			passwords := &passwordHasherStub{hash: "$argon2id$test-hash"}
			service := NewBootstrapService(repository, passwords)

			_, err := service.CreateAdmin(context.Background(), test.input)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("CreateAdmin() error = %v, want %v", err, test.wantError)
			}
			if passwords.calls != 0 || repository.createCalls != 0 {
				t.Fatalf("invalid input reached dependencies: hash calls %d, create calls %d", passwords.calls, repository.createCalls)
			}
		})
	}
}

func TestBootstrapServiceMapsClosedSetup(t *testing.T) {
	repository := &bootstrapRepositoryStub{needsSetup: true, createError: domainauth.ErrFirstUserAlreadyExists}
	service := NewBootstrapService(repository, &passwordHasherStub{hash: "$argon2id$test-hash"})

	_, err := service.CreateAdmin(context.Background(), CreateAdminInput{
		Name:            "Owner",
		EmailOrUsername: "owner",
		Password:        "a long owner passphrase",
	})
	if !errors.Is(err, ErrSetupClosed) {
		t.Fatalf("CreateAdmin() error = %v, want ErrSetupClosed", err)
	}
}

func TestBootstrapServiceSkipsHashingWhenSetupIsAlreadyClosed(t *testing.T) {
	repository := &bootstrapRepositoryStub{}
	passwords := &passwordHasherStub{hash: "$argon2id$test-hash"}
	service := NewBootstrapService(repository, passwords)

	_, err := service.CreateAdmin(context.Background(), CreateAdminInput{
		Name:            "Owner",
		EmailOrUsername: "owner",
		Password:        "a long owner passphrase",
	})
	if !errors.Is(err, ErrSetupClosed) {
		t.Fatalf("CreateAdmin() error = %v, want ErrSetupClosed", err)
	}
	if passwords.calls != 0 || repository.createCalls != 0 {
		t.Fatalf("closed setup reached expensive dependencies: hash calls %d, create calls %d", passwords.calls, repository.createCalls)
	}
}

func TestBootstrapServiceDoesNotPersistWhenHashingFails(t *testing.T) {
	repository := &bootstrapRepositoryStub{needsSetup: true}
	passwords := &passwordHasherStub{err: errors.New("hashing unavailable")}
	service := NewBootstrapService(repository, passwords)

	_, err := service.CreateAdmin(context.Background(), CreateAdminInput{
		Name:            "Owner",
		EmailOrUsername: "owner",
		Password:        "a long owner passphrase",
	})
	if err == nil {
		t.Fatal("CreateAdmin() error = nil")
	}
	if repository.createCalls != 0 {
		t.Fatalf("repository create calls = %d, want 0", repository.createCalls)
	}
}

type bootstrapRepositoryStub struct {
	needsSetup  bool
	statusErr   error
	statusCalls int
	params      domainauth.CreateUserParams
	createdUser domainauth.User
	createError error
	createCalls int
}

func (stub *bootstrapRepositoryStub) NeedsSetup(context.Context) (bool, error) {
	stub.statusCalls++
	return stub.needsSetup, stub.statusErr
}

func (stub *bootstrapRepositoryStub) CreateFirst(
	_ context.Context,
	params domainauth.CreateUserParams,
) (domainauth.User, error) {
	stub.createCalls++
	stub.params = params
	return stub.createdUser, stub.createError
}

type passwordHasherStub struct {
	hash             string
	err              error
	calls            int
	receivedNonEmpty bool
}

func (stub *passwordHasherStub) Hash(password []byte) (string, error) {
	stub.calls++
	stub.receivedNonEmpty = len(password) > 0
	return stub.hash, stub.err
}
