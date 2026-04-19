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

type loginRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type authHandler struct {
	logger  *zerolog.Logger
	service *auth.AuthService
}

func NewAuthHandler(logger *zerolog.Logger, service *auth.AuthService) *authHandler {
	return &authHandler{
		logger:  logger,
		service: service,
	}
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequestBody
	if err := internal.DecodeBodyJson(w, r, &body); err != nil {
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

func (h *authHandler) RegisterRoutes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/refresh", h.refreshToken)
}
