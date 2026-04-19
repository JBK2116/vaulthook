// Package api serves as the entry point to this application
//
// `main.go` is where the application is launched from.
// It serves as a wrapper, importing all necessary code from sub packages to build the application.
//
// This package should have as minimal code as possible.
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

func main() {
	ctx := context.Background()
	// Check the database connection
	ctxDB, cancelDB := context.WithTimeout(ctx, time.Second*10)
	defer cancelDB()
	db, err := db.NewPG(ctxDB)
	if err != nil {
		panic(err)
	}
	if err := db.Ping(ctxDB); err != nil {
		panic(err)
	}
	// Check the logger connection
	logger, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	// Configure the auth package
	refreshTokenRepo := auth.NewRefreshTokenRepo(db.DB)
	authService := auth.NewAuthService(config.Envs.TOKEN_SECRET, config.Envs.ACCESS_TOKEN_TLL, config.Envs.REFRESH_TOKEN_TTL, refreshTokenRepo, logger)
	authHandler := handler.NewAuthHandler(logger, authService)
	// Configure the router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.ThrottleBacklog(config.Envs.THROTTLE_MAX_CONCURRENT, config.Envs.THROTTLE_MAX_BACKLOG, time.Duration(config.Envs.THROTTLE_BACKLOG_TIMEOUT)*time.Second))
	r.Use(middleware.Timeout(time.Duration(config.Envs.MAX_REQUEST_TIME_LENGTH) * time.Second))
	r.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			// TODO: Add JWT Middleware
			authHandler.RegisterRoutes(r)
		})
	})
	// Run the server
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		logger.Fatal().Stack().Err(err).Msg("error starting http server")
	}
}
