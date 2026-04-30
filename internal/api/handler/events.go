package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

var (
	ErrClientDisconnected = errors.New("client disconnected from backend sse ")
)

// eventsHandler handles sse event logic for sending webhook related data to the frontend
type eventsHandler struct {
	logger  *zerolog.Logger
	service *events.EventService
}

// NewEventsHandler returns an eventsHandler configured with the provided logger and service.
func NewEventsHandler(logger *zerolog.Logger, service *events.EventService) *eventsHandler {
	return &eventsHandler{
		logger:  logger,
		service: service,
	}
}

// sse establishes a long-lived Server-Sent Events connection with the client.
// It subscribes to the event service hub and streams incoming webhook events
// to the connected client in real time. The connection is closed when the
// client disconnects.
func (h *eventsHandler) sse(w http.ResponseWriter, r *http.Request) {
	// sse headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	// client disconncection handling
	clientGone := r.Context().Done()
	// subscriber handling
	ch, unsub := h.service.Subscribe()
	rc := http.NewResponseController(w)
	// needs to be sent immediately to confirm connection and keep it running
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {}\n\n"); err != nil {
		h.logger.Error().Err(err).Msg("failed to send initial connection string to frontend via sse")
	}
	if err := rc.Flush(); err != nil {
		h.logger.Error().Err(err).Msg("failed to flush sse buffer")
	}
	h.logger.Info().Msg("client sse connected")

	// heartbeat for sse to keep it running
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-clientGone:
			h.logger.Error().Err(ErrClientDisconnected).Msg(ErrClientDisconnected.Error())
			unsub()
			return
		case <-ticker.C:
			// heartbeat (comment line in SSE spec)
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				h.logger.Error().Err(err).Msg("failed to send heartbeat")
				continue
			}
			if err := rc.Flush(); err != nil {
				h.logger.Error().Err(err).Msg("failed to flush sse buffer")
				continue
			}
		case event := <-ch:
			data, err := json.Marshal(event)
			if err != nil {
				h.logger.Error().Err(err).Msg("failed to marshal event to send to frontend")
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				h.logger.Error().Err(err).Msg("failed to send webhook event to frontend via sse")
				continue
			}
			if err := rc.Flush(); err != nil {
				h.logger.Error().Err(err).Msg("failed to flush sse buffer")
				continue
			}
		}
	}
}

// getAll handles GET /events, returning all webhook events as JSON.
func (h *eventsHandler) getAll(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	events, err := h.service.GetAll(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("error retrieving all webhook events from the database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(events)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("error marshaling webhook events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rBody); err != nil {
		h.logger.Error().Stack().Err(err).Msg("error sending webhook events to frontend")
		return
	}
}

// RegisterRoutes mounts the webhook event related endpoints onto the provided router
//
// Endpoints:
//
//	GET /api/events/stream
//	GET /api/events
func (h *eventsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/events/stream", h.sse)
	r.Get("/events", h.getAll)
}
