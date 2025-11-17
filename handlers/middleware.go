package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	dbpkg "sudocrypt25/db"
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
			email, err := dbpkg.Get(dbConn, "sessions", sid)
			if err != nil || email == "" {
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
			ctx := context.WithValue(r.Context(), userContextKey, map[string]interface{}{"session_id": sid, "email": email})
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
	start, err := time.Parse(time.RFC3339, ds)
	if err != nil {
		return true
	}
	de := os.Getenv("TIMEGATE_END")
	var end time.Time
	var endSet bool
	if de != "" {
		if e, err2 := time.Parse(time.RFC3339, de); err2 == nil {
			end = e
			endSet = true
		}
	}
	now := time.Now()
	if now.Before(start) {
		return false
	}
	if endSet && now.After(end) {
		return false
	}
	return true
}

func EventPhase() int {
	ds := os.Getenv("TIMEGATE_START")
	if ds == "" {
		ds = "2025-11-07T09:00:00+05:30"
	}
	start, err := time.Parse(time.RFC3339, ds)
	if err != nil {
		return 0
	}
	de := os.Getenv("TIMEGATE_END")
	var end time.Time
	var endSet bool
	if de != "" {
		if e, err2 := time.Parse(time.RFC3339, de); err2 == nil {
			end = e
			endSet = true
		}
	}
	now := time.Now()
	if now.Before(start) {
		return -1
	}
	if endSet && now.After(end) {
		return 1
	}
	return 0
}

func DuringEvent() bool {
	return EventPhase() == 0
}
