// Package auth contains authentication domain records and value definitions.
package auth

import (
	"errors"
	"time"
)

// ErrFirstUserAlreadyExists indicates that bootstrap has permanently closed.
var ErrFirstUserAlreadyExists = errors.New("first user already exists")

// UserStatus controls whether a persisted user may authenticate.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// User is the persisted server-side identity record.
type User struct {
	ID              string
	Name            string
	EmailOrUsername string
	PasswordHash    string
	Status          UserStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastLoginAt     *time.Time
}

// CreateUserParams contains hashed identity data ready for persistence.
type CreateUserParams struct {
	Name            string
	EmailOrUsername string
	PasswordHash    string
	Status          UserStatus
}
