package shopify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/model"

	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers"
)

// Service provides the main business logic for handling webhook events
// pertaining to the Shopify Provider.
type Service struct {
	logger    *zerolog.Logger
	events    *events.EventRepo
	providers *providers.ProviderRepo
}

// NewShopifyService returns a Service configured with the provided
// logger, event repository and provider repository.
func NewShopifyService(logger *zerolog.Logger, events *events.EventRepo, providers *providers.ProviderRepo) *Service {
	return &Service{
		logger:    logger,
		events:    events,
		providers: providers,
	}
}

// SetForwardHeaders applies the appropriate Shopify-specific HTTP headers
// to the outgoing forward request. Only a curated allowlist of headers
// from the original incoming webhook/http request are forwarded.
func SetForwardHeaders(r *http.Request, headers []byte) error {
	allowed := map[string]struct{}{
		"Content-Type":           {},
		"User-Agent":             {},
		"X-Shopify-Topic":        {},
		"X-Shopify-Hmac-Sha256":  {},
		"X-Shopify-Shop-Domain":  {},
		"X-Shopify-Api-Version":  {},
		"X-Shopify-Webhook-Id":   {},
		"X-Shopify-Triggered-At": {},
		"X-Shopify-Event-Id":     {},
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

// ValidateSecret receives a Shopify signature from the `X-Shopify-Hmac-Sha256`
// header and ensures that it matches the secret key used for Shopify endpoints.
func (s *Service) ValidateSecret(ctx context.Context, signature string, payload []byte) error {
	prov, err := providers.Cache.Get(ctx, model.Shopify)
	if err != nil {
		return err
	}
	decrypted, err := crypto.DecryptSigningKey(prov.SigningSecret)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(decrypted))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return providers.ErrInvalidSignature
	}
	if !hmac.Equal(sigBytes, expectedMAC) {
		return providers.ErrInvalidSignature
	}
	return nil
}

// InsertWebhook creates and stores a Shopify webhook using the provided data request.
func (s *Service) InsertWebhook(
	ctx context.Context,
	headers []byte,
	payload []byte,
	id string,
	event string,
) (model.Webhook, error) {
	prov, err := providers.Cache.Get(ctx, model.Shopify)
	if err != nil {
		return model.Webhook{}, err
	}
	params := model.CreateWebhookParams{
		ProviderID:  prov.ID,
		Provider:    string(model.Shopify),
		EventID:     &id,
		EventType:   event,
		Headers:     headers,
		Payload:     payload,
		ForwardedTo: prov.DestinationURL,
		ReceivedAt:  time.Now().UTC(),
	}
	hook, err := s.events.InsertWebhook(ctx, params)
	if err != nil {
		return model.Webhook{}, err
	}
	return hook, nil
}

// Exists checks if a Shopify webhook with the provided event_id already exists in the database.
func (s *Service) Exists(ctx context.Context, evID string) (bool, error) {
	prov, err := providers.Cache.Get(ctx, model.Shopify)
	if err != nil {
		return false, err
	}
	exists, err := s.events.Exists(ctx, prov.ID, evID)
	return exists, err
}
