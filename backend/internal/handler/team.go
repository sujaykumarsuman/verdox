package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/encryption"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
	v "github.com/sujaykumarsuman/verdox/backend/pkg/validator"
)

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

type TeamHandler struct {
	teamService    *service.TeamService
	teamRepo       repository.TeamRepository
	teamMemberRepo repository.TeamMemberRepository
	githubService  *service.GitHubService
	cfg            *config.Config
	log            zerolog.Logger
}

func NewTeamHandler(
	teamService *service.TeamService,
	teamRepo repository.TeamRepository,
	teamMemberRepo repository.TeamMemberRepository,
	githubService *service.GitHubService,
	cfg *config.Config,
	log zerolog.Logger,
) *TeamHandler {
	return &TeamHandler{
		teamService:    teamService,
		teamRepo:       teamRepo,
		teamMemberRepo: teamMemberRepo,
		githubService:  githubService,
		cfg:            cfg,
		log:            log,
	}
}

// ListMyTeams handles GET /v1/teams
func (h *TeamHandler) ListMyTeams(c echo.Context) error {
	userID := mw.GetUserID(c)
	teams, err := h.teamService.ListMyTeams(c.Request().Context(), userID)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusOK, teams)
}

// CreateTeam handles POST /v1/teams
func (h *TeamHandler) CreateTeam(c echo.Context) error {
	var req dto.CreateTeamRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	userID := mw.GetUserID(c)
	resp, err := h.teamService.CreateTeam(c.Request().Context(), userID, &req)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusCreated, resp)
}

// DiscoverTeams handles GET /v1/teams/discover
func (h *TeamHandler) DiscoverTeams(c echo.Context) error {
	userID := mw.GetUserID(c)
	teams, err := h.teamService.ListDiscoverable(c.Request().Context(), userID)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusOK, teams)
}

// GetTeam handles GET /v1/teams/:id
func (h *TeamHandler) GetTeam(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	userID := mw.GetUserID(c)
	resp, err := h.teamService.GetTeamDetail(c.Request().Context(), teamID, userID)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusOK, resp)
}

// UpdateTeam handles PUT /v1/teams/:id
func (h *TeamHandler) UpdateTeam(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	var req dto.UpdateTeamRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	// Generate slug from name
	newSlug := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(req.Name)), "-")
	newSlug = strings.Trim(newSlug, "-")

	resp, err := h.teamService.UpdateTeam(c.Request().Context(), teamID, &req, newSlug)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusOK, resp)
}

// DeleteTeam handles DELETE /v1/teams/:id
func (h *TeamHandler) DeleteTeam(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	if err := h.teamService.DeleteTeam(c.Request().Context(), teamID); err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusOK, map[string]string{"message": "Team deleted"})
}

// InviteMember handles POST /v1/teams/:id/members
func (h *TeamHandler) InviteMember(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	var req dto.InviteMemberRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
	}

	inviterID := mw.GetUserID(c)
	resp, err := h.teamService.InviteMember(c.Request().Context(), teamID, inviterID, targetUserID, req.Role)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusCreated, resp)
}

// UpdateMember handles PUT /v1/teams/:id/members/:userId
func (h *TeamHandler) UpdateMember(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
	}

	var req dto.UpdateMemberRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	actorID := mw.GetUserID(c)
	if err := h.teamService.UpdateMember(c.Request().Context(), teamID, actorID, targetUserID, &req); err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusOK, map[string]string{"message": "Member updated"})
}

// RemoveMember handles DELETE /v1/teams/:id/members/:userId
func (h *TeamHandler) RemoveMember(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
	}

	actorID := mw.GetUserID(c)

	// Self-removal: any member can remove themselves
	// Otherwise: require admin/maintainer (checked by middleware on non-self routes)
	if actorID != targetUserID {
		// Check if actor is admin or maintainer
		member, err := h.teamMemberRepo.GetByTeamAndUser(c.Request().Context(), teamID, actorID)
		if err != nil || (member.Role != "admin" && member.Role != "maintainer") {
			if mw.GetUserRole(c) != "root" {
				return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Insufficient team role")
			}
		}
	}

	if err := h.teamService.RemoveMember(c.Request().Context(), teamID, actorID, targetUserID); err != nil {
		return h.mapError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// AssignRepo handles POST /v1/teams/:id/repositories
func (h *TeamHandler) AssignRepo(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	var req dto.AssignRepoRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	repoID, err := uuid.Parse(req.RepositoryID)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	addedBy := mw.GetUserID(c)
	resp, err := h.teamService.AssignRepo(c.Request().Context(), teamID, repoID, addedBy)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusCreated, resp)
}

// UnassignRepo handles DELETE /v1/teams/:id/repositories/:repoId
func (h *TeamHandler) UnassignRepo(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	repoID, err := uuid.Parse(c.Param("repoId"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	if err := h.teamService.UnassignRepo(c.Request().Context(), teamID, repoID); err != nil {
		return h.mapError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// SubmitJoinRequest handles POST /v1/teams/:id/join-requests
func (h *TeamHandler) SubmitJoinRequest(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	var req dto.SubmitJoinRequestDTO
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	userID := mw.GetUserID(c)
	resp, err := h.teamService.SubmitJoinRequest(c.Request().Context(), teamID, userID, req.Message)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusCreated, resp)
}

// ListJoinRequests handles GET /v1/teams/:id/join-requests
func (h *TeamHandler) ListJoinRequests(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	statusFilter := c.QueryParam("status")
	resp, err := h.teamService.ListJoinRequests(c.Request().Context(), teamID, statusFilter)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusOK, resp)
}

// ReviewJoinRequest handles PATCH /v1/teams/:id/join-requests/:requestId
func (h *TeamHandler) ReviewJoinRequest(c echo.Context) error {
	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	requestID, err := uuid.Parse(c.Param("requestId"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid request ID")
	}

	var req dto.ReviewJoinRequestDTO
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	reviewerID := mw.GetUserID(c)
	if err := h.teamService.ReviewJoinRequest(c.Request().Context(), requestID, reviewerID, req.Status, req.Role); err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, http.StatusOK, map[string]string{"message": "Join request reviewed"})
}

// --- PAT handlers (kept from Phase 2) ---

// SetPAT handles PUT /v1/teams/:id/pat
func (h *TeamHandler) SetPAT(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	var req dto.SetPATRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	userID := mw.GetUserID(c)

	// Validate PAT against GitHub
	githubUsername, err := h.githubService.ValidatePAT(ctx, req.Token)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPAT) {
			return response.Error(c, http.StatusUnprocessableEntity, "INVALID_PAT", "GitHub PAT is invalid or expired")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate PAT")
	}

	encrypted, nonce, err := encryption.Encrypt(req.Token, h.cfg.GithubTokenEncryptionKey)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to encrypt PAT")
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to store PAT")
	}

	if err := h.teamRepo.UpdatePAT(ctx, teamID, encrypted, nonce, userID, githubUsername); err != nil {
		h.log.Error().Err(err).Msg("failed to update PAT")
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to store PAT")
	}

	return response.Success(c, http.StatusOK, dto.PATInfoResponse{
		IsConfigured:   true,
		GithubUsername: githubUsername,
	})
}

// GetPATStatus handles GET /v1/teams/:id/pat/validate
func (h *TeamHandler) GetPATStatus(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	ctx := c.Request().Context()
	team, err := h.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Team not found")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load team")
	}

	if !team.HasPAT() {
		return response.Success(c, http.StatusOK, dto.PATInfoResponse{IsConfigured: false})
	}

	pat, err := encryption.Decrypt(*team.GithubPATEncrypted, team.GithubPATNonce, h.cfg.GithubTokenEncryptionKey)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to decrypt PAT")
	}

	username, err := h.githubService.ValidatePAT(ctx, pat)
	if err != nil {
		return response.Success(c, http.StatusOK, dto.PATValidationResponse{
			Valid: false,
			Error: "PAT is no longer valid",
		})
	}

	return response.Success(c, http.StatusOK, dto.PATValidationResponse{
		Valid:          true,
		GithubUsername: username,
	})
}

// RevokePAT handles DELETE /v1/teams/:id/pat
func (h *TeamHandler) RevokePAT(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	if err := h.teamRepo.ClearPAT(c.Request().Context(), teamID); err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke PAT")
	}

	return c.NoContent(http.StatusNoContent)
}

// mapError translates service-level errors to HTTP responses.
func (h *TeamHandler) mapError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, service.ErrTeamNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Team not found")
	case errors.Is(err, service.ErrSlugConflict):
		return response.Error(c, http.StatusConflict, "CONFLICT", "A team with this slug already exists")
	case errors.Is(err, service.ErrAlreadyMember):
		return response.Error(c, http.StatusConflict, "CONFLICT", "User is already a team member")
	case errors.Is(err, service.ErrNotTeamMember):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Member not found")
	case errors.Is(err, service.ErrLastAdmin):
		return response.Error(c, http.StatusConflict, "CONFLICT", "Cannot remove or demote the last admin")
	case errors.Is(err, service.ErrAlreadyRequested):
		return response.Error(c, http.StatusConflict, "CONFLICT", "Join request already pending")
	case errors.Is(err, service.ErrRequestNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Join request not found")
	case errors.Is(err, service.ErrRepoAlreadyAssigned):
		return response.Error(c, http.StatusConflict, "CONFLICT", "Repository already assigned to team")
	case errors.Is(err, service.ErrRepoNotAssigned):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Repository not assigned to this team")
	case errors.Is(err, service.ErrTeamRepoNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Repository not found")
	case errors.Is(err, service.ErrInvalidRole):
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid role")
	case errors.Is(err, service.ErrSelfRoleChange):
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Cannot change your own role")
	default:
		h.log.Error().Err(err).Msg("team handler error")
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}
