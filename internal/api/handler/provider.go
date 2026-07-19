package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/helpers"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type configureRequestBody struct {
	SigningSecret  string `json:"signing_secret"`
	DestinationURL string `json:"destination_url"`
}

// ProviderHandler handles HTTP requests for provider operations.
type ProviderHandler struct {
	logger  *zerolog.Logger
	service *providers.ProviderService
}

// NewProviderHandler returns a new providerHandler with the given logger and service.
func NewProviderHandler(logger *zerolog.Logger, service *providers.ProviderService) *ProviderHandler {
	return &ProviderHandler{
		logger:  logger,
		service: service,
	}
}

// GetAll handles GET /providers, returning all providers as JSON.
func (h *ProviderHandler) getAll(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	providers, err := h.service.GetAll(ctx)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error retrieving all providers from database")
		http.Error(w, "error retrieving all providers from database", http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(providers)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error marshaling providers")
		http.Error(w, "error marshaling providers", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rBody); err != nil {
		h.logger.Error().Stack().Err(err).Msg("error sending providers json to frontend")
		return
	}
}

// Configure handles PATCH /providers/{id}, updating a provider's signing secret
// and destination URL, setting is_configured to true on success.
func (h *ProviderHandler) configure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body configureRequestBody
	if err := helpers.DecodeBodyJSON(w, r, &body); err != nil {
		http.Error(w, err.Error(), err.Status)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	provider, err := h.service.Configure(ctx, id, body.SigningSecret, body.DestinationURL)
	if err != nil {
		if errors.Is(err, providers.ErrMissingSigningSecret) || errors.Is(err, providers.ErrMissingDestination) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.logger.Error().Stack().Err(err).Msg("error occurred updating provider config")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(provider)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error marshaling provider")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rBody); err != nil {
		h.logger.Error().Stack().Err(err).Msg("error sending providers json to frontend")
		return
	}
}

// RegisterRoutes mounts the provider endpoints onto the provided router.
//
// Endpoints:
//
//	GET /api/providers
//	PATCH /api/providers/{id}
func (h *ProviderHandler) RegisterRoutes(r chi.Router) {
	r.Get("/providers", h.getAll)
	r.Patch("/providers/{id}", h.configure)
}
