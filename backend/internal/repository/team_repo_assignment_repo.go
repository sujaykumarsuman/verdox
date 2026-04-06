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

// TeamRepoWithDetail is a read-only projection joining team_repositories with repositories.
type TeamRepoWithDetail struct {
	ID             uuid.UUID  `db:"id"`
	TeamID         uuid.UUID  `db:"team_id"`
	RepositoryID   uuid.UUID  `db:"repository_id"`
	AddedBy        *uuid.UUID `db:"added_by"`
	CreatedAt      time.Time  `db:"created_at"`
	RepositoryName string     `db:"repository_name"`
	GithubFullName string     `db:"github_full_name"`
	IsActive       bool       `db:"is_active"`
}

type TeamRepoAssignmentRepository interface {
	Create(ctx context.Context, tr *model.TeamRepository) error
	Delete(ctx context.Context, teamID, repoID uuid.UUID) error
	GetByTeamAndRepo(ctx context.Context, teamID, repoID uuid.UUID) (*model.TeamRepository, error)
	ListByTeam(ctx context.Context, teamID uuid.UUID) ([]TeamRepoWithDetail, error)
	CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error)
}

type teamRepoAssignmentRepo struct {
	db *sqlx.DB
}

func NewTeamRepoAssignmentRepository(db *sqlx.DB) TeamRepoAssignmentRepository {
	return &teamRepoAssignmentRepo{db: db}
}

func (r *teamRepoAssignmentRepo) Create(ctx context.Context, tr *model.TeamRepository) error {
	query := `INSERT INTO team_repositories (team_id, repository_id, added_by)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		tr.TeamID, tr.RepositoryID, tr.AddedBy,
	).Scan(&tr.ID, &tr.CreatedAt)
}

func (r *teamRepoAssignmentRepo) Delete(ctx context.Context, teamID, repoID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM team_repositories WHERE team_id = $1 AND repository_id = $2",
		teamID, repoID)
	return err
}

func (r *teamRepoAssignmentRepo) GetByTeamAndRepo(ctx context.Context, teamID, repoID uuid.UUID) (*model.TeamRepository, error) {
	var tr model.TeamRepository
	err := r.db.GetContext(ctx, &tr,
		"SELECT * FROM team_repositories WHERE team_id = $1 AND repository_id = $2",
		teamID, repoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &tr, err
}

func (r *teamRepoAssignmentRepo) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]TeamRepoWithDetail, error) {
	query := `SELECT tr.id, tr.team_id, tr.repository_id, tr.added_by, tr.created_at,
			repo.name AS repository_name, repo.github_full_name, repo.is_active
		FROM team_repositories tr
		JOIN repositories repo ON repo.id = tr.repository_id
		WHERE tr.team_id = $1
		ORDER BY tr.created_at`
	var repos []TeamRepoWithDetail
	err := r.db.SelectContext(ctx, &repos, query, teamID)
	return repos, err
}

func (r *teamRepoAssignmentRepo) CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM team_repositories WHERE team_id = $1", teamID)
	return count, err
}
