package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/pkg/hash"
	"github.com/sujaykumarsuman/verdox/backend/pkg/jwt"
)

var (
	ErrEmailTaken          = errors.New("email already in use")
	ErrUsernameTaken       = errors.New("username already in use")
	ErrInvalidCredentials  = errors.New("invalid email/username or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
)

type AuthService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	resetRepo   repository.PasswordResetRepository
	rdb         *redis.Client
	cfg         *config.Config
	log         zerolog.Logger
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	resetRepo repository.PasswordResetRepository,
	rdb *redis.Client,
	cfg *config.Config,
	log zerolog.Logger,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		resetRepo:   resetRepo,
		rdb:         rdb,
		cfg:         cfg,
		log:         log,
	}
}

func (s *AuthService) Signup(ctx context.Context, req *dto.SignupRequest) (*dto.AuthResponse, string, error) {
	// Check email uniqueness
	if _, err := s.userRepo.GetByEmail(ctx, req.Email); err == nil {
		return nil, "", ErrEmailTaken
	}

	// Check username uniqueness
	if _, err := s.userRepo.GetByUsername(ctx, req.Username); err == nil {
		return nil, "", ErrUsernameTaken
	}

	// Hash password
	passwordHash, err := hash.HashPassword(req.Password, s.cfg.BcryptCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	// Create user
	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         model.RoleUser,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	// Generate tokens and session
	accessToken, refreshToken, err := s.createSession(ctx, user)
	if err != nil {
		return nil, "", err
	}

	return &dto.AuthResponse{
		User:        dto.NewUserResponse(user),
		AccessToken: accessToken,
	}, refreshToken, nil
}

func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, string, error) {
	user, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if err := hash.CheckPassword(req.Password, user.PasswordHash); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	accessToken, refreshToken, err := s.createSession(ctx, user)
	if err != nil {
		return nil, "", err
	}

	return &dto.AuthResponse{
		User:        dto.NewUserResponse(user),
		AccessToken: accessToken,
	}, refreshToken, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*dto.TokenResponse, string, error) {
	tokenHash := hash.SHA256(refreshToken)

	session, err := s.sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, "", ErrInvalidRefreshToken
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessionRepo.DeleteByID(ctx, session.ID)
		return nil, "", ErrInvalidRefreshToken
	}

	// Load user for fresh claims
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("get user: %w", err)
	}

	// Delete old session (rotation)
	_ = s.sessionRepo.DeleteByID(ctx, session.ID)
	s.rdb.Del(ctx, fmt.Sprintf("session:%s", session.UserID.String()))

	// Create new session
	accessToken, newRefreshToken, err := s.createSession(ctx, user)
	if err != nil {
		return nil, "", err
	}

	return &dto.TokenResponse{AccessToken: accessToken}, newRefreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	_ = s.sessionRepo.DeleteByUserID(ctx, userID)
	s.rdb.Del(ctx, fmt.Sprintf("session:%s", userID.String()))
	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Always succeed — prevent email enumeration
		return nil
	}

	// Invalidate existing tokens
	_ = s.resetRepo.InvalidateForUser(ctx, user.ID)

	// Generate token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	pr := &model.PasswordReset{
		UserID:    user.ID,
		TokenHash: hash.SHA256(token),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := s.resetRepo.Create(ctx, pr); err != nil {
		return fmt.Errorf("create reset: %w", err)
	}

	// Log reset URL (email out of scope for v1)
	frontendURL := s.cfg.FrontendURL
	s.log.Info().
		Str("user_id", user.ID.String()).
		Str("reset_url", fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)).
		Msg("password reset token generated")

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error {
	tokenHash := hash.SHA256(req.Token)

	pr, err := s.resetRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return ErrInvalidResetToken
	}

	if pr.UsedAt != nil {
		return ErrInvalidResetToken
	}

	if time.Now().After(pr.ExpiresAt) {
		return ErrInvalidResetToken
	}

	// Hash new password
	passwordHash, err := hash.HashPassword(req.NewPassword, s.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update user password
	user, err := s.userRepo.GetByID(ctx, pr.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	user.PasswordHash = passwordHash
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	// Mark token as used
	_ = s.resetRepo.MarkUsed(ctx, pr.ID)

	// Invalidate all sessions
	_ = s.sessionRepo.DeleteByUserID(ctx, pr.UserID)
	s.rdb.Del(ctx, fmt.Sprintf("session:%s", pr.UserID.String()))

	return nil
}

func (s *AuthService) createSession(ctx context.Context, user *model.User) (string, string, error) {
	accessToken, err := jwt.GenerateAccessToken(
		s.cfg.JWTSecret,
		user.ID,
		user.Username,
		string(user.Role),
		s.cfg.JWTAccessExpiry,
	)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := jwt.GenerateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	refreshDays := s.cfg.JWTRefreshDays
	if refreshDays == 0 {
		refreshDays = 7
	}

	session := &model.Session{
		UserID:           user.ID,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(time.Duration(refreshDays) * 24 * time.Hour),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", "", fmt.Errorf("create session: %w", err)
	}

	// Cache session in Redis
	s.rdb.Set(ctx,
		fmt.Sprintf("session:%s", user.ID.String()),
		session.ID.String(),
		time.Duration(refreshDays)*24*time.Hour,
	)

	return accessToken, refreshToken, nil
}
