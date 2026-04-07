package handler

import (
	"math"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type NotificationHandler struct {
	notifService *service.NotificationService
}

func NewNotificationHandler(notifService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

func (h *NotificationHandler) List(c echo.Context) error {
	userID := mw.GetUserID(c)

	page := 1
	perPage := 20
	if p := c.QueryParam("page"); p != "" {
		if v, err := uuid.Parse(p); err == nil {
			_ = v // not a uuid, parse as int below
		}
	}
	// Simple int parsing
	if p := c.QueryParam("page"); p != "" {
		var v int
		if _, err := parseIntParam(p, &v); err == nil && v > 0 {
			page = v
		}
	}
	if pp := c.QueryParam("per_page"); pp != "" {
		var v int
		if _, err := parseIntParam(pp, &v); err == nil && v > 0 && v <= 100 {
			perPage = v
		}
	}

	notifications, total, err := h.notifService.List(c.Request().Context(), userID, page, perPage)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list notifications")
	}

	items := make([]dto.NotificationResponse, len(notifications))
	for i, n := range notifications {
		items[i] = dto.NewNotificationResponse(&n)
	}

	return response.Success(c, http.StatusOK, dto.NotificationListResponse{
		Notifications: items,
		Total:         total,
		Page:          page,
		PerPage:       perPage,
	})
}

func (h *NotificationHandler) UnreadCount(c echo.Context) error {
	userID := mw.GetUserID(c)

	count, err := h.notifService.CountUnread(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count unread notifications")
	}

	return response.Success(c, http.StatusOK, dto.UnreadCountResponse{Count: count})
}

func (h *NotificationHandler) MarkRead(c echo.Context) error {
	userID := mw.GetUserID(c)

	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID")
	}

	if err := h.notifService.MarkRead(c.Request().Context(), notifID, userID); err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark notification as read")
	}

	return response.Success(c, http.StatusOK, dto.MessageResponse{Message: "Notification marked as read"})
}

func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	userID := mw.GetUserID(c)

	if err := h.notifService.MarkAllRead(c.Request().Context(), userID); err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark all notifications as read")
	}

	return response.Success(c, http.StatusOK, dto.MessageResponse{Message: "All notifications marked as read"})
}

func parseIntParam(s string, v *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid integer")
		}
		n = n*10 + int(c-'0')
		if n > math.MaxInt32 {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "integer overflow")
		}
	}
	*v = n
	return n, nil
}
