// Package crypto provides AES-256-GCM encryption and decryption utilities
// for securing sensitive provider secrets at rest.
//
// Encrypted values are stored as hex-encoded strings with the nonce prepended
// to the ciphertext, allowing safe storage in PostgreSQL without a separate
// nonce column. The encryption key is sourced from MasterKey in the
// application config and must be exactly 32 bytes.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/JBK2116/vaulthook/internal/config"
)

var (
	ErrEncryption = errors.New("error encrypting provider signing key")
	ErrDecryption = errors.New("error decrypting provider signing key")
)

// EncryptSigningKey takes a plaintext string and returns a hex-encoded string containing
// the nonce prepended to the AES-256-GCM ciphertext.
//
// A unique nonce is generated per call, ensuring that encrypting the same
// plaintext twice produces different ciphertext. The MasterKey from config
// must be exactly 32 bytes or an error is returned.
func EncryptSigningKey(plaintext string) (string, error) {
	plaintextBytes := []byte(plaintext)
	// create the encryption key and iv key used for AES encrpytion
	keyBytes := []byte(config.Envs.MasterKey) // MasterKey is 32 bytes
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

// DecryptSigningKey takes a hex-encoded string produced by Encrypt and returns the
// original plaintext string.
//
// The nonce is extracted from the first 12 bytes of the decoded ciphertext.
// Returns an error if the input is malformed, the MasterKey does not match,
// or the ciphertext has been tampered with.
func DecryptSigningKey(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	decodedCipherText, err := hex.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryption, err)
	}
	keyBytes := []byte(config.Envs.MasterKey)
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
