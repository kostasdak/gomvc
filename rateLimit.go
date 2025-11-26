package gomvc

import (
	"sync"
	"time"
)

// RateLimiter tracks failed login attempts
type RateLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*attemptRecord
	// Configuration
	MaxAttempts   int           // Max attempts before blocking
	BlockDuration time.Duration // How long to block
	CleanupPeriod time.Duration // How often to cleanup old records
}

type attemptRecord struct {
	Count        int
	FirstAttempt time.Time
	BlockedUntil time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxAttempts int, blockDuration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts:      make(map[string]*attemptRecord),
		MaxAttempts:   maxAttempts,
		BlockDuration: blockDuration,
		CleanupPeriod: time.Minute * 5,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// IsBlocked checks if an identifier (IP or username) is currently blocked
func (rl *RateLimiter) IsBlocked(identifier string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	record, exists := rl.attempts[identifier]
	if !exists {
		return false
	}

	// Check if block has expired
	if time.Now().Before(record.BlockedUntil) {
		return true
	}

	return false
}

// RecordFailedAttempt records a failed login attempt
func (rl *RateLimiter) RecordFailedAttempt(identifier string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	record, exists := rl.attempts[identifier]

	if !exists {
		rl.attempts[identifier] = &attemptRecord{
			Count:        1,
			FirstAttempt: now,
			BlockedUntil: time.Time{},
		}
		return
	}

	// If previous block expired, reset
	if !record.BlockedUntil.IsZero() && now.After(record.BlockedUntil) {
		record.Count = 1
		record.FirstAttempt = now
		record.BlockedUntil = time.Time{}
		return
	}

	// Increment count
	record.Count++

	// Block if exceeded max attempts
	if record.Count >= rl.MaxAttempts {
		record.BlockedUntil = now.Add(rl.BlockDuration)
		InfoMessage("Rate limit exceeded for: " + identifier +
			" - Blocked until: " + record.BlockedUntil.Format(time.RFC3339))
	}
}

// ResetAttempts clears attempts for an identifier (on successful login)
func (rl *RateLimiter) ResetAttempts(identifier string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, identifier)
}

// GetRemainingAttempts returns how many attempts are left before blocking
func (rl *RateLimiter) GetRemainingAttempts(identifier string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	record, exists := rl.attempts[identifier]
	if !exists {
		return rl.MaxAttempts
	}

	remaining := rl.MaxAttempts - record.Count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetBlockedUntil returns when the identifier will be unblocked
func (rl *RateLimiter) GetBlockedUntil(identifier string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	record, exists := rl.attempts[identifier]
	if !exists {
		return time.Time{}
	}

	return record.BlockedUntil
}

// cleanupLoop periodically removes old records
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.CleanupPeriod)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes expired records
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for identifier, record := range rl.attempts {
		// Remove if block expired and no recent attempts
		if !record.BlockedUntil.IsZero() &&
			now.After(record.BlockedUntil.Add(rl.BlockDuration)) {
			delete(rl.attempts, identifier)
		}
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	blocked := 0
	for _, record := range rl.attempts {
		if time.Now().Before(record.BlockedUntil) {
			blocked++
		}
	}

	return map[string]interface{}{
		"total_tracked":          len(rl.attempts),
		"currently_blocked":      blocked,
		"max_attempts":           rl.MaxAttempts,
		"block_duration_minutes": rl.BlockDuration.Minutes(),
	}
}
