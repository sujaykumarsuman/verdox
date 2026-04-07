package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type RepositoryRepository interface {
	Create(ctx context.Context, repo *model.Repository) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Repository, error)
	GetByGithubRepoID(ctx context.Context, githubRepoID int64) (*model.Repository, error)
	ListByTeamID(ctx context.Context, teamID uuid.UUID, search string, page, perPage int) ([]model.Repository, int, error)
	Update(ctx context.Context, repo *model.Repository) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Reactivate(ctx context.Context, repo *model.Repository) error
	AddTeamRepository(ctx context.Context, teamID, repoID uuid.UUID, addedBy uuid.UUID) error
	GetTeamIDForRepository(ctx context.Context, repoID uuid.UUID) (uuid.UUID, error)
	ListByUserTeams(ctx context.Context, userID uuid.UUID, search string, page, perPage int) ([]model.Repository, int, error)
}

type repositoryRepo struct {
	db *sqlx.DB
}

func NewRepositoryRepository(db *sqlx.DB) RepositoryRepository {
	return &repositoryRepo{db: db}
}

func (r *repositoryRepo) Create(ctx context.Context, repo *model.Repository) error {
	query := `INSERT INTO repositories (github_repo_id, github_full_name, name, description, default_branch)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, clone_status, fork_status, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		repo.GithubRepoID, repo.GithubFullName, repo.Name,
		repo.Description, repo.DefaultBranch,
	).Scan(&repo.ID, &repo.CloneStatus, &repo.ForkStatus, &repo.CreatedAt, &repo.UpdatedAt)
}

func (r *repositoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Repository, error) {
	var repo model.Repository
	err := r.db.GetContext(ctx, &repo, "SELECT * FROM repositories WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &repo, err
}

func (r *repositoryRepo) GetByGithubRepoID(ctx context.Context, githubRepoID int64) (*model.Repository, error) {
	var repo model.Repository
	err := r.db.GetContext(ctx, &repo, "SELECT * FROM repositories WHERE github_repo_id = $1", githubRepoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &repo, err
}

func (r *repositoryRepo) ListByTeamID(ctx context.Context, teamID uuid.UUID, search string, page, perPage int) ([]model.Repository, int, error) {
	offset := (page - 1) * perPage

	countQuery := `SELECT COUNT(*) FROM repositories r
		JOIN team_repositories tr ON tr.repository_id = r.id
		WHERE tr.team_id = $1 AND r.is_active = true`
	listQuery := `SELECT r.* FROM repositories r
		JOIN team_repositories tr ON tr.repository_id = r.id
		WHERE tr.team_id = $1 AND r.is_active = true`

	args := []interface{}{teamID}
	argIdx := 2

	if search != "" {
		filter := fmt.Sprintf(` AND (r.name ILIKE $%d OR r.github_full_name ILIKE $%d)`, argIdx, argIdx)
		countQuery += filter
		listQuery += filter
		args = append(args, "%"+search+"%")
		argIdx++
	}

	listQuery += fmt.Sprintf(` ORDER BY r.created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	listArgs := append(args, perPage, offset)
	var repos []model.Repository
	if err := r.db.SelectContext(ctx, &repos, listQuery, listArgs...); err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}

func (r *repositoryRepo) Update(ctx context.Context, repo *model.Repository) error {
	query := `UPDATE repositories SET description = $1, updated_at = now() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, repo.Description, repo.ID)
	return err
}

func (r *repositoryRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE repositories SET is_active = false, updated_at = now() WHERE id = $1", id)
	return err
}

func (r *repositoryRepo) Reactivate(ctx context.Context, repo *model.Repository) error {
	query := `UPDATE repositories
		SET is_active = true, description = $1, default_branch = $2, fork_status = 'none', updated_at = now()
		WHERE id = $3
		RETURNING updated_at`
	return r.db.QueryRowxContext(ctx, query,
		repo.Description, repo.DefaultBranch, repo.ID,
	).Scan(&repo.UpdatedAt)
}

func (r *repositoryRepo) AddTeamRepository(ctx context.Context, teamID, repoID uuid.UUID, addedBy uuid.UUID) error {
	query := `INSERT INTO team_repositories (team_id, repository_id, added_by) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, teamID, repoID, addedBy)
	return err
}

func (r *repositoryRepo) GetTeamIDForRepository(ctx context.Context, repoID uuid.UUID) (uuid.UUID, error) {
	var teamID uuid.UUID
	err := r.db.GetContext(ctx, &teamID,
		"SELECT team_id FROM team_repositories WHERE repository_id = $1 LIMIT 1", repoID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return teamID, err
}

func (r *repositoryRepo) ListByUserTeams(ctx context.Context, userID uuid.UUID, search string, page, perPage int) ([]model.Repository, int, error) {
	offset := (page - 1) * perPage

	countQuery := `SELECT COUNT(DISTINCT r.id) FROM repositories r
		JOIN team_repositories tr ON tr.repository_id = r.id
		JOIN team_members tm ON tm.team_id = tr.team_id
		WHERE tm.user_id = $1 AND tm.status = 'approved' AND r.is_active = true`
	listQuery := `SELECT DISTINCT r.* FROM repositories r
		JOIN team_repositories tr ON tr.repository_id = r.id
		JOIN team_members tm ON tm.team_id = tr.team_id
		WHERE tm.user_id = $1 AND tm.status = 'approved' AND r.is_active = true`

	args := []interface{}{userID}
	argIdx := 2

	if search != "" {
		filter := fmt.Sprintf(` AND (r.name ILIKE $%d OR r.github_full_name ILIKE $%d)`, argIdx, argIdx)
		countQuery += filter
		listQuery += filter
		args = append(args, "%"+search+"%")
		argIdx++
	}

	listQuery += fmt.Sprintf(` ORDER BY r.created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	listArgs := append(args, perPage, offset)
	var repos []model.Repository
	if err := r.db.SelectContext(ctx, &repos, listQuery, listArgs...); err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}
