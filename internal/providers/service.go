package providers

import (
	"context"
	"errors"

	crypto "github.com/JBK2116/vaulthook/internal/crypto"
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
	provs, err := s.repo.getAll(ctx)
	if err != nil {
		return nil, err
	}
	for i, prov := range provs {
		if !prov.IsConfigured {
			continue
		}
		decKey, err := crypto.DecryptSigningKey(prov.SigningSecret)
		if err != nil {
			return nil, err
		}
		provs[i].SigningSecret = decKey
	}
	return provs, nil
}

// Configure updates a provider's signing secret and destination URL by ID,
// setting is_configured to true. Returns an error if the ID is invalid,
// either field is empty or a database error occurs.
func (s *ProviderService) Configure(ctx context.Context, ID string, sec string, des string) (Provider, error) {
	uuidS, err := uuid.Parse(ID)
	if err != nil {
		return Provider{}, err
	}
	if len(sec) <= 0 {
		return Provider{}, ErrMissingSigningSecret
	}
	if len(des) <= 0 {
		return Provider{}, ErrMissingDestination
	}
	encKey, err := crypto.EncryptSigningKey(sec)
	if err != nil {
		return Provider{}, err
	}
	prov, err := s.repo.configure(ctx, uuidS, encKey, des)
	if err != nil {
		return Provider{}, err
	}
	return prov, nil
}
