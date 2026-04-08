package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type ImportHandler struct {
	importService *service.ImportService
}

func NewImportHandler(importService *service.ImportService) *ImportHandler {
	return &ImportHandler{importService: importService}
}

// ListWorkflows handles GET /v1/repositories/:id/workflows
func (h *ImportHandler) ListWorkflows(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	resp, err := h.importService.ListWorkflowFiles(c.Request().Context(), repoID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "LIST_WORKFLOWS_FAILED", err.Error())
	}

	return response.Success(c, http.StatusOK, resp)
}

// ImportSuite handles POST /v1/repositories/:id/import-suite
func (h *ImportHandler) ImportSuite(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	var req dto.ImportSuiteRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Validate exactly one source is provided
	hasFile := req.WorkflowFile != nil && *req.WorkflowFile != ""
	hasYAML := req.WorkflowYAML != nil && *req.WorkflowYAML != ""
	if hasFile == hasYAML {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Provide exactly one of workflow_file or workflow_yaml")
	}

	resp, err := h.importService.ImportSuite(c.Request().Context(), repoID, &req)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "IMPORT_FAILED", err.Error())
	}

	return response.Success(c, http.StatusOK, resp)
}
