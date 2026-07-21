package crypto

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/JBK2116/vaulthook/internal/config"
)

func init() {
	// Set a test master key (32 bytes) so Encrypt/Decrypt work without a .env file.
	config.Envs.MasterKey = "0123456789abcdef0123456789abcdef" // 32 bytes
}

func TestEncryptSigningKey(t *testing.T) {
	tests := map[string]struct {
		plaintext string
		wantErr   bool
	}{
		"normal string":  {plaintext: "whsec_abc123", wantErr: false},
		"empty string":   {plaintext: "", wantErr: false},
		"long string":    {plaintext: strings.Repeat("x", 1000), wantErr: false},
		"special chars":  {plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?", wantErr: false},
		"unicode chars":  {plaintext: "héllo wörld 世界 🌍", wantErr: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncryptSigningKey(tt.plaintext)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && got == "" {
				t.Fatal("expected non-empty ciphertext")
			}
			// Verify output is valid hex.
			if !tt.wantErr {
				if _, hexErr := hex.DecodeString(got); hexErr != nil {
					t.Fatalf("output is not valid hex: %v", hexErr)
				}
			}
		})
	}
}

func TestDecryptSigningKey_Empty(t *testing.T) {
	got, err := DecryptSigningKey("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestDecryptSigningKey_InvalidHex(t *testing.T) {
	_, err := DecryptSigningKey("not-valid-hex!!!")
	if err == nil {
		t.Fatal("expected error for invalid hex, got nil")
	}
}

func TestDecryptSigningKey_TruncatedCiphertext(t *testing.T) {
	// Provide a valid nonce (12 bytes = 24 hex chars) plus just 1 byte of
	// ciphertext. GCM requires the ciphertext to be at least as long as the
	// plaintext was, so a 1-byte ciphertext from a longer original will fail
	// authentication.
	_, err := DecryptSigningKey("aabbccddeeff00112233445566778899aabbccdd" + "ff")
	if err == nil {
		t.Fatal("expected error for truncated ciphertext, got nil")
	}
}

func TestRoundtrip(t *testing.T) {
	tests := []string{
		"whsec_test_secret",
		"",
		"a",
		strings.Repeat("secret!", 100),
	}
	for _, plaintext := range tests {
		t.Run("", func(t *testing.T) {
			encrypted, err := EncryptSigningKey(plaintext)
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}
			decrypted, err := DecryptSigningKey(encrypted)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}
			if decrypted != plaintext {
				t.Fatalf("roundtrip mismatch: got %q, want %q", decrypted, plaintext)
			}
		})
	}
}

func TestEncryptIsNonDeterministic(t *testing.T) {
	// Same plaintext encrypted twice should produce different ciphertext
	// because each call generates a unique nonce.
	plaintext := "whsec_stable"
	c1, err := EncryptSigningKey(plaintext)
	if err != nil {
		t.Fatalf("first encrypt failed: %v", err)
	}
	c2, err := EncryptSigningKey(plaintext)
	if err != nil {
		t.Fatalf("second encrypt failed: %v", err)
	}
	if c1 == c2 {
		t.Fatal("expected different ciphertexts for same plaintext (unique nonce per call)")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	plaintext := "whsec_tamper_test"
	encrypted, err := EncryptSigningKey(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	// Flip the last byte of the ciphertext to simulate tampering.
	decoded, _ := hex.DecodeString(encrypted)
	decoded[len(decoded)-1] ^= 0xFF
	tampered := hex.EncodeToString(decoded)
	_, err = DecryptSigningKey(tampered)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext, got nil")
	}
}
