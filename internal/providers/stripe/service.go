package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	crypto "github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
)

// safePrefix returns a truncated version of s safe for logging.
func safePrefix(s string) string {
	if len(s) <= 6 {
		return s
	}
	return s[:6]
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
	for k, val := range parsed {
		if _, ok := allowed[k]; ok {
			for _, v := range val {
				r.Header.Add(k, v)
			}
		}
	}
	return nil
}

// StripeService provides the main business logic for handling webhook events
// pertaining to the Stripe provider.
type StripeService struct {
	logger       *zerolog.Logger
	eventRepo    *events.EventRepo
	providerRepo *providers.ProviderRepo
}

// NewStripeService returns a StripeService configured with the provided
// logger, event repository, and provider repository.
func NewStripeService(logger *zerolog.Logger, eventRepo *events.EventRepo, providerRepo *providers.ProviderRepo) *StripeService {
	return &StripeService{
		logger:       logger,
		eventRepo:    eventRepo,
		providerRepo: providerRepo,
	}
}

// ValidateSecret receives a stripe signature from the `Stripe-Signature` header
// and ensures that it matches the secret key used for stripe endpoints.
func (s *StripeService) ValidateSecret(ctx context.Context, signatureHeader string, payload []byte) (stripe.Event, error) {
	endpointSecret, err := s.providerRepo.GetSigningKey(ctx, string(model.Stripe))
	if err != nil {
		return stripe.Event{}, err
	}
	decrytedSecret, err := crypto.DecryptSigningKey(endpointSecret)
	if err != nil {
		return stripe.Event{}, err
	}
	event, err := webhook.ConstructEvent(payload, signatureHeader, decrytedSecret)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int("secret_len", len(decrytedSecret)).
			Str("secret_prefix", safePrefix(decrytedSecret)).
			Str("sig_prefix", safePrefix(signatureHeader)).
			Int("payload_len", len(payload)).
			Msg("[Stripe] failed to validate stripe webhook secret")
		return stripe.Event{}, err
	}
	return event, nil
}

// InsertWebhook creates and stores a Stripe webhook using the incoming request
// data and parsed event. It resolves the provider routing, builds the insert
// parameters, and persists the webhook.
//
// Returns the stored webhook record or an error if any step fails.
func (s *StripeService) InsertWebhook(ctx context.Context, headers []byte, payload []byte, event stripe.Event) (model.Webhook, error) {
	providerRouting, err := s.providerRepo.GetProviderRouting(ctx, string(model.Stripe))
	if err != nil {
		return model.Webhook{}, err
	}
	params := model.CreateWebhookParams{
		ProviderID:  providerRouting.ID,
		Provider:    string(model.Stripe),
		EventID:     &event.ID,
		EventType:   string(event.Type),
		Headers:     headers,
		Payload:     payload,
		ForwardedTo: providerRouting.ForwardedTo,
		ReceivedAt:  time.Now().UTC(),
	}
	stripeWebhook, err := s.eventRepo.InsertWebhook(ctx, params)
	if err != nil {
		return model.Webhook{}, err
	}
	return stripeWebhook, nil
}
