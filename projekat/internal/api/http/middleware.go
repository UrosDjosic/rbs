package httpapi

import (
	"net/http"
	"strings"
	"time"

	"oblak/internal/api/store/sqlite"
	"oblak/internal/common/httpx"
)

func AuthMiddleware(db *sqlite.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		token := strings.TrimSpace(h[len("Bearer "):])
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		t, err := db.GetToken(r.Context(), token)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "token lookup failed")
			return
		}
		if t == nil || time.Now().After(t.ExpiresAt) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), t.UserID)))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func AuditMiddleware(db *sqlite.DB, action string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		var actor *string
		if uid, ok := UserIDFromContext(r.Context()); ok {
			actor = &uid
		}
		ip := r.RemoteAddr
		ua := r.UserAgent()
		_ = db.InsertAudit(r.Context(), sqlite.AuditEvent{
			ID:          mustID(),
			Ts:          time.Now(),
			ActorUserID: actor,
			Action:      action,
			Path:        r.URL.Path,
			Method:      r.Method,
			Status:      rec.status,
			IP:          &ip,
			UserAgent:   &ua,
			Details:     nil,
		})
	})
}
