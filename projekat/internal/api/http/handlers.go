package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"oblak/internal/api/store/sqlite"
	"oblak/internal/common/httpx"
	"oblak/internal/common/ids"
	"oblak/internal/runner"
	"oblak/internal/verifier"
)

type Server struct {
	DB            *sqlite.DB
	PublicBaseURL string // npr. http://127.0.0.1:8080 — za invoke_url u odgovorima
	Runner        runner.Runner // Execution backend (local, firecracker, etc.)
}

func (s *Server) invokeURL(functionID string) string {
	base := strings.TrimRight(s.PublicBaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	return base + "/invoke/" + functionID
}

func (s *Server) Register(mux *http.ServeMux, uiFS http.Handler) {
	mux.HandleFunc("/health", s.handleHealth)
	// FileServer radi od korena direktorijuma, pa treba stripovati "/ui/" prefix
	mux.HandleFunc("/ui", func(w http.ResponseWriter, r *http.Request) {
		// Redirect na `/ui/` da FileServer nađe `index.html`
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})
	mux.Handle("/ui/", http.StripPrefix("/ui/", uiFS))

	mux.Handle("/auth/login", AuditMiddleware(s.DB, "login", http.HandlerFunc(s.handleLogin)))
	mux.Handle("/auth/register", AuditMiddleware(s.DB, "register", http.HandlerFunc(s.handleRegister)))
	mux.Handle("/me", AuditMiddleware(s.DB, "me", AuthMiddleware(s.DB, http.HandlerFunc(s.handleMe))))
	mux.Handle("/functions", AuditMiddleware(s.DB, "functions", AuthMiddleware(s.DB, http.HandlerFunc(s.handleFunctions))))
	mux.Handle("POST /functions/{id}/deploy", AuditMiddleware(s.DB, "function_deploy", AuthMiddleware(s.DB, http.HandlerFunc(s.handleFunctionDeploy))))
	mux.Handle("POST /invoke/{function_id}", AuditMiddleware(s.DB, "invoke", http.HandlerFunc(s.handleInvoke)))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"time": time.Now().Format(time.RFC3339Nano),
	})
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	Token string `json:"token"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Username == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "username and password required")
		return
	}

	u, err := s.DB.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "user lookup failed")
		return
	}
	if u == nil {
		// constant-ish time compare to avoid trivial oracle
		_ = subtle.ConstantTimeCompare([]byte("x"), []byte(req.Username))
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(req.Password)); err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	tok, err := ids.NewToken(32)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	now := time.Now()
	tt := sqlite.Token{
		Token:     tok,
		UserID:    u.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := s.DB.InsertToken(r.Context(), tt); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "token insert failed")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResp{Token: tok})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Username == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "username and password required")
		return
	}

	// Check if user already exists
	existing, err := s.DB.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "user lookup failed")
		return
	}
	if existing != nil {
		httpx.WriteError(w, http.StatusConflict, "username already taken")
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "password hashing failed")
		return
	}

	// Create user
	uid, err := ids.NewID()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "id generation failed")
		return
	}
	u := sqlite.User{
		ID:           uid,
		Username:     req.Username,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}
	if err := s.DB.InsertUser(r.Context(), u); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "user creation failed")
		return
	}

	// Generate token
	tok, err := ids.NewToken(32)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	now := time.Now()
	tt := sqlite.Token{
		Token:     tok,
		UserID:    uid,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := s.DB.InsertToken(r.Context(), tt); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "token insert failed")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResp{Token: tok})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	uid, ok := UserIDFromContext(r.Context())
	if !ok || uid == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}
	row := s.DB.SQL.QueryRowContext(r.Context(), `SELECT username FROM users WHERE id = ?`, uid)
	var username string
	if err := row.Scan(&username); err != nil {
		if err == sql.ErrNoRows {
			httpx.WriteError(w, http.StatusUnauthorized, "unknown user")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"user_id":  uid,
		"username": username,
	})
}

func (s *Server) handleFunctions(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok || uid == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := s.DB.ListFunctions(r.Context(), uid, 50)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "db error")
			return
		}
		for i := range rows {
			if dep, err := s.DB.GetDeployedFunction(r.Context(), rows[i].FunctionID); err == nil && dep != nil {
				u := s.invokeURL(rows[i].FunctionID)
				rows[i].InvokeURL = &u
			}
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
		return

	case http.MethodPost:
		// multipart/form-data with file field "zip"
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}
		f, hdr, err := r.FormFile("zip")
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "missing file field 'zip'")
			return
		}
		defer f.Close()

		fnID, err := ids.NewToken(16)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "id generation failed")
			return
		}
		verID, err := ids.NewToken(16)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "id generation failed")
			return
		}

		if err := os.MkdirAll(filepath.Join("storage", "functions", fnID, verID), 0o755); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "storage mkdir failed")
			return
		}
		dstPath := filepath.Join("storage", "functions", fnID, verID, "src.zip")
		dst, err := os.Create(dstPath)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "storage create failed")
			return
		}
		defer dst.Close()

		h := sha256.New()
		// write to disk + hash
		if _, err := io.Copy(io.MultiWriter(dst, h), f); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "storage write failed")
			return
		}
		sha := hex.EncodeToString(h.Sum(nil))

		now := time.Now()
		var name *string
		if n := r.FormValue("name"); n != "" {
			name = &n
		} else if hdr != nil && hdr.Filename != "" {
			n := hdr.Filename
			name = &n
		}

		if err := s.DB.InsertFunction(r.Context(), sqlite.Function{
			ID:          fnID,
			OwnerUserID: uid,
			Name:        name,
			CreatedAt:   now,
		}); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "db insert failed")
			return
		}
		if err := s.DB.InsertFunctionVersion(r.Context(), sqlite.FunctionVersion{
			ID:         verID,
			FunctionID: fnID,
			CreatedAt:  now,
			Status:     "uploaded",
			SrcZipPath: dstPath,
			SrcSHA256:  sha,
		}); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "db insert failed")
			return
		}

		workDir := filepath.Join("storage", "functions", fnID, verID, "work")
		vr, err := verifier.Verify(dstPath, workDir, nil)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "verification failed")
			return
		}
		if err := s.DB.UpdateFunctionVersionStatus(r.Context(), verID, vr.Status); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "status update failed")
			return
		}

		resp := map[string]any{
			"function_id": fnID,
			"version_id":  verID,
			"status":      vr.Status,
			"sha256":      sha,
			"verified":    vr.OK,
			"layers":      vr.Layers,
		}
		httpx.WriteJSON(w, http.StatusOK, resp)
		return

	default:
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
}
