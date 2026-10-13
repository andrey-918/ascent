package api

import (
	"ascent/store"
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func NewLoggerMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc((func(w http.ResponseWriter, r *http.Request) {
			logger.Info("http request", "path", r.Method+" "+r.URL.Path)
			next.ServeHTTP(w, r)
		}))
	}
}

type userCtxKey struct{}

func ContextWithUser(ctx context.Context, user *store.User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, user)
}

func NewAuthMiddleware(jwtManger *JWTManager, userStore *store.UserStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/auth") {
				next.ServeHTTP(w, r)
				return
			}
			authHeader := r.Header.Get("Authorization")
			var token string
			if parts := strings.Split(authHeader, "Bearer "); len(parts) == 2 {
				token = parts[1]
			}
			if token == "" {
				writeUnauthorized(w)
				return
			}

			parsedToken, err := jwtManger.Parse(token)
			if err != nil {
				slog.Error("failed to parse token", "error", err)
				writeUnauthorized(w)
				return
			}

			if !jwtManger.IsAccessToken(parsedToken) {
				writeUnauthorized(w)
				return
			}

			userIdStr, err := parsedToken.Claims.GetSubject()
			if err != nil {
				slog.Error("failed to extract subject claim from token", "error", err)
				writeUnauthorized(w)
				return
			}

			userId, err := uuid.Parse(userIdStr)
			if err != nil {
				slog.Error("token subject is not valie uuid", "error", err)
				writeUnauthorized(w)
				return
			}

			user, err := userStore.ById(r.Context(), userId)
			if err != nil {
				slog.Error("failed to get user by id", "error", err)
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithUser(r.Context(), user)))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	if err := encode(ApiResponse[struct{}]{
		Message: http.StatusText(http.StatusUnauthorized),
	}, http.StatusUnauthorized, w); err != nil {
		slog.Error("failed to encode unauthorized response", "error", err)
	}
}
