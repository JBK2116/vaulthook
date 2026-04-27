package providers

import (
	"context"
	"errors"

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
		return Provider{}, err
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
