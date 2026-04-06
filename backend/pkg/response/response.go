package response

import (
	"github.com/labstack/echo/v4"
)

type successEnvelope struct {
	Data interface{} `json:"data"`
}

type errorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func Success(c echo.Context, status int, data interface{}) error {
	return c.JSON(status, successEnvelope{Data: data})
}

func Error(c echo.Context, status int, code string, message string) error {
	return c.JSON(status, errorEnvelope{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}

func ErrorWithData(c echo.Context, status int, code string, message string, data interface{}) error {
	return c.JSON(status, errorEnvelope{
		Error: errorBody{
			Code:    code,
			Message: message,
			Details: data,
		},
	})
}

func ValidationError(c echo.Context, details interface{}) error {
	return c.JSON(400, errorEnvelope{
		Error: errorBody{
			Code:    "VALIDATION_ERROR",
			Message: "Request validation failed",
			Details: details,
		},
	})
}
