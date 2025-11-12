package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptionKey stores the encryption key derived from the master secret
var encryptionKey []byte

// InitializeEncryption initializes the encryption system with a master secret
// The master secret should be stored securely (e.g., environment variable, secrets manager)
func InitializeEncryption(masterSecret string) error {
	if masterSecret == "" {
		return errors.New("master secret cannot be empty")
	}

	// Derive a 32-byte key using PBKDF2
	// Using a fixed salt for now - in production, consider per-installation salt
	salt := []byte("nofx-encryption-salt-v1") // TODO: Make this configurable
	encryptionKey = pbkdf2.Key([]byte(masterSecret), salt, 100000, 32, sha256.New)

	return nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func Encrypt(plaintext string) (string, error) {
	if encryptionKey == nil {
		return "", errors.New("encryption not initialized - call InitializeEncryption first")
	}

	if plaintext == "" {
		return "", nil // Empty string stays empty
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce (must be unique for each encryption with the same key)
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func Decrypt(ciphertext string) (string, error) {
	if encryptionKey == nil {
		return "", errors.New("encryption not initialized - call InitializeEncryption first")
	}

	if ciphertext == "" {
		return "", nil // Empty string stays empty
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GetMasterSecretFromEnv retrieves the encryption master secret from environment
// Falls back to a file-based secret for backwards compatibility
func GetMasterSecretFromEnv() (string, error) {
	// First try environment variable (recommended)
	secret := os.Getenv("NOFX_ENCRYPTION_SECRET")
	if secret != "" {
		return secret, nil
	}

	// Fall back to reading from a secure file
	secretFile := os.Getenv("NOFX_ENCRYPTION_SECRET_FILE")
	if secretFile != "" {
		data, err := os.ReadFile(secretFile)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Generate a new secret if none exists (first-time setup)
	// In production, this should prompt the user or fail
	return generateAndStoreSecret()
}

// generateAndStoreSecret generates a new random encryption secret
// and stores it in a file for persistence
func generateAndStoreSecret() (string, error) {
	// Generate 32 random bytes
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", err
	}

	secret := base64.StdEncoding.EncodeToString(secretBytes)

	// Store in a hidden file with restricted permissions
	secretFile := ".nofx_encryption_secret"
	if err := os.WriteFile(secretFile, []byte(secret), 0600); err != nil {
		return "", err
	}

	return secret, nil
}

// IsEncrypted checks if a string appears to be encrypted (base64 with certain length)
// This is a heuristic to support migration from plaintext to encrypted
func IsEncrypted(value string) bool {
	// Empty values are not encrypted
	if value == "" {
		return false
	}

	// Try to decode as base64 and check if it has the right characteristics
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return false // Not valid base64, assume plaintext
	}

	// AES-GCM nonce is 12 bytes, minimum ciphertext would be nonce + tag (28 bytes)
	return len(decoded) >= 28
}
