package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers/stripe"
	"github.com/JBK2116/vaulthook/internal/worker"
)

const (
	StripeMaxBodyBytes = int64(65539)
)

// StripeHandler handles webhook logic for all events that reach `/webhooks/stripe endpoint`.
type StripeHandler struct {
	logger   *zerolog.Logger
	svc      *stripe.Service
	eventSvc *events.EventService
	pool     *worker.Pool
}

// NewStripeHandler returns an stripeHandler configured with the provided logger and service.
func NewStripeHandler(
	logger *zerolog.Logger,
	svc *stripe.Service,
	eventSvc *events.EventService,
	pool *worker.Pool,
) *StripeHandler {
	return &StripeHandler{
		logger:   logger,
		svc:      svc,
		eventSvc: eventSvc,
		pool:     pool,
	}
}

// Receive handles /api/webhooks/stripe. It receives the incoming webhook,
// validates it using the signing key, saves it to the database if necessary and
// sets its status for processing.
//
//nolint:funlen // function will be refactored later.
func (h *StripeHandler) Receive(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), receiveTimeout)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, StripeMaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Stripe] error receiving webhook request")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err := h.svc.ValidateSecret(ctx, signatureHeader, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Stripe] failed to validate webhook secret")
		if errors.Is(err, crypto.ErrDecryption) {
			h.logger.Error().Err(err).Msg("[Stripe] failed to decrypt signing key")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if errors.As(err, &config.ErrPg) {
			h.logger.Error().Err(err).Msg("[Stripe] database error validating webhook")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	exists, err := h.svc.Exists(ctx, event.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Stripe] error checking if webhook exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if exists {
		h.logger.Debug().Msgf("[Stripe] event already exists in database: %s", event.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	h.logger.Debug().Msgf("[Stripe] event validated: %s", event.ID)
	headersJSON, err := json.Marshal(r.Header)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Stripe] failed to marshal webhook request headers")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	stripeWebhook, err := h.svc.InsertWebhook(ctx, headersJSON, payload, event)
	if err != nil {
		if errors.Is(err, events.ErrHookExists) {
			h.logger.Error().Err(err).Msgf("[Stripe] event already exists in database: %s", event.ID)
			w.WriteHeader(http.StatusOK)
			return
		}
		h.logger.Error().Err(err).Msg("[Stripe] error inserting webhook into database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// notify the frontend
	h.eventSvc.Send(stripeWebhook)
	// alert the workers to begin processing
	h.pool.Notify()
	// send a response back to stripe
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Explicitly set 200
	if cErr := json.NewEncoder(w).Encode(map[string]string{
		"status": "queued",
		"id":     event.ID,
	}); cErr != nil {
		h.logger.Error().Stack().Err(cErr).Msg("[Stripe] error encoding response")
	}
}

// RegisterRoutes mounts the stripe endpoints onto the provided router.
//
// Endpoints:
//
//	POST /api/webhooks/stripe
func (h *StripeHandler) RegisterRoutes(r chi.Router) {
	r.Post("/webhooks/stripe", h.Receive)
}
