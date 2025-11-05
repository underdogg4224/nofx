# Solana Perpetuals Trading - Quick Start Guide

**Status:** Phase 2 Complete (Foundation + Drift Protocol Integration)
**Progress:** 30% (2 of 7 phases)

---

## 🎯 What is This?

This integration enables NOFX to trade perpetual futures on **Drift Protocol**, Solana's leading decentralized perpetuals exchange. Unlike centralized exchanges (Binance) or EVM-based DEXs (Hyperliquid, Aster), Drift runs natively on Solana blockchain with:

- ✅ **Full transparency** - All trades on-chain
- ✅ **Lower fees** - No custodial intermediaries
- ✅ **Cross-margin trading** - Up to 101x leverage
- ✅ **High liquidity** - Hybrid orderbook + AMM design
- ✅ **Fast execution** - Solana's 400ms block times

---

## 📦 Components

```
trader/
├── solana_wallet.go      - Wallet management, transaction signing
├── solana_rpc.go         - Solana blockchain RPC client
├── drift_client.go       - Drift Protocol API wrapper
└── drift_trader.go       - Trader interface implementation

docs/
├── SOLANA_PERPS_INTEGRATION_PLAN.md  - Complete 7-phase roadmap
├── PHASE2_IMPLEMENTATION_SUMMARY.md   - Phase 2 detailed review
├── PHASE3_ANCHOR_INSTRUCTIONS.md      - Phase 3 implementation guide
└── SOLANA_INTEGRATION_README.md       - This file
```

---

## 🚀 Quick Start (Devnet Testing)

### 1. Get Devnet SOL

```bash
# Install Solana CLI (if not already installed)
sh -c "$(curl -sSfL https://release.solana.com/stable/install)"

# Set to devnet
solana config set --url devnet

# Create wallet (or use existing)
solana-keygen new --outfile ~/devnet-wallet.json

# Get airdrop
solana airdrop 2 --keypair ~/devnet-wallet.json
```

### 2. Initialize Drift User Account

Currently requires using Drift UI or TypeScript SDK:

```bash
# Visit Drift devnet UI
https://app.drift.trade/?cluster=devnet

# Or use TypeScript SDK
# (See docs/PHASE3_ANCHOR_INSTRUCTIONS.md for details)
```

### 3. Configure NOFX for Drift

```go
config := trader.AutoTraderConfig{
    // Trader info
    ID:   "drift-test-1",
    Name: "Drift Devnet Test",

    // Exchange selection
    Exchange: "drift",

    // Drift configuration
    DriftNetwork:      "devnet",
    DriftPrivateKey:   "YOUR_BASE58_PRIVATE_KEY",
    DriftRPCURL:       "", // Empty = use default
    DriftWSURL:        "", // Empty = use default
    DriftSubAccountID: 0,  // Use subaccount 0

    // AI configuration
    AIModel:     "deepseek",
    DeepSeekKey: "YOUR_DEEPSEEK_KEY",

    // Trading parameters
    InitialBalance:   1000.0,
    ScanInterval:     3 * time.Minute,
    BTCETHLeverage:   5,
    AltcoinLeverage:  5,
    IsCrossMargin:    true,
}

trader, err := trader.NewAutoTrader(config)
if err != nil {
    log.Fatal(err)
}
```

### 4. Test Connection

```go
// Health check
if err := trader.HealthCheck(); err != nil {
    log.Fatalf("Health check failed: %v", err)
}

// Get balance
balance, err := trader.GetBalance()
if err != nil {
    log.Fatalf("Failed to get balance: %v", err)
}
log.Printf("SOL Balance: %.4f", balance["sol_balance"])

// Get wallet info
log.Printf("Wallet: %s", trader.GetWalletAddress())
log.Printf("User PDA: %s", trader.GetUserPDA())
```

---

## 📊 Current Capabilities

### ✅ What Works (Phase 1 & 2)

| Feature | Status | Description |
|---------|--------|-------------|
| Wallet Management | ✅ Complete | Load/create wallets, sign transactions |
| RPC Connectivity | ✅ Complete | Mainnet/devnet support, WebSocket |
| Balance Queries | ✅ Partial | SOL balance works, USDC pending |
| Market Data | ✅ Complete | Fetch markets from Drift API |
| Health Checks | ✅ Complete | Monitor RPC and SOL balance |
| Trader Interface | ✅ Complete | All 14 methods implemented |
| Database Support | ✅ Complete | Drift added to exchanges |
| Auto Trader Integration | ✅ Complete | Works with existing system |

### ⏳ What's Pending (Phase 3+)

| Feature | Status | Timeline |
|---------|--------|----------|
| Trading Operations | 🚧 Phase 3 | Week 5-6 |
| Position Queries | 🚧 Phase 3 | Week 5-6 |
| USDC Balance | 🚧 Phase 3 | Week 5-6 |
| Market Prices | 🚧 Phase 4 | Week 7 |
| WebSocket Updates | 🚧 Phase 4 | Week 7 |
| Frontend UI | 🚧 Phase 5 | Week 8 |
| Full Testing | 🚧 Phase 6 | Week 9-10 |
| Mainnet Deployment | 🚧 Phase 7 | Week 11 |

---

## 🔧 Configuration Options

### Network Selection

```go
// Mainnet (real money!)
DriftNetwork: "mainnet-beta"

// Devnet (testnet SOL)
DriftNetwork: "devnet"
```

### Custom RPC Endpoints

For better performance, use paid RPC providers:

```go
// Helius
DriftRPCURL: "https://mainnet.helius-rpc.com/?api-key=YOUR_KEY"

// QuickNode
DriftRPCURL: "https://YOUR_ENDPOINT.solana-mainnet.quiknode.pro/YOUR_KEY/"

// Triton
DriftRPCURL: "https://YOUR_ENDPOINT.rpcpool.com/YOUR_KEY"
```

### Subaccount Selection

Drift supports 0-9 subaccounts per wallet:

```go
DriftSubAccountID: 0  // Main account
DriftSubAccountID: 1  // Subaccount 1 (separate positions/margin)
// ...
DriftSubAccountID: 9  // Subaccount 9
```

---

## 🔒 Security Best Practices

### Private Key Management

**DO:**
- ✅ Use dedicated trading wallets (not main wallet)
- ✅ Store private keys encrypted at rest
- ✅ Use environment variables for keys
- ✅ Limit wallet balance to trading capital + gas fees
- ✅ Test on devnet first

**DON'T:**
- ❌ Use your main wallet for trading bots
- ❌ Store private keys in plain text
- ❌ Commit private keys to git
- ❌ Share private keys in logs
- ❌ Start on mainnet without testing

### SOL for Gas Fees

- Keep **minimum 0.01 SOL** for transaction fees
- Typical transaction cost: ~0.00005 SOL (5,000 lamports)
- Budget: 0.5 SOL = ~10,000 transactions
- Monitor balance regularly

### RPC Rate Limits

- **Public RPCs:** Rate limited, may be slow
- **Recommended:** Use paid RPC for production
- **Cost:** ~$50-200/month for production usage

---

## 📚 Documentation Index

| Document | Purpose | Audience |
|----------|---------|----------|
| [SOLANA_PERPS_INTEGRATION_PLAN.md](SOLANA_PERPS_INTEGRATION_PLAN.md) | Complete 7-phase roadmap | Product, Dev |
| [PHASE2_IMPLEMENTATION_SUMMARY.md](PHASE2_IMPLEMENTATION_SUMMARY.md) | Phase 2 detailed review | Code reviewers |
| [PHASE3_ANCHOR_INSTRUCTIONS.md](PHASE3_ANCHOR_INSTRUCTIONS.md) | Anchor implementation guide | Developers |
| **SOLANA_INTEGRATION_README.md** | Quick start guide | **All users** |

---

## 🧪 Testing Checklist

Before deploying to mainnet:

- [ ] Test wallet creation on devnet
- [ ] Verify SOL airdrop works
- [ ] Initialize Drift user account
- [ ] Confirm health check passes
- [ ] Test balance queries
- [ ] Verify market data fetching
- [ ] Check RPC connectivity
- [ ] Monitor transaction confirmations
- [ ] Test with small positions first
- [ ] Verify logging and monitoring
- [ ] Review transaction fees
- [ ] Test error handling
- [ ] Verify stop-loss mechanisms
- [ ] Run for 24 hours on devnet
- [ ] Review all trade decisions

---

## 🐛 Troubleshooting

### Issue: "Insufficient SOL for gas fees"

**Solution:**
```bash
# Get more devnet SOL
solana airdrop 2 --keypair ~/devnet-wallet.json

# Check current balance
solana balance --keypair ~/devnet-wallet.json
```

### Issue: "User account not initialized"

**Solution:**
Visit Drift UI to initialize user account:
```
https://app.drift.trade/?cluster=devnet
```

### Issue: "RPC health check failed"

**Solutions:**
1. Check network connectivity
2. Verify RPC endpoint is correct
3. Try alternative RPC endpoint
4. Check Solana network status: https://status.solana.com/

### Issue: "Transaction failed: 0x1 (Insufficient funds)"

**Solution:**
Deposit USDC to Drift account via Drift UI

### Issue: "Market not found: SOL-PERP"

**Solution:**
Verify symbol format:
- Correct: "SOL-PERP"
- Incorrect: "SOLUSDT", "SOL/USDC"

---

## 📊 Architecture Overview

```
┌─────────────────────────────────────────┐
│         NOFX AI Trading System          │
├─────────────────────────────────────────┤
│                                          │
│  AutoTrader                              │
│      ↓                                   │
│  DriftTrader (implements Trader)         │
│      ↓                                   │
│  DriftClient (Drift API wrapper)         │
│      ↓                                   │
│  SolanaRPCClient + SolanaWallet          │
│      ↓                                   │
│  ┌──────────────────────────────────┐   │
│  │  Solana Blockchain Network       │   │
│  │  ├─ Drift Protocol Program       │   │
│  │  ├─ User Accounts                │   │
│  │  ├─ Perp Markets                 │   │
│  │  └─ Oracle Feeds (Pyth)          │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

---

## 🔗 External Resources

**Drift Protocol:**
- Website: https://www.drift.trade/
- Docs: https://docs.drift.trade/
- Discord: https://discord.gg/drift
- GitHub: https://github.com/drift-labs/protocol-v2
- Devnet UI: https://app.drift.trade/?cluster=devnet
- Mainnet UI: https://app.drift.trade/

**Solana:**
- Docs: https://docs.solana.com/
- Status: https://status.solana.com/
- Explorer (Mainnet): https://explorer.solana.com/
- Explorer (Devnet): https://explorer.solana.com/?cluster=devnet
- Faucet: https://faucet.solana.com/

**Go SDKs:**
- solana-go: https://github.com/gagliardetto/solana-go
- anchor-go: https://github.com/gagliardetto/anchor-go

---

## ❓ FAQ

**Q: Can I use this on mainnet now?**
A: Not yet. Trading operations require Phase 3 (Anchor instructions). Currently only market data and balance queries work.

**Q: Do I need SOL to trade?**
A: Yes, for transaction fees. ~0.01 SOL minimum, but 0.5 SOL recommended for active trading.

**Q: What about USDC for trading?**
A: You need to deposit USDC to your Drift account via the Drift UI.

**Q: Is my private key safe?**
A: Private keys are stored in memory only. Use encrypted storage and dedicated trading wallets.

**Q: What's the transaction cost?**
A: ~0.00005 SOL (~$0.01) per transaction at current prices.

**Q: Can I use this with existing NOFX traders?**
A: Yes! Drift integrates seamlessly with the existing AutoTrader system.

**Q: When will full trading support be ready?**
A: Phase 3 (trading operations) is planned for Week 5-6. See roadmap for details.

---

## 🤝 Contributing

Interested in contributing to Solana integration?

1. Review the integration plan: `SOLANA_PERPS_INTEGRATION_PLAN.md`
2. Check Phase 3 guide: `PHASE3_ANCHOR_INSTRUCTIONS.md`
3. Test on devnet and provide feedback
4. Submit issues or PRs to the repository

---

## 📞 Support

**For NOFX-specific issues:**
- GitHub Issues: https://github.com/underdogg4224/nofx/issues

**For Drift Protocol questions:**
- Discord: https://discord.gg/drift
- Docs: https://docs.drift.trade/

**For Solana questions:**
- Stack Exchange: https://solana.stackexchange.com/
- Discord: https://discord.gg/solana

---

**Document Version:** 1.0
**Last Updated:** November 5, 2025
**Maintained By:** NOFX Development Team
**License:** Same as NOFX project
