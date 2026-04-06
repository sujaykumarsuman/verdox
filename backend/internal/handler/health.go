package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type HealthHandler struct {
	db  *sqlx.DB
	rdb *redis.Client
}

func NewHealthHandler(db *sqlx.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

func (h *HealthHandler) Liveness(c echo.Context) error {
	return response.Success(c, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Readiness(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		return response.Error(c, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "Database is not reachable")
	}

	if err := h.rdb.Ping(ctx).Err(); err != nil {
		return response.Error(c, http.StatusServiceUnavailable, "REDIS_UNAVAILABLE", "Redis is not reachable")
	}

	return response.Success(c, http.StatusOK, map[string]string{"status": "ready"})
}
