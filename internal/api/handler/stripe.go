package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	crypto "github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/db"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers"
	stripeProvider "github.com/JBK2116/vaulthook/internal/providers/stripe"
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
}

// NewStripeHandler returns an stripeHandler configured with the provided logger and service.
func NewStripeHandler(logger *zerolog.Logger, service *stripeProvider.StripeService, eventService *events.EventService) *StripeHandler {
	return &StripeHandler{
		logger:       logger,
		service:      service,
		eventService: eventService,
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
		h.logger.Error().Err(err).Msg("error receiving webhook request")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err := h.service.ValidateSecret(ctx, signatureHeader, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to validate stripe webhook secret")
		if errors.Is(err, crypto.ErrDecryption) {
			h.logger.Error().Err(err).Msg(fmt.Sprintf("failed to decrypt signing key for provider: %s", providers.Stripe))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if errors.As(err, &db.PgErr) {
			h.logger.Error().Err(err).Msg("database error validating webhook")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.logger.Debug().Msgf("stripe event validated: %s", event.ID)
	headersJSON, err := json.Marshal(r.Header)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal stripe webhook request headers")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	stripeWebhook, err := h.service.InsertWebhook(ctx, headersJSON, payload, event)
	if err != nil {
		h.logger.Error().Err(err).Msg("error inserting webhook into database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.eventService.Send(stripeWebhook)
	w.WriteHeader(http.StatusOK)
}

// RegisterRoutes mounts the stripe endpoints onto the provided router.
//
// Endpoints:
//
//	POST /api/webhooks/stripe
func (h *StripeHandler) RegisterRoutes(r chi.Router) {
	r.Post("/webhooks/stripe", h.Receive)
}
