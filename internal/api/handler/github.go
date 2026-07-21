package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers/github"
	"github.com/JBK2116/vaulthook/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// GitHandler handles webhook logic for all events that reach `/webhooks/github`
type GitHandler struct {
	logger       *zerolog.Logger
	service      *github.GitService
	eventService *events.EventService
	workerPool   *worker.WorkerPool
}

// NewGitHandler returns an GitHandler configured with the provided logger and services.
func NewGitHandler(logger *zerolog.Logger, service *github.GitService, eventService *events.EventService, workerPool *worker.WorkerPool) *GitHandler {
	return &GitHandler{
		logger:       logger,
		service:      service,
		eventService: eventService,
		workerPool:   workerPool,
	}
}

func (h *GitHandler) Receive(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()
	const maxBodyBytes = int64(25_000_000)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Github] error receiving webhook request")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	hid := r.Header.Get("X-GitHub-Hook-ID")
	if hid == "" {
		h.logger.Debug().Msg("[Github] error webhook missing X-GitHub-Hook-ID")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	event := r.Header.Get("X-GitHub-Event")
	if event == "" {
		h.logger.Debug().Msg("[Github] error webhook missing X-GitHub-Event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	signature := r.Header.Get("X-Hub-Signature-256")
	if secErr := h.service.ValidateSecret(ctx, signature, payload); secErr != nil {
		h.logger.Error().Err(secErr).Msg("[Github] failed to validate webhook secret")
		if errors.Is(secErr, crypto.ErrDecryption) {
			h.logger.Error().Err(secErr).Msg("[Github] failed to decrypt signing key")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if errors.As(secErr, &config.PgErr) {
			h.logger.Error().Err(secErr).Msg("[Github] database error validating webhook")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.logger.Debug().Msgf("[Github] event validated: %s", hid)
	headers, err := json.Marshal(r.Header)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Github] failed to marshal webhook request headers")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hook, err := h.service.InsertWebhook(ctx, headers, payload, hid, event)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Github] error inserting webhook into database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// notify the frontend
	h.eventService.Send(hook)
	// alert the workers to begin processing
	h.workerPool.Notify()
	// send a response back to stripe
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Explicitly set 200
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "queued",
		"id":     hid,
	}); err != nil {
		h.logger.Error().Stack().Err(err).Msg("[Github] error encoding response")
	}

}

// RegisterRoutes mounts the github endpoints onto the provided router
func (h *GitHandler) RegisterRoutes(r chi.Router) {
	r.Post("/webhooks/github", h.Receive)
}
