package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidUUID = errors.New("invalid uuid received")
)

// maxBatchSize caps the number of webhook events sent per SSE batch to prevent
// overwhelming the browser with multi-megabyte JSON payloads at high throughput.
const maxBatchSize = 200

// EventService provides the main universal business logic for handling webhook events pertaining to all providers.
type EventService struct {
	logger      *zerolog.Logger
	mu          sync.Mutex
	repo        *EventRepo
	buffer      []model.Webhook
	broadcast   chan model.Webhook
	subscribers map[chan []model.Webhook]struct{}
	dropped     atomic.Int64 // events dropped from SSE batches due to capping
}

// NewEventService returns a EventService configured with the provided logger and repo.
func NewEventService(logger *zerolog.Logger, repo *EventRepo) *EventService {
	return &EventService{
		logger:      logger,
		repo:        repo,
		buffer:      make([]model.Webhook, 0),
		broadcast:   make(chan model.Webhook, 10000),
		subscribers: make(map[chan []model.Webhook]struct{}),
	}
}

// Start kicks off the background batching loop for processing webhooks.
//
// On each ticker pulse it drains all pending events from the broadcast channel
// in bulk (holding the mutex once) before flushing. Between ticks it consumes
// events one at a time. This avoids the original single-event-per-iteration
// bottleneck and prevents the ticker from starving event consumption.
func (s *EventService) Start(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond) // 10 flushes per second. Update as necessary.
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Drain all pending events before flushing so the batch
			// includes everything received since the last tick.
			s.drainBroadcast()
			s.flush()
		case event := <-s.broadcast:
			s.mu.Lock()
			s.buffer = append(s.buffer, event)
			s.mu.Unlock()
		}
	}
}

// drainBroadcast empties the broadcast channel into the buffer in a single
// mutex acquisition, avoiding the O(n) lock/unlock overhead of the previous
// per-event approach.
func (s *EventService) drainBroadcast() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		select {
		case event := <-s.broadcast:
			s.buffer = append(s.buffer, event)
		default:
			return
		}
	}
}

// flush resets the current buffer and broadcasts webhooks to frontend SSE subscribers.
//
// When the buffer exceeds maxBatchSize the oldest events are dropped from the
// SSE feed (they remain persisted in the database) and the dropped count is
// accumulated so the SSE handler can signal an overflow to the client.
func (s *EventService) flush() {
	s.mu.Lock()
	if len(s.buffer) == 0 {
		s.mu.Unlock()
		return
	}
	// Capture current buffer and reset it
	batch := s.buffer
	if len(batch) > maxBatchSize {
		// Keep the newest events; older ones are still queryable via REST.
		excess := len(batch) - maxBatchSize
		batch = batch[excess:]
		s.dropped.Add(int64(excess))
	}
	s.buffer = make([]model.Webhook, 0)
	s.mu.Unlock()
	// Broadcast the batch to all subscribers
	s.mu.Lock()
	for ch := range s.subscribers {
		select {
		case ch <- batch:
		default:
			// subscriber is too slow, drop the batch for them to avoid blocking the hub
		}
	}
	s.mu.Unlock()
}

// Subscribe now returns a channel that receives SLICES of webhooks
func (s *EventService) Subscribe() (<-chan []model.Webhook, func()) {
	ch := make(chan []model.Webhook, 100)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()

	unsub := func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		close(ch)
		s.mu.Unlock()
	}
	return ch, unsub
}

// Send enqueues a webhook event for SSE broadcast. It is non-blocking: if the
// internal channel is full the event is dropped from the live feed (it remains
// safely persisted in the database). This prevents a slow SSE consumer from
// blocking incoming HTTP handlers.
func (s *EventService) Send(event model.Webhook) {
	select {
	case s.broadcast <- event:
	default:
		// Channel full — event is still in the DB; frontend can fetch via REST.
		s.logger.Warn().Msg("[EventService] channel full, dropping event from SSE feed")
	}
}

// GetAll retrieves all webhooks from the repository, optionally filtered by the provided creation timestamp.
func (s *EventService) GetAll(ctx context.Context, createdAt *time.Time) ([]model.Webhook, error) {
	return s.repo.getAll(ctx, createdAt)
}

// GetStats retrieves and returns the aggregated statistics from the repository.
func (s *EventService) GetStats(ctx context.Context) (*model.Stats, error) {
	stats, err := s.repo.getStats(ctx)
	if err != nil {
		return nil, err
	}
	return stats, nil

}

// Dropped returns the number of events that were dropped from SSE batches due
// to the maxBatchSize cap since the last call. The counter is reset atomically.
func (s *EventService) Dropped() int64 {
	return s.dropped.Swap(0)
}

// ReplayEvent marks the webhook event as "queued", allowing it to be picked by up workers to be replayed
func (s *EventService) ReplayEvent(ctx context.Context, id string) error {
	uuidS, err := uuid.Parse(id)
	if err != nil {
		return ErrInvalidUUID
	}
	err = s.repo.replayEvent(ctx, uuidS)
	if err != nil {
		return err
	}
	return nil
}

// Search returns all webhooks that meet thre requirements listed in the provided options payload
func (s *EventService) Search(ctx context.Context, opts model.SearchRequest) ([]model.Webhook, error) {
	var found []model.Webhook
	if opts.Type == model.LookUp {
		lookOpts := model.LookupOpts{
			WebhookID: opts.WebhookID,
			EventID:   opts.EventID,
		}
		hooks, err := s.repo.lookup(ctx, lookOpts)
		if err != nil {
			return nil, err
		}
		found = append(found, hooks...)
	}
	if opts.Type == model.Filter {
		filterOpts := model.FilterOpts{
			Providers:        opts.Providers,
			EventType:        opts.EventType,
			DeliveryStatuses: opts.DeliveryStatuses,
			ResponseCode:     opts.ResponseCode,
			PayloadSearch:    opts.PayloadSearch,
			HasRetries:       opts.HasRetries,
			HasError:         opts.HasError,
		}
		if opts.FromTime != nil {
			filterOpts.FromTime = convTime(*opts.FromTime)
		}
		if opts.ToTime != nil {
			filterOpts.ToTime = convTime(*opts.ToTime)
		}
		hooks, err := s.repo.filter(ctx, filterOpts)
		if err != nil {
			return nil, err
		}
		found = append(found, hooks...)
	}
	return found, nil
}

// convTime is a helper function to convert a valid ISO8601 string to a time object
func convTime(iso string) *time.Time {
	time, _ := time.Parse(time.RFC3339, iso)
	return &time
}
