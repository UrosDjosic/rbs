package httpapi

import (
	"net/http"
	"strings"

	"oblak/internal/common/httpx"
)

const maxRunDisplayBytes = 1024

func (s *Server) handleRunGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	uid, ok := UserIDFromContext(r.Context())
	if !ok || uid == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	runID := r.PathValue("run_id")
	if runID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing run id")
		return
	}

	run, err := s.DB.GetRun(r.Context(), runID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	if run == nil {
		httpx.WriteError(w, http.StatusNotFound, "run not found")
		return
	}

	owner, err := s.DB.GetFunctionOwner(r.Context(), run.FunctionID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	if owner != uid {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	stdout, stdoutCut := clipForDisplay(ptrStr(run.Stdout))
	stderr, stderrCut := clipForDisplay(ptrStr(run.Stderr))

	resp := map[string]any{
		"run_id":            run.ID,
		"function_id":       run.FunctionID,
		"version_id":        run.VersionID,
		"status":            run.Status,
		"exit_code":         run.ExitCode,
		"stdout":            stdout,
		"stderr":            stderr,
		"stdout_truncated":  stdoutCut,
		"stderr_truncated":  stderrCut,
		"message":           run.Message,
		"created_at":        run.CreatedAt,
	}
	if run.FinishedAt != nil {
		resp["finished_at"] = *run.FinishedAt
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func clipForDisplay(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) <= maxRunDisplayBytes {
		return s, false
	}
	return s[:maxRunDisplayBytes] + "…", true
}
