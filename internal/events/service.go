package events

import (
	"context"
	"sync"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/rs/zerolog"
)

// EventService provides the main universal business logic for handling webhook events pertaining to all providers.
type EventService struct {
	logger      *zerolog.Logger
	mu          sync.Mutex
	repo        *EventRepo
	subscribers map[chan model.Webhook]struct{}
}

// NewEventService returns a EventService configured with the provided logger and repo.
func NewEventService(logger *zerolog.Logger, repo *EventRepo) *EventService {
	return &EventService{
		logger:      logger,
		repo:        repo,
		subscribers: make(map[chan model.Webhook]struct{}),
	}
}

// Subscribe creates a buffered webhook channel and adds it to the `subscribers` map.
// Each webhook channel is protected by a mutex, ensuring that it is concurrency safe.
// An `unsub` function is provided to delete the created buffered webhook channel when needed.
func (s *EventService) Subscribe() (<-chan model.Webhook, func()) {
	ch := make(chan model.Webhook, 20)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	// delete channel function
	unsub := func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		close(ch)
		s.mu.Unlock()
	}
	return ch, unsub
}

// Send inserts the provided webhook event into all subcriber channels.
func (s *EventService) Send(event model.Webhook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subscribers {
		ch <- event
	}
}

// GetAll returns all the webhook events stored in the database.
func (s *EventService) GetAll(ctx context.Context) ([]model.Webhook, error) {
	events, err := s.repo.getAll(ctx)
	if err != nil {
		return nil, err
	}
	return events, nil

}
