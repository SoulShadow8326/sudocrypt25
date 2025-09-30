package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"
)

type contextKey string

const userContextKey contextKey = "user"

func AuthMiddleware(dbConn *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("session_id")
			if err != nil || c.Value == "" {
				accept := r.Header.Get("Accept")
				if strings.Contains(accept, "application/json") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"unauthenticated"}`))
					return
				}
				http.Redirect(w, r, "/auth", http.StatusFound)
				return
			}

			sid := c.Value
			if sid == "" {
				accept := r.Header.Get("Accept")
				if strings.Contains(accept, "application/json") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"invalid_session"}`))
					return
				}
				http.Redirect(w, r, "/auth", http.StatusFound)
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, map[string]interface{}{"session_id": sid})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) (map[string]interface{}, bool) {
	u := ctx.Value(userContextKey)
	if u == nil {
		return nil, false
	}
	m, ok := u.(map[string]interface{})
	return m, ok
}

func IsTimeGateOpen() bool {
	ds := os.Getenv("TIMEGATE_START")
	if ds == "" {
		ds = "2025-11-07T09:00:00+05:30"
	}
	t, err := time.Parse(time.RFC3339, ds)
	if err != nil {
		return true
	}
	now := time.Now()
	return !now.Before(t)
}
