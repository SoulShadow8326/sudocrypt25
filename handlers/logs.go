package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	dbpkg "sudocrypt25/db"
)

type LogEntry struct {
	ID        int    `json:"id"`
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Event     string `json:"event"`
	Data      string `json:"data"`
	CreatedAt int64  `json:"created_at"`
}

func LogsHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		requester, err := GetEmailFromRequest(dbConn, r)
		if err != nil || requester == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		requester = strings.ToLower(strings.TrimSpace(requester))

		q := r.URL.Query()
		user := strings.ToLower(strings.TrimSpace(q.Get("user")))
		ns := strings.TrimSpace(q.Get("namespace"))

		isAdmin := admins != nil && admins.IsAdmin(requester)
		if user != "" && user != requester && !isAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		all, err := dbpkg.GetAll(dbConn, "logs")
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		out := []LogEntry{}
		for _, v := range all {
			var e LogEntry
			if err := json.Unmarshal([]byte(v), &e); err == nil {
				if user != "" && strings.ToLower(e.Key) != user {
					continue
				}
				if ns != "" && e.Namespace != ns {
					continue
				}
				out = append(out, e)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	}
}
