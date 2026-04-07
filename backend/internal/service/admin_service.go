package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/sse"
)

var (
	ErrSelfDemotion                  = errors.New("cannot change your own role")
	ErrLastRoot                      = errors.New("cannot demote the last root user")
	ErrSelfDeactivation              = errors.New("cannot deactivate your own account")
	ErrModeratorCannotChangeRoles    = errors.New("only root users can change roles")
	ErrModeratorCannotDeactivateRoot = errors.New("moderators cannot deactivate root users")
	ErrCannotAssignRoot              = errors.New("cannot assign root role")
	ErrSelfBan                       = errors.New("cannot ban your own account")
	ErrCannotBanRoot                 = errors.New("cannot ban a root user")
	ErrBanReasonRequired             = errors.New("ban reason is required")
	ErrReviewNotFound                = errors.New("ban review not found")
	ErrReviewAlreadyProcessed        = errors.New("ban review has already been processed")
)

type AdminService struct {
	userRepo       repository.UserRepository
	sessionRepo    repository.SessionRepository
	banReviewRepo  repository.BanReviewRepository
	teamMemberRepo repository.TeamMemberRepository
	teamRepo       repository.TeamRepository
	notifService   *NotificationService
	rdb            *redis.Client
	db             *sqlx.DB
	log            zerolog.Logger
}

func NewAdminService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	banReviewRepo repository.BanReviewRepository,
	teamMemberRepo repository.TeamMemberRepository,
	teamRepo repository.TeamRepository,
	notifService *NotificationService,
	rdb *redis.Client,
	db *sqlx.DB,
	log zerolog.Logger,
) *AdminService {
	return &AdminService{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		banReviewRepo:  banReviewRepo,
		teamMemberRepo: teamMemberRepo,
		teamRepo:       teamRepo,
		notifService:   notifService,
		rdb:            rdb,
		db:             db,
		log:            log,
	}
}

func (s *AdminService) ListUsers(ctx context.Context, req *dto.AdminUserListRequest) (*dto.AdminUserListResponse, error) {
	req.Defaults()
	offset := (req.Page - 1) * req.PerPage

	users, total, err := s.userRepo.ListFiltered(ctx, req.Search, req.Role, req.Status, req.Sort, req.Order, offset, req.PerPage)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	resp := &dto.AdminUserListResponse{
		Users:      make([]dto.AdminUserResponse, len(users)),
		Total:      total,
		Page:       req.Page,
		PerPage:    req.PerPage,
		TotalPages: int(math.Ceil(float64(total) / float64(req.PerPage))),
	}
	for i, u := range users {
		resp.Users[i] = dto.NewAdminUserResponse(&u)
		count, _ := s.teamMemberRepo.CountByUser(ctx, u.ID)
		resp.Users[i].TeamCount = count
	}

	return resp, nil
}

func (s *AdminService) UpdateUser(ctx context.Context, callerID uuid.UUID, callerRole string, targetID uuid.UUID, req *dto.UpdateUserRequest) error {
	// Handle role change — root and admin can change roles
	if req.Role != nil {
		canChangeRoles := callerRole == string(model.RoleRoot) || callerRole == string(model.RoleAdmin)
		if !canChangeRoles {
			return ErrModeratorCannotChangeRoles
		}
		if callerID == targetID {
			return ErrSelfDemotion
		}
		if *req.Role == model.RoleRoot {
			return ErrCannotAssignRoot
		}

		target, err := s.userRepo.GetByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("get target user: %w", err)
		}

		// Only root can change another root's role
		if target.Role == model.RoleRoot && callerRole != string(model.RoleRoot) {
			return ErrModeratorCannotChangeRoles
		}

		// Prevent demoting the last root
		if target.Role == model.RoleRoot {
			count, err := s.userRepo.CountByRole(ctx, string(model.RoleRoot))
			if err != nil {
				return fmt.Errorf("count root users: %w", err)
			}
			if count <= 1 {
				return ErrLastRoot
			}
		}

		target.Role = *req.Role
		if err := s.userRepo.Update(ctx, target); err != nil {
			return fmt.Errorf("update user role: %w", err)
		}

		s.log.Info().
			Str("caller_id", callerID.String()).
			Str("target_id", targetID.String()).
			Str("new_role", string(*req.Role)).
			Msg("admin: user role changed")
	}

	// Handle activation/deactivation
	if req.IsActive != nil {
		if callerID == targetID {
			return ErrSelfDeactivation
		}

		target, err := s.userRepo.GetByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("get target user: %w", err)
		}

		// Only root can deactivate root users; moderators can't deactivate root or admin
		if target.Role == model.RoleRoot && callerRole != string(model.RoleRoot) {
			return ErrModeratorCannotDeactivateRoot
		}
		if callerRole == string(model.RoleModerator) && (target.Role == model.RoleRoot || target.Role == model.RoleAdmin) {
			return ErrModeratorCannotDeactivateRoot
		}

		if *req.IsActive {
			if err := s.userRepo.ReactivateUser(ctx, targetID); err != nil {
				return fmt.Errorf("reactivate user: %w", err)
			}
			s.log.Info().
				Str("caller_id", callerID.String()).
				Str("target_id", targetID.String()).
				Msg("admin: user reactivated")
		} else {
			if err := s.userRepo.DeactivateUser(ctx, targetID); err != nil {
				return fmt.Errorf("deactivate user: %w", err)
			}
			// Invalidate all sessions
			if err := s.sessionRepo.DeleteByUserID(ctx, targetID); err != nil {
				s.log.Error().Err(err).Str("target_id", targetID.String()).Msg("failed to delete sessions on deactivation")
			}
			sessionKey := fmt.Sprintf("session:%s", targetID.String())
			s.rdb.Del(ctx, sessionKey)

			s.log.Info().
				Str("caller_id", callerID.String()).
				Str("target_id", targetID.String()).
				Msg("admin: user deactivated")
		}
	}

	// Handle ban/unban — root and admin only
	if req.IsBanned != nil {
		canBan := callerRole == string(model.RoleRoot) || callerRole == string(model.RoleAdmin)
		if !canBan {
			return ErrModeratorCannotChangeRoles
		}
		if callerID == targetID {
			return ErrSelfBan
		}

		target, err := s.userRepo.GetByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("get target user: %w", err)
		}

		if target.Role == model.RoleRoot {
			return ErrCannotBanRoot
		}

		if *req.IsBanned {
			// Ban reason is required
			if req.BanReason == nil || *req.BanReason == "" {
				return ErrBanReasonRequired
			}
			if err := s.userRepo.BanUser(ctx, targetID, *req.BanReason); err != nil {
				return fmt.Errorf("ban user: %w", err)
			}
			// Invalidate all sessions
			if err := s.sessionRepo.DeleteByUserID(ctx, targetID); err != nil {
				s.log.Error().Err(err).Str("target_id", targetID.String()).Msg("failed to delete sessions on ban")
			}
			sessionKey := fmt.Sprintf("session:%s", targetID.String())
			s.rdb.Del(ctx, sessionKey)

			// Push SSE event to the banned user for immediate redirect
			_ = sse.PublishEvent(ctx, s.rdb, targetID, sse.EventBanned, map[string]string{
				"reason": *req.BanReason,
			})

			// Create notification for the banned user (visible after unban)
			if s.notifService != nil {
				_ = s.notifService.CreateAndPublish(ctx, &model.Notification{
					UserID:  targetID,
					Type:    model.NotificationSystem,
					Subject: "Your account has been banned",
					Body:    fmt.Sprintf("Reason: %s", *req.BanReason),
				})
			}

			s.log.Info().
				Str("caller_id", callerID.String()).
				Str("target_id", targetID.String()).
				Msg("admin: user banned")
		} else {
			if err := s.userRepo.UnbanUser(ctx, targetID); err != nil {
				return fmt.Errorf("unban user: %w", err)
			}
			// Clear ban reviews so the user gets fresh attempts if banned again
			_ = s.banReviewRepo.DeleteByUser(ctx, targetID)

			// Push SSE event to the unbanned user
			_ = sse.PublishEvent(ctx, s.rdb, targetID, sse.EventUnbanned, nil)

			s.log.Info().
				Str("caller_id", callerID.String()).
				Str("target_id", targetID.String()).
				Msg("admin: user unbanned")
		}
	}

	return nil
}

func (s *AdminService) GetStats(ctx context.Context) (*dto.StatsResponse, error) {
	stats := &dto.StatsResponse{}

	// User counts
	total, active, err := s.userRepo.CountByActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	stats.TotalUsers = total
	stats.ActiveUsers = active

	// Repository count
	err = s.db.GetContext(ctx, &stats.TotalRepos, "SELECT COUNT(*) FROM repositories WHERE is_active = true")
	if err != nil {
		return nil, fmt.Errorf("count repos: %w", err)
	}

	// Team count
	err = s.db.GetContext(ctx, &stats.TotalTeams, "SELECT COUNT(*) FROM teams WHERE deleted_at IS NULL")
	if err != nil {
		return nil, fmt.Errorf("count teams: %w", err)
	}

	// Test suite count
	err = s.db.GetContext(ctx, &stats.TotalSuites, "SELECT COUNT(*) FROM test_suites")
	if err != nil {
		return nil, fmt.Errorf("count suites: %w", err)
	}

	// Total test runs
	err = s.db.GetContext(ctx, &stats.TotalTestRuns, "SELECT COUNT(*) FROM test_runs")
	if err != nil {
		return nil, fmt.Errorf("count runs: %w", err)
	}

	// Runs today
	today := time.Now().Truncate(24 * time.Hour)
	err = s.db.GetContext(ctx, &stats.RunsToday,
		"SELECT COUNT(*) FROM test_runs WHERE created_at >= $1", today)
	if err != nil {
		return nil, fmt.Errorf("count runs today: %w", err)
	}

	// Pass rate last 7 days
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	var passRateResult struct {
		Total  int `db:"total"`
		Passed int `db:"passed"`
	}
	err = s.db.GetContext(ctx, &passRateResult,
		`SELECT COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'passed') AS passed
		FROM test_runs WHERE created_at >= $1`, sevenDaysAgo)
	if err != nil {
		return nil, fmt.Errorf("calculate pass rate: %w", err)
	}
	if passRateResult.Total > 0 {
		stats.PassRate7d = math.Round(float64(passRateResult.Passed)/float64(passRateResult.Total)*10000) / 100
	}

	return stats, nil
}

func (s *AdminService) ListPendingBanReviews(ctx context.Context) (*dto.PendingBanReviewsResponse, error) {
	reviews, err := s.banReviewRepo.ListPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pending reviews: %w", err)
	}

	resp := &dto.PendingBanReviewsResponse{
		Reviews: make([]dto.BanReviewResponse, len(reviews)),
		Count:   len(reviews),
	}
	for i, r := range reviews {
		resp.Reviews[i] = dto.BanReviewResponse{
			ID:            r.ID.String(),
			UserID:        r.UserID.String(),
			Username:      r.Username,
			Email:         r.Email,
			BanReason:     r.BanReason,
			Clarification: r.Clarification,
			Status:        r.Status,
			CreatedAt:     r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if r.ReviewedAt != nil {
			t := r.ReviewedAt.Format("2006-01-02T15:04:05Z")
			resp.Reviews[i].ReviewedAt = &t
		}
	}

	return resp, nil
}

func (s *AdminService) ReviewBan(ctx context.Context, callerID uuid.UUID, reviewID uuid.UUID, decision string) error {
	review, err := s.banReviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return ErrReviewNotFound
	}

	if review.Status != "pending" {
		return ErrReviewAlreadyProcessed
	}

	if err := s.banReviewRepo.UpdateStatus(ctx, reviewID, decision, callerID); err != nil {
		return fmt.Errorf("update review status: %w", err)
	}

	// If approved, unban the user
	if decision == "approved" {
		if err := s.userRepo.UnbanUser(ctx, review.UserID); err != nil {
			return fmt.Errorf("unban user: %w", err)
		}
		// Clear all ban reviews for fresh attempts if banned again
		_ = s.banReviewRepo.DeleteByUser(ctx, review.UserID)

		// Notify the user their review was approved
		_ = sse.PublishEvent(ctx, s.rdb, review.UserID, sse.EventUnbanned, nil)
		if s.notifService != nil {
			_ = s.notifService.CreateAndPublish(ctx, &model.Notification{
				UserID:  review.UserID,
				Type:    model.NotificationSystem,
				Subject: "Your ban appeal has been approved",
				Body:    "Your account has been unbanned. You can now log in again.",
			})
		}

		s.log.Info().
			Str("caller_id", callerID.String()).
			Str("review_id", reviewID.String()).
			Str("user_id", review.UserID.String()).
			Msg("admin: ban review approved, user unbanned")
	} else {
		// Notify the user their review was denied
		if s.notifService != nil {
			_ = s.notifService.CreateAndPublish(ctx, &model.Notification{
				UserID:  review.UserID,
				Type:    model.NotificationSystem,
				Subject: "Your ban appeal has been denied",
				Body:    "An administrator has reviewed and denied your ban appeal.",
			})
		}

		s.log.Info().
			Str("caller_id", callerID.String()).
			Str("review_id", reviewID.String()).
			Str("user_id", review.UserID.String()).
			Msg("admin: ban review denied")
	}

	return nil
}

func (s *AdminService) CountPendingReviews(ctx context.Context) (int, error) {
	return s.banReviewRepo.CountPending(ctx)
}

func (s *AdminService) GetUserTeams(ctx context.Context, userID uuid.UUID) ([]dto.UserTeamEntry, error) {
	teams, err := s.teamMemberRepo.ListTeamsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user teams: %w", err)
	}

	// Get the user's role in each team
	entries := make([]dto.UserTeamEntry, 0, len(teams))
	for _, t := range teams {
		member, err := s.teamMemberRepo.GetByTeamAndUser(ctx, t.ID, userID)
		if err != nil {
			continue
		}
		entries = append(entries, dto.UserTeamEntry{
			TeamID:   t.ID.String(),
			TeamName: t.Name,
			TeamSlug: t.Slug,
			Role:     string(member.Role),
		})
	}

	return entries, nil
}

func (s *AdminService) ListAllTeams(ctx context.Context) ([]dto.AdminTeamEntry, error) {
	teams, err := s.teamRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all teams: %w", err)
	}

	entries := make([]dto.AdminTeamEntry, len(teams))
	for i, t := range teams {
		count, _ := s.teamMemberRepo.CountMembers(ctx, t.ID)
		entries[i] = dto.AdminTeamEntry{
			ID:          t.ID.String(),
			Name:        t.Name,
			Slug:        t.Slug,
			MemberCount: count,
		}
	}

	return entries, nil
}

func (s *AdminService) UpdateUserTeams(ctx context.Context, callerID, targetUserID uuid.UUID, teamIDs []uuid.UUID) error {
	currentIDs, err := s.teamMemberRepo.ListTeamIDsByUser(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("list current teams: %w", err)
	}

	currentSet := make(map[uuid.UUID]bool, len(currentIDs))
	for _, id := range currentIDs {
		currentSet[id] = true
	}

	desiredSet := make(map[uuid.UUID]bool, len(teamIDs))
	for _, id := range teamIDs {
		desiredSet[id] = true
	}

	// Add to new teams
	for _, id := range teamIDs {
		if !currentSet[id] {
			member := &model.TeamMember{
				TeamID:    id,
				UserID:    targetUserID,
				Role:      model.TeamMemberRoleViewer,
				Status:    model.TeamMemberStatusApproved,
				InvitedBy: &callerID,
			}
			if err := s.teamMemberRepo.Create(ctx, member); err != nil {
				s.log.Error().Err(err).Str("team_id", id.String()).Msg("failed to add user to team")
			}
		}
	}

	// Remove from deselected teams
	for _, id := range currentIDs {
		if !desiredSet[id] {
			if err := s.teamMemberRepo.Delete(ctx, id, targetUserID); err != nil {
				s.log.Error().Err(err).Str("team_id", id.String()).Msg("failed to remove user from team")
			}
		}
	}

	s.log.Info().
		Str("caller_id", callerID.String()).
		Str("target_user_id", targetUserID.String()).
		Int("team_count", len(teamIDs)).
		Msg("admin: user teams updated")

	return nil
}
