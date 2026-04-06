package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type TeamRepository interface {
	Create(ctx context.Context, team *model.Team) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Team, error)
	GetBySlug(ctx context.Context, slug string) (*model.Team, error)
	SoftDelete(ctx context.Context, teamID uuid.UUID) error
	UpdatePAT(ctx context.Context, teamID uuid.UUID, encrypted string, nonce []byte, setBy uuid.UUID, githubUsername string) error
	ClearPAT(ctx context.Context, teamID uuid.UUID) error
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
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		team.Name, team.Slug, team.CreatedBy,
	).Scan(&team.ID, &team.CreatedAt, &team.UpdatedAt)
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

func (r *teamRepo) SoftDelete(ctx context.Context, teamID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE teams SET deleted_at = now(), updated_at = now() WHERE id = $1", teamID)
	return err
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
