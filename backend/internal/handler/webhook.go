package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/runner"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type WebhookHandler struct {
	runRepo    repository.TestRunRepository
	resultRepo repository.TestResultRepository
}

func NewWebhookHandler(runRepo repository.TestRunRepository, resultRepo repository.TestResultRepository) *WebhookHandler {
	return &WebhookHandler{runRepo: runRepo, resultRepo: resultRepo}
}

// GHACallback handles POST /v1/webhooks/gha/:run_id
// The run_id UUID acts as a one-time token (no auth middleware).
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

	// Parse the results payload (matching .verdox/results.json schema)
	var payload runner.VerdoxResultsFile
	if err := c.Bind(&payload); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	ctx := c.Request().Context()

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
