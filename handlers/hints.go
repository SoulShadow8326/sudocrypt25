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

type HintEntry struct {
	Time    float64 `json:"time"`
	Content string  `json:"content"`
	ID      string  `json:"id"`
	Author  string  `json:"author"`
	Type    string  `json:"type"`
}

func HintsHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		q := r.URL.Query()
		level := q.Get("level")
		if level == "" {
			http.Error(w, "missing level", http.StatusBadRequest)
			return
		}
		rows, err := dbConn.Query(`SELECT hint_id, data FROM hints WHERE level_id = ? ORDER BY created_at ASC`, level)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := make([]HintEntry, 0)
		for rows.Next() {
			var id string
			var data sql.NullString
			if err := rows.Scan(&id, &data); err != nil {
				continue
			}
			if !data.Valid {
				continue
			}
			var he HintEntry
			if err := json.Unmarshal([]byte(data.String), &he); err != nil {
				he = HintEntry{Time: float64(time.Now().Unix()), Content: data.String, ID: id, Author: "Exun Clan", Type: "cryptic"}
			}
			out = append(out, he)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"hints": out})
	}
}

func AdminHintsHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		email, err := GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodPost:
			var payload map[string]string
			if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				defer r.Body.Close()
				json.NewDecoder(r.Body).Decode(&payload)
			} else {
				r.ParseForm()
				payload = map[string]string{"level": r.FormValue("level"), "content": r.FormValue("content"), "type": r.FormValue("type")}
			}
			level := payload["level"]
			content := payload["content"]
			typ := payload["type"]
			if typ == "" {
				typ = "cryptic"
			}
			id := strconv.FormatInt(time.Now().UnixNano(), 10)
			he := HintEntry{Time: float64(time.Now().Unix()), Content: content, ID: id, Author: "Exun Clan", Type: typ}
			b, _ := json.Marshal(he)
			if err := dbpkg.Set(dbConn, "hints", level+"/"+id, string(b)); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": id})
			return
		case http.MethodPut:
			var payload map[string]string
			defer r.Body.Close()
			json.NewDecoder(r.Body).Decode(&payload)
			level := payload["level"]
			id := payload["id"]
			content := payload["content"]
			typ := payload["type"]
			if id == "" || level == "" {
				http.Error(w, "missing id or level", http.StatusBadRequest)
				return
			}
			he := HintEntry{Time: float64(time.Now().Unix()), Content: content, ID: id, Author: "Exun Clan", Type: typ}
			b, _ := json.Marshal(he)
			if err := dbpkg.Set(dbConn, "hints", level+"/"+id, string(b)); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		case http.MethodDelete:
			q := r.URL.Query()
			level := q.Get("level")
			id := q.Get("id")
			if level == "" || id == "" {
				http.Error(w, "missing level or id", http.StatusBadRequest)
				return
			}
			if err := dbpkg.Delete(dbConn, "hints", level+"/"+id); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}
