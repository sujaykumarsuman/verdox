package dto

import (
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type AdminUserListRequest struct {
	Search  string `query:"search"`
	Role    string `query:"role"`
	Status  string `query:"status"` // "active" or "inactive"
	Page    int    `query:"page"`
	PerPage int    `query:"per_page"`
	Sort    string `query:"sort"`
	Order   string `query:"order"` // "asc" or "desc"
}

func (r *AdminUserListRequest) Defaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PerPage < 1 || r.PerPage > 100 {
		r.PerPage = 20
	}
	if r.Sort == "" {
		r.Sort = "created_at"
	}
	if r.Order == "" {
		r.Order = "desc"
	}
}

type UpdateUserRequest struct {
	Role      *model.UserRole `json:"role"`
	IsActive  *bool           `json:"is_active"`
	IsBanned  *bool           `json:"is_banned"`
	BanReason *string         `json:"ban_reason"`
}

type AdminUserResponse struct {
	ID        string         `json:"id"`
	Username  string         `json:"username"`
	Email     string         `json:"email"`
	Role      model.UserRole `json:"role"`
	AvatarURL *string        `json:"avatar_url"`
	IsActive  bool           `json:"is_active"`
	IsBanned  bool           `json:"is_banned"`
	BanReason *string        `json:"ban_reason"`
	TeamCount int            `json:"team_count"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

func NewAdminUserResponse(u *model.User) AdminUserResponse {
	return AdminUserResponse{
		ID:        u.ID.String(),
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		AvatarURL: u.AvatarURL,
		IsActive:  u.IsActive,
		IsBanned:  u.IsBanned,
		BanReason: u.BanReason,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

type AdminUserListResponse struct {
	Users      []AdminUserResponse `json:"users"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PerPage    int                 `json:"per_page"`
	TotalPages int                 `json:"total_pages"`
}

type BanReviewResponse struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	BanReason     string  `json:"ban_reason"`
	Clarification string  `json:"clarification"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	ReviewedAt    *string `json:"reviewed_at"`
}

type ReviewBanDecisionRequest struct {
	Status string `json:"status" validate:"required,oneof=approved denied"`
}

type PendingBanReviewsResponse struct {
	Reviews []BanReviewResponse `json:"reviews"`
	Count   int                 `json:"count"`
}

type UserTeamEntry struct {
	TeamID   string `json:"team_id"`
	TeamName string `json:"team_name"`
	TeamSlug string `json:"team_slug"`
	Role     string `json:"role"`
}

type AdminTeamEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	MemberCount int    `json:"member_count"`
}

type UpdateUserTeamsRequest struct {
	TeamIDs []string `json:"team_ids" validate:"required"`
}

type StatsResponse struct {
	TotalUsers    int     `json:"total_users"`
	ActiveUsers   int     `json:"active_users"`
	TotalRepos    int     `json:"total_repos"`
	TotalTeams    int     `json:"total_teams"`
	TotalTestRuns int     `json:"total_test_runs"`
	PassRate7d    float64 `json:"pass_rate_7d"`
	RunsToday     int     `json:"runs_today"`
	TotalSuites   int     `json:"total_suites"`
}
