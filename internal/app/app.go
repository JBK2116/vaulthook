// Package app provides the application bootstrap and lifecycle management.
//
// The App struct encapsulates all infrastructure dependencies, services,
// and the HTTP router. It is constructed once at startup and provides a
// single Run method that blocks until the server terminates.
package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/api/handler"
	"github.com/JBK2116/vaulthook/internal/auth"
	"github.com/JBK2116/vaulthook/internal/cache"
	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/health"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/JBK2116/vaulthook/internal/providers/github"
	"github.com/JBK2116/vaulthook/internal/providers/stripe"
	"github.com/JBK2116/vaulthook/internal/worker"
)

// App holds all runtime dependencies for the vaulthook application.
// It is constructed via New and started via Run.
type App struct {
	db     *pgxpool.Pool
	logger *zerolog.Logger

	// lifecycle
	cancelApp     context.CancelFunc
	cancelWorkers context.CancelFunc

	// services
	authService     *auth.AuthService
	providerService *providers.ProviderService
	eventService    *events.EventService
	stripeService   *stripe.Service
	gitService      *github.GitService

	// workers
	workerPool *worker.Pool

	// HTTP
	router chi.Router
}

const (
	// ThrottleMaxConcurrent is the maximum number of requests handled concurrently.
	ThrottleMaxConcurrent = 500
	// ThrottleMaxBacklog is the maximum number of requests queued while at capacity.
	ThrottleMaxBacklog = 2000
	// ThrottleBacklogTimeout is the number of seconds a queued request may wait before timing out.
	ThrottleBacklogTimeout = 5
	// MaxRequestTime is the maximum number of seconds a request may run end-to-end.
	MaxRequestTime = 5
)

// New initializes all infrastructure, wires dependencies, and returns a
// fully configured App ready to serve requests.
//
// Any failure during initialization panics to prevent a misconfigured
// server from accepting traffic.
func New() *App {
	// Environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env file not found, using system environment: %v", err)
	}
	config.Init()

	// calculate application resource allocations
	resources := health.ComputeResources()
	worker.QueueWorkerBatch = resources.QueueBatch
	worker.RetryWorkerBatch = resources.RetryBatch
	worker.TotalQueueWorkers = resources.QueueWorkers
	worker.TotalRetryWorkers = resources.RetryWorkers

	// Infrastructure: Database & Logger
	dbCtx, cancelDB := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelDB()

	pg, dbErr := config.NewPG(dbCtx, resources.DBMaxConns)
	if dbErr != nil {
		panic(dbErr)
	}
	if err := pg.Ping(dbCtx); err != nil {
		panic(err)
	}

	logger, err := config.NewLogger()
	if err != nil {
		panic(err)
	}

	// Application context — stored so Run() can cancel on shutdown.
	appCtx, cancelApp := context.WithCancel(context.Background())

	// Repositories
	refreshTokenRepo := auth.NewRefreshTokenRepo(pg.DB)
	providerRepo := providers.NewProviderRepo(pg.DB)
	eventRepo := events.NewEventRepo(pg.DB)

	// Services
	authSvc := auth.NewAuthService(
		config.Envs.JWTSecret,
		auth.AccessTokenTTL,
		auth.RefreshTokenTTL,
		refreshTokenRepo,
		logger,
	)
	providerSvc := providers.NewProviderService(providerRepo)
	eventSvc := events.NewEventService(logger, eventRepo)
	stripeSvc := stripe.NewStripeService(logger, eventRepo, providerRepo)
	gitSvc := github.NewGitService(logger, eventRepo, providerRepo)

	// Caches
	if err := cache.InitRedisCache(appCtx); err != nil {
		panic(err)
	}
	if err := providers.InitProviderCache(appCtx, providerRepo); err != nil {
		panic(err)
	}

	// Rate Limiting
	worker.InitRateLimiter(cache.Cache.GetCache())

	// Health Checks
	healthSvc := health.NewHealthService(pg.DB, cache.Cache.GetCache())

	// Background Workers
	go eventSvc.Start(appCtx)
	workerCtx, cancelWorkers := context.WithCancel(appCtx)
	workerPool := worker.NewWorkerPool(workerCtx, eventSvc, logger, pg.DB)

	// HTTP handlers
	healthH := handler.NewHealthHandler(healthSvc, logger)
	authH := handler.NewAuthHandler(logger, authSvc)
	providerH := handler.NewProviderHandler(logger, providerSvc)
	eventsH := handler.NewEventsHandler(logger, eventSvc)
	stripeH := handler.NewStripeHandler(logger, stripeSvc, eventSvc, workerPool)
	gitH := handler.NewGitHandler(logger, gitSvc, eventSvc, workerPool)

	// Router
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.CleanPath)
	router.Use(middleware.StripSlashes)
	router.Use(middleware.ThrottleBacklog(
		ThrottleMaxConcurrent,
		ThrottleMaxBacklog,
		time.Duration(ThrottleBacklogTimeout)*time.Second,
	))

	router.Route("/api", func(r chi.Router) {
		r.Use(middleware.Timeout(time.Duration(MaxRequestTime) * time.Second))

		// Public endpoints
		authH.RegisterRoutes(r)
		stripeH.RegisterRoutes(r)
		gitH.RegisterRoutes(r)
		healthH.RegisterRoutes(r)

		// JWT-protected endpoints
		r.Group(func(r chi.Router) {
			r.Use(auth.Jwt(authSvc))
			providerH.RegisterRoutes(r)
			eventsH.RegisterRoutes(r)
		})
	})

	// SSE stream
	router.With(auth.Jwt(authSvc)).
		Get("/api/events/stream", eventsH.SSE)

	return &App{
		db:              pg.DB,
		logger:          logger,
		cancelApp:       cancelApp,
		cancelWorkers:   cancelWorkers,
		authService:     authSvc,
		providerService: providerSvc,
		eventService:    eventSvc,
		stripeService:   stripeSvc,
		gitService:      gitSvc,
		workerPool:      workerPool,
		router:          router,
	}
}

// Run starts the HTTP server and blocks until it terminates.
// On return all background contexts are canceled, shutting down
// workers and the SSE event loop cleanly.
func (a *App) Run() {
	defer a.cancelWorkers()
	defer a.cancelApp()

	server := &http.Server{
		Addr:    ":8080",
		Handler: a.router,
	}
	a.logger.Info().Str("addr", server.Addr).Msg("[Main] server starting")
	if err := server.ListenAndServe(); err != nil {
		a.logger.Fatal().Stack().Err(err).Msg("[Main] server stopped unexpectedly")
	}
}
