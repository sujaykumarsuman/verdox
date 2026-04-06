package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type TestSuiteRepository interface {
	Create(ctx context.Context, suite *model.TestSuite) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.TestSuite, error)
	ListByRepositoryID(ctx context.Context, repoID uuid.UUID) ([]model.TestSuite, error)
	Update(ctx context.Context, suite *model.TestSuite) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type testSuiteRepo struct {
	db *sqlx.DB
}

func NewTestSuiteRepository(db *sqlx.DB) TestSuiteRepository {
	return &testSuiteRepo{db: db}
}

func (r *testSuiteRepo) Create(ctx context.Context, suite *model.TestSuite) error {
	query := `INSERT INTO test_suites (repository_id, name, type, execution_mode, docker_image, test_command, gha_workflow_id, env_vars, config_path, timeout_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		suite.RepositoryID, suite.Name, suite.Type, suite.ExecutionMode,
		suite.DockerImage, suite.TestCommand, suite.GHAWorkflowID, suite.EnvVars,
		suite.ConfigPath, suite.TimeoutSeconds,
	).Scan(&suite.ID, &suite.CreatedAt, &suite.UpdatedAt)
}

func (r *testSuiteRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.TestSuite, error) {
	var suite model.TestSuite
	err := r.db.GetContext(ctx, &suite, "SELECT * FROM test_suites WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &suite, err
}

func (r *testSuiteRepo) ListByRepositoryID(ctx context.Context, repoID uuid.UUID) ([]model.TestSuite, error) {
	var suites []model.TestSuite
	err := r.db.SelectContext(ctx, &suites,
		"SELECT * FROM test_suites WHERE repository_id = $1 ORDER BY created_at", repoID)
	if err != nil {
		return nil, err
	}
	return suites, nil
}

func (r *testSuiteRepo) Update(ctx context.Context, suite *model.TestSuite) error {
	query := `UPDATE test_suites SET name = $1, type = $2, execution_mode = $3, docker_image = $4, test_command = $5,
		gha_workflow_id = $6, env_vars = $7, config_path = $8, timeout_seconds = $9, updated_at = now()
		WHERE id = $10`
	_, err := r.db.ExecContext(ctx, query,
		suite.Name, suite.Type, suite.ExecutionMode, suite.DockerImage, suite.TestCommand,
		suite.GHAWorkflowID, suite.EnvVars, suite.ConfigPath, suite.TimeoutSeconds, suite.ID)
	return err
}

func (r *testSuiteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM test_suites WHERE id = $1", id)
	return err
}
