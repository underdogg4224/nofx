package trader

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// SolanaRPCClient wraps the Solana RPC and WebSocket clients
type SolanaRPCClient struct {
	rpcClient *rpc.Client
	wsClient  *ws.Client
	network   string // "mainnet-beta", "devnet", or "testnet"
	rpcURL    string
	wsURL     string
}

// Network constants
const (
	MainnetBeta = "mainnet-beta"
	Devnet      = "devnet"
	Testnet     = "testnet"
)

// Default RPC endpoints
var DefaultRPCEndpoints = map[string]string{
	MainnetBeta: "https://api.mainnet-beta.solana.com",
	Devnet:      "https://api.devnet.solana.com",
	Testnet:     "https://api.testnet.solana.com",
}

// Default WebSocket endpoints
var DefaultWSEndpoints = map[string]string{
	MainnetBeta: "wss://api.mainnet-beta.solana.com",
	Devnet:      "wss://api.devnet.solana.com",
	Testnet:     "wss://api.testnet.solana.com",
}

// NewSolanaRPCClient creates a new Solana RPC client
func NewSolanaRPCClient(network, rpcURL, wsURL string) (*SolanaRPCClient, error) {
	// Use default endpoints if not provided
	if rpcURL == "" {
		var ok bool
		rpcURL, ok = DefaultRPCEndpoints[network]
		if !ok {
			return nil, fmt.Errorf("unknown network: %s", network)
		}
	}

	if wsURL == "" {
		var ok bool
		wsURL, ok = DefaultWSEndpoints[network]
		if !ok {
			return nil, fmt.Errorf("unknown network: %s", network)
		}
	}

	// Create RPC client
	rpcClient := rpc.New(rpcURL)

	// Create WebSocket client
	wsClient, err := ws.Connect(context.Background(), wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	return &SolanaRPCClient{
		rpcClient: rpcClient,
		wsClient:  wsClient,
		network:   network,
		rpcURL:    rpcURL,
		wsURL:     wsURL,
	}, nil
}

// GetRPCClient returns the underlying RPC client
func (c *SolanaRPCClient) GetRPCClient() *rpc.Client {
	return c.rpcClient
}

// GetWSClient returns the underlying WebSocket client
func (c *SolanaRPCClient) GetWSClient() *ws.Client {
	return c.wsClient
}

// GetNetwork returns the network name
func (c *SolanaRPCClient) GetNetwork() string {
	return c.network
}

// GetBalance returns the SOL balance for an address (in lamports)
func (c *SolanaRPCClient) GetBalance(ctx context.Context, address solana.PublicKey) (uint64, error) {
	balance, err := c.rpcClient.GetBalance(
		ctx,
		address,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance.Value, nil
}

// GetBalanceSOL returns the SOL balance in SOL (not lamports)
func (c *SolanaRPCClient) GetBalanceSOL(ctx context.Context, address solana.PublicKey) (float64, error) {
	lamports, err := c.GetBalance(ctx, address)
	if err != nil {
		return 0, err
	}

	// Convert lamports to SOL (1 SOL = 1e9 lamports)
	return float64(lamports) / 1e9, nil
}

// GetTokenBalance returns the SPL token balance for a token account
func (c *SolanaRPCClient) GetTokenBalance(ctx context.Context, tokenAccount solana.PublicKey) (*rpc.GetTokenAccountBalanceResult, error) {
	balance, err := c.rpcClient.GetTokenAccountBalance(
		ctx,
		tokenAccount,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get token balance: %w", err)
	}

	return balance, nil
}

// GetRecentBlockhash gets a recent blockhash for transaction creation
func (c *SolanaRPCClient) GetRecentBlockhash(ctx context.Context) (solana.Hash, error) {
	recent, err := c.rpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Hash{}, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	return recent.Value.Blockhash, nil
}

// SendTransaction sends a signed transaction to the network
func (c *SolanaRPCClient) SendTransaction(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	sig, err := c.rpcClient.SendTransactionWithOpts(
		ctx,
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: rpc.CommitmentConfirmed,
		},
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return sig, nil
}

// ConfirmTransaction waits for a transaction to be confirmed
func (c *SolanaRPCClient) ConfirmTransaction(ctx context.Context, sig solana.Signature, commitment rpc.CommitmentType) error {
	// Set timeout for confirmation
	confirmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Poll for transaction status
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-confirmCtx.Done():
			return fmt.Errorf("transaction confirmation timeout")
		case <-ticker.C:
			// Check transaction status
			status, err := c.rpcClient.GetSignatureStatuses(confirmCtx, true, sig)
			if err != nil {
				continue // Retry on error
			}

			if len(status.Value) > 0 && status.Value[0] != nil {
				txStatus := status.Value[0]

				// Check if transaction failed
				if txStatus.Err != nil {
					return fmt.Errorf("transaction failed: %v", txStatus.Err)
				}

				// Check confirmation level
				if txStatus.ConfirmationStatus == commitment {
					return nil // Transaction confirmed
				}
			}
		}
	}
}

// SendAndConfirmTransaction sends a transaction and waits for confirmation
func (c *SolanaRPCClient) SendAndConfirmTransaction(
	ctx context.Context,
	tx *solana.Transaction,
	commitment rpc.CommitmentType,
) (solana.Signature, error) {
	// Send transaction
	sig, err := c.SendTransaction(ctx, tx)
	if err != nil {
		return solana.Signature{}, err
	}

	// Wait for confirmation
	if err := c.ConfirmTransaction(ctx, sig, commitment); err != nil {
		return sig, err
	}

	return sig, nil
}

// GetAccountInfo retrieves account information
func (c *SolanaRPCClient) GetAccountInfo(ctx context.Context, account solana.PublicKey) (*rpc.GetAccountInfoResult, error) {
	info, err := c.rpcClient.GetAccountInfo(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	return info, nil
}

// SubscribeAccount subscribes to account changes via WebSocket
func (c *SolanaRPCClient) SubscribeAccount(
	account solana.PublicKey,
	commitment rpc.CommitmentType,
) (*ws.AccountSubscription, error) {
	sub, err := c.wsClient.AccountSubscribe(account, commitment)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to account: %w", err)
	}

	return sub, nil
}

// GetSlot returns the current slot
func (c *SolanaRPCClient) GetSlot(ctx context.Context) (uint64, error) {
	slot, err := c.rpcClient.GetSlot(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, fmt.Errorf("failed to get slot: %w", err)
	}

	return slot, nil
}

// GetVersion returns the Solana version running on the node
func (c *SolanaRPCClient) GetVersion(ctx context.Context) (*rpc.GetVersionResult, error) {
	version, err := c.rpcClient.GetVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	return version, nil
}

// HealthCheck performs a health check on the RPC endpoint
func (c *SolanaRPCClient) HealthCheck(ctx context.Context) error {
	// Try to get the current slot
	_, err := c.GetSlot(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close closes the WebSocket connection
func (c *SolanaRPCClient) Close() error {
	if c.wsClient != nil {
		return c.wsClient.Close()
	}
	return nil
}

// RequestAirdrop requests an airdrop of SOL (devnet/testnet only)
func (c *SolanaRPCClient) RequestAirdrop(
	ctx context.Context,
	address solana.PublicKey,
	lamports uint64,
) (solana.Signature, error) {
	if c.network == MainnetBeta {
		return solana.Signature{}, fmt.Errorf("airdrops not available on mainnet")
	}

	sig, err := c.rpcClient.RequestAirdrop(
		ctx,
		address,
		lamports,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to request airdrop: %w", err)
	}

	return sig, nil
}
