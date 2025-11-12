package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSetJWTSecret(t *testing.T) {
	testSecret := "test-secret-key-123"
	SetJWTSecret(testSecret)

	if string(JWTSecret) != testSecret {
		t.Errorf("Expected JWT secret to be %s, got %s", testSecret, string(JWTSecret))
	}
}

func TestSetAdminMode(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "Enable admin mode",
			enabled:  true,
			expected: true,
		},
		{
			name:     "Disable admin mode",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAdminMode(tt.enabled)
			if IsAdminMode() != tt.expected {
				t.Errorf("Expected admin mode to be %v, got %v", tt.expected, IsAdminMode())
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	password := "testPassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	if hash == password {
		t.Error("Hash should not equal plain password")
	}

	// Verify hash starts with bcrypt prefix
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		t.Errorf("Expected bcrypt hash format, got %s", hash)
	}
}

func TestCheckPassword(t *testing.T) {
	password := "correctPassword123!"
	wrongPassword := "wrongPassword456!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		expected bool
	}{
		{
			name:     "Correct password",
			password: password,
			hash:     hash,
			expected: true,
		},
		{
			name:     "Wrong password",
			password: wrongPassword,
			hash:     hash,
			expected: false,
		},
		{
			name:     "Empty password",
			password: "",
			hash:     hash,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPassword(tt.password, tt.hash)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateOTPSecret(t *testing.T) {
	secret, err := GenerateOTPSecret()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if secret == "" {
		t.Error("Expected non-empty OTP secret")
	}

	// Generate another secret to ensure uniqueness
	secret2, err := GenerateOTPSecret()
	if err != nil {
		t.Fatalf("Expected no error on second generation, got %v", err)
	}

	if secret == secret2 {
		t.Error("Expected unique OTP secrets on each generation")
	}
}

func TestGenerateJWT(t *testing.T) {
	// Set a test JWT secret
	SetJWTSecret("test-jwt-secret-key")

	userID := "user-123"
	email := "test@example.com"

	token, err := GenerateJWT(userID, email)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty JWT token")
	}

	// Verify token has 3 parts (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("Expected JWT to have 3 parts, got %d", len(parts))
	}
}

func TestValidateJWT(t *testing.T) {
	// Set a test JWT secret
	SetJWTSecret("test-jwt-secret-key")

	userID := "user-123"
	email := "test@example.com"

	// Generate a valid token
	token, err := GenerateJWT(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		shouldErr bool
		checkFunc func(*Claims) bool
	}{
		{
			name:      "Valid token",
			token:     token,
			shouldErr: false,
			checkFunc: func(claims *Claims) bool {
				return claims.UserID == userID && claims.Email == email
			},
		},
		{
			name:      "Invalid token format",
			token:     "invalid.token.format",
			shouldErr: true,
			checkFunc: nil,
		},
		{
			name:      "Empty token",
			token:     "",
			shouldErr: true,
			checkFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateJWT(tt.token)

			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if claims == nil {
					t.Fatal("Expected claims, got nil")
				}
				if tt.checkFunc != nil && !tt.checkFunc(claims) {
					t.Error("Claims validation failed")
				}
			}
		})
	}
}

func TestJWTExpiration(t *testing.T) {
	// Set a test JWT secret
	SetJWTSecret("test-jwt-secret-key")

	userID := "user-123"
	email := "test@example.com"

	token, err := GenerateJWT(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	// Check that token expires in approximately 24 hours
	expiresAt := claims.ExpiresAt.Time
	now := time.Now()
	expectedExpiry := now.Add(24 * time.Hour)

	// Allow 1 minute tolerance
	if expiresAt.Before(expectedExpiry.Add(-1*time.Minute)) ||
		expiresAt.After(expectedExpiry.Add(1*time.Minute)) {
		t.Errorf("Expected expiry around %v, got %v", expectedExpiry, expiresAt)
	}
}

func TestJWTWithDifferentSecret(t *testing.T) {
	// Generate token with first secret
	SetJWTSecret("secret-1")
	token, err := GenerateJWT("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Try to validate with different secret
	SetJWTSecret("secret-2")
	_, err = ValidateJWT(token)
	if err == nil {
		t.Error("Expected error when validating with wrong secret, got nil")
	}
}

func TestClaims_Structure(t *testing.T) {
	claims := Claims{
		UserID: "test-user",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "nofxAI",
		},
	}

	if claims.UserID != "test-user" {
		t.Errorf("Expected UserID to be test-user, got %s", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Expected Email to be test@example.com, got %s", claims.Email)
	}
	if claims.Issuer != "nofxAI" {
		t.Errorf("Expected Issuer to be nofxAI, got %s", claims.Issuer)
	}
}

// Benchmark tests
func BenchmarkHashPassword(b *testing.B) {
	password := "testPassword123!"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	password := "testPassword123!"
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CheckPassword(password, hash)
	}
}

func BenchmarkGenerateJWT(b *testing.B) {
	SetJWTSecret("test-secret-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateJWT("user-123", "test@example.com")
	}
}

func BenchmarkValidateJWT(b *testing.B) {
	SetJWTSecret("test-secret-key")
	token, _ := GenerateJWT("user-123", "test@example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateJWT(token)
	}
}
