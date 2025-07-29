package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/sirupsen/logrus"
)

// EncryptionService handles encryption and decryption of secret values
type EncryptionService struct {
	key  []byte
	aead cipher.AEAD
	log  *logrus.Logger
}

// NewEncryptionService creates a new encryption service with the provided key
func NewEncryptionService(key string, logger *logrus.Logger) (*EncryptionService, error) {
	// Decode the base64 key
	decodedKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %w", err)
	}

	// Check key length (AES-256 requires 32 bytes)
	if len(decodedKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (AES-256), got %d bytes", len(decodedKey))
	}

	// Create AES cipher
	block, err := aes.NewCipher(decodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}

	return &EncryptionService{
		key:  decodedKey,
		aead: aead,
		log:  logger,
	}, nil
}

// Encrypt encrypts a plaintext value and returns the encrypted bytes
func (e *EncryptionService) Encrypt(plaintext string) ([]byte, error) {
	e.log.Debug("Encrypting secret value")

	// Generate random nonce
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		e.log.Errorf("Failed to generate nonce: %v", err)
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := e.aead.Seal(nonce, nonce, []byte(plaintext), nil)

	e.log.Debug("Successfully encrypted secret value")
	return ciphertext, nil
}

// Decrypt decrypts encrypted bytes and returns the plaintext value
func (e *EncryptionService) Decrypt(encryptedData []byte) (string, error) {
	e.log.Debug("Decrypting secret value")

	// Check minimum length
	if len(encryptedData) < e.aead.NonceSize() {
		e.log.Error("Encrypted data too short")

		return "", apiErrors.ErrDecryptionFailed
	}

	// Extract nonce and ciphertext
	nonce := encryptedData[:e.aead.NonceSize()]
	ciphertext := encryptedData[e.aead.NonceSize():]

	// Decrypt the ciphertext
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		e.log.Errorf("Failed to decrypt secret value: %v", err)
		return "", apiErrors.ErrDecryptionFailed
	}

	e.log.Debug("Successfully decrypted secret value")
	return string(plaintext), nil
}

// ValidateSecretName validates a secret name
func (e *EncryptionService) ValidateSecretName(name string) error {
	if name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	// Check for invalid characters (basic validation)
	for _, char := range name {
		if char < 32 || char > 126 {
			return apiErrors.ErrInvalidSecretName
		}
	}

	return nil
}

// ValidateSecretValue validates a secret value
func (e *EncryptionService) ValidateSecretValue(value string) error {
	if value == "" {
		return fmt.Errorf("secret value cannot be empty")
	}

	// Check maximum length (1MB)
	const maxSecretValueLength = 1024 * 1024
	if len(value) > maxSecretValueLength {
		return apiErrors.ErrSecretValueTooLong
	}

	return nil
}
