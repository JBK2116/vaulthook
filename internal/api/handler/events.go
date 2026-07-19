package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
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
func (h *eventsHandler) SSE(w http.ResponseWriter, r *http.Request) {
	// sse headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	if config.Envs.IsDevelopment {
		// needed since client sse doesn't work optimally through vite proxy
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	// client disconncection handling
	clientGone := r.Context().Done()
	// subcriber handling
	ch, unsub := h.service.Subscribe()
	defer unsub()
	rc := http.NewResponseController(w)
	// needs to be sent immediately to confirm connection and keep it running
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {}\n\n"); err != nil {
		h.logger.Error().Err(err).Msg("failed to send initial connection string to frontend via sse")
	}
	if err := rc.Flush(); err != nil {
		h.logger.Error().Err(err).Msg("failed to flush sse buffer")
	}
	h.logger.Info().Msg("client sse connected")
	// heartbeat for sse to keep it running when no events are coming in
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-clientGone:
			h.logger.Error().Err(ErrClientDisconnected).Msg(ErrClientDisconnected.Error())
			return
		case <-ticker.C:
			// heartbeat
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				h.logger.Error().Err(err).Msg("failed to send heartbeat")
				return
			}
			if err := rc.Flush(); err != nil {
				h.logger.Error().Err(err).Msg("failed to flush sse buffer")
				return
			}
		case batch := <-ch:
			data, err := json.Marshal(batch)
			if err != nil {
				h.logger.Error().Err(err).Msg("failed to marshal batch")
				continue
			}
			// send the entire array under one 'data:' identifier
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				h.logger.Error().Err(err).Msg("failed to send batch to frontend")
				return
			}
			if err := rc.Flush(); err != nil {
				h.logger.Error().Err(err).Msg("failed to flush sse buffer")
				return
			}
		}
	}
}

// getAll handles GET /events, returning all webhook events as JSON.
func (h *eventsHandler) getAll(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	var cursor *time.Time
	if c := r.URL.Query().Get("cursor"); c != "" {
		t, err := time.Parse(time.RFC3339Nano, c)
		if err != nil {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		cursor = &t
	}
	events, err := h.service.GetAll(ctx, cursor)
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

// replayEvent sets the webhook with the provided id to status 'queued' allowing it to be replayed by queue workers
func (h *eventsHandler) replayEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	if err := h.service.ReplayEvent(ctx, id); err != nil {
		if errors.Is(err, events.ErrInvalidUUID) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.logger.Error().Stack().Err(err).Msg("error replaying event")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// RegisterRoutes mounts the webhook event related endpoints onto the provided router
//
// NOTE: SSE mounting is handled explicitly in main as it requires special configuration.
//
// Endpoints:
//
//	GET /api/events
func (h *eventsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/events", h.getAll)
	r.Post("/events/{id}/replay", h.replayEvent)
}
