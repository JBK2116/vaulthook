package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers/github"
	"github.com/JBK2116/vaulthook/internal/worker"
)

const (
	gitMaxBodyBytes = 25_000_000
	Queued          = "queued"

	// receiveTimeout bounds the time spent receiving and persisting a webhook.
	receiveTimeout = time.Second * 3
)

// GitHandler handles webhook logic for all events that reach `/webhooks/github`.
type GitHandler struct {
	logger *zerolog.Logger
	gitSvc *github.Service
	evSvc  *events.Service
	pool   *worker.Pool
}

// NewGitHandler returns an GitHandler configured with the provided logger and services.
func NewGitHandler(
	logger *zerolog.Logger,
	svc *github.Service,
	evSvc *events.Service,
	pool *worker.Pool,
) *GitHandler {
	return &GitHandler{
		logger: logger,
		gitSvc: svc,
		evSvc:  evSvc,
		pool:   pool,
	}
}

//nolint:funlen // function will be refactored later
func (h *GitHandler) Receive(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), receiveTimeout)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, gitMaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Github] error receiving webhook request")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	hid := r.Header.Get("X-Github-Hook-Id")
	if hid == "" {
		h.logger.Debug().Msg("[Github] error webhook missing X-GitHub-Hook-ID")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	event := r.Header.Get("X-Github-Event")
	if event == "" {
		h.logger.Debug().Msg("[Github] error webhook missing X-GitHub-Event")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	signature := r.Header.Get("X-Hub-Signature-256")
	if secErr := h.gitSvc.ValidateSecret(ctx, signature, payload); secErr != nil {
		h.logger.Error().Err(secErr).Msg("[Github] failed to validate webhook secret")
		if errors.Is(secErr, crypto.ErrDecryption) {
			h.logger.Error().Err(secErr).Msg("[Github] failed to decrypt signing key")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if errors.As(secErr, &config.ErrPg) {
			h.logger.Error().Err(secErr).Msg("[Github] database error validating webhook")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	evID := r.Header.Get("X-Github-Delivery")
	exists, err := h.gitSvc.Exists(ctx, evID)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Github] error checking if webhook exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if exists {
		h.logger.Debug().Msgf("[Github] event already exists in database: %s", evID)
		w.WriteHeader(http.StatusOK)
		return
	}
	h.logger.Debug().Msgf("[Github] event validated: %s", hid)
	headers, err := json.Marshal(r.Header)
	if err != nil {
		h.logger.Error().Err(err).Msg("[Github] failed to marshal webhook request headers")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hook, err := h.gitSvc.InsertWebhook(ctx, headers, payload, hid, event)
	if err != nil {
		if errors.Is(err, events.ErrHookExists) {
			h.logger.Error().Err(err).Msgf("[Github] event already exists in database: %s", evID)
			w.WriteHeader(http.StatusOK)
			return
		}
		h.logger.Error().Err(err).Msg("[Github] error inserting webhook into database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// notify the frontend
	h.evSvc.Send(hook)
	// alert the workers to begin processing
	h.pool.Notify()
	// send a response back to github
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Explicitly set 200
	if cErr := json.NewEncoder(w).Encode(map[string]string{
		"status": Queued,
		"id":     hid,
	}); cErr != nil {
		h.logger.Error().Stack().Err(cErr).Msg("[Github] error encoding response")
	}
}

// RegisterRoutes mounts the GitHub endpoints onto the provided router.
func (h *GitHandler) RegisterRoutes(r chi.Router) {
	r.Post("/webhooks/github", h.Receive)
}
