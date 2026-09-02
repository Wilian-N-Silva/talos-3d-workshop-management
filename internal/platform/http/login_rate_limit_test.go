package httpplatform

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoginRateLimiterEnforcesAndResetsPeerWindow(t *testing.T) {
	now := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.UTC)
	limiter, err := newLoginRateLimiter(2, time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatalf("newLoginRateLimiter() error = %v", err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		if allowed, retryAfter := limiter.Allow("127.0.0.1"); !allowed || retryAfter != 0 {
			t.Fatalf("attempt %d = %t, %s, want allowed", attempt, allowed, retryAfter)
		}
	}
	if allowed, retryAfter := limiter.Allow("127.0.0.1"); allowed || retryAfter != time.Minute {
		t.Fatalf("limited attempt = %t, %s, want false and 1m", allowed, retryAfter)
	}
	if allowed, _ := limiter.Allow("127.0.0.2"); !allowed {
		t.Fatal("separate peer was incorrectly rate limited")
	}

	now = now.Add(time.Minute)
	if allowed, retryAfter := limiter.Allow("127.0.0.1"); !allowed || retryAfter != 0 {
		t.Fatalf("attempt after reset = %t, %s, want allowed", allowed, retryAfter)
	}
}

func TestLoginRateLimiterIsConcurrencySafe(t *testing.T) {
	limiter, err := newLoginRateLimiter(5, time.Minute, func() time.Time { return time.Now().UTC() })
	if err != nil {
		t.Fatalf("newLoginRateLimiter() error = %v", err)
	}

	var allowed atomic.Int64
	var wait sync.WaitGroup
	for range 100 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if ok, _ := limiter.Allow("shared-peer"); ok {
				allowed.Add(1)
			}
		}()
	}
	wait.Wait()
	if got := allowed.Load(); got != 5 {
		t.Fatalf("allowed attempts = %d, want 5", got)
	}
}

func TestNewLoginRateLimiterRejectsInvalidPolicy(t *testing.T) {
	if _, err := NewLoginRateLimiter(0, time.Minute); err != ErrInvalidLoginRateLimit {
		t.Fatalf("zero attempts error = %v", err)
	}
	if _, err := NewLoginRateLimiter(1, 0); err != ErrInvalidLoginRateLimit {
		t.Fatalf("zero window error = %v", err)
	}
}
