package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv  string `mapstructure:"APP_ENV"`
	AppPort string `mapstructure:"APP_PORT"`

	DatabaseURL   string `mapstructure:"DATABASE_URL"`
	DBMaxOpenConn int    `mapstructure:"DB_MAX_OPEN_CONN"`
	DBMaxIdleConn int    `mapstructure:"DB_MAX_IDLE_CONN"`
	DBMaxLifetime int    `mapstructure:"DB_MAX_LIFETIME"`

	RedisURL string `mapstructure:"REDIS_URL"`

	JWTSecret       string        `mapstructure:"JWT_SECRET"`
	JWTAccessExpiry time.Duration `mapstructure:"JWT_ACCESS_EXPIRY"`
	JWTRefreshDays  int           `mapstructure:"JWT_REFRESH_DAYS"`

	BcryptCost int `mapstructure:"BCRYPT_COST"`

	RootEmail    string `mapstructure:"ROOT_EMAIL"`
	RootPassword string `mapstructure:"ROOT_PASSWORD"`

	FrontendURL string `mapstructure:"FRONTEND_URL"`
	CORSOrigins string `mapstructure:"CORS_ORIGINS"`

	LogLevel string `mapstructure:"LOG_LEVEL"`

	RepoBasePath string `mapstructure:"VERDOX_REPO_BASE_PATH"`

	GithubTokenEncryptionKey string `mapstructure:"GITHUB_TOKEN_ENCRYPTION_KEY"`
}

func (c *Config) CORSOriginsList() []string {
	if c.CORSOrigins == "" {
		return []string{c.FrontendURL}
	}
	return strings.Split(c.CORSOrigins, ",")
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("DB_MAX_OPEN_CONN", 25)
	viper.SetDefault("DB_MAX_IDLE_CONN", 5)
	viper.SetDefault("DB_MAX_LIFETIME", 300)
	viper.SetDefault("JWT_ACCESS_EXPIRY", 15*time.Minute)
	viper.SetDefault("JWT_REFRESH_DAYS", 7)
	viper.SetDefault("BCRYPT_COST", 12)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("FRONTEND_URL", "http://localhost:3000")
	viper.SetDefault("VERDOX_REPO_BASE_PATH", "./data/repositories")

	// Explicitly bind all config keys so AutomaticEnv picks up env vars
	for _, key := range []string{
		"APP_ENV", "APP_PORT",
		"DATABASE_URL", "DB_MAX_OPEN_CONN", "DB_MAX_IDLE_CONN", "DB_MAX_LIFETIME",
		"REDIS_URL",
		"JWT_SECRET", "JWT_ACCESS_EXPIRY", "JWT_REFRESH_DAYS",
		"BCRYPT_COST",
		"ROOT_EMAIL", "ROOT_PASSWORD",
		"FRONTEND_URL", "CORS_ORIGINS",
		"LOG_LEVEL",
		"VERDOX_REPO_BASE_PATH",
		"GITHUB_TOKEN_ENCRYPTION_KEY",
	} {
		_ = viper.BindEnv(key)
	}

	// Ignore error if .env file doesn't exist — env vars may be set directly
	_ = viper.ReadInConfig()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
