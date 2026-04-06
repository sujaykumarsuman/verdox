package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type DiscoveryHandler struct {
	discoveryService *service.DiscoveryService
}

func NewDiscoveryHandler(discoveryService *service.DiscoveryService) *DiscoveryHandler {
	return &DiscoveryHandler{discoveryService: discoveryService}
}

// Discover handles POST /v1/repositories/:id/discover
func (h *DiscoveryHandler) Discover(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	resp, err := h.discoveryService.Discover(c.Request().Context(), repoID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "DISCOVERY_FAILED", err.Error())
	}

	return response.Success(c, http.StatusOK, resp)
}

// GetDiscovery handles GET /v1/repositories/:id/discovery
// Returns the latest discovery results for a repository.
func (h *DiscoveryHandler) GetDiscovery(c echo.Context) error {
	// For now, discovery results are not persisted — they're computed on demand.
	// A future enhancement would cache results in the test_discoveries table.
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	resp, err := h.discoveryService.Discover(c.Request().Context(), repoID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "DISCOVERY_FAILED", err.Error())
	}

	return response.Success(c, http.StatusOK, resp)
}
