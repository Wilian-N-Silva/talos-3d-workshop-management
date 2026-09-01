package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var errConnection = errors.New("connect to PostgreSQL: database unavailable or credentials invalid")

// Open creates and verifies the server's PostgreSQL connection pool.
func Open(ctx context.Context, serverConfig config.Config) (*sql.DB, error) {
	database, err := sql.Open("pgx", serverConfig.DatabaseURL)
	if err != nil {
		return nil, connectionError{cause: err}
	}

	database.SetMaxOpenConns(serverConfig.DatabaseMaxOpenConnections)
	database.SetMaxIdleConns(serverConfig.DatabaseMaxIdleConnections)
	database.SetConnMaxLifetime(serverConfig.DatabaseConnectionMaxLifetime)
	database.SetConnMaxIdleTime(serverConfig.DatabaseConnectionMaxIdleTime)

	pingContext, cancel := context.WithTimeout(ctx, serverConfig.DatabasePingTimeout)
	defer cancel()

	if err := database.PingContext(pingContext); err != nil {
		_ = database.Close()
		return nil, connectionError{cause: err}
	}

	return database, nil
}

type connectionError struct {
	cause error
}

func (connectionError) Error() string {
	return errConnection.Error()
}

func (err connectionError) Unwrap() error {
	return err.cause
}
