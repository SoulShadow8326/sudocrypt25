package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"sudocrypt25/db"
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
			v, err := db.Get(dbConn, "sessions", sid)
			if err != nil || v == "" {
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

			var user map[string]interface{}
			_ = json.Unmarshal([]byte(v), &user)
			ctx := context.WithValue(r.Context(), userContextKey, user)
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
