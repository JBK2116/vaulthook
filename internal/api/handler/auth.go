package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal"
	"github.com/JBK2116/vaulthook/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// loginRequestBody holds the expected JSON fields for a login request.
type loginRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// tokenResponse holds the access and refresh tokens returned after a
// successful login or token refresh.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// authHandler handles authentication-related HTTP requests.
// It relies on an AuthService for business logic and a logger for
// structured error reporting.
type authHandler struct {
	logger  *zerolog.Logger
	service *auth.AuthService
}

// NewAuthHandler returns an authHandler configured with the provided
// logger and AuthService.
func NewAuthHandler(logger *zerolog.Logger, service *auth.AuthService) *authHandler {
	return &authHandler{
		logger:  logger,
		service: service,
	}
}

// login handles POST /login. It decodes the request body, delegates
// credential validation to the AuthService, and writes a JSON response
// containing an access token and a refresh token on success.
func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequestBody
	if err := internal.DecodeBodyJSON(w, r, &body); err != nil {
		http.Error(w, err.Error(), err.Status)
		return
	}
	ctx := r.Context()
	accessT, refreshT, err := h.service.Login(ctx, body.Email, body.Password)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg(err.Error())
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := tokenResponse{
		AccessToken:  accessT,
		RefreshToken: refreshT,
	}
	rBody, err := json.Marshal(&response)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error marshaling token response struct in login endpoint")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rBody); err != nil {
		h.logger.Error().Stack().Err(err).Msg("error writing body to login endpoint")
		return
	}
}

// refreshToken handles POST /refresh. It extracts a bearer token from
// the Authorization header, passes it to the AuthService for validation,
// and writes a JSON response with a new access token and refresh token.
func (h *authHandler) refreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := internal.ExtractBearerToken(r)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error extracting bearer token in refresh token endpoint")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()

	accessT, refreshT, err := h.service.RefreshToken(ctx, token)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error refreshing token in refresh token endpoint")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	response := tokenResponse{
		AccessToken:  accessT,
		RefreshToken: refreshT,
	}
	rBody, err := json.Marshal(&response)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error marshaling token response struct in login endpoint")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rBody); err != nil {
		h.logger.Error().Stack().Err(err).Msg("error writing body to login endpoint")
		return
	}
}

// RegisterPublicRoutes mounts the authentication endpoints onto the provided router.
func (h *authHandler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/refresh", h.refreshToken)
}
