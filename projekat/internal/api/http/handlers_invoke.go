package httpapi

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"oblak/internal/api/store/sqlite"
	"oblak/internal/common/httpx"
	"oblak/internal/common/ids"
	"oblak/internal/runner"
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

	// Prepare work directory path
	workDir := filepath.Join("storage", "functions", fnID, dep.ActiveVersionID, "work")

	// Invoke the function using the configured runner
	ctx := r.Context()
	result, err := s.Runner.Invoke(ctx, runner.InvokeRequest{
		FunctionID: fnID,
		VersionID:  dep.ActiveVersionID,
		WorkDir:    workDir,
		Payload:    nil, // TODO: Read from request body if needed
	})

	now := time.Now()
	status := "done"
	var message *string

	if err != nil {
		status = "error"
		errMsg := "execution failed: " + err.Error()
		message = &errMsg
		result = &runner.InvokeResult{
			ExitCode: 1,
			Error:    err.Error(),
			Stderr:   err.Error(),
		}
	} else if result == nil {
		status = "error"
		errMsg := "execution failed: runner returned no result"
		message = &errMsg
		result = &runner.InvokeResult{
			ExitCode: 1,
			Error:    errMsg,
			Stderr:   errMsg,
		}
	} else if result.ExitCode != 0 {
		status = "error"
		errMsg := fmt.Sprintf("function exited with code: %d", result.ExitCode)
		message = &errMsg
	}

	// Record the run in the database
	if err := s.DB.InsertRun(r.Context(), sqlite.Run{
		ID:         runID,
		FunctionID: fnID,
		VersionID:  dep.ActiveVersionID,
		Status:     status,
		CreatedAt:  now,
		FinishedAt: &now,
		Message:    message,
	}); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "run insert failed")
		return
	}

	// Return result to client
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"run_id":      runID,
		"function_id": fnID,
		"version_id":  dep.ActiveVersionID,
		"status":      status,
		"exit_code":   result.ExitCode,
		"stdout":      result.Stdout,
		"stderr":      result.Stderr,
		"message":     message,
	})
}
