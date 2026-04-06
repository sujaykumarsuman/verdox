package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

var ErrNotFound = errors.New("record not found")

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	List(ctx context.Context, offset, limit int) ([]model.User, int, error)
	ListFiltered(ctx context.Context, search, role, status, sort, order string, offset, limit int) ([]model.User, int, error)
	CountByActive(ctx context.Context) (total int, active int, err error)
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	ReactivateUser(ctx context.Context, id uuid.UUID) error
	BanUser(ctx context.Context, id uuid.UUID, reason string) error
	UnbanUser(ctx context.Context, id uuid.UUID) error
	CountByRole(ctx context.Context, role string) (int, error)
}

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (username, email, password_hash, role, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.Role, user.AvatarURL,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE email = $1", email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE username = $1", username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user,
		"SELECT * FROM users WHERE email = $1 OR username = $1", login)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepo) Update(ctx context.Context, user *model.User) error {
	query := `UPDATE users SET username = $1, email = $2, password_hash = $3, role = $4,
		avatar_url = $5, is_active = $6, is_banned = $7, ban_reason = $8, updated_at = now() WHERE id = $9`
	_, err := r.db.ExecContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.Role, user.AvatarURL, user.IsActive, user.IsBanned, user.BanReason, user.ID)
	return err
}

func (r *userRepo) List(ctx context.Context, offset, limit int) ([]model.User, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM users")
	if err != nil {
		return nil, 0, err
	}

	var users []model.User
	err = r.db.SelectContext(ctx, &users,
		"SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)
	return users, total, err
}

// validSortColumns prevents SQL injection in ORDER BY clauses.
var validSortColumns = map[string]bool{
	"created_at": true,
	"username":   true,
	"email":      true,
}

func (r *userRepo) ListFiltered(ctx context.Context, search, role, status, sort, order string, offset, limit int) ([]model.User, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(username ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, role)
		argIdx++
	}

	switch status {
	case "active":
		conditions = append(conditions, "is_active = true AND is_banned = false")
	case "inactive":
		conditions = append(conditions, "is_active = false AND is_banned = false")
	case "banned":
		conditions = append(conditions, "is_banned = true")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Validate sort column
	if !validSortColumns[sort] {
		sort = "created_at"
	}
	if order != "asc" {
		order = "desc"
	}

	// Count query
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", where)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := fmt.Sprintf("SELECT * FROM users %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, sort, order, argIdx, argIdx+1)
	args = append(args, limit, offset)

	var users []model.User
	err = r.db.SelectContext(ctx, &users, dataQuery, args...)
	return users, total, err
}

func (r *userRepo) CountByActive(ctx context.Context) (int, int, error) {
	var result struct {
		Total  int `db:"total"`
		Active int `db:"active"`
	}
	err := r.db.GetContext(ctx, &result,
		"SELECT COUNT(*) AS total, COUNT(*) FILTER (WHERE is_active) AS active FROM users")
	return result.Total, result.Active, err
}

func (r *userRepo) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET is_active = false, updated_at = now() WHERE id = $1", id)
	return err
}

func (r *userRepo) ReactivateUser(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET is_active = true, updated_at = now() WHERE id = $1", id)
	return err
}

func (r *userRepo) BanUser(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET is_banned = true, is_active = false, ban_reason = $1, updated_at = now() WHERE id = $2",
		reason, id)
	return err
}

func (r *userRepo) UnbanUser(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET is_banned = false, is_active = true, ban_reason = NULL, updated_at = now() WHERE id = $1", id)
	return err
}

func (r *userRepo) CountByRole(ctx context.Context, role string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users WHERE role = $1", role)
	return count, err
}
