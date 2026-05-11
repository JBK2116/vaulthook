// Package main is the entry point for the vaulthook application.
//
// It wires together all sub-packages, bootstraps infrastructure dependencies,
// and starts the HTTP server. Logic here should remain minimal.
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/api/handler"
	"github.com/JBK2116/vaulthook/internal/auth"
	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/JBK2116/vaulthook/internal/providers/stripe"
	"github.com/JBK2116/vaulthook/internal/worker"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

// main initializes infrastructure, wires dependencies, and starts the HTTP server.
//
// Any failure during initialization panics to prevent a misconfigured server
// from accepting traffic.
func main() {
	// initialize the environment variables
	godotenv.Load()
	config.Init()
	ctx := context.Background()
	// Initialize and verify the database connection pool.
	ctxDB, cancelDB := context.WithTimeout(ctx, time.Second*10)
	defer cancelDB()
	db, err := config.NewPG(ctxDB)
	if err != nil {
		panic(err)
	}
	if err := db.Ping(ctxDB); err != nil {
		panic(err)
	}
	// Initialize the application-wide logger.
	logger, err := config.NewLogger()
	if err != nil {
		panic(err)
	}
	// Wire auth dependencies.
	refreshTokenRepo := auth.NewRefreshTokenRepo(db.DB)
	authService := auth.NewAuthService(config.Envs.JWTSecret, config.Envs.AccessTokenTTL, config.Envs.RefreshTokenTTL, refreshTokenRepo, logger)
	authHandler := handler.NewAuthHandler(logger, authService)
	// Wire provider dependencies.
	providerRepo := providers.NewProviderRepo(db.DB)
	providerService := providers.NewProviderService(providerRepo)
	providerHandler := handler.NewProviderHandler(logger, providerService)
	// Wire events & sse dependencies.
	eventRepo := events.NewEventRepo(db.DB)
	eventService := events.NewEventService(logger, eventRepo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go eventService.Start(ctx)
	eventHandler := handler.NewEventsHandler(logger, eventService)
	// Wire worker dependencies.
	ctxWorker, cancelCtxWorker := context.WithCancel(ctx)
	defer cancelCtxWorker()
	workerPool := worker.NewWorkerPool(ctxWorker, eventService, logger, db.DB)
	// Wire stripe dependencies
	stripeRepo := stripe.NewStripeRepo(db.DB)
	stripeService := stripe.NewStripeService(logger, stripeRepo, providerRepo)
	stripeHandler := handler.NewStripeHandler(logger, stripeService, eventService, workerPool)
	// Configure the router with global middleware.
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.CleanPath)
	r.Use(chimiddleware.StripSlashes)
	r.Use(chimiddleware.ThrottleBacklog(config.Envs.ThrottleMaxConcurrent, config.Envs.ThrottleMaxBacklog, time.Duration(config.Envs.ThrottleBacklogTimeout)*time.Second))
	// Register API routes.
	r.Route("/api", func(r chi.Router) {
		r.Use(chimiddleware.Timeout(time.Duration(config.Envs.MaxRequestTime) * time.Second))
		// User Authentication related routes.
		authHandler.RegisterRoutes(r)
		// Stripe related routes
		stripeHandler.RegisterRoutes(r)
		// Protected routes.
		r.Group(func(r chi.Router) {
			r.Use(auth.Jwt(authService))
			providerHandler.RegisterRoutes(r)
			eventHandler.RegisterRoutes(r)
		})
	})
	r.With(auth.Jwt(authService)).
		Get("/api/events/stream", eventHandler.SSE)
	// Start the HTTP server.
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		logger.Fatal().Stack().Err(err).Msg("error starting http server")
	}
}
