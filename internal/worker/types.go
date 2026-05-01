package worker

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/providers"
)

// Worker interface represents an asynchronous goroutine worker. Responsible for handling webhook events.
type Worker interface {
	// Run kicks off the Worker to begin working on webhooks.
	Run(ctx context.Context) error
	// QueryLatest retrieves the latest webhook event required for processing.
	QueryLatest(context.Context) (*providers.Webhook, error)
	// ForwardEvent attempts to forward the received webhook event to its destination.
	ForwardEvent(context.Context, *providers.Webhook) (*providers.Webhook, error)
	// UpdateEvent updates the received events data in the database.
	UpdateEvent(ctx context.Context, event *providers.Webhook) (*providers.Webhook, error)
	// Send pushes the received updated event to the frontend via the sse pipeline.
	Send(ctx context.Context, event *providers.Webhook) error
}

// QueueWorkerRepository interface represents a repository
// that handles dataabase events for queued events.
type QueueWorkerRepository interface {
	// GetQueuedEvent queries the database for the next event in the queue.
	GetQueuedEvent(ctx context.Context) (*providers.Webhook, error)
	// UpdateEventStatus updates the necessary values of the provided webhook event.
	UpdateEventStatus(ctx context.Context, event *providers.Webhook) (*providers.Webhook, error)
}

// RetryWorkerRepository interface represents a repository
// that handles dataabase events for retrying/failed events.
type RetryWorkerRepository interface {
	// GetRetryEvent queries the database for the next event in the queue.
	GetRetryEvent(ctx context.Context) (*providers.Webhook, error)
	// UpdateEventStatus updates the necessary values of the provided webhook event.
	UpdateEventStatus(ctx context.Context, event *providers.Webhook) error
}

// QueueWorker struct is responsible for processing events that are queued.
type QueueWorker struct {
	interval int
	evt      *providers.Webhook
	svc      *events.EventService
	repo     *QueueWorkerRepository
}

// RetryWorker struct is responsible for processing events that have failed.
type RetryWorker struct {
	interval int
	evt      *providers.Webhook
	svc      *events.EventService
	repo     *RetryWorkerRepository
}
