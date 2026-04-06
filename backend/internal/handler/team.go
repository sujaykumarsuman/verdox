package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/encryption"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
	v "github.com/sujaykumarsuman/verdox/backend/pkg/validator"
)

type TeamHandler struct {
	teamRepo       repository.TeamRepository
	teamMemberRepo repository.TeamMemberRepository
	githubService  *service.GitHubService
	cfg            *config.Config
	log            zerolog.Logger
}

func NewTeamHandler(
	teamRepo repository.TeamRepository,
	teamMemberRepo repository.TeamMemberRepository,
	githubService *service.GitHubService,
	cfg *config.Config,
	log zerolog.Logger,
) *TeamHandler {
	return &TeamHandler{
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
	teams, err := h.teamMemberRepo.ListTeamsByUser(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load teams")
	}

	resp := make([]dto.TeamResponse, len(teams))
	for i, t := range teams {
		resp[i] = dto.NewTeamResponse(&t)
	}
	return response.Success(c, http.StatusOK, resp)
}

// CreateTeam handles POST /v1/teams
func (h *TeamHandler) CreateTeam(c echo.Context) error {
	var req dto.CreateTeamRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	userID := mw.GetUserID(c)

	// Check slug uniqueness
	if _, err := h.teamRepo.GetBySlug(ctx, req.Slug); err == nil {
		return response.Error(c, http.StatusConflict, "CONFLICT", "A team with this slug already exists")
	}

	// Create team
	team := &model.Team{
		Name:      req.Name,
		Slug:      req.Slug,
		CreatedBy: &userID,
	}
	if err := h.teamRepo.Create(ctx, team); err != nil {
		h.log.Error().Err(err).Msg("failed to create team")
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create team")
	}

	// Add creator as admin member
	member := &model.TeamMember{
		TeamID: team.ID,
		UserID: userID,
		Role:   model.TeamMemberRoleAdmin,
		Status: model.TeamMemberStatusApproved,
	}
	if err := h.teamMemberRepo.Create(ctx, member); err != nil {
		h.log.Error().Err(err).Msg("failed to add creator as team admin")
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Team created but failed to add you as member")
	}

	return response.Success(c, http.StatusCreated, dto.NewTeamResponse(team))
}

// GetTeam handles GET /v1/teams/:id
func (h *TeamHandler) GetTeam(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	ctx := c.Request().Context()
	userID := mw.GetUserID(c)

	// Verify membership (or root)
	isMember, err := h.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check membership")
	}
	if !isMember && mw.GetUserRole(c) != "root" {
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Not a member of this team")
	}

	team, err := h.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Team not found")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load team")
	}

	return response.Success(c, http.StatusOK, dto.NewTeamResponse(team))
}

// DeleteTeam handles DELETE /v1/teams/:id
func (h *TeamHandler) DeleteTeam(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	userID := mw.GetUserID(c)
	ctx := c.Request().Context()

	if err := h.requireTeamAdmin(ctx, teamID, userID, mw.GetUserRole(c)); err != nil {
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Team admin role required")
	}

	if err := h.teamRepo.SoftDelete(ctx, teamID); err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete team")
	}

	return c.NoContent(http.StatusNoContent)
}

// SetPAT handles PUT /v1/teams/:id/pat
func (h *TeamHandler) SetPAT(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	userID := mw.GetUserID(c)
	ctx := c.Request().Context()

	// Check authorization: team admin or root
	if err := h.requireTeamAdmin(ctx, teamID, userID, mw.GetUserRole(c)); err != nil {
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Team admin role required")
	}

	var req dto.SetPATRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	// Validate PAT against GitHub
	githubUsername, err := h.githubService.ValidatePAT(ctx, req.Token)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPAT) {
			return response.Error(c, http.StatusUnprocessableEntity, "INVALID_PAT", "GitHub PAT is invalid or expired")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate PAT")
	}

	// Encrypt PAT
	encrypted, nonce, err := encryption.Encrypt(req.Token, h.cfg.GithubTokenEncryptionKey)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to encrypt PAT")
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to store PAT")
	}

	// Store
	if err := h.teamRepo.UpdatePAT(ctx, teamID, encrypted, nonce, userID, githubUsername); err != nil {
		h.log.Error().Err(err).Msg("failed to update PAT")
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to store PAT")
	}

	return response.Success(c, http.StatusOK, dto.PATInfoResponse{
		IsConfigured:   true,
		GithubUsername: githubUsername,
	})
}

// ValidatePAT handles GET /v1/teams/:id/pat/validate
func (h *TeamHandler) ValidatePAT(c echo.Context) error {
	teamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
	}

	userID := mw.GetUserID(c)
	ctx := c.Request().Context()

	// Check team membership
	isMember, err := h.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check membership")
	}
	if !isMember && mw.GetUserRole(c) != "root" {
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Not a member of this team")
	}

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

	// Decrypt and validate
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

	userID := mw.GetUserID(c)
	ctx := c.Request().Context()

	if err := h.requireTeamAdmin(ctx, teamID, userID, mw.GetUserRole(c)); err != nil {
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Team admin role required")
	}

	if err := h.teamRepo.ClearPAT(ctx, teamID); err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke PAT")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *TeamHandler) requireTeamAdmin(ctx context.Context, teamID, userID uuid.UUID, role string) error {
	// Root users bypass team admin check
	if role == "root" {
		return nil
	}

	isAdmin, err := h.teamMemberRepo.IsTeamAdmin(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return errors.New("not a team admin")
	}
	return nil
}
