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

		from, err := GetEmailFromRequest(dbConn, r)
		if err != nil || from == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
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
		if !isAdmin {
			phase := EventPhase()
			if phase == -1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "The event has not commenced yet", "when": "before"})
				return
			}
			if phase == 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "The event has concluded", "when": "after"})
				return
			}
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
		requesterRaw, err := GetEmailFromRequest(dbConn, r)
		if err != nil || requesterRaw == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
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
		sort.Slice(msgs, func(i, j int) bool {
			if msgs[i].CreatedAt == msgs[j].CreatedAt {
				return msgs[i].ID < msgs[j].ID
			}
			return msgs[i].CreatedAt < msgs[j].CreatedAt
		})
		h := sha256.New()
		for _, m := range msgs {
			h.Write([]byte(strconv.Itoa(m.ID)))
			h.Write([]byte(m.From))
			h.Write([]byte(m.To))
			h.Write([]byte(strconv.FormatInt(m.CreatedAt, 10)))
		}

		levelSet := map[string]struct{}{}
		for _, m := range msgs {
			if m.LevelID != "" {
				levelSet[m.LevelID] = struct{}{}
			}
		}

		typ := r.URL.Query().Get("type")
		leadsEnabledForType := true
		if typ != "" {
			acctRaw, err := dbpkg.Get(dbConn, "accounts", requesterRaw)
			curr := 0
			if err == nil {
				var acct map[string]interface{}
				if json.Unmarshal([]byte(acctRaw), &acct) == nil {
					if lm, ok := acct["levels"].(map[string]interface{}); ok {
						if v, ok := lm[typ].(float64); ok {
							curr = int(v)
						}
					}
				}
			}
			levelID := fmt.Sprintf("%s-%d", typ, curr)
			if levelID != "" {
				levelSet[levelID] = struct{}{}
			}
			if levelID != "" {
				if lvlObj, err := GetLevel(dbConn, levelID); err == nil && lvlObj != nil {
					leadsEnabledForType = lvlObj.LeadsEnabled
					if leadsEnabledForType {
						h.Write([]byte("leads_enabled:true"))
					} else {
						h.Write([]byte("leads_enabled:false"))
					}
				}
			}
		}
		hintsList := make([]map[string]interface{}, 0)
		for lvl := range levelSet {
			hintsStr, err := dbpkg.Get(dbConn, "hints", lvl)
			if err == nil {
				hintsMap := map[string]string{}
				json.Unmarshal([]byte(hintsStr), &hintsMap)
				for k, v := range hintsMap {
					var he map[string]interface{}
					if json.Unmarshal([]byte(v), &he) == nil {
						hintsList = append(hintsList, he)
						if c, ok := he["content"].(string); ok {
							h.Write([]byte(c))
						}
						if t, ok := he["time"]; ok {
							switch tv := t.(type) {
							case float64:
								h.Write([]byte(strconv.FormatInt(int64(tv), 10)))
							case string:
								h.Write([]byte(tv))
							}
						}
						h.Write([]byte(k))
					}
				}
			}
		}
		checksum := hex.EncodeToString(h.Sum(nil))
		clientChecksum := r.URL.Query().Get("checksum")
		annItems, _ := dbpkg.GetAll(dbConn, "announcements")
		a := sha256.New()
		keys := make([]string, 0, len(annItems))
		for k := range annItems {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := annItems[k]
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(v), &m); err == nil {
				if c, ok := m["content"].(string); ok {
					a.Write([]byte(c))
				}
				if t, ok := m["time"]; ok {
					switch tv := t.(type) {
					case float64:
						a.Write([]byte(strconv.FormatInt(int64(tv), 10)))
					case string:
						a.Write([]byte(tv))
					case int64:
						a.Write([]byte(strconv.FormatInt(tv, 10)))
					}
				}
				a.Write([]byte(k))
			} else {
				a.Write([]byte(v))
			}
		}
		annChecksum := hex.EncodeToString(a.Sum(nil))
		clientAnnChecksum := r.URL.Query().Get("announcements_checksum")

		if clientChecksum != "" && clientChecksum == checksum {
			if clientAnnChecksum == "" || clientAnnChecksum == annChecksum {
				w.WriteHeader(http.StatusNotModified)
				return
			}
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

			entry := map[string]interface{}{
				"id":         m.ID,
				"from":       displayFrom,
				"to":         m.To,
				"level_id":   m.LevelID,
				"level":      m.LevelID,
				"lvl":        m.LevelID,
				"LevelID":    m.LevelID,
				"Level":      m.LevelID,
				"type":       m.Type,
				"content":    m.Content,
				"from_name":  "",
				"to_name":    "",
				"created_at": m.CreatedAt,
				"read":       m.Read,
				"is_me":      isMe,
				"from_label": fromLabel,
			}

			fromEmail := strings.ToLower(displayFrom)
			if acctRaw, err := dbpkg.Get(dbConn, "accounts", fromEmail); err == nil {
				var acct map[string]interface{}
				if json.Unmarshal([]byte(acctRaw), &acct) == nil {
					if n, ok := acct["name"].(string); ok && n != "" {
						entry["from_name"] = n
					}
				}
			}
			toEmail := strings.ToLower(m.To)
			if acctRaw, err := dbpkg.Get(dbConn, "accounts", toEmail); err == nil {
				var acct map[string]interface{}
				if json.Unmarshal([]byte(acctRaw), &acct) == nil {
					if n, ok := acct["name"].(string); ok && n != "" {
						entry["to_name"] = n
					}
				}
			}

			out = append(out, entry)
		}
		aiLeadsEnabled := true
		if v, err := dbpkg.Get(dbConn, "settings", "ai_leads"); err == nil {
			s := strings.TrimSpace(strings.ToLower(v))
			if s == "0" || s == "false" || s == "off" {
				aiLeadsEnabled = false
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"checksum": checksum, "announcements_checksum": annChecksum, "messages": out, "hints": hintsList, "leads_enabled": leadsEnabledForType, "ai_leads": aiLeadsEnabled})
	}
}

func MarkMessagesReadHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		requester, err := GetEmailFromRequest(dbConn, r)
		if err != nil || requester == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		if admins == nil || !admins.IsAdmin(requester) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		var payload map[string]interface{}
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			defer r.Body.Close()
			json.NewDecoder(r.Body).Decode(&payload)
		} else {
			r.ParseForm()
			payload = map[string]interface{}{"email": r.FormValue("email"), "upto_id": r.FormValue("upto_id")}
		}
		emailRaw, _ := payload["email"].(string)
		email := strings.ToLower(strings.TrimSpace(emailRaw))
		if email == "" {
			http.Error(w, "missing email", http.StatusBadRequest)
			return
		}

		uptoVal := int64(0)
		if v, ok := payload["upto_id"]; ok && v != nil {
			switch tv := v.(type) {
			case float64:
				uptoVal = int64(tv)
			case string:
				if s := strings.TrimSpace(tv); s != "" {
					if parsed, err := strconv.ParseInt(s, 10, 64); err == nil {
						uptoVal = parsed
					}
				}
			}
		}

		adminInbox := "admin@sudocrypt.com"
		var res sql.Result
		var execErr error
		if uptoVal > 0 {
			res, execErr = dbConn.Exec(`UPDATE messages SET read = 1 WHERE json_extract(data, '$.from') = ? AND json_extract(data, '$.to') = ? AND id <= ? AND read = 0`, email, adminInbox, uptoVal)
		} else {
			res, execErr = dbConn.Exec(`UPDATE messages SET read = 1 WHERE json_extract(data, '$.from') = ? AND json_extract(data, '$.to') = ? AND read = 0`, email, adminInbox)
		}
		if execErr != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		affected := int64(0)
		if res != nil {
			if n, err := res.RowsAffected(); err == nil {
				affected = n
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "updated": affected})
	}
}
