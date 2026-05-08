package model

// DeliveryStatus Enum represents the current delivery status of a webhook event.
type DeliveryStatus string

const (
	DeliveryStatusQueued     DeliveryStatus = "queued"
	DeliveryStatusProcessing DeliveryStatus = "processing"
	DeliveryStatusDelivered  DeliveryStatus = "delivered"
	DeliveryStatusRetrying   DeliveryStatus = "retrying"
	DeliveryStatusFailed     DeliveryStatus = "failed"
)
