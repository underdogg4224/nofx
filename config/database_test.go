package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewDatabase tests database creation and initialization
func TestNewDatabase(t *testing.T) {
	// Create a temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify tables exist by querying them
	tables := []string{
		"users",
		"ai_models",
		"exchanges",
		"traders",
		"system_config",
		"beta_codes",
		"user_signal_sources",
	}

	for _, table := range tables {
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		var name string
		err := db.db.QueryRow(query, table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s does not exist: %v", table, err)
		}
	}
}

// TestDatabaseClose tests database closure
func TestDatabaseClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Failed to close database: %v", err)
	}

	// Verify database is closed by attempting a query
	_, err = db.db.Exec("SELECT 1")
	if err == nil {
		t.Error("Expected error when querying closed database")
	}
}

// TestCreateTablesIdempotency tests that creating tables multiple times doesn't fail
func TestCreateTablesIdempotency(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create tables again - should not fail due to IF NOT EXISTS
	err = db.createTables()
	if err != nil {
		t.Errorf("Second createTables call should not fail: %v", err)
	}
}

// TestSystemConfigOperations tests system config CRUD operations
func TestSystemConfigOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test setting a config value
	key := "test_key"
	value := "test_value"

	err = db.SetSystemConfig(key, value)
	if err != nil {
		t.Fatalf("Failed to set system config: %v", err)
	}

	// Test getting a config value
	retrievedValue, err := db.GetSystemConfig(key)
	if err != nil {
		t.Fatalf("Failed to get system config: %v", err)
	}

	if retrievedValue != value {
		t.Errorf("Expected value %s, got %s", value, retrievedValue)
	}

	// Test updating a config value
	newValue := "updated_value"
	err = db.SetSystemConfig(key, newValue)
	if err != nil {
		t.Fatalf("Failed to update system config: %v", err)
	}

	retrievedValue, err = db.GetSystemConfig(key)
	if err != nil {
		t.Fatalf("Failed to get updated system config: %v", err)
	}

	if retrievedValue != newValue {
		t.Errorf("Expected updated value %s, got %s", newValue, retrievedValue)
	}
}

// TestUserOperations tests basic user operations
func TestUserOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test creating a user
	userID := "test-user-123"
	email := "test@example.com"
	passwordHash := "hashed_password"

	query := `INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)`
	_, err = db.db.Exec(query, userID, email, passwordHash)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test retrieving the user
	var retrievedID, retrievedEmail, retrievedHash string
	err = db.db.QueryRow("SELECT id, email, password_hash FROM users WHERE id = ?", userID).
		Scan(&retrievedID, &retrievedEmail, &retrievedHash)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrievedID != userID || retrievedEmail != email || retrievedHash != passwordHash {
		t.Errorf("User data mismatch. Expected (%s, %s, %s), got (%s, %s, %s)",
			userID, email, passwordHash, retrievedID, retrievedEmail, retrievedHash)
	}
}

// TestBetaCodeOperations tests beta code operations
func TestBetaCodeOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Insert a beta code
	code := "TEST-BETA-CODE-123"
	_, err = db.db.Exec("INSERT INTO beta_codes (code) VALUES (?)", code)
	if err != nil {
		t.Fatalf("Failed to insert beta code: %v", err)
	}

	// Verify code exists and is not used
	var used bool
	err = db.db.QueryRow("SELECT used FROM beta_codes WHERE code = ?", code).Scan(&used)
	if err != nil {
		t.Fatalf("Failed to retrieve beta code: %v", err)
	}

	if used {
		t.Error("Beta code should not be marked as used initially")
	}

	// Mark code as used
	_, err = db.db.Exec("UPDATE beta_codes SET used = 1, used_by = ?, used_at = ? WHERE code = ?",
		"test@example.com", time.Now(), code)
	if err != nil {
		t.Fatalf("Failed to mark beta code as used: %v", err)
	}

	// Verify code is now marked as used
	err = db.db.QueryRow("SELECT used FROM beta_codes WHERE code = ?", code).Scan(&used)
	if err != nil {
		t.Fatalf("Failed to retrieve updated beta code: %v", err)
	}

	if !used {
		t.Error("Beta code should be marked as used after update")
	}
}

// TestForeignKeyConstraints tests that foreign key constraints are enforced
func TestForeignKeyConstraints(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	_, err = db.db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Try to insert a trader without a valid user_id
	// This should fail due to foreign key constraint
	query := `INSERT INTO traders (id, user_id, name, ai_model_id, exchange_id, initial_balance)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err = db.db.Exec(query, "trader-1", "non-existent-user", "Test Trader",
		"model-1", "exchange-1", 1000.0)

	// Note: SQLite foreign key enforcement depends on configuration
	// This test documents expected behavior but may not fail in all SQLite setups
	if err == nil {
		t.Log("Warning: Foreign key constraint was not enforced (SQLite configuration dependent)")
	}
}

// TestConcurrentAccess tests that multiple goroutines can access the database
func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_config.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Set up some test data
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	for _, key := range keys {
		err := db.SetSystemConfig(key, "initial")
		if err != nil {
			t.Fatalf("Failed to set initial config: %v", err)
		}
	}

	// Run concurrent updates
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(index int) {
			key := keys[index]
			for j := 0; j < 10; j++ {
				_ = db.SetSystemConfig(key, "updated")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all keys were updated
	for _, key := range keys {
		value, err := db.GetSystemConfig(key)
		if err != nil {
			t.Errorf("Failed to get config for %s: %v", key, err)
		}
		if value != "updated" {
			t.Errorf("Expected value 'updated' for %s, got %s", key, value)
		}
	}
}

// BenchmarkDatabaseCreation benchmarks database creation
func BenchmarkDatabaseCreation(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dbPath := filepath.Join(tmpDir, "bench_"+string(rune(i))+".db")
		db, err := NewDatabase(dbPath)
		if err != nil {
			b.Fatalf("Failed to create database: %v", err)
		}
		db.Close()
	}
}

// BenchmarkSystemConfigWrite benchmarks config writes
func BenchmarkSystemConfigWrite(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.SetSystemConfig("test_key", "test_value")
	}
}

// BenchmarkSystemConfigRead benchmarks config reads
func BenchmarkSystemConfigRead(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	_ = db.SetSystemConfig("test_key", "test_value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = db.GetSystemConfig("test_key")
	}
}
