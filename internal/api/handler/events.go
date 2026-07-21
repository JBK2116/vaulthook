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
	ErrClientDisconnected = errors.New("[Events] client disconnected from backend sse")
)

// EventsHandler handles SSE event logic for sending webhook related data to the frontend.
type EventsHandler struct {
	logger  *zerolog.Logger
	service *events.EventService
}

// NewEventsHandler returns an EventsHandler configured with the provided logger and service.
func NewEventsHandler(logger *zerolog.Logger, service *events.EventService) *EventsHandler {
	return &EventsHandler{
		logger:  logger,
		service: service,
	}
}

// SSE establishes a long-lived Server-Sent Events connection with the client.
// It subscribes to the event service hub and streams incoming webhook events
// to the connected client in real time. The connection is closed when the
// client disconnects.
func (h *EventsHandler) SSE(w http.ResponseWriter, r *http.Request) {
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
		h.logger.Error().Err(err).Msg("[Events] failed to send initial connection string to frontend via sse")
	}
	if err := rc.Flush(); err != nil {
		h.logger.Error().Err(err).Msg("[Events] failed to flush sse buffer")
	}
	h.logger.Info().Msg("[Events] client sse connected")
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
				h.logger.Error().Err(err).Msg("[Events] failed to send heartbeat")
				return
			}
			if err := rc.Flush(); err != nil {
				h.logger.Error().Err(err).Msg("[Events] failed to flush sse buffer")
				return
			}
		case batch := <-ch:
			data, err := json.Marshal(batch)
			if err != nil {
				h.logger.Error().Err(err).Msg("[Events] failed to marshal batch")
				continue
			}
			// send the entire array under one 'data:' identifier
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				h.logger.Error().Err(err).Msg("[Events] failed to send batch to frontend")
				return
			}
			if err := rc.Flush(); err != nil {
				h.logger.Error().Err(err).Msg("[Events] failed to flush sse buffer")
				return
			}
			// Signal overflow so the client can resync from the REST API
			if dropped := h.service.Dropped(); dropped > 0 {
				if _, err := fmt.Fprintf(w, "event: overflow\ndata: {\"count\":%d}\n\n", dropped); err != nil {
					h.logger.Error().Err(err).Msg("[Events] failed to send overflow event")
					return
				}
				if err := rc.Flush(); err != nil {
					h.logger.Error().Err(err).Msg("[Events] failed to flush overflow")
					return
				}
			}
		}
	}
}

// getAll handles GET /events, returning all webhook events as JSON.
func (h *EventsHandler) getAll(w http.ResponseWriter, r *http.Request) {
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
		h.logger.Error().Err(err).Msg("[Events] error retrieving all webhook events from the database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(events)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Events] error marshaling webhook events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rBody); err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Events] error sending webhook events to frontend")
		return
	}
}

// getStats retrieves statistics for webhooks
func (h *EventsHandler) getStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	stats, err := h.service.GetStats(ctx)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Events] error retrieving stats for webhooks")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(stats)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Events] error marshaling stats")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rBody); err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Events] error sending stats to frontend")
		return
	}
}

// replayEvent sets the webhook with the provided id to status 'queued' allowing it to be replayed by queue workers
func (h *EventsHandler) replayEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	if err := h.service.ReplayEvent(ctx, id); err != nil {
		if errors.Is(err, events.ErrInvalidUUID) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.logger.Error().Stack().Err(err).Msg("[Events] error replaying event")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// RegisterRoutes mounts the webhook event related endpoints onto the provided router
//
// NOTE: SSE mounting is handled explicitly in main as it requires special configuration.
func (h *EventsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/events", h.getAll)
	r.Post("/events/{id}/replay", h.replayEvent)
	r.Get("/events/stats", h.getStats)
}
