package worker

import (
	"context"
	"time"

	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// updateWebhook is a struct representing all the necessary fields that may be used to update a webhook following a forwarding attempt.
type updateWebhook struct {
	id             uuid.UUID
	nextRetryAt    *time.Time
	deliveryStatus providers.DeliveryStatus
	responseCode   *int
	lastError      *string
}

// WorkerRepository interface represents a repository
// that handles database events for webhooks.
type WorkerRepository interface {
	// GetEvent queries the database for the next event in the queue.
	GetEvent(ctx context.Context) (*providers.Webhook, error)
	// GetDestinationURL queries the database for the destination_url of the provider with the provided ID.
	GetDestinationURL(ctx context.Context, provID uuid.UUID) (string, error)
	// UpdateEvent updates the necessary values of the provided webhook event.
	UpdateEvent(ctx context.Context, updates updateWebhook) (*providers.Webhook, error)
}

// Worker struct is responsible for processing all webhook events that are stored in the database.
type Worker struct {
	sse    *events.EventService
	repo   WorkerRepository
	logger *zerolog.Logger
}

// QueueWorkerRepo struct is responsible for processing events that are queued.
//
// Implementing the WorkerRepository interface
type QueueWorkerRepo struct {
	db *pgxpool.Pool
}

// RetryWorkerRepo struct is responsible for processing events that need to be retried.
//
// Implementing the WorkerRepository interface
type RetryWorkerRepo struct {
	db *pgxpool.Pool
}
