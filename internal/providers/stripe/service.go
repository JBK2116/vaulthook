package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"

	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/JBK2116/vaulthook/internal/providers"
)

const (
	secretPrefixLength = 6
)

// safePrefix returns a truncated version of s safe for logging.
func safePrefix(s string) string {
	if len(s) <= secretPrefixLength {
		return s
	}
	return s[:secretPrefixLength]
}

// Service provides the main business logic for handling webhook events
// pertaining to the Stripe provider.
type Service struct {
	logger    *zerolog.Logger
	events    *events.EventRepo
	providers *providers.ProviderRepo
}

// NewStripeService returns a Service configured with the provided
// logger, event repository, and provider repository.
func NewStripeService(
	logger *zerolog.Logger,
	events *events.EventRepo,
	providers *providers.ProviderRepo,
) *Service {
	return &Service{
		logger:    logger,
		events:    events,
		providers: providers,
	}
}

// SetForwardHeaders applies the appropriate Stripe-specific HTTP headers
// to the outgoing forward request. Only a curated allowlist of headers
// from the original incoming webhook are forwarded.
func SetForwardHeaders(r *http.Request, headers []byte) error {
	allowed := map[string]struct{}{
		"Content-Type":     {},
		"Stripe-Signature": {},
		"User-Agent":       {},
		"Cache-Control":    {},
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

// ValidateSecret receives a stripe signature from the `Stripe-Signature` header
// and ensures that it matches the secret key used for stripe endpoints.
func (s *Service) ValidateSecret(
	ctx context.Context,
	signature string,
	payload []byte,
) (stripe.Event, error) {
	prov, err := providers.Cache.Get(ctx, model.Stripe)
	if err != nil {
		return stripe.Event{}, err
	}
	decrypted, err := crypto.DecryptSigningKey(prov.SigningSecret)
	if err != nil {
		return stripe.Event{}, err
	}
	event, err := webhook.ConstructEvent(payload, signature, decrypted)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int("secret_len", len(decrypted)).
			Str("secret_prefix", safePrefix(decrypted)).
			Str("sig_prefix", safePrefix(signature)).
			Int("payload_len", len(payload)).
			Msg("[Stripe] failed to validate stripe webhook secret")
		return stripe.Event{}, err
	}
	return event, nil
}

// Exists checks if a stripe webhook with the provided event ID already exists in the database.
func (s *Service) Exists(ctx context.Context, evID string) (bool, error) {
	prov, err := providers.Cache.Get(ctx, model.Stripe)
	if err != nil {
		return false, err
	}
	exists, err := s.events.Exists(ctx, prov.ID, evID)
	return exists, err
}

// InsertWebhook creates and stores a Stripe webhook using the incoming request
// data and parsed event. It resolves the provider routing, builds the insert
// parameters, and persists the webhook.
//
// Returns the stored webhook record or an error if any step fails.
func (s *Service) InsertWebhook(
	ctx context.Context,
	headers []byte,
	payload []byte,
	event stripe.Event,
) (model.Webhook, error) {
	prov, err := providers.Cache.Get(ctx, model.Stripe)
	if err != nil {
		return model.Webhook{}, err
	}
	params := model.CreateWebhookParams{
		ProviderID:  prov.ID,
		Provider:    string(model.Stripe),
		EventID:     &event.ID,
		EventType:   string(event.Type),
		Headers:     headers,
		Payload:     payload,
		ForwardedTo: prov.DestinationURL,
		ReceivedAt:  time.Now().UTC(),
	}
	stripeWebhook, err := s.events.InsertWebhook(ctx, params)
	if err != nil {
		return model.Webhook{}, err
	}
	return stripeWebhook, nil
}
