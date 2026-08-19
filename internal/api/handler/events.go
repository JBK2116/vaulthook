package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
)

var (
	ErrClientDisconnected = errors.New("[Events] client disconnected from backend sse")
)

const (
	// SSEHeartbeatInterval is how often a comment-only frame is sent to
	// keep the SSE connection alive when no events are arriving.
	SSEHeartbeatInterval = time.Second * 15

	// requestTimeout bounds the service calls made by the non-SSE event
	// endpoints.
	requestTimeout = time.Second * 3
)

// EventsHandler handles SSE event logic for sending webhook related data to the frontend.
type EventsHandler struct {
	logger  *zerolog.Logger
	service *events.Service
}

// NewEventsHandler returns an EventsHandler configured with the provided logger and service.
func NewEventsHandler(logger *zerolog.Logger, service *events.Service) *EventsHandler {
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
	// client disconnection handling
	clientGone := r.Context().Done()
	// subscriber handling
	ch, unsub := h.service.Subscribe()
	defer unsub()
	rc := http.NewResponseController(w)
	// needs to be sent immediately to confirm connection and keep it running
	h.announceConnected(w, rc)
	h.logger.Info().Msg("[Events] client sse connected")
	// heartbeat for sse to keep it running when no events are coming in
	ticker := time.NewTicker(SSEHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-clientGone:
			h.logger.Error().Err(ErrClientDisconnected).Msg(ErrClientDisconnected.Error())
			return
		case <-ticker.C:
			if err := h.sendHeartbeat(w, rc); err != nil {
				return
			}
		case batch := <-ch:
			if err := h.streamBatch(w, rc, batch); err != nil {
				return
			}
		}
	}
}

// announceConnected sends the initial SSE frame that confirms the connection
// to the client. Errors are logged but not fatal, since the connection is
// kept alive regardless.
func (h *EventsHandler) announceConnected(w io.Writer, rc *http.ResponseController) {
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {}\n\n"); err != nil {
		h.logger.Error().Err(err).Msg("[Events] failed to send initial connection string to frontend via sse")
	}
	if err := rc.Flush(); err != nil {
		h.logger.Error().Err(err).Msg("[Events] failed to flush sse buffer")
	}
}

// sendHeartbeat writes a comment-only SSE frame to keep the connection alive
// and flushes the response. It returns an error when the connection should
// be closed.
func (h *EventsHandler) sendHeartbeat(w io.Writer, rc *http.ResponseController) error {
	if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
		h.logger.Error().Err(err).Msg("[Events] failed to send heartbeat")
		return err
	}
	if err := rc.Flush(); err != nil {
		h.logger.Error().Err(err).Msg("[Events] failed to flush sse buffer")
		return err
	}
	return nil
}

// streamBatch marshals the webhook batch and writes it as a single SSE data
// frame, followed by an overflow frame when events were dropped. It returns
// an error when the connection should be closed; marshal failures are logged
// and the batch is skipped.
func (h *EventsHandler) streamBatch(w io.Writer, rc *http.ResponseController, batch []model.Webhook) error {
	data, err := json.Marshal(batch)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Events] failed to marshal batch")
		return nil
	}
	// send the entire array under one 'data:' identifier
	if _, pErr := fmt.Fprintf(w, "data: %s\n\n", data); pErr != nil {
		h.logger.Error().Err(pErr).Msg("[Events] failed to send batch to frontend")
		return pErr
	}
	if fErr := rc.Flush(); fErr != nil {
		h.logger.Error().Err(fErr).Msg("[Events] failed to flush sse buffer")
		return fErr
	}
	// Signal overflow so the client can resync from the REST API
	if dropped := h.service.Dropped(); dropped > 0 {
		if _, dErr := fmt.Fprintf(w, "event: overflow\ndata: {\"count\":%d}\n\n", dropped); dErr != nil {
			h.logger.Error().Err(dErr).Msg("[Events] failed to send overflow event")
			return dErr
		}
		if oErr := rc.Flush(); oErr != nil {
			h.logger.Error().Err(oErr).Msg("[Events] failed to flush overflow")
			return oErr
		}
	}
	return nil
}

func (h *EventsHandler) search(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	var opts model.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		h.logger.Error().Err(err).Msg("[Events] error decoding payload")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := opts.Validate(); err != nil {
		h.logger.Warn().Err(err).Msg("[Events] error validating payload")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := h.service.Search(ctx, opts)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Events] error executing search")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(result)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Events] error marshaling response payload")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, wErr := w.Write(rBody); wErr != nil {
		h.logger.Error().Err(wErr).Msg("[Events] error sending webhook events to frontend")
		return
	}
}

// getAll handles GET /events, returning all webhook events as JSON.
func (h *EventsHandler) getAll(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
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
	evs, err := h.service.GetAll(ctx, cursor)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Events] error retrieving all webhook events from the database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rBody, err := json.Marshal(evs)
	if err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Events] error marshaling webhook events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, wErr := w.Write(rBody); wErr != nil {
		h.logger.Error().Stack().Err(wErr).Msg("[Events] error sending webhook events to frontend")
		return
	}
}

// getStats retrieves statistics for webhooks.
func (h *EventsHandler) getStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
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
	if _, wErr := w.Write(rBody); wErr != nil {
		h.logger.Error().Stack().Err(wErr).Msg("[Events] error sending stats to frontend")
		return
	}
}

// replayEvent sets the webhook with the provided id to status 'queued' allowing it to be replayed by queue workers.
func (h *EventsHandler) replayEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
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
	r.Post("/events", h.search)
	r.Post("/events/{id}/replay", h.replayEvent)
	r.Get("/events/stats", h.getStats)
}
