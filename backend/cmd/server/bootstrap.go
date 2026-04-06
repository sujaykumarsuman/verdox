package main

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/pkg/hash"
)

func bootstrapRootUser(db *sqlx.DB, cfg *config.Config, log zerolog.Logger) {
	if cfg.RootEmail == "" || cfg.RootPassword == "" {
		log.Warn().Msg("ROOT_EMAIL or ROOT_PASSWORD not set, skipping root user bootstrap")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userRepo := repository.NewUserRepository(db)

	// Check if root user already exists
	if _, err := userRepo.GetByEmail(ctx, cfg.RootEmail); err == nil {
		log.Info().Str("email", cfg.RootEmail).Msg("root user already exists")
		return
	}

	// Hash password
	passwordHash, err := hash.HashPassword(cfg.RootPassword, cfg.BcryptCost)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash root password")
		return
	}

	user := &model.User{
		Username:     "root",
		Email:        cfg.RootEmail,
		PasswordHash: passwordHash,
		Role:         model.RoleRoot,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		log.Error().Err(err).Msg("failed to create root user")
		return
	}

	log.Info().Str("email", cfg.RootEmail).Msg("root user created successfully")
}
