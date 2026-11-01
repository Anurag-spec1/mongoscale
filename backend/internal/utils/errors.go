package utils

import (
	"context"
	"errors"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
)

type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func BadRequest(message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "bad_request", Message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "forbidden", Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "conflict", Message: message}
}

func TooManyRequests(message string) *AppError {
	return &AppError{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: message}
}

func Unavailable(message string) *AppError {
	return &AppError{Status: http.StatusServiceUnavailable, Code: "unavailable", Message: message}
}

func Classify(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	if errors.Is(err, context.DeadlineExceeded) || mongo.IsTimeout(err) {
		return &AppError{Status: http.StatusGatewayTimeout, Code: "timeout", Message: "the request timed out before the database could answer"}
	}
	if errors.Is(err, context.Canceled) {
		return &AppError{Status: http.StatusServiceUnavailable, Code: "canceled", Message: "the request was canceled"}
	}
	if mongo.IsNetworkError(err) {
		return &AppError{Status: http.StatusServiceUnavailable, Code: "database_unavailable", Message: "the database is unreachable"}
	}
	return &AppError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "an unexpected error occurred"}
}
