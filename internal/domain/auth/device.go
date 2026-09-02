package auth

import (
	"errors"
	"time"
)

// ErrClientDeviceNotFound indicates that no registered installation matches a lookup.
var ErrClientDeviceNotFound = errors.New("client device not found")

// ClientDevice is an authorized desktop installation recorded for audit.
type ClientDevice struct {
	ID          string
	DisplayName string
	OS          string
	AppVersion  string
	CreatedAt   time.Time
	LastSeenAt  time.Time
}

// CreateClientDeviceParams contains installation metadata ready for persistence.
type CreateClientDeviceParams struct {
	DisplayName string
	OS          string
	AppVersion  string
}
