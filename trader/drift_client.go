package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
)

// DriftConfig holds Drift Protocol configuration
type DriftConfig struct {
	Network         string // "mainnet-beta" or "devnet"
	ProgramID       solana.PublicKey
	DLOBAPI         string // Decentralized Limit Order Book API
	DataAPI         string // Market data API
	USDCMint        solana.PublicKey
	HTTPClient      *http.Client
}

// Default Drift program IDs
var (
	DriftProgramIDMainnet = solana.MustPublicKeyFromBase58("dRiftyHA39MWEi3m9aunc5MzRF1JYuBsbn6VPcn33UH")
	DriftProgramIDDevnet  = solana.MustPublicKeyFromBase58("dRiftyHA39MWEi3m9aunc5MzRF1JYuBsbn6VPcn33UH")
	USDCMintMainnet       = solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
	USDCMintDevnet        = solana.MustPublicKeyFromBase58("8zGuJQqwhZafTah7Uc7Z4tXRnguqkn5KLFAP8oV6PHe2")
)

// Default API endpoints
const (
	DLOBAPIMainnet = "https://dlob.drift.trade"
	DLOBAPIDevnet  = "https://master.dlob.drift.trade"
	DataAPIMainnet = "https://data.api.drift.trade"
	DataAPIDevnet  = "https://data.api.drift.trade" // Same for both
)

// DriftClient handles interactions with Drift Protocol
type DriftClient struct {
	config     *DriftConfig
	rpcClient  *SolanaRPCClient
	wallet     *SolanaWallet
	userPDA    solana.PublicKey // User account PDA
	httpClient *http.Client
}

// NewDriftClient creates a new Drift Protocol client
func NewDriftClient(
	network string,
	rpcClient *SolanaRPCClient,
	wallet *SolanaWallet,
) (*DriftClient, error) {
	config, err := NewDriftConfig(network)
	if err != nil {
		return nil, err
	}

	return &DriftClient{
		config:     config,
		rpcClient:  rpcClient,
		wallet:     wallet,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// NewDriftConfig creates a Drift configuration for the specified network
func NewDriftConfig(network string) (*DriftConfig, error) {
	var programID, usdcMint solana.PublicKey
	var dlobAPI, dataAPI string

	switch network {
	case "mainnet-beta", "mainnet":
		programID = DriftProgramIDMainnet
		usdcMint = USDCMintMainnet
		dlobAPI = DLOBAPIMainnet
		dataAPI = DataAPIMainnet
	case "devnet":
		programID = DriftProgramIDDevnet
		usdcMint = USDCMintDevnet
		dlobAPI = DLOBAPIDevnet
		dataAPI = DataAPIDevnet
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	return &DriftConfig{
		Network:    network,
		ProgramID:  programID,
		DLOBAPI:    dlobAPI,
		DataAPI:    dataAPI,
		USDCMint:   usdcMint,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// DriftMarket represents a perpetual market on Drift
type DriftMarket struct {
	MarketIndex           uint16  `json:"marketIndex"`
	Symbol                string  `json:"symbol"`
	BaseAssetSymbol       string  `json:"baseAssetSymbol"`
	QuoteAssetSymbol      string  `json:"quoteAssetSymbol"`
	MarketName            string  `json:"marketName"`
	ContractTier          string  `json:"contractTier"`
	InitialMarginFraction float64 `json:"initialMarginFraction"`
	MaintenanceMarginFrac float64 `json:"maintenanceMarginFraction"`
	MinOrderSize          float64 `json:"minOrderSize"`
	MaxPositionSize       float64 `json:"maxPositionSize"`
}

// DriftPosition represents a user's position
type DriftPosition struct {
	MarketIndex      uint16  `json:"marketIndex"`
	BaseAssetAmount  int64   `json:"baseAssetAmount"`  // In base precision (1e9)
	QuoteAssetAmount int64   `json:"quoteAssetAmount"` // In quote precision (1e6)
	LastCumulativeFundingRate int64   `json:"lastCumulativeFundingRate"`
	OpenBids         int64   `json:"openBids"`
	OpenAsks         int64   `json:"openAsks"`
}

// DriftUserAccount represents a Drift user account
type DriftUserAccount struct {
	Authority   solana.PublicKey `json:"authority"`
	Delegate    solana.PublicKey `json:"delegate"`
	Name        [32]byte         `json:"name"`
	SubAccountID uint16           `json:"subAccountId"`
	Positions   []DriftPosition  `json:"positions"`
	// More fields as needed
}

// Precision constants (from Drift SDK)
const (
	QUOTE_PRECISION int64 = 1_000_000       // 1e6
	BASE_PRECISION  int64 = 1_000_000_000   // 1e9
	PRICE_PRECISION int64 = 1_000_000       // 1e6
	AMM_RESERVE_PRECISION int64 = 1_000_000_000 // 1e9
)

// PositionDirection represents long or short
type PositionDirection uint8

const (
	PositionDirectionLong PositionDirection = iota
	PositionDirectionShort
)

// OrderType represents different order types
type OrderType uint8

const (
	OrderTypeMarket OrderType = iota
	OrderTypeLimit
	OrderTypeTriggerMarket
	OrderTypeTriggerLimit
	OrderTypeOracle
)

// GetUserAccountPDA derives the user account PDA
func (c *DriftClient) GetUserAccountPDA(subAccountID uint16) (solana.PublicKey, error) {
	seeds := [][]byte{
		[]byte("user"),
		c.wallet.GetPublicKey().Bytes(),
		{byte(subAccountID), byte(subAccountID >> 8)}, // Little-endian u16
	}

	pda, _, err := solana.FindProgramAddress(seeds, c.config.ProgramID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("failed to derive user PDA: %w", err)
	}

	return pda, nil
}

// InitializeUser creates a Drift user account (one-time setup)
func (c *DriftClient) InitializeUser(
	ctx context.Context,
	subAccountID uint16,
	name string,
) (solana.Signature, error) {
	// Get user PDA
	userPDA, err := c.GetUserAccountPDA(subAccountID)
	if err != nil {
		return solana.Signature{}, err
	}

	// Store for later use
	c.userPDA = userPDA

	// TODO: Build initialize_user instruction
	// This requires creating the proper Anchor instruction format
	// For now, return error indicating manual initialization needed
	return solana.Signature{}, fmt.Errorf("user initialization requires Drift UI or TypeScript SDK")
}

// GetMarkets fetches available perpetual markets
func (c *DriftClient) GetMarkets(ctx context.Context) ([]DriftMarket, error) {
	url := fmt.Sprintf("%s/markets", c.config.DataAPI)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch markets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var markets []DriftMarket
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, fmt.Errorf("failed to decode markets: %w", err)
	}

	return markets, nil
}

// GetMarketBySymbol finds a market by its symbol (e.g., "SOL-PERP")
func (c *DriftClient) GetMarketBySymbol(ctx context.Context, symbol string) (*DriftMarket, error) {
	markets, err := c.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}

	for _, market := range markets {
		if market.Symbol == symbol || market.MarketName == symbol {
			return &market, nil
		}
	}

	return nil, fmt.Errorf("market not found: %s", symbol)
}

// ConvertToBaseAssetAmount converts decimal amount to base precision
func ConvertToBaseAssetAmount(amount float64) int64 {
	return int64(amount * float64(BASE_PRECISION))
}

// ConvertFromBaseAssetAmount converts base precision to decimal
func ConvertFromBaseAssetAmount(baseAmount int64) float64 {
	return float64(baseAmount) / float64(BASE_PRECISION)
}

// ConvertToQuoteAssetAmount converts decimal amount to quote precision
func ConvertToQuoteAssetAmount(amount float64) int64 {
	return int64(amount * float64(QUOTE_PRECISION))
}

// ConvertFromQuoteAssetAmount converts quote precision to decimal
func ConvertFromQuoteAssetAmount(quoteAmount int64) float64 {
	return float64(quoteAmount) / float64(QUOTE_PRECISION)
}

// GetConfig returns the Drift configuration
func (c *DriftClient) GetConfig() *DriftConfig {
	return c.config
}

// GetUserPDA returns the cached user PDA
func (c *DriftClient) GetUserPDA() solana.PublicKey {
	return c.userPDA
}
