package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type RateLimitConfig struct {
	Prefix     string
	MaxReqs    int
	WindowSecs int
	KeyFunc    func(c echo.Context) string
}

func IPKeyFunc(c echo.Context) string {
	return c.RealIP()
}

func RateLimit(rdb *redis.Client, cfg RateLimitConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := fmt.Sprintf("rate:%s:%s", cfg.Prefix, cfg.KeyFunc(c))
			window := time.Duration(cfg.WindowSecs) * time.Second

			ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
			defer cancel()

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// On Redis error, allow the request through
				return next(c)
			}

			if count == 1 {
				rdb.Expire(ctx, key, window)
			}

			if count > int64(cfg.MaxReqs) {
				c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", cfg.WindowSecs))
				return response.Error(c, http.StatusTooManyRequests, "RATE_LIMITED",
					fmt.Sprintf("Too many requests. Please try again in %d seconds.", cfg.WindowSecs))
			}

			return next(c)
		}
	}
}

func SignupRateLimit(rdb *redis.Client) echo.MiddlewareFunc {
	return RateLimit(rdb, RateLimitConfig{
		Prefix:     "signup",
		MaxReqs:    5,
		WindowSecs: 60,
		KeyFunc:    IPKeyFunc,
	})
}

func LoginRateLimit(rdb *redis.Client) echo.MiddlewareFunc {
	return RateLimit(rdb, RateLimitConfig{
		Prefix:     "login",
		MaxReqs:    5,
		WindowSecs: 60,
		KeyFunc:    IPKeyFunc,
	})
}

func ForgotPasswordRateLimit(rdb *redis.Client) echo.MiddlewareFunc {
	return RateLimit(rdb, RateLimitConfig{
		Prefix:     "forgot",
		MaxReqs:    3,
		WindowSecs: 60,
		KeyFunc:    IPKeyFunc,
	})
}

func BanReviewRateLimit(rdb *redis.Client) echo.MiddlewareFunc {
	return RateLimit(rdb, RateLimitConfig{
		Prefix:     "ban-review",
		MaxReqs:    3,
		WindowSecs: 300,
		KeyFunc:    IPKeyFunc,
	})
}

func RefreshRateLimit(rdb *redis.Client) echo.MiddlewareFunc {
	return RateLimit(rdb, RateLimitConfig{
		Prefix:     "refresh",
		MaxReqs:    10,
		WindowSecs: 60,
		KeyFunc:    IPKeyFunc,
	})
}
