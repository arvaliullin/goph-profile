package middleware

import (
	"context"
	"net/http"
)

type ctxKey string

const userIDKey ctxKey = "userID"

// UserIDFromHeader читает X-User-ID в контекст.
func UserIDFromHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := r.Header.Get("X-User-ID")
		if uid != "" {
			r = r.WithContext(context.WithValue(r.Context(), userIDKey, uid))
		}
		next.ServeHTTP(w, r)
	})
}

// UserID возвращает идентификатор пользователя из заголовка, если задан.
func UserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok && v != ""
}
