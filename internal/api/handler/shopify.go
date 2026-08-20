package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/JBK2116/vaulthook/internal/providers/shopify"
	"github.com/JBK2116/vaulthook/internal/worker"
)

const (
	shopifyMaxBodyBytes = 5_000_000
)

// ShopifyHandler handles webhook logic for all events that reach `/webhooks/shopify`.
type ShopifyHandler struct {
	logger  *zerolog.Logger
	service *shopify.Service
	events  *events.Service
	pool    *worker.Pool
}

// NewShopifyHandler returns a ShopifyHandler configured with the provided logger and services.
func NewShopifyHandler(
	logger *zerolog.Logger,
	svc *shopify.Service,
	evSvc *events.Service,
	pool *worker.Pool,
) *ShopifyHandler {
	return &ShopifyHandler{
		logger:  logger,
		service: svc,
		events:  evSvc,
		pool:    pool,
	}
}

//nolint:funlen // function will be refactored later.
func (h *ShopifyHandler) Receive(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), receiveTimeout)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, shopifyMaxBodyBytes)
	payload, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		h.logger.Error().Err(readErr).Msg("[Shopify] error receiving webhook request")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	hid := r.Header.Get("X-Shopify-Webhook-Id") // unique webhook identifier
	if hid == "" {
		h.logger.Debug().Msg("[Shopify] error webhook missing X-Shopify-Webhook-Id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	headers, hErr := json.Marshal(r.Header)
	if hErr != nil {
		h.logger.Error().Err(hErr).Msg("[Shopify] failed to marshal webhook request headers")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	event := r.Header.Get("X-Shopify-Topic") // webhook event type
	if event == "" {
		h.logger.Debug().Msg("[Shopify] error webhook missing X-Shopify-Topic")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	signature := r.Header.Get("X-Shopify-Hmac-Sha256") // webhook verification signature
	sigErr := h.service.ValidateSecret(ctx, signature, payload)
	if sigErr != nil {
		if errors.Is(sigErr, crypto.ErrDecryption) {
			h.logger.Error().Err(sigErr).Msg("[Shopify] failed to decrypt signing key")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if errors.Is(sigErr, providers.ErrInvalidSignature) {
			h.logger.Debug().Err(sigErr).Msg("[Shopify] failed to validate webhook secret")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.logger.Error().Err(sigErr).Msg("[Shopify] error validating webhook")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	exists, exErr := h.service.Exists(ctx, hid)
	if exErr != nil {
		h.logger.Error().Err(exErr).Msg("[Shopify] error checking if webhook exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if exists {
		h.logger.Debug().Msgf("[Shopify] event already exists in database %s", hid)
		w.WriteHeader(http.StatusOK)
		return
	}
	hook, inErr := h.service.InsertWebhook(ctx, headers, payload, hid, event)
	if inErr != nil {
		if errors.Is(inErr, events.ErrHookExists) {
			h.logger.Error().Err(inErr).Msgf("[Shopify] event already exists in database %s", hid)
			w.WriteHeader(http.StatusOK)
			return
		}
		h.logger.Error().Err(inErr).Msg("[Shopify] error inserting webhook into database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// notify the frontend
	h.events.Send(hook)
	// alert the workers to begin processing
	h.pool.Notify()
	// send a response back to shopify
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if cErr := json.NewEncoder(w).Encode(map[string]string{
		"status": Queued,
		"id":     hid,
	}); cErr != nil {
		h.logger.Error().Err(cErr).Msg("[Shopify] error encoding response")
	}
}

// RegisterRoutes mounts the shopify endpoints onto the provided router.
//
// Endpoints:
//
//	POST /api/webhooks/shopify
func (h *ShopifyHandler) RegisterRoutes(r chi.Router) {
	r.Post("/webhooks/shopify", h.Receive)
}
