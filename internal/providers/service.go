package providers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMissingSigningSecret = errors.New("error field is empty: signing_secret")
	ErrMissingDestination   = errors.New("error missing field: destination_url")
)

// ProviderService handles business logic for providers.
type ProviderService struct {
	repo *ProviderRepo
}

// NewProviderService returns an ProviderService configured with the provided repo.
func NewProviderService(repo *ProviderRepo) *ProviderService {
	return &ProviderService{
		repo: repo,
	}
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

// GetAll retrieves all providers.
func (s *ProviderService) GetAll(ctx context.Context) ([]Provider, error) {
	providers, err := s.repo.getAll(ctx)
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// Configure updates a provider's signing secret and destination URL by ID,
// setting is_configured to true. Returns an error if the ID is invalid,
// either field is empty or a database error occurs.
func (s *ProviderService) Configure(ctx context.Context, ID string, signingSecret string, destinationURL string) (Provider, error) {
	uuidS, err := uuid.Parse(ID)
	if err != nil {
		return Provider{}, nil
	}
	if len(signingSecret) <= 0 {
		return Provider{}, ErrMissingSigningSecret
	}
	if len(destinationURL) <= 0 {
		return Provider{}, ErrMissingDestination
	}
	provider, err := s.repo.configure(ctx, uuidS, signingSecret, destinationURL)
	if err != nil {
		return Provider{}, err
	}
	return provider, nil
}
