package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/rs/zerolog"
)

// SetForwardHeaders applies the appropriate Github-specific HTTP headers
// to the outgoing forward request. Only a curated allowlist of headers
// from the original incoming webhook/http request are forwarded.
func SetForwardHeaders(r *http.Request, headers []byte) error {
	allowed := map[string]struct{}{
		"Content-Type":        {},
		"X-Hub-Signature-256": {},
		"X-Hub-Signature":     {},
		"X-Github-Event":      {},
		"X-Github-Delivery":   {},
		"X-Github-Hook-Id":    {},
		"User-Agent":          {},
	}
	var parsed map[string][]string
	if err := json.Unmarshal(headers, &parsed); err != nil {
		return err
	}
	for k, vals := range parsed {
		if _, ok := allowed[http.CanonicalHeaderKey(k)]; ok {
			for _, v := range vals {
				r.Header.Add(k, v)
			}
		}
	}
	return nil
}

// ErrInvalidSignature is returned when a webhook signature does not match
// the expected HMAC.
var ErrInvalidSignature = errors.New("invalid GitHub webhook signature")

// GitService provides the main business logic for handling webhook events
// pertaining to the GitHub provider.
type GitService struct {
	logger       *zerolog.Logger
	eventRepo    *events.EventRepo
	providerRepo *providers.ProviderRepo
}

// NewGitService returns a GitService configured with the provided logger,
// event repository, and provider repository.
func NewGitService(logger *zerolog.Logger, eventRepo *events.EventRepo, providerRepo *providers.ProviderRepo) *GitService {
	return &GitService{
		logger:       logger,
		eventRepo:    eventRepo,
		providerRepo: providerRepo,
	}
}

// ValidateSecret receives a GitHub signature from the `X-Hub-Signature-256`
// header and ensures that it matches the secret key used for GitHub endpoints.
func (s *GitService) ValidateSecret(ctx context.Context, signature string, payload []byte) (err error) {
	prov, err := providers.Cache.Get(ctx, model.Github)
	if err != nil {
		return err
	}
	decrypted, err := crypto.DecryptSigningKey(prov.SigningSecret)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(decrypted))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}

// InsertWebhook creates and stores a Github webhook using the provided data request
func (s *GitService) InsertWebhook(ctx context.Context, headers []byte, payload []byte, id string, event string) (model.Webhook, error) {
	prov, err := providers.Cache.Get(ctx, model.Github)
	if err != nil {
		return model.Webhook{}, err
	}
	params := model.CreateWebhookParams{
		ProviderID:  prov.ID,
		Provider:    string(model.Github),
		EventID:     &id,
		EventType:   event,
		Headers:     headers,
		Payload:     payload,
		ForwardedTo: prov.DestinationURL,
		ReceivedAt:  time.Now().UTC(),
	}
	hook, err := s.eventRepo.InsertWebhook(ctx, params)
	if err != nil {
		return model.Webhook{}, err
	}
	return hook, nil
}

// Exists checks if a github webhook with the provided event_id already exists in the database
func (s *GitService) Exists(ctx context.Context, evID string) (bool, error) {
	prov, err := providers.Cache.Get(ctx, model.Github)
	if err != nil {
		return false, err
	}
	exists, err := s.eventRepo.Exists(ctx, prov.ID, evID)
	return exists, err
}
