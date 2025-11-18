package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	dbpkg "sudocrypt25/db"
)

func AttemptLog(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		email, err := GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method == "POST" {
			var req struct {
				Log   string `json:"logs"`
				Typpe string `json:"type"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			acctRaw, err := dbpkg.Get(dbConn, "attempt_logs", email)
			if err != nil {
				acctRaw = `{"email":"` + email + `","logs":""}`
			}

			var acct map[string]interface{}
			if err := json.Unmarshal([]byte(acctRaw), &acct); err != nil {
				http.Error(w, "Error loading account", http.StatusInternalServerError)
				return
			}

			acct["logs"] = acct["logs"].(string) + "\n" + req.Log + "+" + req.Typpe + "+" + strconv.FormatInt(time.Now().Unix(), 10)

			acctBytes, _ := json.Marshal(acct)
			if err := dbpkg.Set(dbConn, "attempt_logs", email, string(acctBytes)); err != nil {
				http.Error(w, "Failed to update", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Bio updated successfully",
			})
		}
		if r.Method == "GET" {
			target := strings.TrimSpace(r.URL.Query().Get("email"))
			if target != "" {
				if admins == nil || !admins.IsAdmin(email) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				email = target
			}
			acctRaw, err := dbpkg.Get(dbConn, "attempt_logs", email)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
					"message": "Bio updated successfully",
					"data":    "",
				})
				return
			}

			var acct map[string]interface{}
			if err := json.Unmarshal([]byte(acctRaw), &acct); err != nil {
				http.Error(w, "Error loading account", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Bio updated successfully",
				"data":    acct["logs"].(string),
			})
		}
	}
}
