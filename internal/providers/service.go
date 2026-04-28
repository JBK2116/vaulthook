package providers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/google/uuid"
)

var (
	ErrMissingSigningSecret = errors.New("error field is empty: signing_secret")
	ErrMissingDestination   = errors.New("error missing field: destination_url")
	ErrEncryption           = errors.New("error encrypting provider signing key")
	ErrDecryption           = errors.New("error decrypting provider signing key")
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
	for index, provider := range providers {
		if !provider.IsConfigured {
			continue
		}
		decryptedSigningKey, err := s.decryptSigningKey(provider.SigningSecret)
		if err != nil {
			return nil, err
		}
		providers[index].SigningSecret = decryptedSigningKey
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
	encryptedSigningKey, err := s.encryptSigningKey(signingSecret)
	if err != nil {
		return Provider{}, err
	}
	provider, err := s.repo.configure(ctx, uuidS, encryptedSigningKey, destinationURL)
	if err != nil {
		return Provider{}, err
	}
	return provider, nil
}

// encryptSigningKey transforms the provided plaintext into an encrypted nonce + ciphertext combination string
//
// The nonce is dynamically created per encryption, ensuring unique ciphertext for every operation.
func (s *ProviderService) encryptSigningKey(plaintext string) (string, error) {
	plaintextBytes := []byte(plaintext)
	// create the encryption key and iv key used for AES encrpytion
	keyBytes := []byte(config.Envs.MASTER_KEY) // MASTER_KEY is 32 bytes
	// block serves as the lock for keeping the plaintext data secure
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryption, err)
	}
	// introduce gcm to further enhance encrpytion
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryption, err)
	}
	// nonce is a unique random number used for each encryption process
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryption, err)
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintextBytes, nil)
	encoded := hex.EncodeToString(ciphertext)
	return encoded, nil
}

func (s *ProviderService) decryptSigningKey(encoded string) (string, error) {
	decodedCipherText, err := hex.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryption, err)
	}
	keyBytes := []byte(config.Envs.MASTER_KEY)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryption, err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryption, err)
	}
	decryptedData, err := gcm.Open(nil, decodedCipherText[:gcm.NonceSize()], decodedCipherText[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryption, err)
	}
	return string(decryptedData), nil
}
