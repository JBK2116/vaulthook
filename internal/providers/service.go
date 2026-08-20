package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/model"
)

var (
	ErrMissingSigningSecret = errors.New("error field is empty: signing_secret")
	ErrMissingDestination   = errors.New("error missing field: destination_url")
	ErrInvalidRetryCount    = fmt.Errorf("error field is invalid: retry_count must be between 0 and %d", maxRetryCount)
	ErrInvalidReqSecond     = fmt.Errorf(
		"error field is invalid: max_req_second must be between 0 and %d",
		maxReqSecond,
	)
)

const (
	maxRetryCount = 20
	maxReqSecond  = 1000
)

var ErrInvalidSignature = errors.New("invalid webhook signature")

// Service handles business logic for providers.
type Service struct {
	repo *ProviderRepo
}

// NewProviderService returns a Service configured with the provided repo.
func NewProviderService(repo *ProviderRepo) *Service {
	return &Service{
		repo: repo,
	}
}

// GetAll retrieves all providers.
func (s *Service) GetAll(ctx context.Context) ([]model.Provider, error) {
	provs, err := s.repo.getAll(ctx)
	if err != nil {
		return nil, err
	}
	for i, prov := range provs {
		if !prov.IsConfigured {
			continue
		}
		var decKey string
		decKey, err = crypto.DecryptSigningKey(prov.SigningSecret)
		if err != nil {
			return nil, err
		}
		provs[i].SigningSecret = decKey
	}
	return provs, nil
}

// Configure updates a providers configuration settings by ID,
// setting is_configured to true. Returns an error if the ID is invalid,
// any field is empty or a database error occurs.
func (s *Service) Configure(
	ctx context.Context,
	id string,
	sec string,
	des string,
	maxRetry int,
	maxReqSec int,
) (model.Provider, error) {
	uuidS, err := uuid.Parse(id)
	sec = strings.TrimSpace(sec)
	if err != nil {
		return model.Provider{}, err
	}
	if len(sec) == 0 {
		return model.Provider{}, ErrMissingSigningSecret
	}
	if len(des) == 0 {
		return model.Provider{}, ErrMissingDestination
	}
	if maxRetry < 0 || maxRetry > maxRetryCount {
		return model.Provider{}, ErrInvalidRetryCount
	}
	if maxReqSec < 0 || maxReqSec > maxReqSecond {
		return model.Provider{}, ErrInvalidReqSecond
	}
	encKey, err := crypto.EncryptSigningKey(sec)
	if err != nil {
		return model.Provider{}, err
	}
	prov, err := s.repo.configure(ctx, uuidS, encKey, des, maxRetry, maxReqSec)
	if err != nil {
		return model.Provider{}, err
	}
	err = Cache.Set(ctx, model.ProviderName(prov.Name), prov)
	if err != nil {
		return model.Provider{}, err
	}
	return prov, nil
}
