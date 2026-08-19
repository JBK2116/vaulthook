package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/health"
)

const (
	// healthTimeout bounds the amount of time spent conducting a health check response.
	healthTimeout = time.Second * 3
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	svc    *health.HealthService
	logger *zerolog.Logger
}

// NewHealthHandler returns a `HealthHandler` configured with the provided svc.
func NewHealthHandler(svc *health.HealthService, logger *zerolog.Logger) *HealthHandler {
	return &HealthHandler{
		svc:    svc,
		logger: logger,
	}
}

// getHealth handles GET /health requests providing a health check covering the applications' status.
func (h *HealthHandler) getHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()

	check := h.svc.GetHealthCheck(ctx)
	rBody, err := json.Marshal(check)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Health] error marshaling health check")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if check.Status == health.HEALTHY {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if _, wErr := w.Write(rBody); wErr != nil {
		h.logger.Error().Err(wErr).Msg("[Health] error sending health check response")
		return
	}
}

// RegisterRoutes mounts the health check endpoint onto the provided router.
func (h *HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.getHealth)
}
