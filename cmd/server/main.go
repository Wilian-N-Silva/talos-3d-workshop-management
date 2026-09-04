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

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	applicationenergy "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/energy"
	applicationfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/files"
	applicationinventory "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/inventory"
	applicationjobs "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/jobs"
	applicationprinters "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/printers"
	applicationsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/settings"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/buildinfo"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/config"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/infrastructure/filestorage"
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
	passwordService, err := applicationauth.NewPasswordService(applicationauth.DefaultPasswordParameters())
	if err != nil {
		logger.Printf("server startup failed: %v", err)
		os.Exit(1)
	}
	userRepository := postgres.NewUserRepository(database)
	workshopSettingsService, err := applicationsettings.NewService(postgres.NewWorkshopSettingsRepository(database))
	if err != nil {
		logger.Printf("server startup failed: initialize workshop settings service: %v", err)
		os.Exit(1)
	}
	if _, err := workshopSettingsService.Initialize(ctx, domainsettings.Values{
		WorkshopName:    serverConfig.WorkshopName,
		DefaultLocale:   serverConfig.DefaultLocale,
		DefaultCurrency: serverConfig.DefaultCurrency,
		DisplayTimezone: serverConfig.DefaultTimezone,
		DefaultTheme:    domainsettings.ThemeSystem,
	}); err != nil {
		logger.Printf("server startup failed: initialize workshop settings: %v", err)
		os.Exit(1)
	}
	objectStore, err := filestorage.NewLocalFilesystemStorage(serverConfig.DataDirectory)
	if err != nil {
		logger.Printf("server startup failed: initialize object storage: %v", err)
		os.Exit(1)
	}
	fileTransferService, err := applicationfiles.NewService(
		postgres.NewFileRepository(database),
		objectStore,
		serverConfig.UploadMaxBytes,
	)
	if err != nil {
		logger.Printf("server startup failed: initialize file transfer service: %v", err)
		os.Exit(1)
	}
	catalogItemService, err := applicationcatalog.NewService(postgres.NewCatalogItemRepository(database))
	if err != nil {
		logger.Printf("server startup failed: initialize catalog item service: %v", err)
		os.Exit(1)
	}
	catalogDesignService, err := applicationcatalog.NewDesignService(postgres.NewCatalogDesignRepository(database))
	if err != nil {
		logger.Printf("server startup failed: initialize catalog design service: %v", err)
		os.Exit(1)
	}
	filamentInventoryService, err := applicationinventory.NewFilamentService(postgres.NewFilamentInventoryRepository(database))
	if err != nil {
		logger.Printf("server startup failed: initialize filament inventory service: %v", err)
		os.Exit(1)
	}
	supplyInventoryService, err := applicationinventory.NewSupplyService(postgres.NewSupplyInventoryRepository(database))
	if err != nil {
		logger.Printf("server startup failed: initialize supply inventory service: %v", err)
		os.Exit(1)
	}
	catalogBOMService, err := applicationcatalog.NewBOMService(postgres.NewCatalogBOMRepository(database))
	if err != nil {
		logger.Printf("server startup failed: initialize catalog BOM service: %v", err)
		os.Exit(1)
	}
	printerService, err := applicationprinters.NewService(postgres.NewPrinterRepository(database))
	if err != nil {
		logger.Printf("server startup failed: initialize printer service: %v", err)
		os.Exit(1)
	}
	jobRepository := postgres.NewJobRepository(database)
	jobService, err := applicationjobs.NewService(jobRepository)
	if err != nil {
		logger.Printf("server startup failed: initialize print job service: %v", err)
		os.Exit(1)
	}
	jobMaterialUsageService, err := applicationjobs.NewMaterialUsageService(jobRepository)
	if err != nil {
		logger.Printf("server startup failed: initialize job material usage service: %v", err)
		os.Exit(1)
	}
	energyService, err := applicationenergy.NewService(postgres.NewEnergyRepository(database))
	if err != nil {
		logger.Printf("server startup failed: initialize energy service: %v", err)
		os.Exit(1)
	}
	maximumLogoBytes := int64(applicationsettings.DefaultMaximumLogoBytes)
	if serverConfig.UploadMaxBytes < maximumLogoBytes {
		maximumLogoBytes = serverConfig.UploadMaxBytes
	}
	workshopLogoService, err := applicationsettings.NewLogoService(
		postgres.NewWorkshopLogoRepository(database),
		objectStore,
		maximumLogoBytes,
	)
	if err != nil {
		logger.Printf("server startup failed: initialize workshop logo service: %v", err)
		os.Exit(1)
	}
	bootstrapService := applicationauth.NewBootstrapService(userRepository, passwordService)
	dummyPassword := []byte("invalid login timing equalization")
	dummyPasswordHash, err := passwordService.Hash(dummyPassword)
	clear(dummyPassword)
	if err != nil {
		logger.Printf("server startup failed: initialize login verification: %v", err)
		os.Exit(1)
	}
	sessionRepository := postgres.NewSessionRepository(database)
	sessionService := applicationauth.NewSessionService(sessionRepository)
	loginService, err := applicationauth.NewLoginService(
		userRepository,
		postgres.NewClientDeviceRepository(database),
		sessionService,
		passwordService,
		dummyPasswordHash,
		serverConfig.SessionTTL,
	)
	if err != nil {
		logger.Printf("server startup failed: initialize login service: %v", err)
		os.Exit(1)
	}
	authenticationService, err := applicationauth.NewAuthenticationService(
		sessionRepository,
		applicationauth.DefaultSessionLastUsedInterval,
	)
	if err != nil {
		logger.Printf("server startup failed: initialize authentication service: %v", err)
		os.Exit(1)
	}
	sessionManagementService, err := applicationauth.NewSessionManagementService(sessionRepository)
	if err != nil {
		logger.Printf("server startup failed: initialize session management service: %v", err)
		os.Exit(1)
	}
	loginRateLimiter, err := httpplatform.NewLoginRateLimiter(
		serverConfig.LoginRateLimitAttempts,
		serverConfig.LoginRateLimitWindow,
	)
	if err != nil {
		logger.Printf("server startup failed: initialize login rate limit: %v", err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", serverConfig.ListenAddress())
	if err != nil {
		logger.Printf("server startup failed: %v", err)
		os.Exit(1)
	}

	metadata := httpplatform.MetaResponse{
		APIVersion:            httpplatform.APIVersion,
		ServerVersion:         buildinfo.ServerVersion,
		WorkshopName:          serverConfig.WorkshopName,
		MinimumDesktopVersion: buildinfo.MinimumDesktopVersion,
	}

	logger.Printf("server listening on %s", listener.Addr())
	if err := run(ctx, listener, logger, newHandler(
		readiness,
		metadata,
		bootstrapService,
		loginService,
		loginRateLimiter,
		authenticationService,
		sessionManagementService,
		workshopSettingsService,
		workshopLogoService,
		maximumLogoBytes,
		fileTransferService,
		serverConfig.UploadMaxBytes,
		catalogItemService,
		catalogDesignService,
		filamentInventoryService,
		supplyInventoryService,
		catalogBOMService,
		printerService,
		jobService,
		jobMaterialUsageService,
		energyService,
	)); err != nil {
		logger.Printf("server stopped with error: %v", err)
		os.Exit(1)
	}
}

func newHandler(
	readiness httpplatform.ReadinessChecker,
	metadata httpplatform.MetaResponse,
	setup httpplatform.SetupService,
	login httpplatform.LoginService,
	loginRateLimiter *httpplatform.LoginRateLimiter,
	authentication httpplatform.BearerAuthenticationService,
	sessions httpplatform.SessionManagementService,
	settings httpplatform.WorkshopSettingsService,
	logo httpplatform.WorkshopLogoService,
	maximumLogoBytes int64,
	files httpplatform.FileTransferService,
	maximumFileBytes int64,
	catalogItems httpplatform.CatalogItemService,
	catalogDesigns httpplatform.CatalogDesignService,
	filamentInventory httpplatform.FilamentInventoryService,
	supplyInventory httpplatform.SupplyInventoryService,
	catalogBOM httpplatform.CatalogBOMService,
	printers httpplatform.PrinterService,
	jobs httpplatform.JobService,
	jobMaterialUsage httpplatform.JobMaterialUsageService,
	energy httpplatform.EnergyService,
) http.Handler {
	mux := http.NewServeMux()
	httpplatform.RegisterLiveness(mux)
	httpplatform.RegisterReadiness(mux, readiness)
	httpplatform.RegisterSetup(mux, setup)
	apiRouter := httpplatform.NewAPIV1Router()
	httpplatform.RegisterMeta(apiRouter, metadata, settings)
	httpplatform.RegisterLogin(apiRouter, login, loginRateLimiter)
	httpplatform.RegisterSessionManagement(apiRouter, authentication, sessions)
	httpplatform.RegisterWorkshopSettings(apiRouter, authentication, settings)
	httpplatform.RegisterWorkshopLogo(apiRouter, authentication, logo, maximumLogoBytes)
	httpplatform.RegisterFiles(apiRouter, authentication, files, maximumFileBytes)
	httpplatform.RegisterCatalogItems(apiRouter, authentication, catalogItems)
	httpplatform.RegisterCatalogDesigns(apiRouter, authentication, catalogDesigns)
	httpplatform.RegisterFilamentInventory(apiRouter, authentication, filamentInventory)
	httpplatform.RegisterSupplyInventory(apiRouter, authentication, supplyInventory)
	httpplatform.RegisterCatalogBOM(apiRouter, authentication, catalogBOM)
	httpplatform.RegisterPrinters(apiRouter, authentication, printers)
	httpplatform.RegisterJobs(apiRouter, authentication, jobs)
	httpplatform.RegisterJobMaterialUsage(apiRouter, authentication, jobMaterialUsage)
	httpplatform.RegisterEnergy(apiRouter, authentication, energy)
	httpplatform.RegisterAPIV1(mux, apiRouter)

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
