package httpplatform

import (
	"errors"
	"sync"
	"time"
)

const maximumLoginRateLimitEntries = 10_000

// ErrInvalidLoginRateLimit indicates an unusable login throttling policy.
var ErrInvalidLoginRateLimit = errors.New("invalid login rate limit")

type loginRateLimitWindow struct {
	startedAt time.Time
	attempts  int
}

// LoginRateLimiter applies a bounded, in-memory fixed window per socket peer.
type LoginRateLimiter struct {
	mutex       sync.Mutex
	entries     map[string]loginRateLimitWindow
	maxAttempts int
	window      time.Duration
	now         func() time.Time
}

// NewLoginRateLimiter creates a concurrency-safe login limiter.
func NewLoginRateLimiter(maxAttempts int, window time.Duration) (*LoginRateLimiter, error) {
	return newLoginRateLimiter(maxAttempts, window, time.Now)
}

func newLoginRateLimiter(
	maxAttempts int,
	window time.Duration,
	now func() time.Time,
) (*LoginRateLimiter, error) {
	if maxAttempts <= 0 || window <= 0 || now == nil {
		return nil, ErrInvalidLoginRateLimit
	}
	return &LoginRateLimiter{
		entries:     make(map[string]loginRateLimitWindow),
		maxAttempts: maxAttempts,
		window:      window,
		now:         now,
	}, nil
}

// Allow consumes one attempt and reports how long a rejected peer must wait.
func (limiter *LoginRateLimiter) Allow(key string) (bool, time.Duration) {
	now := limiter.now().UTC()
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	entry, exists := limiter.entries[key]
	if exists && now.Sub(entry.startedAt) >= limiter.window {
		delete(limiter.entries, key)
		exists = false
	}
	if !exists {
		if len(limiter.entries) >= maximumLoginRateLimitEntries {
			limiter.pruneExpired(now)
		}
		if len(limiter.entries) >= maximumLoginRateLimitEntries {
			return false, limiter.window
		}
		limiter.entries[key] = loginRateLimitWindow{startedAt: now, attempts: 1}
		return true, 0
	}

	if entry.attempts >= limiter.maxAttempts {
		return false, limiter.window - now.Sub(entry.startedAt)
	}
	entry.attempts++
	limiter.entries[key] = entry
	return true, 0
}

func (limiter *LoginRateLimiter) pruneExpired(now time.Time) {
	for key, entry := range limiter.entries {
		if now.Sub(entry.startedAt) >= limiter.window {
			delete(limiter.entries, key)
		}
	}
}
