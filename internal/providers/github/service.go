package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/rs/zerolog"
)

// GitService provides the main business logic for handling webhook events pertaining to the git provider
type GitService struct {
	logger       *zerolog.Logger
	repo         *GitRepo
	providerRepo *providers.ProviderRepo
}

// NewGitService returns a Git service configured with the provided logger and repo
func NewGitService(logger *zerolog.Logger, repo *GitRepo, providerRepo *providers.ProviderRepo) *GitService {
	return &GitService{
		logger:       logger,
		repo:         repo,
		providerRepo: providerRepo,
	}
}

// ValidateSecret receives a github signature from the `X-Hub-Signature-256` header and ensures that it matches the
// secret key used for github endpoints.
func (s *GitService) ValidateSecret(ctx context.Context, signature string, payload []byte) (err error) {
	key, err := s.repo.getSigningKey(ctx, string(model.Github))
	if err != nil {
		return err
	}
	decrypted, err := crypto.DecryptSigningKey(key)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(decrypted))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil
	}
	return nil
}

// InsertWebhook creates and stores a Github webhook using the provided data request
func (s *GitService) InsertWebhook(ctx context.Context, headers []byte, payload []byte, id string, event string) (model.Webhook, error) {
	routing, err := s.providerRepo.GetProviderRouting(ctx, string(model.Github))
	if err != nil {
		return model.Webhook{}, err
	}
	params := model.CreateWebhookParams{
		ProviderID:  routing.ID,
		Provider:    string(model.Github),
		EventID:     &id,
		EventType:   event,
		Headers:     headers,
		Payload:     payload,
		ForwardedTo: routing.ForwardedTo,
		ReceivedAt:  time.Now().UTC(),
	}
	hook, err := s.repo.insertWebhook(ctx, params)
	if err != nil {
		return model.Webhook{}, err
	}
	return hook, nil
}
