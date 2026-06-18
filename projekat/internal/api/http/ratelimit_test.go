package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPRateLimiterAllowsWithinWindow(t *testing.T) {
	lim := NewIPRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		ok, _ := lim.Allow("1.2.3.4")
		if !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	ok, retry := lim.Allow("1.2.3.4")
	if ok {
		t.Fatal("fourth request should be denied")
	}
	if retry <= 0 {
		t.Fatalf("expected positive retry, got %v", retry)
	}
}

func TestIPRateLimiterDisabled(t *testing.T) {
	var lim *IPRateLimiter
	ok, _ := lim.Allow("x")
	if !ok {
		t.Fatal("nil limiter should allow all")
	}
}

func TestRateLimitMiddlewareReturns429(t *testing.T) {
	lim := NewIPRateLimiter(1, time.Minute)
	var hits int
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	})
	h := RateLimitMiddleware(lim, "test", next)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request: got %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: got %d want 429", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	if hits != 1 {
		t.Fatalf("handler should run once, got %d", hits)
	}
}
