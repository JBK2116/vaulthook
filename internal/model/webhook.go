package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CreateWebhookParams contains only fields required to insert a webhook.
type CreateWebhookParams struct {
	// Provider ID
	ProviderID uuid.UUID
	// Provider Name
	Provider string
	// Webhook Event ID (NULLABLE)
	EventID *string
	// Webhook Event Type
	EventType string
	// Webhook Event Headers JSON
	Headers json.RawMessage
	// Webhook Event Payload JSON
	Payload json.RawMessage
	// Webhook Forwarded To Destination
	ForwardedTo string
	// Webhook Received At Time
	ReceivedAt time.Time
}

// Webhook struct represents a webhook event received by a provider.
type Webhook struct {
	// Webhook ID
	ID uuid.UUID `json:"id"`
	// Webhook Provider ID
	ProviderID uuid.UUID `json:"provider_id"`
	// Webhook Provider Name
	Provider string `json:"provider"`
	// Webhook Event ID (NULLABLE)
	EventID *string `json:"event_id"`
	// Webhook Event Type
	EventType string `json:"event_type"`
	// Webhook Headers JSON
	Headers json.RawMessage `json:"headers"`
	// Webhook Payload JSON
	Payload json.RawMessage `json:"payload"`
	// Webhook Delivery Status Enum
	DeliveryStatus DeliveryStatus `json:"delivery_status"`
	// Webhook Forwarded To URL
	ForwardedTo string `json:"forwarded_to"`
	// Webhook Forwarded To Response Code
	ResponseCode *int `json:"response_code"`
	// Webhook Retry Count Of Forwarding
	RetryCount int `json:"retry_count"`
	// Webhook Next Scheduled Retry At Time (NULLABLE)
	NextRetryAt *time.Time `json:"next_retry_at"`
	// Webhook Last Error Message (NULLABLE)
	LastError *string `json:"last_error"`
	// Webhook Received At Time
	ReceivedAt time.Time `json:"received_at"`
	// Webhook Updated At Time
	UpdatedAt time.Time `json:"updated_at"`
	// Webhook Created At Time
	CreatedAt time.Time `json:"created_at"`
}
