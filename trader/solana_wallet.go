package trader

import (
	"crypto/ed25519"
	"encoding/base58"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
)

// SolanaWallet represents a Solana wallet for transaction signing
type SolanaWallet struct {
	privateKey solana.PrivateKey
	publicKey  solana.PublicKey
}

// NewSolanaWalletFromPrivateKey creates a wallet from a base58-encoded private key
func NewSolanaWalletFromPrivateKey(privateKeyBase58 string) (*SolanaWallet, error) {
	// Decode base58 private key
	decoded := base58.Decode(privateKeyBase58)
	if len(decoded) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key length: expected %d, got %d",
			ed25519.PrivateKeySize, len(decoded))
	}

	// Create Solana private key
	privateKey := solana.PrivateKey(decoded)
	publicKey := privateKey.PublicKey()

	return &SolanaWallet{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// NewSolanaWalletFromJSON creates a wallet from a Solana CLI JSON keypair file
func NewSolanaWalletFromJSON(filePath string) (*SolanaWallet, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read keypair file: %w", err)
	}

	// Parse JSON array (Solana CLI format is [byte, byte, ...])
	var keyBytes []byte
	if err := json.Unmarshal(data, &keyBytes); err != nil {
		return nil, fmt.Errorf("failed to parse keypair JSON: %w", err)
	}

	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid keypair length: expected %d, got %d",
			ed25519.PrivateKeySize, len(keyBytes))
	}

	// Create Solana private key
	privateKey := solana.PrivateKey(keyBytes)
	publicKey := privateKey.PublicKey()

	return &SolanaWallet{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// GenerateNewWallet creates a new random Solana wallet
func GenerateNewWallet() (*SolanaWallet, error) {
	// Generate new account
	account := solana.NewWallet()

	return &SolanaWallet{
		privateKey: account.PrivateKey,
		publicKey:  account.PublicKey(),
	}, nil
}

// GetPublicKey returns the wallet's public key
func (w *SolanaWallet) GetPublicKey() solana.PublicKey {
	return w.publicKey
}

// GetPrivateKey returns the wallet's private key (use with caution!)
func (w *SolanaWallet) GetPrivateKey() solana.PrivateKey {
	return w.privateKey
}

// GetAddress returns the wallet address as a base58 string
func (w *SolanaWallet) GetAddress() string {
	return w.publicKey.String()
}

// SignTransaction signs a Solana transaction
func (w *SolanaWallet) SignTransaction(tx *solana.Transaction) error {
	// Sign the transaction with this wallet's private key
	_, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(w.publicKey) {
			return &w.privateKey
		}
		return nil
	})

	return err
}

// SignMessage signs an arbitrary message
func (w *SolanaWallet) SignMessage(message []byte) ([]byte, error) {
	// Sign using ed25519
	signature := ed25519.Sign(ed25519.PrivateKey(w.privateKey), message)
	return signature, nil
}

// VerifySignature verifies a signature for a message
func (w *SolanaWallet) VerifySignature(message, signature []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(w.publicKey[:]), message, signature)
}

// ExportPrivateKeyBase58 exports the private key as base58 string
func (w *SolanaWallet) ExportPrivateKeyBase58() string {
	return base58.Encode(w.privateKey)
}

// ExportKeypairJSON exports the keypair in Solana CLI JSON format
func (w *SolanaWallet) ExportKeypairJSON() ([]byte, error) {
	keyBytes := []byte(w.privateKey)
	return json.Marshal(keyBytes)
}
