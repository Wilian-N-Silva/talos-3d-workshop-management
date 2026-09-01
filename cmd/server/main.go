package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/config"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/infrastructure/postgres"
	healthplatform "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/platform/health"
	httpplatform "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/platform/http"
)

const (
	shutdownTimeout  = 5 * time.Second
	readinessTimeout = 5 * time.Second
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	serverConfig, err := config.Load()
	if err != nil {
		logger.Printf("invalid server configuration: %v", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := postgres.Open(ctx, serverConfig)
	if err != nil {
		logger.Printf("server startup failed: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := postgres.Migrate(ctx, database); err != nil {
		logger.Printf("server startup failed: %v", err)
		os.Exit(1)
	}
	readiness := healthplatform.NewReadiness(
		database.PingContext,
		func(ctx context.Context) error {
			state, err := postgres.GetMigrationState(ctx, database)
			if err != nil {
				return err
			}
			if !state.IsCurrent() {
				return errors.New("database migrations are not current")
			}
			return nil
		},
		healthplatform.StorageDirectory(serverConfig.DataDirectory),
		readinessTimeout,
	)

	listener, err := net.Listen("tcp", serverConfig.ListenAddress())
	if err != nil {
		logger.Printf("server startup failed: %v", err)
		os.Exit(1)
	}

	logger.Printf("server listening on %s", listener.Addr())
	if err := run(ctx, listener, logger, newHandler(readiness)); err != nil {
		logger.Printf("server stopped with error: %v", err)
		os.Exit(1)
	}
}

func newHandler(readiness httpplatform.ReadinessChecker) http.Handler {
	mux := http.NewServeMux()
	httpplatform.RegisterLiveness(mux)
	httpplatform.RegisterReadiness(mux, readiness)

	return mux
}

func run(ctx context.Context, listener net.Listener, logger *log.Logger, handler http.Handler) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.Serve(listener)
	}()

	select {
	case err := <-serveResult:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		logger.Print("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	if err := <-serveResult; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}

	logger.Print("server stopped")
	return nil
}
