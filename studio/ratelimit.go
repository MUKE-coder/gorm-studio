package studio

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig configures Studio-specific rate limiting on the most
// expensive endpoints, independent of any rate limiting the host app applies.
// A zero value for a field means "no limit" for that category.
type RateLimitConfig struct {
	// SQLPerMinute limits requests to the raw SQL endpoint, per client IP.
	SQLPerMinute int
	// ImportPerMinute limits requests to the import endpoints, per client IP.
	ImportPerMinute int
}

// tokenBucket is a single client's bucket.
type tokenBucket struct {
	tokens float64
	last   time.Time
}

// rateLimiter is a per-client-IP token-bucket limiter.
type rateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*tokenBucket
	ratePerSec float64
	burst      float64
}

// newRateLimiter returns a limiter allowing perMinute requests per client IP,
// or nil when perMinute <= 0 (no limit).
func newRateLimiter(perMinute int) *rateLimiter {
	if perMinute <= 0 {
		return nil
	}
	return &rateLimiter{
		buckets:    make(map[string]*tokenBucket),
		ratePerSec: float64(perMinute) / 60.0,
		burst:      float64(perMinute),
	}
}

// allow reports whether a request from key may proceed, consuming a token.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &tokenBucket{tokens: rl.burst - 1, last: now}
		return true
	}

	// Refill based on elapsed time, capped at burst.
	b.tokens += now.Sub(b.last).Seconds() * rl.ratePerSec
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// middleware returns a Gin middleware enforcing the limit.
func (rl *rateLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded, slow down"})
			return
		}
		c.Next()
	}
}

// withRateLimit prepends rl's middleware to h when rl is non-nil.
func withRateLimit(rl *rateLimiter, h gin.HandlerFunc) []gin.HandlerFunc {
	if rl == nil {
		return []gin.HandlerFunc{h}
	}
	return []gin.HandlerFunc{rl.middleware(), h}
}
