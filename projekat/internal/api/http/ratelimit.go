package httpapi

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"oblak/internal/common/httpx"
)

// IPRateLimiter enforces a fixed-window request count per key (typically client IP).
type IPRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]ipRateEntry
}

type ipRateEntry struct {
	windowStart time.Time
	count       int
}

// NewIPRateLimiter returns a limiter allowing at most limit requests per window per key.
// limit <= 0 disables limiting.
func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	if limit <= 0 {
		return nil
	}
	return &IPRateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]ipRateEntry),
	}
}

// Allow reports whether key may proceed and how long to wait when denied.
func (l *IPRateLimiter) Allow(key string) (bool, time.Duration) {
	if l == nil {
		return true, 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	e, ok := l.entries[key]
	if !ok || now.Sub(e.windowStart) >= l.window {
		l.entries[key] = ipRateEntry{windowStart: now, count: 1}
		l.pruneLocked(now)
		return true, 0
	}
	if e.count >= l.limit {
		retry := l.window - now.Sub(e.windowStart)
		if retry < time.Second {
			retry = time.Second
		}
		return false, retry
	}
	e.count++
	l.entries[key] = e
	return true, 0
}

func (l *IPRateLimiter) pruneLocked(now time.Time) {
	if len(l.entries) <= 1024 {
		return
	}
	cutoff := now.Add(-2 * l.window)
	for k, e := range l.entries {
		if e.windowStart.Before(cutoff) {
			delete(l.entries, k)
		}
	}
}

// RateLimitMiddleware limits requests per client IP within the limiter's window.
func RateLimitMiddleware(limiter *IPRateLimiter, keyPrefix string, next http.Handler) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := keyPrefix + ":" + clientIP(r)
		ok, retry := limiter.Allow(key)
		if !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())))
			httpx.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
