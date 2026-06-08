package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// visitor is the per-client token-bucket state.
type visitor struct {
	tokens float64
	last   time.Time // last refill instant
	seen   time.Time // last request instant (for eviction)
}

// RateLimiter is an in-memory, per-IP token-bucket limiter. It is suitable for
// a single-node, on-prem deployment (a small metalworking shop) where a shared
// store like Redis would be overkill. Buckets idle for longer than ttl are
// evicted by a background sweeper so memory stays bounded.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      float64 // refill rate (tokens per second)
	burst    float64 // bucket capacity
	ttl      time.Duration
}

// NewRateLimiter builds a limiter allowing, in steady state, rps requests per
// second per IP with bursts up to burst. A sweeper goroutine evicts idle
// buckets every ttl. If rps <= 0 the limiter is effectively disabled (Middleware
// becomes a pass-through).
func NewRateLimiter(rps, burst float64) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rps:      rps,
		burst:    burst,
		ttl:      10 * time.Minute,
	}
	if rps > 0 {
		go rl.sweep()
	}
	return rl
}

func (rl *RateLimiter) sweep() {
	ticker := time.NewTicker(rl.ttl)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-rl.ttl)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if v.seen.Before(cutoff) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// allow consumes one token for ip, refilling the bucket based on elapsed time.
func (rl *RateLimiter) allow(ip string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[ip]
	if !ok {
		rl.visitors[ip] = &visitor{tokens: rl.burst - 1, last: now, seen: now}
		return true
	}

	// Refill proportionally to elapsed time, capped at burst.
	v.tokens += now.Sub(v.last).Seconds() * rl.rps
	if v.tokens > rl.burst {
		v.tokens = rl.burst
	}
	v.last = now
	v.seen = now

	if v.tokens >= 1 {
		v.tokens--
		return true
	}
	return false
}

// Middleware enforces the limit. On rejection it returns 429 with a small JSON
// body and a Retry-After hint. When rps <= 0 the limiter is a no-op.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	if rl.rps <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(realIP(r)) {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded, slow down",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
