package events

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidUUID = errors.New("invalid uuid received")
)

// EventService provides the main universal business logic for handling webhook events pertaining to all providers.
type EventService struct {
	logger      *zerolog.Logger
	mu          sync.Mutex
	repo        *EventRepo
	buffer      []model.Webhook
	broadcast   chan model.Webhook
	subscribers map[chan []model.Webhook]struct{}
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

// Start kicks off the background batching loop for processing webhooks
func (s *EventService) Start(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond) // 10 flushes per second. Update as necessary.
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-s.broadcast:
			s.mu.Lock()
			s.buffer = append(s.buffer, event)
			s.mu.Unlock()
		case <-ticker.C:
			s.flush()
		}
	}
}

// flush resets the current buffer and flushes all webhooks to the frontend sse
func (s *EventService) flush() {
	s.mu.Lock()
	if len(s.buffer) == 0 {
		s.mu.Unlock()
		return
	}
	// Capture current buffer and reset it
	batch := s.buffer
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

// Send is now non-blocking and concurrency-safe without a long-held mutex
func (s *EventService) Send(event model.Webhook) {
	s.broadcast <- event
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
