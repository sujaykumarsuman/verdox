package validator

import (
	"net/http"
	"regexp"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type CustomValidator struct {
	validator *validator.Validate
}

func New() *CustomValidator {
	v := validator.New()

	_ = v.RegisterValidation("strong_password", strongPassword)
	_ = v.RegisterValidation("github_url", githubURL)
	_ = v.RegisterValidation("uuid", validUUID)

	return &CustomValidator{validator: v}
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, formatErrors(err))
	}
	return nil
}

func formatErrors(err error) map[string]string {
	errs := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			errs[fe.Field()] = msgForTag(fe)
		}
	}
	return errs
}

func msgForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Must be at least " + fe.Param() + " characters"
	case "max":
		return "Must be at most " + fe.Param() + " characters"
	case "strong_password":
		return "Must be at least 8 characters with 1 uppercase, 1 lowercase, and 1 digit"
	case "github_url":
		return "Must be a valid GitHub repository URL"
	case "uuid":
		return "Must be a valid UUID"
	default:
		return "Invalid value"
	}
}

func strongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

var githubURLRegex = regexp.MustCompile(`^https?://github\.com/[\w.-]+/[\w.-]+$`)

func githubURL(fl validator.FieldLevel) bool {
	return githubURLRegex.MatchString(fl.Field().String())
}

func validUUID(fl validator.FieldLevel) bool {
	_, err := uuid.Parse(fl.Field().String())
	return err == nil
}

func BindAndValidate(c echo.Context, dto interface{}) error {
	if err := c.Bind(dto); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if err := c.Validate(dto); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			if details, ok := he.Message.(map[string]string); ok {
				return response.ValidationError(c, details)
			}
		}
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	return nil
}
