package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

// RateLimiter implements a simple in-memory rate limiter per tenant
type RateLimiter struct {
	mu          sync.Mutex
	limits      map[string][]time.Time
	maxRequests int
	window      time.Duration
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limits:      make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:      window,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := authctx.TenantID(r.Context()).String()

		rl.mu.Lock()
		now := time.Now()
		rl.limits[tenantID] = rl.filterOldRequests(rl.limits[tenantID], now)

		if len(rl.limits[tenantID]) >= rl.maxRequests {
			rl.mu.Unlock()
			RespondError(w, domain.ErrRateLimitExceeded)
			return
		}

		rl.limits[tenantID] = append(rl.limits[tenantID], now)
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) filterOldRequests(requests []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-rl.window)
	filtered := requests[:0]
	for _, req := range requests {
		if req.After(cutoff) {
			filtered = append(filtered, req)
		}
	}
	return filtered
}
