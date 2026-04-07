package handler

import (
	"math"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type HierarchyHandler struct {
	runRepo   repository.TestRunRepository
	groupRepo repository.TestGroupRepository
	caseRepo  repository.TestCaseRepository
}

func NewHierarchyHandler(
	runRepo repository.TestRunRepository,
	groupRepo repository.TestGroupRepository,
	caseRepo repository.TestCaseRepository,
) *HierarchyHandler {
	return &HierarchyHandler{
		runRepo:   runRepo,
		groupRepo: groupRepo,
		caseRepo:  caseRepo,
	}
}

// ListGroups handles GET /v1/runs/:id/groups
func (h *HierarchyHandler) ListGroups(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid run ID")
	}

	ctx := c.Request().Context()
	groups, err := h.groupRepo.ListByRunID(ctx, runID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list groups")
	}

	resp := make([]dto.TestGroupResponse, len(groups))
	for i := range groups {
		resp[i] = dto.NewTestGroupResponse(&groups[i])
	}

	return response.Success(c, http.StatusOK, resp)
}

// ListGroupCases handles GET /v1/runs/:id/groups/:groupId/cases
func (h *HierarchyHandler) ListGroupCases(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid run ID")
	}

	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid group ID")
	}

	// Verify group belongs to this run
	group, err := h.groupRepo.GetByID(c.Request().Context(), groupID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Group not found")
	}
	if group.TestRunID != runID {
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Group not found for this run")
	}

	page := intQueryParam(c, "page", 1)
	perPage := intQueryParam(c, "per_page", 100)

	ctx := c.Request().Context()
	cases, total, err := h.caseRepo.ListByGroupID(ctx, groupID, page, perPage)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list cases")
	}

	resp := make([]dto.TestCaseResponse, len(cases))
	for i := range cases {
		resp[i] = dto.NewTestCaseResponse(&cases[i])
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return response.Success(c, http.StatusOK, dto.GroupCasesResponse{
		Cases: resp,
		Meta: dto.PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// ListFailedCases handles GET /v1/runs/:id/cases/failed
func (h *HierarchyHandler) ListFailedCases(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid run ID")
	}

	ctx := c.Request().Context()
	cases, err := h.caseRepo.ListFailedByRunID(ctx, runID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list failed cases")
	}

	resp := make([]dto.TestCaseResponse, len(cases))
	for i := range cases {
		resp[i] = dto.NewTestCaseResponse(&cases[i])
	}

	return response.Success(c, http.StatusOK, resp)
}

// GetReport handles GET /v1/reports/:reportId
func (h *HierarchyHandler) GetReport(c echo.Context) error {
	reportID := c.Param("reportId")
	if reportID == "" {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Report ID is required")
	}

	ctx := c.Request().Context()
	runs, err := h.runRepo.ListByReportID(ctx, reportID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list runs for report")
	}

	if len(runs) == 0 {
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Report not found")
	}

	runResponses := make([]dto.TestRunResponse, len(runs))
	for i := range runs {
		runResponses[i] = dto.NewTestRunResponse(&runs[i])
	}

	return response.Success(c, http.StatusOK, dto.ReportResponse{
		ReportID: reportID,
		Runs:     runResponses,
	})
}

func intQueryParam(c echo.Context, name string, defaultVal int) int {
	v := c.QueryParam(name)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil || i < 1 {
		return defaultVal
	}
	return i
}
