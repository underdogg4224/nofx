package pool

import (
	"testing"
	"time"
)

func TestSetCoinPoolAPI(t *testing.T) {
	testURL := "https://test.api.com/coins"
	SetCoinPoolAPI(testURL)

	if coinPoolConfig.APIURL != testURL {
		t.Errorf("Expected API URL to be %s, got %s", testURL, coinPoolConfig.APIURL)
	}
}

func TestSetUseDefaultCoins(t *testing.T) {
	tests := []struct {
		name     string
		useDefault bool
		expected bool
	}{
		{
			name:     "Enable default coins",
			useDefault: true,
			expected: true,
		},
		{
			name:     "Disable default coins",
			useDefault: false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetUseDefaultCoins(tt.useDefault)
			if coinPoolConfig.UseDefaultCoins != tt.expected {
				t.Errorf("Expected UseDefaultCoins to be %v, got %v", tt.expected, coinPoolConfig.UseDefaultCoins)
			}
		})
	}
}

func TestSetDefaultCoins(t *testing.T) {
	tests := []struct {
		name     string
		coins    []string
		expected []string
	}{
		{
			name:     "Set custom coin list",
			coins:    []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"},
			expected: []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"},
		},
		{
			name:     "Set empty list (should not change)",
			coins:    []string{},
			expected: []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}, // Should remain unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDefaultCoins(tt.coins)

			if len(tt.coins) == 0 {
				// Empty list should not change the default
				if len(defaultMainstreamCoins) != len(tt.expected) {
					t.Errorf("Expected defaultMainstreamCoins to remain unchanged")
				}
			} else {
				if len(defaultMainstreamCoins) != len(tt.expected) {
					t.Errorf("Expected %d coins, got %d", len(tt.expected), len(defaultMainstreamCoins))
				}
				for i, coin := range tt.expected {
					if defaultMainstreamCoins[i] != coin {
						t.Errorf("Expected coin at index %d to be %s, got %s", i, coin, defaultMainstreamCoins[i])
					}
				}
			}
		})
	}
}

func TestCoinPoolConfig_Defaults(t *testing.T) {
	// Reset to defaults
	coinPoolConfig = CoinPoolConfig{
		APIURL:          "",
		Timeout:         30 * time.Second,
		CacheDir:        "coin_pool_cache",
		UseDefaultCoins: false,
	}

	tests := []struct {
		name     string
		field    string
		expected interface{}
		actual   interface{}
	}{
		{
			name:     "Default timeout",
			field:    "Timeout",
			expected: 30 * time.Second,
			actual:   coinPoolConfig.Timeout,
		},
		{
			name:     "Default cache directory",
			field:    "CacheDir",
			expected: "coin_pool_cache",
			actual:   coinPoolConfig.CacheDir,
		},
		{
			name:     "Default UseDefaultCoins",
			field:    "UseDefaultCoins",
			expected: false,
			actual:   coinPoolConfig.UseDefaultCoins,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("Expected %s to be %v, got %v", tt.field, tt.expected, tt.actual)
			}
		})
	}
}

func TestCoinInfo_Structure(t *testing.T) {
	coin := CoinInfo{
		Pair:            "BTCUSDT",
		Score:           85.5,
		StartTime:       time.Now().Unix(),
		StartPrice:      50000.0,
		LastScore:       90.0,
		MaxScore:        95.0,
		MaxPrice:        55000.0,
		IncreasePercent: 10.0,
		IsAvailable:     true,
	}

	if coin.Pair != "BTCUSDT" {
		t.Errorf("Expected Pair to be BTCUSDT, got %s", coin.Pair)
	}
	if coin.Score != 85.5 {
		t.Errorf("Expected Score to be 85.5, got %f", coin.Score)
	}
	if coin.IncreasePercent != 10.0 {
		t.Errorf("Expected IncreasePercent to be 10.0, got %f", coin.IncreasePercent)
	}
	if !coin.IsAvailable {
		t.Error("Expected IsAvailable to be true")
	}
}

func TestGetCoinPool_WithDefaultCoins(t *testing.T) {
	// Enable default coins
	SetUseDefaultCoins(true)
	SetDefaultCoins([]string{"BTCUSDT", "ETHUSDT", "SOLUSDT"})

	coins, err := GetCoinPool()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(coins) != 3 {
		t.Errorf("Expected 3 coins, got %d", len(coins))
	}

	expectedPairs := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	for i, coin := range coins {
		if coin.Pair != expectedPairs[i] {
			t.Errorf("Expected coin %d to be %s, got %s", i, expectedPairs[i], coin.Pair)
		}
	}

	// Reset for other tests
	SetUseDefaultCoins(false)
}

// Benchmark tests
func BenchmarkGetCoinPool_DefaultCoins(b *testing.B) {
	SetUseDefaultCoins(true)
	SetDefaultCoins([]string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetCoinPool()
	}
}

func BenchmarkSetDefaultCoins(b *testing.B) {
	coins := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT", "XRPUSDT"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetDefaultCoins(coins)
	}
}
