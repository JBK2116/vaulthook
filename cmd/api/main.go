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
	"github.com/JBK2116/vaulthook/internal/middleware"
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
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	config.Init()
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
	authService := auth.NewAuthService(config.Envs.JWTSecret, config.Envs.AccessTokenTTL, config.Envs.RefreshTokenTTL, refreshTokenRepo, logger)
	authHandler := handler.NewAuthHandler(logger, authService)
	// Configure the router with global middleware.
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.CleanPath)
	r.Use(chimiddleware.StripSlashes)
	r.Use(chimiddleware.ThrottleBacklog(config.Envs.ThrottleMaxConcurrent, config.Envs.ThrottleMaxBacklog, time.Duration(config.Envs.ThrottleBacklogTimeout)*time.Second))
	r.Use(chimiddleware.Timeout(time.Duration(config.Envs.MaxRequestTime) * time.Second))
	// Register API routes.
	r.Route("/api", func(r chi.Router) {
		// User Authentication related routes.
		authHandler.RegisterPublicRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Jwt(authService))
		})
	})
	// Start the HTTP server.
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		logger.Fatal().Stack().Err(err).Msg("error starting http server")
	}
}
