package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/runner"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type WebhookHandler struct {
	runRepo      repository.TestRunRepository
	resultRepo   repository.TestResultRepository
	ingestionSvc *service.IngestionService
}

func NewWebhookHandler(
	runRepo repository.TestRunRepository,
	resultRepo repository.TestResultRepository,
	ingestionSvc *service.IngestionService,
) *WebhookHandler {
	return &WebhookHandler{runRepo: runRepo, resultRepo: resultRepo, ingestionSvc: ingestionSvc}
}

// GHACallback handles POST /v1/webhooks/gha/:run_id
// The run_id UUID acts as a one-time token (no auth middleware).
// Supports both flat (results[]) and hierarchical (suites[]) payloads.
func (h *WebhookHandler) GHACallback(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid run ID")
	}

	run, err := h.runRepo.GetByID(c.Request().Context(), runID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Test run not found")
	}

	if run.Status.IsTerminal() {
		return response.Error(c, http.StatusConflict, "CONFLICT", "Run is already in a terminal state")
	}

	// Read body to detect format
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "Could not read request body")
	}

	// Detect payload format by checking for "jobs" key
	var probe struct {
		Jobs    json.RawMessage `json:"jobs"`
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body")
	}

	ctx := c.Request().Context()

	if len(probe.Jobs) > 0 && string(probe.Jobs) != "null" {
		// Hierarchical payload
		var payload dto.HierarchicalPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "Invalid hierarchical payload")
		}
		if err := h.ingestionSvc.IngestForRun(ctx, runID, &payload); err != nil {
			return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to ingest hierarchical results")
		}
		return response.Success(c, http.StatusOK, map[string]string{
			"id":      runID.String(),
			"status":  "completed",
			"message": "Hierarchical results received",
		})
	}

	// Flat payload (existing path)
	var payload runner.VerdoxResultsFile
	if err := json.Unmarshal(body, &payload); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Store results
	if len(payload.Results) > 0 {
		results := make([]model.TestResult, len(payload.Results))
		for i, r := range payload.Results {
			results[i] = model.TestResult{
				TestRunID: runID,
				TestName:  r.TestName,
				Status:    model.TestResultStatus(r.Status),
			}
			if r.DurationMs > 0 {
				ms := int(r.DurationMs)
				results[i].DurationMs = &ms
			}
			if r.ErrorMessage != "" {
				results[i].ErrorMessage = &r.ErrorMessage
			}
			if r.LogOutput != "" {
				results[i].LogOutput = &r.LogOutput
			}
		}
		if err := h.resultRepo.BatchCreate(ctx, results); err != nil {
			return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to store results")
		}
	}

	// Update run status
	finalStatus := model.TestRunStatusPassed
	if payload.Status == "failed" {
		finalStatus = model.TestRunStatusFailed
	}

	if err := h.runRepo.UpdateFinished(ctx, runID, finalStatus); err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update run")
	}

	return response.Success(c, http.StatusOK, map[string]string{
		"id":      runID.String(),
		"status":  string(finalStatus),
		"message": "Results received",
	})
}

// Ingest handles POST /v1/webhooks/ingest
// Accepts a full multi-suite HierarchicalPayload and creates runs for each suite.
func (h *WebhookHandler) Ingest(c echo.Context) error {
	var payload dto.HierarchicalPayload
	if err := c.Bind(&payload); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if len(payload.Jobs) == 0 {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "At least one suite is required")
	}

	if payload.Repo == "" {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "repo field is required")
	}

	// Look up repository by GitHub full name
	repoID, err := h.resolveRepoID(c, payload.Repo)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	result, err := h.ingestionSvc.IngestHierarchical(ctx, repoID, &payload)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to ingest results: "+err.Error())
	}

	runIDs := make([]string, len(result.RunIDs))
	for i, id := range result.RunIDs {
		runIDs[i] = id.String()
	}

	return response.Success(c, http.StatusCreated, dto.IngestResponse{
		ReportID: result.ReportID,
		RunIDs:   runIDs,
		Message:  "Results ingested successfully",
	})
}

func (h *WebhookHandler) resolveRepoID(c echo.Context, _ string) (uuid.UUID, error) {
	// For now, require repo_id as a query parameter since we can't look up by GitHub full name
	// without an authenticated context. This will be improved when CI mapping is implemented.
	repoIDStr := c.QueryParam("repo_id")
	if repoIDStr == "" {
		return uuid.Nil, response.Error(c, http.StatusBadRequest, "INVALID_BODY", "repo_id query parameter is required")
	}
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		return uuid.Nil, response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repo_id")
	}
	return repoID, nil
}
