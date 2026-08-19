package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/helpers"
	"github.com/JBK2116/vaulthook/internal/providers"
)

// configureRequestBody is a dto used to update a providers configuration variables.
type configureRequestBody struct {
	SigningSecret  string `json:"signing_secret"`
	DestinationURL string `json:"destination_url"`
	MaxRetries     int    `json:"max_retries"`
	MaxReqSecond   int    `json:"max_req_second"`
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

const (
	// getTimeout bounds the amount of time spent conducting a get providers response.
	getTimeout = time.Second * 2
	// configureTimeout bounds the amount of time spent conducting a configure providers response.
	configureTimeout = time.Second * 2
)

// GetAll handles GET /providers, returning all providers as JSON.
func (h *ProviderHandler) getAll(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), getTimeout)
	defer cancel()
	provs, err := h.service.GetAll(ctx)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Provider] error retrieving all providers from database")
		http.Error(w, "error retrieving all providers from database", http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(provs)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Provider] error marshaling providers")
		http.Error(w, "error marshaling providers", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, wErr := w.Write(rBody); wErr != nil {
		h.logger.Error().Stack().Err(wErr).Msg("[Provider] error sending providers json to frontend")
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
	ctx, cancel := context.WithTimeout(r.Context(), configureTimeout)
	defer cancel()
	provider, err := h.service.Configure(
		ctx,
		id,
		body.SigningSecret,
		body.DestinationURL,
		body.MaxRetries,
		body.MaxReqSecond,
	)
	if err != nil {
		if errors.Is(err, providers.ErrMissingSigningSecret) ||
			errors.Is(err, providers.ErrMissingDestination) ||
			errors.Is(err, providers.ErrInvalidRetryCount) ||
			errors.Is(err, providers.ErrInvalidReqSecond) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.logger.Error().Stack().Err(err).Msg("[Provider] error occurred updating provider config")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(provider)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Provider] error marshaling provider")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, wErr := w.Write(rBody); wErr != nil {
		h.logger.Error().Stack().Err(wErr).Msg("[Provider] error sending providers json to frontend")
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
