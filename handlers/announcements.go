package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"sudocrypt25/db"
)

func AnnouncementsHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		items, err := db.GetAll(dbConn, "announcements")
		if err != nil || len(items) == 0 {
			sample := []map[string]string{{"time": "just now", "text": "Welcome to Sudocrypt 2025"}}
			json.NewEncoder(w).Encode(sample)
			return
		}
		var out []map[string]string
		for _, v := range items {
			var m map[string]string
			if err := json.Unmarshal([]byte(v), &m); err == nil {
				out = append(out, m)
			}
		}
		json.NewEncoder(w).Encode(out)
	}
}
