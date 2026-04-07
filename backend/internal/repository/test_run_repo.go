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

type TestRunRepository interface {
	Create(ctx context.Context, run *model.TestRun) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.TestRun, error)
	ListBySuiteID(ctx context.Context, suiteID uuid.UUID, status, branch string, page, perPage int) ([]model.TestRun, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.TestRunStatus) error
	UpdateStarted(ctx context.Context, id uuid.UUID) error
	UpdateFinished(ctx context.Context, id uuid.UUID, status model.TestRunStatus) error
	NextRunNumber(ctx context.Context, suiteID uuid.UUID) (int, error)
	FindTerminalRun(ctx context.Context, suiteID uuid.UUID, branch, commitHash string) (*model.TestRun, error)
	GetLatestBySuiteID(ctx context.Context, suiteID uuid.UUID) (*model.TestRun, error)
	UpdateGHARunID(ctx context.Context, id uuid.UUID, ghaRunID int64) error
	FindByGHARunID(ctx context.Context, ghaRunID int64) (*model.TestRun, error)
	UpdateLogOutput(ctx context.Context, id uuid.UUID, logOutput string) error
	ListActiveGHARuns(ctx context.Context) ([]model.TestRun, error)
	UpdateSummaryAndFinish(ctx context.Context, id uuid.UUID, summary model.SummaryJSON, status model.TestRunStatus) error
	ListByReportID(ctx context.Context, reportID string) ([]model.TestRun, error)
}

type testRunRepo struct {
	db *sqlx.DB
}

func NewTestRunRepository(db *sqlx.DB) TestRunRepository {
	return &testRunRepo{db: db}
}

func (r *testRunRepo) Create(ctx context.Context, run *model.TestRun) error {
	query := `INSERT INTO test_runs (test_suite_id, triggered_by, run_number, branch, commit_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		run.TestSuiteID, run.TriggeredBy, run.RunNumber, run.Branch, run.CommitHash, run.Status,
	).Scan(&run.ID, &run.CreatedAt)
}

func (r *testRunRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.TestRun, error) {
	var run model.TestRun
	err := r.db.GetContext(ctx, &run, "SELECT * FROM test_runs WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &run, err
}

func (r *testRunRepo) ListBySuiteID(ctx context.Context, suiteID uuid.UUID, status, branch string, page, perPage int) ([]model.TestRun, int, error) {
	offset := (page - 1) * perPage

	countQuery := `SELECT COUNT(*) FROM test_runs WHERE test_suite_id = $1`
	listQuery := `SELECT * FROM test_runs WHERE test_suite_id = $1`

	args := []interface{}{suiteID}
	argIdx := 2

	if status != "" {
		filter := fmt.Sprintf(` AND status = $%d`, argIdx)
		countQuery += filter
		listQuery += filter
		args = append(args, status)
		argIdx++
	}

	if branch != "" {
		filter := fmt.Sprintf(` AND branch = $%d`, argIdx)
		countQuery += filter
		listQuery += filter
		args = append(args, branch)
		argIdx++
	}

	listQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	listArgs := append(args, perPage, offset)
	var runs []model.TestRun
	if err := r.db.SelectContext(ctx, &runs, listQuery, listArgs...); err != nil {
		return nil, 0, err
	}

	return runs, total, nil
}

func (r *testRunRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.TestRunStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE test_runs SET status = $1 WHERE id = $2", status, id)
	return err
}

func (r *testRunRepo) UpdateStarted(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE test_runs SET status = 'running', started_at = now() WHERE id = $1", id)
	return err
}

func (r *testRunRepo) UpdateFinished(ctx context.Context, id uuid.UUID, status model.TestRunStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE test_runs SET status = $1, finished_at = now() WHERE id = $2", status, id)
	return err
}

func (r *testRunRepo) NextRunNumber(ctx context.Context, suiteID uuid.UUID) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n,
		"SELECT COALESCE(MAX(run_number), 0) + 1 FROM test_runs WHERE test_suite_id = $1", suiteID)
	return n, err
}

func (r *testRunRepo) FindTerminalRun(ctx context.Context, suiteID uuid.UUID, branch, commitHash string) (*model.TestRun, error) {
	var run model.TestRun
	err := r.db.GetContext(ctx, &run,
		`SELECT * FROM test_runs
		WHERE test_suite_id = $1 AND branch = $2 AND commit_hash = $3 AND status IN ('passed', 'failed')
		ORDER BY run_number DESC LIMIT 1`,
		suiteID, branch, commitHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *testRunRepo) GetLatestBySuiteID(ctx context.Context, suiteID uuid.UUID) (*model.TestRun, error) {
	var run model.TestRun
	err := r.db.GetContext(ctx, &run,
		"SELECT * FROM test_runs WHERE test_suite_id = $1 ORDER BY created_at DESC LIMIT 1", suiteID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *testRunRepo) UpdateGHARunID(ctx context.Context, id uuid.UUID, ghaRunID int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE test_runs SET gha_run_id = $1 WHERE id = $2", ghaRunID, id)
	return err
}

func (r *testRunRepo) FindByGHARunID(ctx context.Context, ghaRunID int64) (*model.TestRun, error) {
	var run model.TestRun
	err := r.db.GetContext(ctx, &run,
		"SELECT * FROM test_runs WHERE gha_run_id = $1", ghaRunID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &run, err
}

func (r *testRunRepo) UpdateLogOutput(ctx context.Context, id uuid.UUID, logOutput string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE test_runs SET log_output = $1 WHERE id = $2", logOutput, id)
	return err
}

func (r *testRunRepo) ListActiveGHARuns(ctx context.Context) ([]model.TestRun, error) {
	var runs []model.TestRun
	err := r.db.SelectContext(ctx, &runs,
		"SELECT * FROM test_runs WHERE gha_run_id IS NOT NULL AND status IN ('queued', 'running') ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *testRunRepo) UpdateSummaryAndFinish(ctx context.Context, id uuid.UUID, summary model.SummaryJSON, status model.TestRunStatus) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE test_runs SET summary = $1, status = $2, finished_at = now() WHERE id = $3",
		summary, status, id)
	return err
}

func (r *testRunRepo) ListByReportID(ctx context.Context, reportID string) ([]model.TestRun, error) {
	var runs []model.TestRun
	err := r.db.SelectContext(ctx, &runs,
		"SELECT * FROM test_runs WHERE report_id = $1 ORDER BY created_at", reportID)
	if err != nil {
		return nil, err
	}
	return runs, nil
}
