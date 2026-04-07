package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
	v "github.com/sujaykumarsuman/verdox/backend/pkg/validator"
)

type AdminMailHandler struct {
	notifService *service.NotificationService
	userRepo     repository.UserRepository
}

func NewAdminMailHandler(
	notifService *service.NotificationService,
	userRepo repository.UserRepository,
) *AdminMailHandler {
	return &AdminMailHandler{
		notifService: notifService,
		userRepo:     userRepo,
	}
}

// SendMail sends a push notification (in-app message) to selected users.
// Messages appear in each recipient's notification bell and notifications page in real-time via SSE.
func (h *AdminMailHandler) SendMail(c echo.Context) error {
	var req dto.AdminMailRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	callerID := mw.GetUserID(c)

	// Resolve recipients
	recipients, err := h.resolveRecipients(c, &req.Recipients)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve recipients")
	}

	if len(recipients) == 0 {
		return response.Error(c, http.StatusBadRequest, "NO_RECIPIENTS", "No matching recipients found")
	}

	sent := 0
	var errors []string

	for _, user := range recipients {
		senderID := callerID
		n := &model.Notification{
			UserID:   user.ID,
			Type:     model.NotificationAdminMessage,
			Subject:  req.Subject,
			Body:     req.Body,
			SenderID: &senderID,
		}
		if err := h.notifService.CreateAndPublish(c.Request().Context(), n); err != nil {
			errors = append(errors, user.Username+": failed")
			continue
		}
		sent++
	}

	return response.Success(c, http.StatusOK, dto.AdminMailResponse{
		Sent:       sent,
		Failed:     len(errors),
		Errors:     errors,
		Recipients: len(recipients),
	})
}

func (h *AdminMailHandler) resolveRecipients(c echo.Context, r *dto.AdminMailRecipients) ([]model.User, error) {
	ctx := c.Request().Context()

	switch r.Type {
	case "all":
		users, _, err := h.userRepo.ListFiltered(ctx, "", "", "active", "created_at", "asc", 0, 10000)
		return users, err

	case "filtered":
		users, _, err := h.userRepo.ListFiltered(ctx, "", r.Role, r.Status, "created_at", "asc", 0, 10000)
		return users, err

	case "selected":
		var users []model.User
		for _, idStr := range r.UserIDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}
			user, err := h.userRepo.GetByID(ctx, id)
			if err != nil {
				continue
			}
			users = append(users, *user)
		}
		return users, nil

	default:
		return nil, nil
	}
}
