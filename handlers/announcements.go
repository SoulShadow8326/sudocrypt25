package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"sudocrypt25/db"
)

func AnnouncementsHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		items, err := db.GetAll(dbConn, "announcements")
		if err != nil || len(items) == 0 {
			sample := []map[string]interface{}{{"time": "just now", "text": "Welcome to Sudocrypt 2025"}}
			json.NewEncoder(w).Encode(sample)
			return
		}
		var out []map[string]interface{}
		for k, v := range items {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(v), &m); err == nil {
				text := ""
				if c, ok := m["content"].(string); ok {
					text = c
				}
				t := m["time"]
				out = append(out, map[string]interface{}{"id": k, "text": text, "time": t})
			}
		}
		json.NewEncoder(w).Encode(out)
	}
}

func SetAnnouncementHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		emailC, err := r.Cookie("email")
		if err != nil || emailC.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		email := emailC.Value
		acctRaw, _ := db.Get(dbConn, "accounts", email)
		var acct map[string]interface{}
		if acctRaw != "" {
			json.Unmarshal([]byte(acctRaw), &acct)
		}
		isAdmin := false
		if admins != nil && admins.IsAdmin(email) {
			isAdmin = true
		}
		if adm, _ := acct["admin"].(bool); adm {
			isAdmin = true
		}
		if !isAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		id := r.FormValue("id")
		content := r.FormValue("content")
		t := r.FormValue("time")
		if id == "" || content == "" {
			http.Error(w, "missing id or content", http.StatusBadRequest)
			return
		}
		var timeVal interface{}
		if t == "" {
			timeVal = time.Now().Unix()
		} else {
			if n, err := strconv.ParseInt(t, 10, 64); err == nil {
				timeVal = n
			} else {
				timeVal = t
			}
		}
		val := map[string]interface{}{"content": content, "time": timeVal}
		b, _ := json.Marshal(val)
		if err := db.Set(dbConn, "announcements", id, string(b)); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func DeleteAnnouncementHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		emailC, err := r.Cookie("email")
		if err != nil || emailC.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		email := emailC.Value
		acctRaw, _ := db.Get(dbConn, "accounts", email)
		var acct map[string]interface{}
		if acctRaw != "" {
			json.Unmarshal([]byte(acctRaw), &acct)
		}
		isAdmin := false
		if admins != nil && admins.IsAdmin(email) {
			isAdmin = true
		}
		if adm, _ := acct["admin"].(bool); adm {
			isAdmin = true
		}
		if !isAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		id := r.FormValue("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		if err := db.Delete(dbConn, "announcements", id); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func AdminCreateAnnouncementFormHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/admin", http.StatusFound)
			return
		}
		emailC, err := r.Cookie("email")
		if err != nil || emailC.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/admin", http.StatusFound)
			return
		}
		email := emailC.Value
		acctRaw, _ := db.Get(dbConn, "accounts", email)
		var acct map[string]interface{}
		if acctRaw != "" {
			json.Unmarshal([]byte(acctRaw), &acct)
		}
		isAdmin := false
		if admins != nil && admins.IsAdmin(email) {
			isAdmin = true
		}
		if adm, _ := acct["admin"].(bool); adm {
			isAdmin = true
		}
		if !isAdmin {
			http.Redirect(w, r, "/timegate?toast=1&from=/admin", http.StatusFound)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		id := r.FormValue("id")
		content := r.FormValue("content")
		t := r.FormValue("time")
		if id == "" || content == "" {
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		var timeVal interface{}
		if t == "" {
			timeVal = time.Now().Unix()
		} else {
			if n, err := strconv.ParseInt(t, 10, 64); err == nil {
				timeVal = n
			} else {
				timeVal = t
			}
		}
		val := map[string]interface{}{"content": content, "time": timeVal}
		b, _ := json.Marshal(val)
		_ = db.Set(dbConn, "announcements", id, string(b))
		http.Redirect(w, r, "/admin", http.StatusFound)
	}
}

func AdminDeleteAnnouncementFormHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/admin", http.StatusFound)
			return
		}
		emailC, err := r.Cookie("email")
		if err != nil || emailC.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/admin", http.StatusFound)
			return
		}
		email := emailC.Value
		acctRaw, _ := db.Get(dbConn, "accounts", email)
		var acct map[string]interface{}
		if acctRaw != "" {
			json.Unmarshal([]byte(acctRaw), &acct)
		}
		isAdmin := false
		if admins != nil && admins.IsAdmin(email) {
			isAdmin = true
		}
		if adm, _ := acct["admin"].(bool); adm {
			isAdmin = true
		}
		if !isAdmin {
			http.Redirect(w, r, "/timegate?toast=1&from=/admin", http.StatusFound)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		id := r.FormValue("id")
		if id != "" {
			_ = db.Delete(dbConn, "announcements", id)
		}
		http.Redirect(w, r, "/admin", http.StatusFound)
	}
}
