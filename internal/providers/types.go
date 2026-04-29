package providers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProviderName Enum represents the name of a provider in the database.
type ProviderName string

const (
	Github ProviderName = "Github"
	Stripe ProviderName = "Stripe"
	SNS    ProviderName = "SNS"
)

// DeliveryStatus Enum represents the current delivery status of a webhook event.
type DeliveryStatus string

const (
	DeliveryStatusQueued     DeliveryStatus = "queued"
	DeliveryStatusProcessing DeliveryStatus = "processing"
	DeliveryStatusDelivered  DeliveryStatus = "delivered"
	DeliveryStatusRetrying   DeliveryStatus = "retrying"
	DeliveryStatusFailed     DeliveryStatus = "failed"
)

// ProviderRouting represents the routing configuration for a webhook provider.
// It contains the provider's unique identifier and the destination address
// where incoming webhooks should be forwarded.
type ProviderRouting struct {
	ID          uuid.UUID
	ForwardedTo string
}

// CreateWebhookParams contains only fields required to insert a webhook.
type CreateWebhookParams struct {
	ProviderID  uuid.UUID
	Provider    string
	EventID     *string
	EventType   string
	Headers     json.RawMessage
	Payload     json.RawMessage
	ForwardedTo string
	ReceivedAt  time.Time
}

// Provider represents a webhook provider.
type Provider struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	SigningSecret  string    `json:"signing_secret"`
	DestinationURL string    `json:"destination_url"`
	IsConfigured   bool      `json:"is_configured"`
	CreatedAt      time.Time `json:"created_at"`
}

// Webhook struct represents a webhook event received by a provider
type Webhook struct {
	// IDs
	ID         uuid.UUID `json:"id"`
	ProviderID uuid.UUID `json:"provider_id"`
	Provider   string    `json:"provider"`
	// Main Information
	EventID   *string         `json:"event_id"` // NULLABLE
	EventType string          `json:"event_type"`
	Headers   json.RawMessage `json:"headers"`
	Payload   json.RawMessage `json:"payload"`
	// Event Statistics
	DeliveryStatus DeliveryStatus `json:"delivery_status"`
	ForwardedTo    string         `json:"forwarded_to"`
	ResponseCode   *int           `json:"response_code"`
	// Event Error Handling
	RetryCount  int        `json:"retry_count"`
	NextRetryAt *time.Time `json:"next_retry_at"` // NULLABLE
	LastError   *string    `json:"last_error"`    // NULLABLE
	// Event Time Stamps
	ReceivedAt time.Time `json:"received_at"`
	CreatedAt  time.Time `json:"created_at"`
}
