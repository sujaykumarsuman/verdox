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

// MemberWithUser is a read-only projection joining team_members with users.
type MemberWithUser struct {
	ID        uuid.UUID              `db:"id"`
	TeamID    uuid.UUID              `db:"team_id"`
	UserID    uuid.UUID              `db:"user_id"`
	Role      model.TeamMemberRole   `db:"role"`
	Status    model.TeamMemberStatus `db:"status"`
	InvitedBy *uuid.UUID             `db:"invited_by"`
	CreatedAt time.Time              `db:"created_at"`
	Username  string                 `db:"username"`
	Email     string                 `db:"email"`
	AvatarURL *string                `db:"avatar_url"`
}

type TeamMemberRepository interface {
	Create(ctx context.Context, member *model.TeamMember) error
	GetByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error)
	IsTeamAdmin(ctx context.Context, teamID, userID uuid.UUID) (bool, error)
	IsTeamMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error)
	ListTeamsByUser(ctx context.Context, userID uuid.UUID) ([]model.Team, error)
	ListMembersByTeam(ctx context.Context, teamID uuid.UUID) ([]model.TeamMember, error)
	ListMembersWithUser(ctx context.Context, teamID uuid.UUID) ([]MemberWithUser, error)
	UpdateRole(ctx context.Context, teamID, userID uuid.UUID, role model.TeamMemberRole) error
	UpdateStatus(ctx context.Context, teamID, userID uuid.UUID, status model.TeamMemberStatus) error
	Delete(ctx context.Context, teamID, userID uuid.UUID) error
	CountAdmins(ctx context.Context, teamID uuid.UUID) (int, error)
	CountMembers(ctx context.Context, teamID uuid.UUID) (int, error)
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

func (r *teamMemberRepo) ListMembersWithUser(ctx context.Context, teamID uuid.UUID) ([]MemberWithUser, error) {
	query := `SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.status, tm.invited_by, tm.created_at,
			u.username, u.email, u.avatar_url
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id = $1
		ORDER BY tm.created_at`
	var members []MemberWithUser
	err := r.db.SelectContext(ctx, &members, query, teamID)
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

func (r *teamMemberRepo) UpdateRole(ctx context.Context, teamID, userID uuid.UUID, role model.TeamMemberRole) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE team_members SET role = $1 WHERE team_id = $2 AND user_id = $3",
		role, teamID, userID)
	return err
}

func (r *teamMemberRepo) UpdateStatus(ctx context.Context, teamID, userID uuid.UUID, status model.TeamMemberStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE team_members SET status = $1 WHERE team_id = $2 AND user_id = $3",
		status, teamID, userID)
	return err
}

func (r *teamMemberRepo) Delete(ctx context.Context, teamID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM team_members WHERE team_id = $1 AND user_id = $2",
		teamID, userID)
	return err
}

func (r *teamMemberRepo) CountAdmins(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND role = 'admin' AND status = 'approved'",
		teamID)
	return count, err
}

func (r *teamMemberRepo) CountMembers(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND status = 'approved'",
		teamID)
	return count, err
}
