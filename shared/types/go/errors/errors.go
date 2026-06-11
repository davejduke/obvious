// Package errors provides common error types for AIAUDITOR services.
package errors

import (
	"fmt"
	"net/http"
)

// AppError is a structured application error with HTTP status mapping.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error constructors
func NotFound(resource, id string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s with id %s not found", resource, id),
		Status:  http.StatusNotFound,
	}
}

func Unauthorized(msg string) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: msg,
		Status:  http.StatusUnauthorized,
	}
}

func Forbidden(msg string) *AppError {
	return &AppError{
		Code:    "FORBIDDEN",
		Message: msg,
		Status:  http.StatusForbidden,
	}
}

func BadRequest(msg string) *AppError {
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: msg,
		Status:  http.StatusBadRequest,
	}
}

func InternalError(msg string) *AppError {
	return &AppError{
		Code:    "INTERNAL_ERROR",
		Message: msg,
		Status:  http.StatusInternalServerError,
	}
}

func Conflict(msg string) *AppError {
	return &AppError{
		Code:    "CONFLICT",
		Message: msg,
		Status:  http.StatusConflict,
	}
}

