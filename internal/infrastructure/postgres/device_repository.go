package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const clientDeviceColumns = `id, display_name, os, app_version, created_at, last_seen_at`

// ClientDeviceRepository persists desktop installation audit records in PostgreSQL.
type ClientDeviceRepository struct {
	database *sql.DB
}

// NewClientDeviceRepository creates a PostgreSQL client-device repository.
func NewClientDeviceRepository(database *sql.DB) *ClientDeviceRepository {
	return &ClientDeviceRepository{database: database}
}

// Create registers a desktop installation and returns database-generated values.
func (repository *ClientDeviceRepository) Create(
	ctx context.Context,
	params auth.CreateClientDeviceParams,
) (auth.ClientDevice, error) {
	device, err := scanClientDevice(repository.database.QueryRowContext(
		ctx,
		`INSERT INTO client_devices (display_name, os, app_version)
		 VALUES ($1, $2, $3)
		 RETURNING `+clientDeviceColumns,
		params.DisplayName,
		params.OS,
		params.AppVersion,
	))
	if err != nil {
		return auth.ClientDevice{}, fmt.Errorf("create client device: %w", err)
	}
	return device, nil
}

// FindByID finds a registered desktop installation by its server-generated ID.
func (repository *ClientDeviceRepository) FindByID(
	ctx context.Context,
	id string,
) (auth.ClientDevice, error) {
	device, err := scanClientDevice(repository.database.QueryRowContext(
		ctx,
		`SELECT `+clientDeviceColumns+` FROM client_devices WHERE id = $1`,
		id,
	))
	if err != nil {
		return auth.ClientDevice{}, fmt.Errorf("find client device: %w", err)
	}
	return device, nil
}

// UpdateLastSeen records client activity without allowing delayed observations
// to move the audit timestamp backward.
func (repository *ClientDeviceRepository) UpdateLastSeen(
	ctx context.Context,
	id string,
	observedAt time.Time,
) (auth.ClientDevice, error) {
	device, err := scanClientDevice(repository.database.QueryRowContext(
		ctx,
		`UPDATE client_devices
		 SET last_seen_at = GREATEST(last_seen_at, $2)
		 WHERE id = $1
		 RETURNING `+clientDeviceColumns,
		id,
		observedAt.UTC(),
	))
	if err != nil {
		return auth.ClientDevice{}, fmt.Errorf("update client device last seen: %w", err)
	}
	return device, nil
}

func scanClientDevice(row rowScanner) (auth.ClientDevice, error) {
	var device auth.ClientDevice
	if err := row.Scan(
		&device.ID,
		&device.DisplayName,
		&device.OS,
		&device.AppVersion,
		&device.CreatedAt,
		&device.LastSeenAt,
	); err != nil {
		return auth.ClientDevice{}, err
	}

	device.CreatedAt = device.CreatedAt.UTC()
	device.LastSeenAt = device.LastSeenAt.UTC()
	return device, nil
}
