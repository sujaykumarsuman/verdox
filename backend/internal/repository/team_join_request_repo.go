package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

// JoinRequestWithUser is a read-only projection joining team_join_requests with users.
type JoinRequestWithUser struct {
	ID           uuid.UUID               `db:"id"`
	TeamID       uuid.UUID               `db:"team_id"`
	UserID       uuid.UUID               `db:"user_id"`
	Message      *string                 `db:"message"`
	Status       model.TeamMemberStatus  `db:"status"`
	ReviewedBy   *uuid.UUID              `db:"reviewed_by"`
	RoleAssigned *model.TeamMemberRole   `db:"role_assigned"`
	CreatedAt    time.Time               `db:"created_at"`
	UpdatedAt    time.Time               `db:"updated_at"`
	Username     string                  `db:"username"`
	Email        string                  `db:"email"`
	AvatarURL    *string                 `db:"avatar_url"`
}

type TeamJoinRequestRepository interface {
	Create(ctx context.Context, req *model.TeamJoinRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.TeamJoinRequest, error)
	GetByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamJoinRequest, error)
	ListByTeam(ctx context.Context, teamID uuid.UUID, statusFilter string) ([]JoinRequestWithUser, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.TeamMemberStatus, reviewedBy uuid.UUID, roleAssigned *model.TeamMemberRole) error
}

type teamJoinRequestRepo struct {
	db *sqlx.DB
}

func NewTeamJoinRequestRepository(db *sqlx.DB) TeamJoinRequestRepository {
	return &teamJoinRequestRepo{db: db}
}

func (r *teamJoinRequestRepo) Create(ctx context.Context, req *model.TeamJoinRequest) error {
	query := `INSERT INTO team_join_requests (team_id, user_id, message)
		VALUES ($1, $2, $3)
		RETURNING id, status, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		req.TeamID, req.UserID, req.Message,
	).Scan(&req.ID, &req.Status, &req.CreatedAt, &req.UpdatedAt)
}

func (r *teamJoinRequestRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.TeamJoinRequest, error) {
	var req model.TeamJoinRequest
	err := r.db.GetContext(ctx, &req, "SELECT * FROM team_join_requests WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &req, err
}

func (r *teamJoinRequestRepo) GetByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamJoinRequest, error) {
	var req model.TeamJoinRequest
	err := r.db.GetContext(ctx, &req,
		"SELECT * FROM team_join_requests WHERE team_id = $1 AND user_id = $2", teamID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &req, err
}

func (r *teamJoinRequestRepo) ListByTeam(ctx context.Context, teamID uuid.UUID, statusFilter string) ([]JoinRequestWithUser, error) {
	query := `SELECT jr.id, jr.team_id, jr.user_id, jr.message, jr.status,
			jr.reviewed_by, jr.role_assigned, jr.created_at, jr.updated_at,
			u.username, u.email, u.avatar_url
		FROM team_join_requests jr
		JOIN users u ON u.id = jr.user_id
		WHERE jr.team_id = $1`
	args := []interface{}{teamID}

	if statusFilter != "" && statusFilter != "all" {
		query += " AND jr.status = $2"
		args = append(args, statusFilter)
	}
	query += " ORDER BY jr.created_at DESC"

	var requests []JoinRequestWithUser
	err := r.db.SelectContext(ctx, &requests, query, args...)
	return requests, err
}

func (r *teamJoinRequestRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.TeamMemberStatus, reviewedBy uuid.UUID, roleAssigned *model.TeamMemberRole) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE team_join_requests
		SET status = $1, reviewed_by = $2, role_assigned = $3, updated_at = now()
		WHERE id = $4`,
		status, reviewedBy, roleAssigned, id)
	return err
}
