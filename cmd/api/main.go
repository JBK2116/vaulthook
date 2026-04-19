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
	"github.com/JBK2116/vaulthook/internal/db"
	"github.com/JBK2116/vaulthook/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// main initializes infrastructure, wires dependencies, and starts the HTTP server.
//
// Any failure during initialization panics to prevent a misconfigured server
// from accepting traffic.
func main() {
	ctx := context.Background()
	// Initialize and verify the database connection pool.
	ctxDB, cancelDB := context.WithTimeout(ctx, time.Second*10)
	defer cancelDB()
	db, err := db.NewPG(ctxDB)
	if err != nil {
		panic(err)
	}
	if err := db.Ping(ctxDB); err != nil {
		panic(err)
	}
	// Initialize the application-wide logger.
	logger, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	// Wire auth dependencies.
	refreshTokenRepo := auth.NewRefreshTokenRepo(db.DB)
	authService := auth.NewAuthService(config.Envs.TOKEN_SECRET, config.Envs.ACCESS_TOKEN_TLL, config.Envs.REFRESH_TOKEN_TTL, refreshTokenRepo, logger)
	authHandler := handler.NewAuthHandler(logger, authService)
	// Configure the router with global middleware.
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.ThrottleBacklog(config.Envs.THROTTLE_MAX_CONCURRENT, config.Envs.THROTTLE_MAX_BACKLOG, time.Duration(config.Envs.THROTTLE_BACKLOG_TIMEOUT)*time.Second))
	r.Use(middleware.Timeout(time.Duration(config.Envs.MAX_REQUEST_TIME_LENGTH) * time.Second))
	// Register API routes.
	r.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			// TODO: Add JWT Middleware
			authHandler.RegisterRoutes(r)
		})
	})
	// Start the HTTP server.
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		logger.Fatal().Stack().Err(err).Msg("error starting http server")
	}
}
