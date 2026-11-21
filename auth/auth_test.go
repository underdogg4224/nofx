package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("s3cur3P@ssword!")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "s3cur3P@ssword!" {
		t.Fatal("password hash should not equal plaintext")
	}
	if !CheckPassword("s3cur3P@ssword!", hash) {
		t.Fatal("expected CheckPassword to succeed for matching password")
	}
	if CheckPassword("wrong-password", hash) {
		t.Fatal("expected CheckPassword to fail for non-matching password")
	}
}

func TestGenerateRandomSecret(t *testing.T) {
	s1, err := GenerateRandomSecret(32)
	if err != nil {
		t.Fatalf("GenerateRandomSecret failed: %v", err)
	}
	s2, err := GenerateRandomSecret(32)
	if err != nil {
		t.Fatalf("GenerateRandomSecret failed: %v", err)
	}
	if len(s1) != 64 {
		t.Fatalf("expected 64 hex chars for 32 bytes, got %d", len(s1))
	}
	if s1 == s2 {
		t.Fatal("expected two random secrets to differ")
	}
}

func TestJWTLifecycleAndValidation(t *testing.T) {
	origSecret := JWTSecret
	t.Cleanup(func() { JWTSecret = origSecret })

	// Empty secret must fail closed.
	SetJWTSecret("")
	if _, err := GenerateJWT("u1", "a@example.com"); err == nil {
		t.Fatal("expected GenerateJWT to fail when JWTSecret is empty")
	}
	if _, err := ValidateJWT("anything"); err == nil {
		t.Fatal("expected ValidateJWT to fail when JWTSecret is empty")
	}

	SetJWTSecret("test-secret-key-for-unit-tests-1234567890")

	token, err := GenerateJWT("user-123", "trader@example.com")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}
	if claims.UserID != "user-123" || claims.Email != "trader@example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// Token signed with wrong issuer must be rejected.
	badIssuerToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: "user-123",
		Email:  "trader@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "other-issuer",
		},
	})
	badSigned, err := badIssuerToken.SignedString(JWTSecret)
	if err != nil {
		t.Fatalf("sign bad issuer token failed: %v", err)
	}
	if _, err := ValidateJWT(badSigned); err == nil {
		t.Fatal("expected token with unexpected issuer to be rejected")
	}
}

func TestOTPChallengeBinding(t *testing.T) {
	origSecret := JWTSecret
	t.Cleanup(func() { JWTSecret = origSecret })

	SetJWTSecret("test-secret-key-for-otp-challenge-1234567890")

	challenge, err := GenerateOTPChallenge("user-1", "u1@example.com")
	if err != nil {
		t.Fatalf("GenerateOTPChallenge failed: %v", err)
	}

	// Challenge token must NOT be accepted as a session access token.
	if _, err := ValidateJWT(challenge); err == nil {
		t.Fatal("expected OTP challenge token to be rejected by ValidateJWT")
	}

	// Challenge token must fail for a different user ID.
	if _, err := ValidateOTPChallenge(challenge, "user-2"); err == nil {
		t.Fatal("expected OTP challenge token to be rejected for wrong user")
	}

	// Challenge token succeeds for the matching user ID.
	claims, err := ValidateOTPChallenge(challenge, "user-1")
	if err != nil {
		t.Fatalf("expected valid OTP challenge to succeed: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("expected user-1, got %s", claims.UserID)
	}

	// Normal access token must NOT be accepted as an OTP challenge token.
	accessToken, err := GenerateJWT("user-1", "u1@example.com")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	if _, err := ValidateOTPChallenge(accessToken, "user-1"); err == nil {
		t.Fatal("expected access token to be rejected by ValidateOTPChallenge")
	}
}

func TestRateLimiter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rl := NewRateLimiter(3, time.Minute)
	rl.nowFn = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !rl.Allow("127.0.0.1:/api/login") {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}
	if rl.Allow("127.0.0.1:/api/login") {
		t.Fatal("expected 4th request within window to be rate limited")
	}

	// Different key is independent.
	if !rl.Allow("10.0.0.2:/api/login") {
		t.Fatal("expected independent client key to be allowed")
	}

	// After window elapses, requests are allowed again.
	now = now.Add(time.Minute)
	if !rl.Allow("127.0.0.1:/api/login") {
		t.Fatal("expected request after window reset to be allowed")
	}

	// Concurrency sanity check.
	rl2 := NewRateLimiter(50, time.Minute)
	var wg sync.WaitGroup
	var allowedCount int
	var mu sync.Mutex
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl2.Allow("concurrent-client") {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowedCount != 50 {
		t.Fatalf("expected exactly 50 concurrent allowed requests, got %d", allowedCount)
	}
}
