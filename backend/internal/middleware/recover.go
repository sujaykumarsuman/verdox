package middleware

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

func Recover(log zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					buf := make([]byte, 2048)
					n := runtime.Stack(buf, false)

					reqID := c.Response().Header().Get(echo.HeaderXRequestID)
					log.Error().
						Str("request_id", reqID).
						Str("panic", fmt.Sprintf("%v", r)).
						Str("stack", string(buf[:n])).
						Msg("panic recovered")

					_ = response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
				}
			}()
			return next(c)
		}
	}
}
