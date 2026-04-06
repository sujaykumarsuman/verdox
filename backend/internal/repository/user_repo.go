package repository

import (
	"context"
	"database/sql"
	"errors"

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
		avatar_url = $5, updated_at = now() WHERE id = $6`
	_, err := r.db.ExecContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.Role, user.AvatarURL, user.ID)
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
