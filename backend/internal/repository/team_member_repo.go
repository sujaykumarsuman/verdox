package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type TeamMemberRepository interface {
	Create(ctx context.Context, member *model.TeamMember) error
	GetByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error)
	IsTeamAdmin(ctx context.Context, teamID, userID uuid.UUID) (bool, error)
	IsTeamMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error)
	ListTeamsByUser(ctx context.Context, userID uuid.UUID) ([]model.Team, error)
	ListMembersByTeam(ctx context.Context, teamID uuid.UUID) ([]model.TeamMember, error)
}

type teamMemberRepo struct {
	db *sqlx.DB
}

func NewTeamMemberRepository(db *sqlx.DB) TeamMemberRepository {
	return &teamMemberRepo{db: db}
}

func (r *teamMemberRepo) Create(ctx context.Context, member *model.TeamMember) error {
	query := `INSERT INTO team_members (team_id, user_id, role, status, invited_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		member.TeamID, member.UserID, member.Role, member.Status, member.InvitedBy,
	).Scan(&member.ID, &member.CreatedAt)
}

func (r *teamMemberRepo) ListMembersByTeam(ctx context.Context, teamID uuid.UUID) ([]model.TeamMember, error) {
	var members []model.TeamMember
	err := r.db.SelectContext(ctx, &members,
		`SELECT * FROM team_members WHERE team_id = $1 ORDER BY created_at`, teamID)
	return members, err
}

func (r *teamMemberRepo) GetByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error) {
	var tm model.TeamMember
	err := r.db.GetContext(ctx, &tm,
		"SELECT * FROM team_members WHERE team_id = $1 AND user_id = $2", teamID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &tm, err
}

func (r *teamMemberRepo) IsTeamAdmin(ctx context.Context, teamID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM team_members
		WHERE team_id = $1 AND user_id = $2 AND role = 'admin' AND status = 'approved'`,
		teamID, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teamMemberRepo) IsTeamMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM team_members
		WHERE team_id = $1 AND user_id = $2 AND status = 'approved'`,
		teamID, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teamMemberRepo) ListTeamsByUser(ctx context.Context, userID uuid.UUID) ([]model.Team, error) {
	var teams []model.Team
	err := r.db.SelectContext(ctx, &teams,
		`SELECT t.* FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = $1 AND tm.status = 'approved' AND t.deleted_at IS NULL
		ORDER BY t.name`, userID)
	return teams, err
}
