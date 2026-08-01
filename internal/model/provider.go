package model

import (
	"time"

	"github.com/google/uuid"
)

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

// Provider represents a webhook provider.
type Provider struct {
	ID uuid.UUID `json:"id"`
	// Provider Name
	Name string `json:"name"`
	// Provider Signing Secret For Validating Webhooks
	SigningSecret string `json:"signing_secret"`
	// Provider Destination URL To Forward Webhooks
	DestinationURL string `json:"destination_url"`
	// Provider Max Retry Count Per Webhook Forwarding Attempt
	MaxRetries int `json:"max_retries"`
	// Provider Max Request Count Allowed Per Second
	MaxReqSecond int `json:"max_req_second"`
	// Provider Manual Configuration Bool
	IsConfigured bool `json:"is_configured"`
	// Provider Created At Time
	CreatedAt time.Time `json:"created_at"`
}
