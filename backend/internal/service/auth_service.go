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
	"encoding/json"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/pkg/hash"
	"github.com/sujaykumarsuman/verdox/backend/pkg/jwt"
)

var (
	ErrEmailTaken             = errors.New("email already in use")
	ErrUsernameTaken          = errors.New("username already in use")
	ErrInvalidCredentials     = errors.New("invalid email/username or password")
	ErrInvalidRefreshToken    = errors.New("invalid or expired refresh token")
	ErrInvalidResetToken      = errors.New("invalid or expired reset token")
	ErrInvalidCurrentPassword = errors.New("current password is incorrect")
	ErrAccountBanned          = errors.New("account has been banned")
	ErrAccountDeactivated     = errors.New("account has been deactivated")
	ErrNotBanned              = errors.New("account is not banned")
	ErrReviewAlreadyPending   = errors.New("a review request is already pending")
	ErrReviewLimitReached     = errors.New("maximum review attempts reached")
)

// BannedUserInfo carries ban details for the login error response.
type BannedUserInfo struct {
	BanReason        string `json:"ban_reason"`
	HasPendingReview bool   `json:"has_pending_review"`
	ReviewsRemaining int    `json:"reviews_remaining"`
}

type AuthService struct {
	userRepo      repository.UserRepository
	sessionRepo   repository.SessionRepository
	resetRepo     repository.PasswordResetRepository
	banReviewRepo repository.BanReviewRepository
	notifService  *NotificationService
	rdb           *redis.Client
	cfg           *config.Config
	log           zerolog.Logger
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	resetRepo repository.PasswordResetRepository,
	banReviewRepo repository.BanReviewRepository,
	rdb *redis.Client,
	cfg *config.Config,
	log zerolog.Logger,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		sessionRepo:   sessionRepo,
		resetRepo:     resetRepo,
		banReviewRepo: banReviewRepo,
		rdb:           rdb,
		cfg:           cfg,
		log:           log,
	}
}

// SetNotificationService sets the notification service for sending admin alerts.
// Called after both services are initialized to avoid circular dependency.
func (s *AuthService) SetNotificationService(ns *NotificationService) {
	s.notifService = ns
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

	if user.IsBanned {
		return nil, "", ErrAccountBanned
	}
	if !user.IsActive {
		return nil, "", ErrAccountDeactivated
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

	// Log reset URL (no email service — password reset links are logged to stdout)
	frontendURL := s.cfg.FrontendURL
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)
	s.log.Info().
		Str("user_id", user.ID.String()).
		Str("reset_url", resetURL).
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

func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *dto.UpdateProfileRequest) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if req.Username != nil && *req.Username != user.Username {
		if existing, err := s.userRepo.GetByUsername(ctx, *req.Username); err == nil && existing.ID != userID {
			return nil, ErrUsernameTaken
		}
		user.Username = *req.Username
	}

	if req.Email != nil && *req.Email != user.Email {
		if existing, err := s.userRepo.GetByEmail(ctx, *req.Email); err == nil && existing.ID != userID {
			return nil, ErrEmailTaken
		}
		user.Email = *req.Email
	}

	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, req *dto.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if err := hash.CheckPassword(req.CurrentPassword, user.PasswordHash); err != nil {
		return ErrInvalidCurrentPassword
	}

	passwordHash, err := hash.HashPassword(req.NewPassword, s.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = passwordHash
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	// Invalidate all sessions
	_ = s.sessionRepo.DeleteByUserID(ctx, userID)
	s.rdb.Del(ctx, fmt.Sprintf("session:%s", userID.String()))

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

func (s *AuthService) GetBannedUserInfo(ctx context.Context, login string) (*BannedUserInfo, error) {
	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}

	reason := ""
	if user.BanReason != nil {
		reason = *user.BanReason
	}

	hasPending, _ := s.banReviewRepo.HasPendingForUser(ctx, user.ID)
	totalReviews, _ := s.banReviewRepo.CountByUser(ctx, user.ID)
	remaining := 3 - totalReviews
	if remaining < 0 {
		remaining = 0
	}

	return &BannedUserInfo{
		BanReason:        reason,
		HasPendingReview: hasPending,
		ReviewsRemaining: remaining,
	}, nil
}

func (s *AuthService) RequestBanReview(ctx context.Context, req *dto.BanReviewRequest) error {
	// Authenticate the banned user
	user, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidCredentials
		}
		return err
	}

	if err := hash.CheckPassword(req.Password, user.PasswordHash); err != nil {
		return ErrInvalidCredentials
	}

	if !user.IsBanned {
		return ErrNotBanned
	}

	// Check for existing pending review
	hasPending, err := s.banReviewRepo.HasPendingForUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("check pending review: %w", err)
	}
	if hasPending {
		return ErrReviewAlreadyPending
	}

	// Max 3 review attempts per user
	totalReviews, err := s.banReviewRepo.CountByUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("count reviews: %w", err)
	}
	if totalReviews >= 3 {
		return ErrReviewLimitReached
	}

	// Create the review request
	banReason := ""
	if user.BanReason != nil {
		banReason = *user.BanReason
	}

	review := &model.BanReview{
		UserID:        user.ID,
		BanReason:     banReason,
		Clarification: req.Clarification,
	}
	if err := s.banReviewRepo.Create(ctx, review); err != nil {
		return fmt.Errorf("create ban review: %w", err)
	}

	s.log.Info().
		Str("user_id", user.ID.String()).
		Str("review_id", review.ID.String()).
		Msg("ban review request submitted")

	// Notify all admins about the new ban review request
	if s.notifService != nil {
		actionType := "ban_review_decision"
		actionPayloadBytes, _ := json.Marshal(map[string]string{
			"review_id": review.ID.String(),
			"user_id":   user.ID.String(),
			"username":  user.Username,
		})
		actionPayload := json.RawMessage(actionPayloadBytes)
		_ = s.notifService.CreateForAdmins(ctx, &model.Notification{
			Type:          model.NotificationBanReview,
			Subject:       fmt.Sprintf("Ban review request from %s", user.Username),
			Body:          fmt.Sprintf("User %s has requested a review of their ban.\n\nBan reason: %s\n\nClarification: %s", user.Username, banReason, req.Clarification),
			ActionType:    &actionType,
			ActionPayload: &actionPayload,
		})
	}

	return nil
}
