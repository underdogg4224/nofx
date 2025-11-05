# Phase 3: Anchor Instruction Builders

**Goal:** Enable full transaction support for Drift Protocol trading operations
**Status:** 🚧 In Progress
**Timeline:** Week 5-6

---

## 📋 Overview

Phase 3 focuses on building Anchor instructions to execute actual trades on Drift Protocol. This requires:

1. Drift Protocol IDL (Interface Description Language) file
2. Instruction discriminator calculation
3. Account meta construction
4. Instruction data serialization
5. Transaction building and signing

---

## 🛠️ Implementation Approach

### Option 1: Use anchor-go (Recommended)

The `anchor-go` tool by gagliardetto can generate Go clients from Anchor IDL files.

**Installation:**
```bash
go install github.com/gagliardetto/anchor-go@latest
```

**Usage:**
```bash
# Generate Drift client from IDL
anchor-go \
  --idl ./drift_idl.json \
  --output ./trader/drift_generated \
  --program-id dRiftyHA39MWEi3m9aunc5MzRF1JYuBsbn6VPcn33UH
```

**Requirements:**
- Drift Protocol IDL file (JSON)
- anchor-go tool supports Anchor v0.30.0+ IDL format
- Generated code will include instruction builders and account deserializers

### Option 2: Manual Implementation

For learning or custom needs, instructions can be built manually.

---

## 📥 Getting Drift IDL

### Method 1: From Drift GitHub
```bash
# Clone Drift protocol-v2
git clone https://github.com/drift-labs/protocol-v2.git
cd protocol-v2

# Build and extract IDL
anchor build
# IDL will be in target/idl/drift.json
```

### Method 2: Fetch from Blockchain
```bash
# Fetch IDL from on-chain account
anchor idl fetch -o drift_idl.json dRiftyHA39MWEi3m9aunc5MzRF1JYuBsbn6VPcn33UH
```

### Method 3: From Drift Gateway
The Drift Gateway project may have the IDL readily available.

---

## 🧩 Instruction Structure

### Anchor Instruction Format

All Anchor instructions follow this structure:

```
[ Discriminator (8 bytes) ][ Instruction Data (variable) ]
```

**Discriminator Calculation:**
```go
import "crypto/sha256"

func instructionDiscriminator(name string) [8]byte {
    preimage := fmt.Sprintf("global:%s", name)
    hash := sha256.Sum256([]byte(preimage))
    var disc [8]byte
    copy(disc[:], hash[:8])
    return disc
}
```

**Example:**
```go
// For "place_perp_order" instruction
disc := instructionDiscriminator("place_perp_order")
// disc = first 8 bytes of SHA256("global:place_perp_order")
```

---

## 📝 Key Instructions to Implement

### 1. place_perp_order

**Purpose:** Open long/short position or place limit order

**Parameters:**
```go
type PlacePerpOrderParams struct {
    OrderType       OrderType       // Market, Limit, etc.
    MarketType      MarketType      // Perp
    Direction       PositionDirection // Long or Short
    UserOrderId     uint8
    BaseAssetAmount uint64          // In BASE_PRECISION (1e9)
    Price           uint64          // In PRICE_PRECISION (1e6)
    MarketIndex     uint16
    ReduceOnly      bool
    PostOnly        PostOnlyParams
    ImmediateOrCancel bool
    MaxTs           int64           // Unix timestamp
    TriggerPrice    uint64
    TriggerCondition TriggerCondition
    OraclePriceOffset int32
    AuctionDuration uint8
    AuctionStartPrice int64
    AuctionEndPrice   int64
}
```

**Accounts Required:**
```go
[]AccountMeta{
    {Pubkey: stateAccount, IsSigner: false, IsWritable: false},
    {Pubkey: userAccount, IsSigner: false, IsWritable: true},
    {Pubkey: userStatsAccount, IsSigner: false, IsWritable: true},
    {Pubkey: authority, IsSigner: true, IsWritable: false},
    // ... oracle accounts, perp market accounts, etc.
}
```

### 2. cancel_order

**Purpose:** Cancel an open order

**Parameters:**
```go
type CancelOrderParams struct {
    OrderId *uint32 // Optional order ID
}
```

### 3. cancel_orders

**Purpose:** Cancel multiple orders

**Parameters:**
```go
type CancelOrdersParams struct {
    MarketType  *MarketType
    MarketIndex *uint16
    Direction   *PositionDirection
}
```

### 4. close_position

**Purpose:** Close entire position in a market

**Parameters:**
```go
type ClosePositionParams struct {
    MarketIndex uint16
}
```

---

## 🏗️ Implementation Example

### Basic Instruction Builder (Manual)

```go
package trader

import (
    "crypto/sha256"
    "fmt"

    "github.com/gagliardetto/solana-go"
    bin "github.com/gagliardetto/binary"
)

// OrderType enum
type OrderType uint8

const (
    OrderTypeMarket OrderType = iota
    OrderTypeLimit
    OrderTypeTriggerMarket
    OrderTypeTriggerLimit
    OrderTypeOracle
)

// PositionDirection enum
type PositionDirection uint8

const (
    PositionDirectionLong PositionDirection = iota
    PositionDirectionShort
)

// PlacePerpOrderParams represents parameters for placing a perp order
type PlacePerpOrderParams struct {
    OrderType         OrderType
    MarketIndex       uint16
    Direction         PositionDirection
    BaseAssetAmount   uint64
    Price             uint64
    ReduceOnly        bool
    PostOnly          uint8
    ImmediateOrCancel bool
    MaxTs             *int64
    TriggerPrice      *uint64
    TriggerCondition  *uint8
    OraclePriceOffset *int32
    AuctionDuration   *uint8
    AuctionStartPrice *int64
    AuctionEndPrice   *int64
}

// BuildPlacePerpOrderInstruction builds the place_perp_order instruction
func BuildPlacePerpOrderInstruction(
    params *PlacePerpOrderParams,
    programID solana.PublicKey,
    stateAccount solana.PublicKey,
    userAccount solana.PublicKey,
    userStatsAccount solana.PublicKey,
    authority solana.PublicKey,
    // ... additional accounts
) (*solana.Instruction, error) {
    // Calculate discriminator
    disc := instructionDiscriminator("place_perp_order")

    // Serialize instruction data
    data := make([]byte, 0, 256)
    data = append(data, disc[:]...)

    // Serialize parameters using borsh encoding
    // This is simplified - real implementation needs proper borsh serialization
    encoder := bin.NewBorshEncoder(data)

    // Encode each field according to Drift's Rust struct layout
    encoder.WriteU8(uint8(params.OrderType))
    encoder.WriteU16(params.MarketIndex, binary.LittleEndian)
    encoder.WriteU8(uint8(params.Direction))
    encoder.WriteU64(params.BaseAssetAmount, binary.LittleEndian)
    encoder.WriteU64(params.Price, binary.LittleEndian)
    // ... encode remaining fields

    data = encoder.Bytes()

    // Build account metas
    accounts := []*solana.AccountMeta{
        {Pubkey: stateAccount, IsSigner: false, IsWritable: false},
        {Pubkey: userAccount, IsSigner: false, IsWritable: true},
        {Pubkey: userStatsAccount, IsSigner: false, IsWritable: true},
        {Pubkey: authority, IsSigner: true, IsWritable: false},
        // ... additional accounts (perp market, oracle, etc.)
    }

    return solana.NewInstruction(
        programID,
        accounts,
        data,
    ), nil
}

// Helper function to calculate instruction discriminator
func instructionDiscriminator(name string) [8]byte {
    preimage := fmt.Sprintf("global:%s", name)
    hash := sha256.Sum256([]byte(preimage))
    var disc [8]byte
    copy(disc[:], hash[:8])
    return disc
}
```

---

## 🔄 Transaction Flow

### Complete Trade Execution

```go
func (d *DriftTrader) ExecuteTrade(
    ctx context.Context,
    symbol string,
    direction PositionDirection,
    quantity float64,
    leverage int,
) (solana.Signature, error) {
    // 1. Get market info
    market, err := d.getMarketInfo(symbol)
    if err != nil {
        return solana.Signature{}, err
    }

    // 2. Convert to base asset amount
    baseAmount := ConvertToBaseAssetAmount(quantity)

    // 3. Get recent blockhash
    blockhash, err := d.rpcClient.GetRecentBlockhash(ctx)
    if err != nil {
        return solana.Signature{}, err
    }

    // 4. Build instruction
    params := &PlacePerpOrderParams{
        OrderType:       OrderTypeMarket,
        MarketIndex:     market.MarketIndex,
        Direction:       direction,
        BaseAssetAmount: uint64(baseAmount),
        Price:           0, // Market order
        ReduceOnly:      false,
    }

    instruction, err := BuildPlacePerpOrderInstruction(
        params,
        d.driftClient.config.ProgramID,
        stateAccount,
        d.driftClient.userPDA,
        userStatsAccount,
        d.wallet.GetPublicKey(),
    )
    if err != nil {
        return solana.Signature{}, err
    }

    // 5. Build transaction
    tx, err := solana.NewTransaction(
        []solana.Instruction{instruction},
        blockhash,
        solana.TransactionPayer(d.wallet.GetPublicKey()),
    )
    if err != nil {
        return solana.Signature{}, err
    }

    // 6. Sign transaction
    if err := d.wallet.SignTransaction(tx); err != nil {
        return solana.Signature{}, err
    }

    // 7. Send and confirm
    sig, err := d.rpcClient.SendAndConfirmTransaction(
        ctx,
        tx,
        rpc.CommitmentConfirmed,
    )
    if err != nil {
        return solana.Signature{}, err
    }

    return sig, nil
}
```

---

## 📦 Account Derivation

### Required PDAs

```go
// User account PDA (already implemented in drift_client.go)
func (c *DriftClient) GetUserAccountPDA(subAccountID uint16) (solana.PublicKey, error) {
    seeds := [][]byte{
        []byte("user"),
        c.wallet.GetPublicKey().Bytes(),
        {byte(subAccountID), byte(subAccountID >> 8)},
    }
    pda, _, err := solana.FindProgramAddress(seeds, c.config.ProgramID)
    return pda, err
}

// User stats account PDA
func (c *DriftClient) GetUserStatsAccountPDA() (solana.PublicKey, error) {
    seeds := [][]byte{
        []byte("user_stats"),
        c.wallet.GetPublicKey().Bytes(),
    }
    pda, _, err := solana.FindProgramAddress(seeds, c.config.ProgramID)
    return pda, err
}

// State account (constant)
func GetStateAccount() solana.PublicKey {
    // Drift state account is a known address
    return solana.MustPublicKeyFromBase58("BXJ4UJU1mYDVNgb5D8x95XLV4P6eBXuJGpvx4oQUv45A")
}
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
func TestInstructionDiscriminator(t *testing.T) {
    disc := instructionDiscriminator("place_perp_order")
    // Verify against known discriminator from TypeScript SDK
    expected := [8]byte{/* expected bytes */}
    assert.Equal(t, expected, disc)
}

func TestBuildPlacePerpOrderInstruction(t *testing.T) {
    // Test instruction building
    params := &PlacePerpOrderParams{
        OrderType:       OrderTypeMarket,
        MarketIndex:     0,
        Direction:       PositionDirectionLong,
        BaseAssetAmount: ConvertToBaseAssetAmount(1.0),
    }

    instruction, err := BuildPlacePerpOrderInstruction(params, ...)
    assert.NoError(t, err)
    assert.NotNil(t, instruction)

    // Verify instruction data format
    assert.Equal(t, 8, len(instruction.Data[:8])) // Discriminator
}
```

### Integration Tests (Devnet)

```go
func TestDevnetTrade(t *testing.T) {
    trader := setupDevnetTrader(t)

    // Open small position
    sig, err := trader.ExecuteTrade(
        context.Background(),
        "SOL-PERP",
        PositionDirectionLong,
        0.01, // 0.01 SOL
        1,    // 1x leverage
    )

    assert.NoError(t, err)
    assert.NotEmpty(t, sig)

    // Wait for confirmation
    time.Sleep(5 * time.Second)

    // Verify position opened
    positions, err := trader.GetPositions()
    assert.NoError(t, err)
    assert.Len(t, positions, 1)
}
```

---

## 🔍 Account Deserialization

### Drift User Account Structure

```go
type DriftUserAccount struct {
    // Account discriminator (first 8 bytes)
    Discriminator [8]byte

    // User fields
    Authority        solana.PublicKey
    Delegate         solana.PublicKey
    Name             [32]byte
    SpotPositions    [8]SpotPosition
    PerpPositions    [8]PerpPosition
    Orders           [32]Order
    LastAddPerpLpSharesTs int64
    TotalDeposits    uint64
    TotalWithdraws   uint64
    TotalSocialLoss  uint64
    SettledPerpPnl   int64
    CumulativeSpo tFees int64
    CumulativePerpFunding int64
    LiquidationMarginFreed uint64
    LastActiveSlot   uint64
    SubAccountId     uint16
    Status           uint8
    IsMarginTradingEnabled bool
    Idle             bool
    OpenOrders       uint8
    HasOpenOrder     bool
    OpenAuctions     uint8
    HasOpenAuction   bool
    Padding          [21]byte
}

type PerpPosition struct {
    LastCumulativeFundingRate int64
    BaseAssetAmount          int64
    QuoteAssetAmount         int64
    QuoteBreakEvenAmount     int64
    QuoteEntryAmount         int64
    OpenBids                 int64
    OpenAsks                 int64
    SettledPnl               int64
    LpShares                 uint64
    LastBaseAssetAmountPerLp int64
    LastQuoteAssetAmountPerLp int64
    RemainderBaseAssetAmount int32
    MarketIndex              uint16
    OpenOrders               uint8
    PerLpBase                int8
}
```

### Deserialization Function

```go
func DeserializeUserAccount(data []byte) (*DriftUserAccount, error) {
    if len(data) < 8 {
        return nil, fmt.Errorf("account data too short")
    }

    // Verify discriminator
    expectedDisc := accountDiscriminator("User")
    actualDisc := [8]byte{}
    copy(actualDisc[:], data[:8])

    if actualDisc != expectedDisc {
        return nil, fmt.Errorf("invalid account discriminator")
    }

    // Deserialize using borsh
    decoder := bin.NewBorshDecoder(data[8:])
    account := &DriftUserAccount{}

    // Decode each field according to Rust struct layout
    decoder.ReadBytes(&account.Authority[:], false)
    decoder.ReadBytes(&account.Delegate[:], false)
    decoder.ReadBytes(&account.Name[:], false)
    // ... decode remaining fields

    return account, nil
}
```

---

## 📚 Resources

**Drift Protocol:**
- GitHub: https://github.com/drift-labs/protocol-v2
- IDL Location: `target/idl/drift.json` (after build)
- Gateway: https://github.com/drift-labs/gateway
- Python SDK: https://github.com/drift-labs/driftpy

**Anchor Tools:**
- anchor-go: https://github.com/gagliardetto/anchor-go
- Anchor Docs: https://www.anchor-lang.com/
- IDL Spec: https://www.anchor-lang.com/docs/basics/idl

**Solana Go:**
- solana-go: https://github.com/gagliardetto/solana-go
- Binary encoding: https://github.com/gagliardetto/binary

---

## ✅ Phase 3 Checklist

- [ ] Obtain Drift Protocol IDL file
- [ ] Generate Go client using anchor-go
- [ ] Implement `place_perp_order` instruction
- [ ] Implement `cancel_order` instruction
- [ ] Implement `close_position` instruction
- [ ] Add account deserialization for User account
- [ ] Add account deserialization for PerpPosition
- [ ] Update `OpenLong()` to use real transactions
- [ ] Update `OpenShort()` to use real transactions
- [ ] Update `CloseLong()` to use real transactions
- [ ] Update `CloseShort()` to use real transactions
- [ ] Update `GetPositions()` to query on-chain data
- [ ] Update `GetBalance()` to include USDC balance
- [ ] Add comprehensive error handling
- [ ] Add transaction retry logic
- [ ] Test on devnet with real trades
- [ ] Document all instructions
- [ ] Create user guide for testing

---

## ⏭️ Next Steps

1. **Get Drift IDL**
   ```bash
   git clone https://github.com/drift-labs/protocol-v2
   cd protocol-v2
   anchor build
   cp target/idl/drift.json /home/user/nofx/drift_idl.json
   ```

2. **Generate Go Client**
   ```bash
   anchor-go \
     --idl drift_idl.json \
     --output trader/drift_generated \
     --program-id dRiftyHA39MWEi3m9aunc5MzRF1JYuBsbn6VPcn33UH
   ```

3. **Integrate Generated Code**
   - Update `drift_trader.go` to use generated instructions
   - Replace stub implementations with real transactions
   - Test on devnet

---

**Document Version:** 1.0
**Last Updated:** November 5, 2025
**Status:** Phase 3 Planning Complete
**Next Action:** Obtain Drift IDL and generate client
