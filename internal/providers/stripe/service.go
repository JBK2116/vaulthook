package stripe

import (
	"context"
	"time"

	crypto "github.com/JBK2116/vaulthook/internal/crpyto"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
)

// StripeService provides the main business logic for handling webhook events pertaining to the stripe provider
type StripeService struct {
	logger       *zerolog.Logger
	repo         *StripeRepo
	providerRepo *providers.ProviderRepo
}

// NewStripeService returns a Stripe service configured with the provided logger and repo
func NewStripeService(logger *zerolog.Logger, repo *StripeRepo, providerRepo *providers.ProviderRepo) *StripeService {
	return &StripeService{
		logger:       logger,
		repo:         repo,
		providerRepo: providerRepo,
	}
}

// ValidateSecret receives a stripe signature from the `Stripe-Signature` header and ensures that it matches the
// secret key used for stripe endpoints.
func (s *StripeService) ValidateSecret(ctx context.Context, signatureHeader string, payload []byte) (stripe.Event, error) {
	endpointSecret, err := s.repo.getSigningKey(ctx, string(providers.Stripe))
	if err != nil {
		return stripe.Event{}, err
	}
	decrytedSecret, err := crypto.DecryptSigningKey(endpointSecret)
	if err != nil {
		return stripe.Event{}, err
	}
	event, err := webhook.ConstructEvent(payload, signatureHeader, decrytedSecret)
	if err != nil {
		return stripe.Event{}, err
	}
	return event, nil
}

// InsertWebhook creates and stores a Stripe webhook using the incoming request
// data and parsed event. It resolves the provider routing, builds the insert
// parameters, and persists the webhook.
//
// Returns the stored webhook record or an error if any step fails.
func (s *StripeService) InsertWebhook(ctx context.Context, headers []byte, payload []byte, event stripe.Event) (providers.Webhook, error) {
	providerRouting, err := s.providerRepo.GetProviderRouting(ctx, string(providers.Stripe))
	if err != nil {
		return providers.Webhook{}, err
	}
	params := providers.CreateWebhookParams{
		ProviderID:  providerRouting.ID,
		Provider:    string(providers.Stripe),
		EventID:     &event.ID,
		EventType:   string(event.Type),
		Headers:     headers,
		Payload:     payload,
		ForwardedTo: providerRouting.ForwardedTo,
		ReceivedAt:  time.Unix(event.Created, 0),
	}
	stripeWebhook, err := s.repo.insertWebhook(ctx, params)
	if err != nil {
		return providers.Webhook{}, err
	}
	return stripeWebhook, nil

}
