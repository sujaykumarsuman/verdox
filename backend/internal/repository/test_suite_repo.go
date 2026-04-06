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
	query := `INSERT INTO test_suites (repository_id, name, type, config_path, timeout_seconds)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		suite.RepositoryID, suite.Name, suite.Type, suite.ConfigPath, suite.TimeoutSeconds,
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
	query := `UPDATE test_suites SET name = $1, type = $2, config_path = $3, timeout_seconds = $4, updated_at = now()
		WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query,
		suite.Name, suite.Type, suite.ConfigPath, suite.TimeoutSeconds, suite.ID)
	return err
}

func (r *testSuiteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM test_suites WHERE id = $1", id)
	return err
}
