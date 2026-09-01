package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const (
	MinimumPasswordLength = 15
	MaximumPasswordLength = 1024
	maximumPasswordBytes  = 4096
	maximumNameLength     = 200
	maximumLoginLength    = 320
)

var (
	ErrSetupClosed     = errors.New("setup is closed")
	ErrInvalidName     = errors.New("invalid name")
	ErrInvalidLogin    = errors.New("invalid login identifier")
	ErrInvalidPassword = errors.New("invalid password")
)

// BootstrapRepository persists the first user under a database-level race
// guarantee and exposes the permanent bootstrap state.
type BootstrapRepository interface {
	NeedsSetup(context.Context) (bool, error)
	CreateFirst(context.Context, domainauth.CreateUserParams) (domainauth.User, error)
}

// PasswordHasher converts plaintext password bytes into persistence-safe data.
type PasswordHasher interface {
	Hash([]byte) (string, error)
}

// CreateAdminInput contains the unauthenticated first-owner setup fields.
type CreateAdminInput struct {
	Name            string
	EmailOrUsername string
	Password        string
}

// BootstrapService owns first-owner validation, hashing, and persistence.
type BootstrapService struct {
	repository BootstrapRepository
	passwords  PasswordHasher
}

// NewBootstrapService creates the first-owner application service.
func NewBootstrapService(repository BootstrapRepository, passwords PasswordHasher) *BootstrapService {
	return &BootstrapService{repository: repository, passwords: passwords}
}

// NeedsSetup reports whether first-owner setup remains available.
func (service *BootstrapService) NeedsSetup(ctx context.Context) (bool, error) {
	needsSetup, err := service.repository.NeedsSetup(ctx)
	if err != nil {
		return false, fmt.Errorf("check bootstrap status: %w", err)
	}
	return needsSetup, nil
}

// CreateAdmin validates and creates the initial active owner identity.
func (service *BootstrapService) CreateAdmin(
	ctx context.Context,
	input CreateAdminInput,
) (domainauth.User, error) {
	needsSetup, err := service.repository.NeedsSetup(ctx)
	if err != nil {
		return domainauth.User{}, fmt.Errorf("check bootstrap before owner creation: %w", err)
	}
	if !needsSetup {
		return domainauth.User{}, ErrSetupClosed
	}

	name := strings.TrimSpace(input.Name)
	if !validBoundedText(name, maximumNameLength) {
		return domainauth.User{}, ErrInvalidName
	}

	login := strings.TrimSpace(input.EmailOrUsername)
	if !validBoundedText(login, maximumLoginLength) {
		return domainauth.User{}, ErrInvalidLogin
	}

	if !validPassword(input.Password) {
		return domainauth.User{}, ErrInvalidPassword
	}
	password := []byte(input.Password)
	defer clear(password)

	passwordHash, err := service.passwords.Hash(password)
	if err != nil {
		return domainauth.User{}, fmt.Errorf("hash first-owner password: %w", err)
	}

	user, err := service.repository.CreateFirst(ctx, domainauth.CreateUserParams{
		Name:            name,
		EmailOrUsername: login,
		PasswordHash:    passwordHash,
		Status:          domainauth.UserStatusActive,
	})
	if errors.Is(err, domainauth.ErrFirstUserAlreadyExists) {
		return domainauth.User{}, ErrSetupClosed
	}
	if err != nil {
		return domainauth.User{}, fmt.Errorf("create first owner: %w", err)
	}
	return user, nil
}

func validBoundedText(value string, maximumLength int) bool {
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > maximumLength {
		return false
	}
	return !containsControlCharacter(value)
}

func validPassword(password string) bool {
	if !utf8.ValidString(password) || len(password) > maximumPasswordBytes || containsControlCharacter(password) {
		return false
	}
	length := utf8.RuneCountInString(password)
	return length >= MinimumPasswordLength && length <= MaximumPasswordLength
}

func containsControlCharacter(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}
