package security

import (
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	requests map[string]*bucketState
	mu       sync.RWMutex

	// Configuration
	maxRequests int           // Maximum requests allowed
	window      time.Duration // Time window
	cleanup     time.Duration // Cleanup interval
}

type bucketState struct {
	count      int
	resetTime  time.Time
	violations int // Track consecutive violations
}

// NewRateLimiter creates a new rate limiter
// maxRequests: maximum number of requests allowed in the time window
// window: time window duration (e.g., 1 minute)
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string]*bucketState),
		maxRequests: maxRequests,
		window:      window,
		cleanup:     window * 2, // Cleanup expired entries
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request from the given identifier should be allowed
// Returns true if allowed, false if rate limit exceeded
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create bucket
	bucket, exists := rl.requests[identifier]
	if !exists {
		rl.requests[identifier] = &bucketState{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	// Reset if window has passed
	if now.After(bucket.resetTime) {
		bucket.count = 1
		bucket.resetTime = now.Add(rl.window)
		bucket.violations = 0
		return true
	}

	// Check if limit exceeded
	if bucket.count >= rl.maxRequests {
		bucket.violations++
		return false
	}

	// Increment counter
	bucket.count++
	return true
}

// GetViolations returns the number of consecutive violations for an identifier
func (rl *RateLimiter) GetViolations(identifier string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if bucket, exists := rl.requests[identifier]; exists {
		return bucket.violations
	}
	return 0
}

// Reset resets the rate limit for a specific identifier
func (rl *RateLimiter) Reset(identifier string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.requests, identifier)
}

// cleanupLoop periodically removes expired entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes expired entries from the map
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, bucket := range rl.requests {
		if now.After(bucket.resetTime.Add(rl.window)) {
			delete(rl.requests, key)
		}
	}
}

// RateLimiters holds rate limiters for different endpoints
type RateLimiters struct {
	login        *RateLimiter
	registration *RateLimiter
	otpVerify    *RateLimiter
	api          *RateLimiter
}

var rateLimiters *RateLimiters

// InitializeRateLimiters initializes all rate limiters
func InitializeRateLimiters() {
	rateLimiters = &RateLimiters{
		// Login: 5 attempts per minute per IP
		login: NewRateLimiter(5, 1*time.Minute),

		// Registration: 3 attempts per hour per IP
		registration: NewRateLimiter(3, 1*time.Hour),

		// OTP verification: 5 attempts per minute per user
		otpVerify: NewRateLimiter(5, 1*time.Minute),

		// General API: 100 requests per minute per user
		api: NewRateLimiter(100, 1*time.Minute),
	}
}

// GetLoginLimiter returns the login rate limiter
func GetLoginLimiter() *RateLimiter {
	if rateLimiters == nil {
		InitializeRateLimiters()
	}
	return rateLimiters.login
}

// GetRegistrationLimiter returns the registration rate limiter
func GetRegistrationLimiter() *RateLimiter {
	if rateLimiters == nil {
		InitializeRateLimiters()
	}
	return rateLimiters.registration
}

// GetOTPLimiter returns the OTP verification rate limiter
func GetOTPLimiter() *RateLimiter {
	if rateLimiters == nil {
		InitializeRateLimiters()
	}
	return rateLimiters.otpVerify
}

// GetAPILimiter returns the general API rate limiter
func GetAPILimiter() *RateLimiter {
	if rateLimiters == nil {
		InitializeRateLimiters()
	}
	return rateLimiters.api
}
