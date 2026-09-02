package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const (
	maximumDeviceDisplayNameLength = 200
	maximumDeviceOSLength          = 200
	maximumDeviceAppVersionLength  = 100
)

var (
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrInvalidLoginDevice        = errors.New("invalid login device")
	ErrInvalidLoginConfiguration = errors.New("invalid login configuration")
)

// LoginUserRepository provides identity lookup and successful-login updates.
type LoginUserRepository interface {
	FindByEmailOrUsername(context.Context, string) (domainauth.User, error)
	UpdateLastLogin(context.Context, string, time.Time) (domainauth.User, error)
}

// LoginDeviceRepository registers or refreshes a desktop installation.
type LoginDeviceRepository interface {
	Create(context.Context, domainauth.CreateClientDeviceParams) (domainauth.ClientDevice, error)
	UpdateForLogin(context.Context, string, domainauth.CreateClientDeviceParams, time.Time) (domainauth.ClientDevice, error)
}

// SessionCreator issues an opaque session after credentials are accepted.
type SessionCreator interface {
	Create(context.Context, CreateSessionInput) (IssuedSession, error)
}

// PasswordVerifier compares a supplied password with an Argon2id hash.
type PasswordVerifier interface {
	Verify([]byte, string) (bool, error)
}

// LoginDeviceInput identifies and describes the calling desktop installation.
// ID is omitted on first login and reused on later logins.
type LoginDeviceInput struct {
	ID          string
	DisplayName string
	OS          string
	AppVersion  string
}

// LoginInput contains credentials and installation audit metadata.
type LoginInput struct {
	EmailOrUsername string
	Password        string
	Device          LoginDeviceInput
}

// LoginResult contains safe identity metadata and the once-returned session token.
type LoginResult struct {
	User    domainauth.User
	Device  domainauth.ClientDevice
	Session IssuedSession
}

// LoginService authenticates users and creates desktop sessions.
type LoginService struct {
	users           LoginUserRepository
	devices         LoginDeviceRepository
	sessions        SessionCreator
	passwords       PasswordVerifier
	dummyHash       string
	sessionLifetime time.Duration
	now             func() time.Time
}

// NewLoginService creates a login service with an explicit session lifetime.
func NewLoginService(
	users LoginUserRepository,
	devices LoginDeviceRepository,
	sessions SessionCreator,
	passwords PasswordVerifier,
	dummyHash string,
	sessionLifetime time.Duration,
) (*LoginService, error) {
	return newLoginService(users, devices, sessions, passwords, dummyHash, sessionLifetime, time.Now)
}

func newLoginService(
	users LoginUserRepository,
	devices LoginDeviceRepository,
	sessions SessionCreator,
	passwords PasswordVerifier,
	dummyHash string,
	sessionLifetime time.Duration,
	now func() time.Time,
) (*LoginService, error) {
	if strings.TrimSpace(dummyHash) == "" || sessionLifetime <= 0 {
		return nil, ErrInvalidLoginConfiguration
	}
	return &LoginService{
		users:           users,
		devices:         devices,
		sessions:        sessions,
		passwords:       passwords,
		dummyHash:       dummyHash,
		sessionLifetime: sessionLifetime,
		now:             now,
	}, nil
}

// Login verifies credentials without distinguishing invalid account states,
// refreshes installation audit data, and returns a newly issued session.
func (service *LoginService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	deviceID, deviceParams, err := normalizeLoginDevice(input.Device)
	if err != nil {
		return LoginResult{}, err
	}

	login := strings.TrimSpace(input.EmailOrUsername)
	password := []byte(input.Password)
	defer clear(password)
	if !validBoundedText(login, maximumLoginLength) || !validPassword(input.Password) {
		return LoginResult{}, service.rejectCredentials(password)
	}

	user, err := service.users.FindByEmailOrUsername(ctx, login)
	if errors.Is(err, domainauth.ErrUserNotFound) {
		return LoginResult{}, service.rejectCredentials(password)
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("find login user: %w", err)
	}

	passwordMatches, err := service.passwords.Verify(password, user.PasswordHash)
	if err != nil {
		return LoginResult{}, fmt.Errorf("verify login password: %w", err)
	}
	if !passwordMatches || user.Status != domainauth.UserStatusActive {
		return LoginResult{}, ErrInvalidCredentials
	}

	loggedInAt := service.now().UTC()
	var device domainauth.ClientDevice
	if deviceID == "" {
		device, err = service.devices.Create(ctx, deviceParams)
	} else {
		device, err = service.devices.UpdateForLogin(ctx, deviceID, deviceParams, loggedInAt)
		if errors.Is(err, domainauth.ErrClientDeviceNotFound) {
			return LoginResult{}, ErrInvalidLoginDevice
		}
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("register login device: %w", err)
	}

	user, err = service.users.UpdateLastLogin(ctx, user.ID, loggedInAt)
	if err != nil {
		return LoginResult{}, fmt.Errorf("update login timestamp: %w", err)
	}

	issued, err := service.sessions.Create(ctx, CreateSessionInput{
		UserID:    user.ID,
		DeviceID:  device.ID,
		ExpiresAt: loggedInAt.Add(service.sessionLifetime),
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("create login session: %w", err)
	}

	return LoginResult{User: user, Device: device, Session: issued}, nil
}

func (service *LoginService) rejectCredentials(password []byte) error {
	if _, err := service.passwords.Verify(password, service.dummyHash); err != nil {
		return fmt.Errorf("verify dummy login password: %w", err)
	}
	return ErrInvalidCredentials
}

func normalizeLoginDevice(input LoginDeviceInput) (string, domainauth.CreateClientDeviceParams, error) {
	id := strings.TrimSpace(input.ID)
	if id != "" && !validUUID(id) {
		return "", domainauth.CreateClientDeviceParams{}, ErrInvalidLoginDevice
	}
	displayName := strings.TrimSpace(input.DisplayName)
	operatingSystem := strings.TrimSpace(input.OS)
	appVersion := strings.TrimSpace(input.AppVersion)
	if !validBoundedText(displayName, maximumDeviceDisplayNameLength) ||
		!validBoundedText(operatingSystem, maximumDeviceOSLength) ||
		!validBoundedText(appVersion, maximumDeviceAppVersionLength) {
		return "", domainauth.CreateClientDeviceParams{}, ErrInvalidLoginDevice
	}
	return id, domainauth.CreateClientDeviceParams{
		DisplayName: displayName,
		OS:          operatingSystem,
		AppVersion:  appVersion,
	}, nil
}
