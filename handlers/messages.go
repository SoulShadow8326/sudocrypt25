package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	dbpkg "sudocrypt25/db"
)

type Message struct {
	ID        int    `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	LevelID   string `json:"level_id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
	Read      int64  `json:"read"`
}

func SendMessageHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]string
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			defer r.Body.Close()
			json.NewDecoder(r.Body).Decode(&payload)
		} else {
			r.ParseForm()
			payload = map[string]string{
				"to":      r.FormValue("to"),
				"type":    r.FormValue("type"),
				"content": r.FormValue("content"),
				"level":   r.FormValue("level"),
			}
		}

		emailC, err := r.Cookie("email")
		if err != nil || emailC.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		from := emailC.Value
		isAdmin := admins != nil && admins.IsAdmin(from)
		displayFrom := from
		to := strings.TrimSpace(payload["to"])
		toLower := strings.ToLower(to)
		isSendToAdminInbox := toLower == "admin@sudocrypt.com"
		if isAdmin && !isSendToAdminInbox {
			displayFrom = "admin@sudocrypt.com"
		}
		if to == "" {
			http.Error(w, "missing to", http.StatusBadRequest)
			return
		}
		mtype := strings.TrimSpace(payload["type"])
		if mtype == "" {
			mtype = "lead"
		}
		content := strings.TrimSpace(payload["content"])
		level := strings.TrimSpace(payload["level"])
		now := time.Now().Unix()
		_ = now

		finalTo := to
		if isSendToAdminInbox {
			finalTo = "admin@sudocrypt.com"
		}
		val := strings.Join([]string{displayFrom, finalTo, level, mtype, content}, "|")
		fmt.Printf("[messages] %s -> %s | level=%s | type=%s | content=%q\n", displayFrom, finalTo, level, mtype, content)
		if err := dbpkg.Set(dbConn, "messages", finalTo, val); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func ListMessagesHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
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
		requesterRaw := emailC.Value
		requester := strings.ToLower(requesterRaw)

		all, err := dbpkg.GetAll(dbConn, "messages")
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		q := r.URL.Query()
		userParam := strings.TrimSpace(q.Get("user"))
		adminMode := false
		if admins != nil && admins.IsAdmin(requesterRaw) && strings.EqualFold(q.Get("mode"), "admin") {
			adminMode = true
		}
		user := requester
		if userParam != "" && adminMode {
			user = userParam
		}
		msgs := make([]Message, 0, len(all))
		requesterIsAdmin := adminMode
		adminInbox := "admin@sudocrypt.com"
		for _, v := range all {
			var m Message
			if err := json.Unmarshal([]byte(v), &m); err == nil {
				// compare
				if requesterIsAdmin && userParam == "" {
					if strings.EqualFold(m.To, requesterRaw) || strings.EqualFold(m.From, requesterRaw) ||
						strings.EqualFold(m.To, adminInbox) || strings.EqualFold(m.From, adminInbox) {
						msgs = append(msgs, m)
					}
				} else {
					if strings.EqualFold(m.To, user) || strings.EqualFold(m.From, user) {
						msgs = append(msgs, m)
					}
				}
			}
		}
		sort.Slice(msgs, func(i, j int) bool { return msgs[i].CreatedAt < msgs[j].CreatedAt })
		h := sha256.New()
		for _, m := range msgs {
			h.Write([]byte(strconv.Itoa(m.ID)))
			h.Write([]byte(m.From))
			h.Write([]byte(m.To))
			h.Write([]byte(strconv.FormatInt(m.CreatedAt, 10)))
		}
		checksum := hex.EncodeToString(h.Sum(nil))
		clientChecksum := r.URL.Query().Get("checksum")
		if clientChecksum != "" && clientChecksum == checksum {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		out := make([]map[string]interface{}, 0, len(msgs))
		for _, m := range msgs {
			isMe := strings.EqualFold(m.From, requesterRaw)
			fromLabel := ""
			if isMe {
				fromLabel = "You"
			} else {
				if requesterIsAdmin {
					fromLabel = m.From
				} else {
					fromLabel = "admin@sudocrypt.com"
				}
			}
			displayFrom := m.From
			if (!isMe && !requesterIsAdmin) || strings.EqualFold(displayFrom, "ADMIN@SUDOCRYPT.COM") {
				displayFrom = "admin@sudocrypt.com"
			}

			out = append(out, map[string]interface{}{
				"id":         m.ID,
				"from":       displayFrom,
				"to":         m.To,
				"level_id":   m.LevelID,
				"type":       m.Type,
				"content":    m.Content,
				"created_at": m.CreatedAt,
				"read":       m.Read,
				"is_me":      isMe,
				"from_label": fromLabel,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"checksum": checksum, "messages": out})
	}
}
