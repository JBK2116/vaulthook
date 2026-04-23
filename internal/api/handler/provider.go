package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// configureRequestBody represents the request body for configuring a provider.
// type configureRequestBody struct {
// 	ID             string `json:"id"`
// 	SigningSecret  string `json:"signing_secret"`
// 	DestinationURL string `json:"destination_url"`
// }

// providerHandler handles HTTP requests for provider operations.
type providerHandler struct {
	logger  *zerolog.Logger
	service *providers.ProviderService
}

// NewProviderHandler returns a new providerHandler with the given logger and service.
func NewProviderHandler(logger *zerolog.Logger, service *providers.ProviderService) *providerHandler {
	return &providerHandler{
		logger:  logger,
		service: service,
	}
}

// GetAll handles GET /providers, returning all providers as JSON.
func (h *providerHandler) getAll(w http.ResponseWriter, r *http.Request) {
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

// RegisterRoutes mounts the provider endpoints onto the provided router.
//
// Endpoints:
//
//	GET /api/providers
func (h *providerHandler) RegisterRoutes(r chi.Router) {
	r.Get("/providers", h.getAll)
}
