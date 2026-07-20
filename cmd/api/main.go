// Package main is the entry point for the vaulthook application.
//
// It wires together all sub-packages, bootstraps infrastructure dependencies,
// and starts the HTTP server. Logic here should remain minimal.
package main

import (
	"context"
	"log"
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

	// Environemnt Variables
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env file not found, using system environment: %v", err)
	}
	config.Init()

	// Infrastructure: Database & Logger
	dbCtx, cancelDB := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelDB()

	db, dbErr := config.NewPG(dbCtx)
	if dbErr != nil {
		panic(dbErr)
	}
	if err := db.Ping(dbCtx); err != nil {
		panic(err)
	}

	logger, err := config.NewLogger()
	if err != nil {
		panic(err)
	}

	// Application context
	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	// Repositories
	refreshTokenRepo := auth.NewRefreshTokenRepo(db.DB)
	providerRepo := providers.NewProviderRepo(db.DB)
	eventRepo := events.NewEventRepo(db.DB)
	stripeRepo := stripe.NewStripeRepo(db.DB)

	// Services
	authService := auth.NewAuthService(config.Envs.JWTSecret, config.Envs.AccessTokenTTL, config.Envs.RefreshTokenTTL, refreshTokenRepo, logger)
	providerService := providers.NewProviderService(providerRepo)
	eventService := events.NewEventService(logger, eventRepo)
	stripeService := stripe.NewStripeService(logger, stripeRepo, providerRepo)

	// Background Workers
	go eventService.Start(appCtx)
	workerCtx, cancelWorkers := context.WithCancel(appCtx)
	defer cancelWorkers()
	workerPool := worker.NewWorkerPool(workerCtx, eventService, logger, db.DB)

	// HTTP handlers
	authHandler := handler.NewAuthHandler(logger, authService)
	providerHandler := handler.NewProviderHandler(logger, providerService)
	eventHandler := handler.NewEventsHandler(logger, eventService)
	stripeHandler := handler.NewStripeHandler(logger, stripeService, eventService, workerPool)

	// Router
	router := chi.NewRouter()
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.CleanPath)
	router.Use(chimiddleware.StripSlashes)
	router.Use(chimiddleware.ThrottleBacklog(
		config.Envs.ThrottleMaxConcurrent,
		config.Envs.ThrottleMaxBacklog,
		time.Duration(config.Envs.ThrottleBacklogTimeout)*time.Second,
	))

	router.Route("/api", func(r chi.Router) {
		r.Use(chimiddleware.Timeout(time.Duration(config.Envs.MaxRequestTime) * time.Second))

		// Public endpoints
		authHandler.RegisterRoutes(r)
		stripeHandler.RegisterRoutes(r)

		// JWT-protected endpoints
		r.Group(func(r chi.Router) {
			r.Use(auth.Jwt(authService))
			providerHandler.RegisterRoutes(r)
			eventHandler.RegisterRoutes(r)
		})
	})

	// SSE stream
	router.With(auth.Jwt(authService)).
		Get("/api/events/stream", eventHandler.SSE)

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	logger.Info().Str("addr", server.Addr).Msg("[Main] server starting")
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal().Stack().Err(err).Msg("[Main] server stopped unexpectedly")
	}
}
