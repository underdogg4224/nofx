package config

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func newTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestDatabaseDefaultAdminModeDisabled(t *testing.T) {
	db := newTestDB(t)

	adminMode, err := db.GetSystemConfig("admin_mode")
	if err != nil {
		t.Fatalf("GetSystemConfig(admin_mode) failed: %v", err)
	}
	if adminMode != "false" {
		t.Fatalf("expected default admin_mode to be 'false', got %q", adminMode)
	}
}

func TestTraderOwnershipAndScopedMutations(t *testing.T) {
	db := newTestDB(t)

	trader := &TraderRecord{
		ID:                  "trader-owner-1",
		UserID:              "user-a",
		Name:                "Alpha Bot",
		AIModelID:           "user-a_deepseek",
		ExchangeID:          "binance",
		InitialBalance:      1000,
		ScanIntervalMinutes: 3,
		BTCETHLeverage:      5,
		AltcoinLeverage:     5,
		CustomPrompt:        "original prompt",
		IsCrossMargin:       true,
	}
	if err := db.CreateTrader(trader); err != nil {
		t.Fatalf("CreateTrader failed: %v", err)
	}

	// Owner check succeeds for user-a and fails for user-b.
	owned, err := db.VerifyTraderOwnership("user-a", trader.ID)
	if err != nil || !owned {
		t.Fatalf("expected user-a to own trader, got owned=%v err=%v", owned, err)
	}
	owned, err = db.VerifyTraderOwnership("user-b", trader.ID)
	if err != nil || owned {
		t.Fatalf("expected user-b NOT to own trader, got owned=%v err=%v", owned, err)
	}

	// Non-owner prompt update must fail with sql.ErrNoRows and leave prompt unchanged.
	err = db.UpdateTraderCustomPrompt("user-b", trader.ID, "malicious prompt", true)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for non-owner prompt update, got %v", err)
	}

	// Non-owner delete must fail with sql.ErrNoRows and leave trader intact.
	err = db.DeleteTrader("user-b", trader.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for non-owner delete, got %v", err)
	}

	traders, err := db.GetTraders("user-a")
	if err != nil || len(traders) != 1 {
		t.Fatalf("expected user-a's trader to still exist, got len=%d err=%v", len(traders), err)
	}
	if traders[0].CustomPrompt != "original prompt" {
		t.Fatalf("expected original prompt to remain intact, got %q", traders[0].CustomPrompt)
	}

	// Owner prompt update and delete succeed.
	if err := db.UpdateTraderCustomPrompt("user-a", trader.ID, "updated prompt", false); err != nil {
		t.Fatalf("owner UpdateTraderCustomPrompt failed: %v", err)
	}
	if err := db.DeleteTrader("user-a", trader.ID); err != nil {
		t.Fatalf("owner DeleteTrader failed: %v", err)
	}
}
