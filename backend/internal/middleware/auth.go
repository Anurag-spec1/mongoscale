package middleware

import (
	"context"
	"net/http"
	"strings"

	"mongoscale/internal/models"
	"mongoscale/internal/services"
	"mongoscale/internal/utils"
)

type identityKey struct{}

type TokenParser interface {
	Parse(token string) (services.Identity, error)
}

func IdentityFrom(ctx context.Context) (services.Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(services.Identity)
	return identity, ok
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if len(header) > 7 && strings.EqualFold(header[:7], "Bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

func Authenticate(parser TokenParser) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				utils.WriteError(w, utils.Unauthorized("authentication required"))
				return
			}
			identity, err := parser.Parse(token)
			if err != nil {
				utils.WriteError(w, utils.Classify(err))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, identity)))
		})
	}
}

func OptionalAuthenticate(parser TokenParser) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := bearerToken(r); token != "" {
				if identity, err := parser.Parse(token); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), identityKey{}, identity))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFrom(r.Context())
			if !ok || identity.Role != models.RoleAdmin {
				utils.WriteError(w, utils.Forbidden("administrator access required"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
