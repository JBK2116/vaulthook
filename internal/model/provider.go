package model

import "github.com/google/uuid"

// ProviderName Enum represents the name of a provider in the database.
type ProviderName string

const (
	Github ProviderName = "Github"
	Stripe ProviderName = "Stripe"
)

// ProviderRouting represents the routing configuration for a webhook provider.
// It contains the provider's unique identifier and the destination address
// where incoming webhooks should be forwarded.
type ProviderRouting struct {
	ID          uuid.UUID
	ForwardedTo string
}
