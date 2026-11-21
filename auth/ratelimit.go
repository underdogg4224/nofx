package auth

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

// RateLimiter provides thread-safe fixed-window rate limiting keyed by client/endpoint.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	limit   int
	window  time.Duration
	nowFn   func() time.Time
}

// NewRateLimiter creates a RateLimiter allowing up to limit requests per window per key.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
		nowFn:   time.Now,
	}
}

// Allow reports whether a request for key is permitted under the rate limit.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.nowFn()

	// Periodically prune expired entries to keep memory bounded.
	if len(rl.entries) > 1000 {
		for k, entry := range rl.entries {
			if !now.Before(entry.resetAt) {
				delete(rl.entries, k)
			}
		}
	}

	entry, exists := rl.entries[key]
	if !exists || !now.Before(entry.resetAt) {
		rl.entries[key] = &rateLimitEntry{
			count:   1,
			resetAt: now.Add(rl.window),
		}
		return true
	}

	if entry.count >= rl.limit {
		return false
	}

	entry.count++
	return true
}

// Reset clears all tracked rate limit counters.
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.entries = make(map[string]*rateLimitEntry)
}
