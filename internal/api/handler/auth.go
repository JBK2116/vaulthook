package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/auth"
	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/helpers"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

// loginRequestBody holds the expected JSON fields for a login request.
type loginRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthHandler handles authentication-related HTTP requests.
// It relies on an AuthService for business logic and a logger for
// structured error reporting.
type AuthHandler struct {
	logger  *zerolog.Logger
	service *auth.AuthService
}

// NewAuthHandler returns an AuthHandler configured with the provided
// logger and AuthService.
func NewAuthHandler(logger *zerolog.Logger, service *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		logger:  logger,
		service: service,
	}
}

// login handles POST /api/login. It decodes the request body, delegates
// credential validation to the AuthService, and writes a JSON response
// containing an access token and a refresh token on success.
func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequestBody
	if err := helpers.DecodeBodyJSON(w, r, &body); err != nil {
		http.Error(w, err.Error(), err.Status)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	accessT, refreshT, err := h.service.Login(ctx, body.Email, body.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		h.logger.Error().Stack().Err(err).Msg(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, auth.NewAccessCookie(accessT, config.Envs.AccessTokenTTL*60))
	http.SetCookie(w, auth.NewRefreshCookie(refreshT, config.Envs.RefreshTokenTTL*60*60))
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	refreshT, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "missing refresh token cookie", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	if err := h.service.DeleteRefreshToken(ctx, refreshT.Value); err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Auth] error occurred deleting refresh token")
		http.Error(w, "error occurred logging out", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, auth.ExpiredAccessCookie())
	http.SetCookie(w, auth.ExpiredRefreshCookie())
	w.WriteHeader(http.StatusOK)
}

// refreshToken handles POST /api/refresh. It extracts a bearer token from
// the Authorization header, passes it to the AuthService for validation,
// and writes a JSON response with a new access token and refresh token.
func (h *AuthHandler) refreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	accessT, refreshT, err := h.service.RefreshToken(ctx, token.Value)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrTokenNotFound) || errors.Is(err, auth.ErrTokenKeyMissing) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		h.logger.Error().Stack().Err(err).Msg("[Auth] error refreshing token in refresh token endpoint")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, auth.NewAccessCookie(accessT, config.Envs.AccessTokenTTL*60))
	http.SetCookie(w, auth.NewRefreshCookie(refreshT, config.Envs.RefreshTokenTTL*60*60))
	w.WriteHeader(http.StatusOK)
}

// me handles GET /api/me. It extracts the access token from the request cookie,
// validates it, and returns 200 OK if the token is valid or 401 Unauthorized if not.
func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie("access_token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_, err = h.service.ValidateAccessToken(token.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// RegisterRoutes mounts the authentication endpoints onto the provided router.
//
// Endpoints:
//
//	POST /api/login
//	POST /api/logout
//	POST /api/refresh
//	GET  /api/me
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Post("/refresh", h.refreshToken)
	r.Get("/me", h.me)
}
