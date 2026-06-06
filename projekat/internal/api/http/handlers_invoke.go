package httpapi

import (
	"net/http"
	"time"

	"oblak/internal/api/store/sqlite"
	"oblak/internal/common/httpx"
	"oblak/internal/common/ids"
)

func (s *Server) handleFunctionDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	uid, ok := UserIDFromContext(r.Context())
	if !ok || uid == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}
	fnID := r.PathValue("id")
	if fnID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing function id")
		return
	}

	owner, err := s.DB.GetFunctionOwner(r.Context(), fnID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	if owner == "" {
		httpx.WriteError(w, http.StatusNotFound, "function not found")
		return
	}
	if owner != uid {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	ver, err := s.DB.GetLatestVersion(r.Context(), fnID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	if ver == nil {
		httpx.WriteError(w, http.StatusNotFound, "no versions for function")
		return
	}
	if ver.Status != "verified" {
		httpx.WriteError(w, http.StatusBadRequest, "function not verified (status: "+ver.Status+")")
		return
	}

	now := time.Now()
	if err := s.DB.DeployFunction(r.Context(), fnID, ver.ID, now); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "deploy failed")
		return
	}

	invokeURL := s.invokeURL(fnID)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"function_id": fnID,
		"version_id":  ver.ID,
		"status":      "deployed",
		"invoke_url":  invokeURL,
	})
}

func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	fnID := r.PathValue("function_id")
	if fnID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing function id")
		return
	}

	dep, err := s.DB.GetDeployedFunction(r.Context(), fnID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	if dep == nil {
		httpx.WriteError(w, http.StatusNotFound, "function not deployed")
		return
	}

	runID, err := ids.NewToken(16)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "id generation failed")
		return
	}
	now := time.Now()
	msg := "stub: Python execution not implemented yet (run recorded only)"
	if err := s.DB.InsertRun(r.Context(), sqlite.Run{
		ID:         runID,
		FunctionID: fnID,
		VersionID:  dep.ActiveVersionID,
		Status:     "done",
		CreatedAt:  now,
		FinishedAt: &now,
		Message:    &msg,
	}); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "run insert failed")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"run_id":      runID,
		"function_id": fnID,
		"version_id":  dep.ActiveVersionID,
		"status":      "done",
		"message":     msg,
	})
}
