package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"mongoscale/internal/middleware"
	"mongoscale/internal/services"
	"mongoscale/internal/utils"
)

const maxBodyBytes = 1 << 20

type Handler func(w http.ResponseWriter, r *http.Request) error

func wrap(logger *slog.Logger, handler Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err == nil {
			return
		}
		appErr := utils.Classify(err)
		if appErr.Status >= http.StatusInternalServerError {
			logger.Error("request error",
				"method", r.Method,
				"path", r.URL.Path,
				"status", appErr.Status,
				"error", err.Error())
		}
		utils.WriteError(w, appErr)
	})
}

func decodeBody(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return utils.BadRequest("request body is required")
		}
		return utils.BadRequest("request body is not valid JSON for this endpoint")
	}
	return nil
}

func decodeOptionalBody(w http.ResponseWriter, r *http.Request, destination any) error {
	if r.ContentLength == 0 {
		return nil
	}
	err := decodeBody(w, r, destination)
	var appErr *utils.AppError
	if errors.As(err, &appErr) && appErr.Message == "request body is required" {
		return nil
	}
	return err
}

func requireIdentity(r *http.Request) (services.Identity, error) {
	identity, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		return services.Identity{}, utils.Unauthorized("authentication required")
	}
	return identity, nil
}
