package trader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// DriftTrader implements the Trader interface for Drift Protocol
type DriftTrader struct {
	wallet       *SolanaWallet
	rpcClient    *SolanaRPCClient
	driftClient  *DriftClient
	subAccountID uint16

	// Caching
	marketCache    map[string]*DriftMarket
	positionCache  map[string]*DriftPosition
	balanceCache   map[string]interface{}
	cacheExpiry    time.Time
	cacheDuration  time.Duration
	cacheMutex     sync.RWMutex

	ctx context.Context
}

// NewDriftTrader creates a new Drift trader
func NewDriftTrader(
	network string,
	rpcURL string,
	wsURL string,
	privateKeyBase58 string,
	subAccountID uint16,
) (*DriftTrader, error) {
	// Create wallet
	wallet, err := NewSolanaWalletFromPrivateKey(privateKeyBase58)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// Create RPC client
	rpcClient, err := NewSolanaRPCClient(network, rpcURL, wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}

	// Create Drift client
	driftClient, err := NewDriftClient(network, rpcClient, wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to create Drift client: %w", err)
	}

	// Derive user PDA
	userPDA, err := driftClient.GetUserAccountPDA(subAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive user PDA: %w", err)
	}
	driftClient.userPDA = userPDA

	return &DriftTrader{
		wallet:        wallet,
		rpcClient:     rpcClient,
		driftClient:   driftClient,
		subAccountID:  subAccountID,
		marketCache:   make(map[string]*DriftMarket),
		positionCache: make(map[string]*DriftPosition),
		balanceCache:  make(map[string]interface{}),
		cacheDuration: 15 * time.Second, // Same as Binance
		ctx:           context.Background(),
	}, nil
}

// GetBalance implements Trader.GetBalance
func (d *DriftTrader) GetBalance() (map[string]interface{}, error) {
	d.cacheMutex.RLock()
	if time.Now().Before(d.cacheExpiry) && d.balanceCache != nil {
		defer d.cacheMutex.RUnlock()
		return d.balanceCache, nil
	}
	d.cacheMutex.RUnlock()

	// Get SOL balance for gas fees
	solBalance, err := d.rpcClient.GetBalanceSOL(d.ctx, d.wallet.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get SOL balance: %w", err)
	}

	// TODO: Get USDC balance from Drift user account
	// For now, return basic balance info
	balance := map[string]interface{}{
		"sol_balance":   solBalance,
		"usdc_balance":  0.0, // TODO: Query from Drift user account
		"available":     0.0,
		"total_equity":  0.0,
		"margin_used":   0.0,
		"margin_ratio":  0.0,
		"wallet_address": d.wallet.GetAddress(),
	}

	// Update cache
	d.cacheMutex.Lock()
	d.balanceCache = balance
	d.cacheExpiry = time.Now().Add(d.cacheDuration)
	d.cacheMutex.Unlock()

	return balance, nil
}

// GetPositions implements Trader.GetPositions
func (d *DriftTrader) GetPositions() ([]map[string]interface{}, error) {
	// TODO: Query on-chain user account for positions
	// For now, return empty positions
	positions := []map[string]interface{}{}

	// Clear position cache as we're fetching fresh data
	d.cacheMutex.Lock()
	d.positionCache = make(map[string]*DriftPosition)
	d.cacheMutex.Unlock()

	return positions, nil
}

// OpenLong implements Trader.OpenLong
func (d *DriftTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	// Get market info
	market, err := d.getMarketInfo(symbol)
	if err != nil {
		return nil, err
	}

	// Convert quantity to base asset amount
	baseAssetAmount := ConvertToBaseAssetAmount(quantity)

	// TODO: Build and send place_perp_order instruction
	// This requires creating proper Anchor instruction format

	result := map[string]interface{}{
		"symbol":           symbol,
		"market_index":     market.MarketIndex,
		"side":             "LONG",
		"quantity":         quantity,
		"base_asset_amount": baseAssetAmount,
		"leverage":         leverage,
		"status":           "pending", // Would be signature in real implementation
		"timestamp":        time.Now().Unix(),
	}

	return result, fmt.Errorf("openLong not fully implemented - requires Anchor instruction builder")
}

// OpenShort implements Trader.OpenShort
func (d *DriftTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	// Get market info
	market, err := d.getMarketInfo(symbol)
	if err != nil {
		return nil, err
	}

	// Convert quantity to base asset amount
	baseAssetAmount := ConvertToBaseAssetAmount(quantity)

	// TODO: Build and send place_perp_order instruction

	result := map[string]interface{}{
		"symbol":           symbol,
		"market_index":     market.MarketIndex,
		"side":             "SHORT",
		"quantity":         quantity,
		"base_asset_amount": baseAssetAmount,
		"leverage":         leverage,
		"status":           "pending",
		"timestamp":        time.Now().Unix(),
	}

	return result, fmt.Errorf("openShort not fully implemented - requires Anchor instruction builder")
}

// CloseLong implements Trader.CloseLong
func (d *DriftTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	// Closing long = opening short of same size
	market, err := d.getMarketInfo(symbol)
	if err != nil {
		return nil, err
	}

	baseAssetAmount := ConvertToBaseAssetAmount(quantity)

	result := map[string]interface{}{
		"symbol":       symbol,
		"market_index": market.MarketIndex,
		"side":         "CLOSE_LONG",
		"quantity":     quantity,
		"base_asset_amount": baseAssetAmount,
		"status":       "pending",
		"timestamp":    time.Now().Unix(),
	}

	return result, fmt.Errorf("closeLong not fully implemented")
}

// CloseShort implements Trader.CloseShort
func (d *DriftTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	// Closing short = opening long of same size
	market, err := d.getMarketInfo(symbol)
	if err != nil {
		return nil, err
	}

	baseAssetAmount := ConvertToBaseAssetAmount(quantity)

	result := map[string]interface{}{
		"symbol":       symbol,
		"market_index": market.MarketIndex,
		"side":         "CLOSE_SHORT",
		"quantity":     quantity,
		"base_asset_amount": baseAssetAmount,
		"status":       "pending",
		"timestamp":    time.Now().Unix(),
	}

	return result, fmt.Errorf("closeShort not fully implemented")
}

// SetLeverage implements Trader.SetLeverage
func (d *DriftTrader) SetLeverage(symbol string, leverage int) error {
	// Drift uses cross-margin by default
	// Leverage is controlled by position size relative to collateral
	// This is a no-op for Drift
	return nil
}

// SetMarginMode implements Trader.SetMarginMode
func (d *DriftTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	// Drift only supports cross-margin mode
	if !isCrossMargin {
		return fmt.Errorf("Drift only supports cross-margin mode")
	}
	return nil
}

// GetMarketPrice implements Trader.GetMarketPrice
func (d *DriftTrader) GetMarketPrice(symbol string) (float64, error) {
	market, err := d.getMarketInfo(symbol)
	if err != nil {
		return 0, err
	}

	// TODO: Query oracle price or last trade price from DLOB API
	// For now, return error
	_ = market
	return 0, fmt.Errorf("getMarketPrice not implemented - use DLOB API")
}

// SetStopLoss implements Trader.SetStopLoss
func (d *DriftTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	// Drift supports trigger orders (stop-loss)
	// TODO: Implement trigger order instruction
	return fmt.Errorf("setStopLoss not implemented")
}

// SetTakeProfit implements Trader.SetTakeProfit
func (d *DriftTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	// Drift supports trigger orders (take-profit)
	// TODO: Implement trigger order instruction
	return fmt.Errorf("setTakeProfit not implemented")
}

// CancelAllOrders implements Trader.CancelAllOrders
func (d *DriftTrader) CancelAllOrders(symbol string) error {
	// TODO: Build cancel_orders instruction
	return fmt.Errorf("cancelAllOrders not implemented")
}

// FormatQuantity implements Trader.FormatQuantity
func (d *DriftTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	market, err := d.getMarketInfo(symbol)
	if err != nil {
		return "", err
	}

	// Round to market's minimum order size
	if market.MinOrderSize > 0 {
		// Round down to nearest min order size
		quantity = float64(int(quantity/market.MinOrderSize)) * market.MinOrderSize
	}

	return fmt.Sprintf("%.8f", quantity), nil
}

// getMarketInfo retrieves market info with caching
func (d *DriftTrader) getMarketInfo(symbol string) (*DriftMarket, error) {
	d.cacheMutex.RLock()
	if market, exists := d.marketCache[symbol]; exists && time.Now().Before(d.cacheExpiry) {
		d.cacheMutex.RUnlock()
		return market, nil
	}
	d.cacheMutex.RUnlock()

	// Fetch from API
	market, err := d.driftClient.GetMarketBySymbol(d.ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Cache it
	d.cacheMutex.Lock()
	d.marketCache[symbol] = market
	d.cacheMutex.Unlock()

	return market, nil
}

// HealthCheck verifies the trader is operational
func (d *DriftTrader) HealthCheck() error {
	// Check RPC connection
	if err := d.rpcClient.HealthCheck(d.ctx); err != nil {
		return fmt.Errorf("RPC health check failed: %w", err)
	}

	// Check SOL balance
	balance, err := d.rpcClient.GetBalanceSOL(d.ctx, d.wallet.GetPublicKey())
	if err != nil {
		return fmt.Errorf("failed to get SOL balance: %w", err)
	}

	if balance < 0.01 { // Minimum 0.01 SOL for gas
		return fmt.Errorf("insufficient SOL for gas fees: %.4f SOL", balance)
	}

	return nil
}

// GetWalletAddress returns the wallet's public address
func (d *DriftTrader) GetWalletAddress() string {
	return d.wallet.GetAddress()
}

// GetUserPDA returns the Drift user account PDA
func (d *DriftTrader) GetUserPDA() string {
	return d.driftClient.GetUserPDA().String()
}

// Close closes connections and cleans up resources
func (d *DriftTrader) Close() error {
	return d.rpcClient.Close()
}

// Helper method to check if user account exists
func (d *DriftTrader) userAccountExists() (bool, error) {
	accountInfo, err := d.rpcClient.GetAccountInfo(d.ctx, d.driftClient.GetUserPDA())
	if err != nil {
		// Account doesn't exist
		return false, nil
	}

	// Check if account has data
	if accountInfo == nil || accountInfo.Value == nil || len(accountInfo.Value.Data.GetBinary()) == 0 {
		return false, nil
	}

	return true, nil
}

// RequestAirdrop requests SOL airdrop on devnet (for testing)
func (d *DriftTrader) RequestAirdrop(amount float64) (solana.Signature, error) {
	if d.rpcClient.GetNetwork() == "mainnet-beta" {
		return solana.Signature{}, fmt.Errorf("airdrops not available on mainnet")
	}

	lamports := uint64(amount * 1e9) // Convert SOL to lamports
	return d.rpcClient.RequestAirdrop(d.ctx, d.wallet.GetPublicKey(), lamports)
}
