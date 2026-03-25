package handler

import (
	"errors"
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type errorResponse struct {
	Error  string            `json:"error"`
	Code   string            `json:"code"`
	Fields map[string]string `json:"fields,omitempty"`
}

func writeDomainError(c *gin.Context, err error) {
	status, code := mapDomainError(err)
	message := messageByCode(code)
	c.JSON(status, errorResponse{Error: message, Code: code})
}

func writeValidationError(c *gin.Context, err error) {
	fields := make(map[string]string)

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fieldErr := range validationErrors {
			fields[fieldErr.Field()] = validationMessage(fieldErr.Tag(), fieldErr.Param())
		}
	}

	if len(fields) == 0 {
		fields["request"] = "invalid request payload"
	}

	c.JSON(http.StatusBadRequest, errorResponse{
		Error:  "validation failed",
		Code:   "ERR_VALIDATION",
		Fields: fields,
	})
}

func writeInternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, errorResponse{Error: messageByCode("ERR_INTERNAL"), Code: "ERR_INTERNAL"})
}

func mapDomainError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, "ERR_INVALID_INPUT"
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "ERR_UNAUTHORIZED"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "ERR_FORBIDDEN"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "ERR_NOT_FOUND"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "ERR_CONFLICT"
	default:
		return http.StatusInternalServerError, "ERR_INTERNAL"
	}
}

func messageByCode(code string) string {
	switch code {
	case "ERR_INVALID_INPUT":
		return "invalid input"
	case "ERR_UNAUTHORIZED":
		return "unauthorized"
	case "ERR_FORBIDDEN":
		return "forbidden"
	case "ERR_NOT_FOUND":
		return "not found"
	case "ERR_CONFLICT":
		return "conflict"
	default:
		return "internal error"
	}
}

func validationMessage(tag, param string) string {
	switch tag {
	case "required":
		return "is required"
	case "max":
		return "must be at most " + param + " characters"
	default:
		return "is invalid"
	}
}
