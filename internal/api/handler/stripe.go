package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	crypto "github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	stripeProvider "github.com/JBK2116/vaulthook/internal/providers/stripe"
	"github.com/JBK2116/vaulthook/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

var (
	ErrStripeUnhandledEvent = errors.New("unhandled stripe webhook event type")
	ErrStripeWebhookParsing = errors.New("error parsing webhook JSON")
)

// StripeHandler handles webhook logic for all events that reach `/webhooks/stripe endpoint`
type StripeHandler struct {
	logger       *zerolog.Logger
	service      *stripeProvider.StripeService
	eventService *events.EventService
	workerPool   *worker.WorkerPool
}

// NewStripeHandler returns an stripeHandler configured with the provided logger and service.
func NewStripeHandler(logger *zerolog.Logger, service *stripeProvider.StripeService, eventService *events.EventService, workerPool *worker.WorkerPool) *StripeHandler {
	return &StripeHandler{
		logger:       logger,
		service:      service,
		eventService: eventService,
		workerPool:   workerPool,
	}
}

// Receive handles /api/webhooks/stripe. It receives the incoming webhook,
// validates it using the signing key, saves it to the database if necessary and
// sets it's status for processing
func (h *StripeHandler) Receive(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	const maxBodyBytes = int64(65539)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Stripe] error receiving webhook request")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err := h.service.ValidateSecret(ctx, signatureHeader, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Stripe] failed to validate webhook secret")
		if errors.Is(err, crypto.ErrDecryption) {
			h.logger.Error().Err(err).Msg("[Stripe] failed to decrypt signing key")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if errors.As(err, &config.PgErr) {
			h.logger.Error().Err(err).Msg("[Stripe] database error validating webhook")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.logger.Debug().Msgf("[Stripe] event validated: %s", event.ID)
	headersJSON, err := json.Marshal(r.Header)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Stripe] failed to marshal webhook request headers")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	stripeWebhook, err := h.service.InsertWebhook(ctx, headersJSON, payload, event)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Stripe] error inserting webhook into database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// notify the frontend
	h.eventService.Send(stripeWebhook)
	// alert the workers to begin processing
	h.workerPool.Notify()
	// send a response back to stripe
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Explicitly set 200
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "queued",
		"id":     event.ID,
	}); err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Stripe] error encoding response")
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
