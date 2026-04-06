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

// DiscoverableTeam is a read-only projection for the discover endpoint.
type DiscoverableTeam struct {
	ID          uuid.UUID  `db:"id"`
	Name        string     `db:"name"`
	Slug        string     `db:"slug"`
	MemberCount int        `db:"member_count"`
	RepoCount   int        `db:"repo_count"`
	CreatedAt   time.Time  `db:"created_at"`
	UserStatus  *string    `db:"user_status"`
}

type TeamRepository interface {
	Create(ctx context.Context, team *model.Team) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Team, error)
	GetBySlug(ctx context.Context, slug string) (*model.Team, error)
	Update(ctx context.Context, teamID uuid.UUID, name, slug string) error
	SoftDelete(ctx context.Context, teamID uuid.UUID) error
	ListDiscoverable(ctx context.Context, userID uuid.UUID) ([]DiscoverableTeam, error)
	UpdatePAT(ctx context.Context, teamID uuid.UUID, encrypted string, nonce []byte, setBy uuid.UUID, githubUsername string) error
	ClearPAT(ctx context.Context, teamID uuid.UUID) error
	ListAll(ctx context.Context) ([]model.Team, error)
}

type teamRepo struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) TeamRepository {
	return &teamRepo{db: db}
}

func (r *teamRepo) Create(ctx context.Context, team *model.Team) error {
	query := `INSERT INTO teams (name, slug, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, is_discoverable, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		team.Name, team.Slug, team.CreatedBy,
	).Scan(&team.ID, &team.IsDiscoverable, &team.CreatedAt, &team.UpdatedAt)
}

func (r *teamRepo) GetBySlug(ctx context.Context, slug string) (*model.Team, error) {
	var team model.Team
	err := r.db.GetContext(ctx, &team, "SELECT * FROM teams WHERE slug = $1 AND deleted_at IS NULL", slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &team, err
}

func (r *teamRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Team, error) {
	var team model.Team
	err := r.db.GetContext(ctx, &team, "SELECT * FROM teams WHERE id = $1 AND deleted_at IS NULL", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &team, err
}

func (r *teamRepo) Update(ctx context.Context, teamID uuid.UUID, name, slug string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE teams SET name = $1, slug = $2, updated_at = now() WHERE id = $3 AND deleted_at IS NULL",
		name, slug, teamID)
	return err
}

func (r *teamRepo) SoftDelete(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE teams SET deleted_at = now(), updated_at = now() WHERE id = $1", teamID)
	return err
}

func (r *teamRepo) ListDiscoverable(ctx context.Context, userID uuid.UUID) ([]DiscoverableTeam, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.created_at,
			(SELECT COUNT(*) FROM team_members WHERE team_id = t.id AND status = 'approved') AS member_count,
			(SELECT COUNT(*) FROM team_repositories WHERE team_id = t.id) AS repo_count,
			COALESCE(
				(SELECT tm.status::text FROM team_members tm WHERE tm.team_id = t.id AND tm.user_id = $1 LIMIT 1),
				(SELECT jr.status::text FROM team_join_requests jr WHERE jr.team_id = t.id AND jr.user_id = $1 LIMIT 1)
			) AS user_status
		FROM teams t
		WHERE t.deleted_at IS NULL AND t.is_discoverable = true
		ORDER BY t.name`
	var teams []DiscoverableTeam
	err := r.db.SelectContext(ctx, &teams, query, userID)
	return teams, err
}

func (r *teamRepo) UpdatePAT(ctx context.Context, teamID uuid.UUID, encrypted string, nonce []byte, setBy uuid.UUID, githubUsername string) error {
	query := `UPDATE teams
		SET github_pat_encrypted = $1,
			github_pat_nonce = $2,
			github_pat_set_at = now(),
			github_pat_set_by = $3,
			github_pat_github_username = $4,
			updated_at = now()
		WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, encrypted, nonce, setBy, githubUsername, teamID)
	return err
}

func (r *teamRepo) ClearPAT(ctx context.Context, teamID uuid.UUID) error {
	query := `UPDATE teams
		SET github_pat_encrypted = NULL,
			github_pat_nonce = NULL,
			github_pat_set_at = NULL,
			github_pat_set_by = NULL,
			github_pat_github_username = NULL,
			updated_at = now()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, teamID)
	return err
}

func (r *teamRepo) ListAll(ctx context.Context) ([]model.Team, error) {
	var teams []model.Team
	err := r.db.SelectContext(ctx, &teams,
		"SELECT * FROM teams WHERE deleted_at IS NULL ORDER BY name")
	return teams, err
}
