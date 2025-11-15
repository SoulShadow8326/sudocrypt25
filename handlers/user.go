package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	dbpkg "sudocrypt25/db"
)

func AdminUpdateUserProgressHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		requester, err := GetEmailFromRequest(dbConn, r)
		if err != nil || requester == "" || admins == nil || !admins.IsAdmin(requester) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			email := r.URL.Query().Get("email")
			if email == "" {
				http.Error(w, "missing email", http.StatusBadRequest)
				return
			}
			acctRaw, err := dbpkg.Get(dbConn, "accounts", email)
			var acct map[string]interface{}
			if err == nil {
				json.Unmarshal([]byte(acctRaw), &acct)
			} else {
				acct = map[string]interface{}{"levels": map[string]float64{"cryptic": 0, "ctf": 0}}
			}
			progMap := map[string][]interface{}{}
			if p, ok := acct["progress"].([]interface{}); ok && len(p) >= 2 {
				progMap["cryptic"] = []interface{}{p[0], p[1]}
				progMap["ctf"] = []interface{}{"ctf-0", float64(0)}
			} else if pm, ok := acct["progress"].(map[string]interface{}); ok {
				for k, v := range pm {
					if arr, ok2 := v.([]interface{}); ok2 && len(arr) >= 2 {
						progMap[k] = []interface{}{arr[0], arr[1]}
					}
				}
				if _, ok := progMap["cryptic"]; !ok {
					levelsMap := map[string]float64{}
					if lm, ok := acct["levels"].(map[string]interface{}); ok {
						for k, v := range lm {
							if vf, ok := v.(float64); ok {
								levelsMap[k] = vf
							}
						}
					}
					curr := int(levelsMap["cryptic"])
					levelID := fmt.Sprintf("%s-%d", "cryptic", curr)
					progMap["cryptic"] = []interface{}{levelID, float64(0)}
				}
				if _, ok := progMap["ctf"]; !ok {
					progMap["ctf"] = []interface{}{"ctf-0", float64(0)}
				}
			} else {
				levelsMap := map[string]float64{}
				if lm, ok := acct["levels"].(map[string]interface{}); ok {
					for k, v := range lm {
						if vf, ok := v.(float64); ok {
							levelsMap[k] = vf
						}
					}
				}
				curr := int(levelsMap["cryptic"])
				levelID := fmt.Sprintf("%s-%d", "cryptic", curr)
				progMap["cryptic"] = []interface{}{levelID, float64(0)}
				progMap["ctf"] = []interface{}{"ctf-0", float64(0)}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"progress": progMap})
			return
		case http.MethodPost:
			var payload map[string]interface{}
			if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				defer r.Body.Close()
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					http.Error(w, "bad payload", http.StatusBadRequest)
					return
				}
			} else {
				r.ParseForm()
				payload = map[string]interface{}{}
				for k := range r.Form {
					payload[k] = r.Form.Get(k)
				}
			}

			targetEmail, _ := payload["email"].(string)
			if targetEmail == "" {
				http.Error(w, "missing email", http.StatusBadRequest)
				return
			}

			acctRaw, err := dbpkg.Get(dbConn, "accounts", targetEmail)
			var acct map[string]interface{}
			if err == nil {
				json.Unmarshal([]byte(acctRaw), &acct)
			} else {
				acct = map[string]interface{}{"levels": map[string]float64{"cryptic": 0, "ctf": 0}}
			}

			action, _ := payload["action"].(string)
			progMap := map[string][]interface{}{}
			if pm, ok := acct["progress"].(map[string]interface{}); ok {
				for k, v := range pm {
					if arr, ok2 := v.([]interface{}); ok2 && len(arr) >= 2 {
						progMap[k] = []interface{}{arr[0], arr[1]}
					}
				}
			} else if p, ok := acct["progress"].([]interface{}); ok && len(p) >= 2 {
				progMap["cryptic"] = []interface{}{p[0], p[1]}
			}

			switch action {
			case "inc":
				typ, _ := payload["type"].(string)
				if typ == "" {
					typ = "cryptic"
				}
				levelsMap := map[string]float64{}
				if lm, ok := acct["levels"].(map[string]interface{}); ok {
					for k, v := range lm {
						if vf, ok := v.(float64); ok {
							levelsMap[k] = vf
						}
					}
				}
				curr := int(levelsMap[typ])
				levelID := fmt.Sprintf("%s-%d", typ, curr)
				var progLevel string
				var progCheckpoint float64
				if pr, ok := progMap[typ]; ok && len(pr) >= 2 {
					if s, ok2 := pr[0].(string); ok2 {
						progLevel = s
					}
					if n, ok2 := pr[1].(float64); ok2 {
						progCheckpoint = n
					}
				} else {
					progLevel = levelID
					progCheckpoint = 0
				}
				if progLevel != levelID {
					progLevel = levelID
					progCheckpoint = 0
				}
				progCheckpoint = progCheckpoint + 1
				if progCheckpoint > 9 {
					progCheckpoint = 9
				}
				progMap[typ] = []interface{}{progLevel, progCheckpoint}
			case "set":
				typ, _ := payload["type"].(string)
				if typ == "" {
					typ = "cryptic"
				}
				if p, ok := payload["progress"].([]interface{}); ok && len(p) >= 2 {
					lvl, ok1 := p[0].(string)
					num, ok2 := p[1].(float64)
					if !ok1 || !ok2 {
						http.Error(w, "bad progress", http.StatusBadRequest)
						return
					}
					if num < 0 {
						num = 0
					}
					if num > 9 {
						num = 9
					}
					progMap[typ] = []interface{}{lvl, num}
				} else {
					http.Error(w, "bad progress", http.StatusBadRequest)
					return
				}
			default:
				http.Error(w, "unknown action", http.StatusBadRequest)
				return
			}

			acct["progress"] = progMap

			b, _ := json.Marshal(acct)
			if err := dbpkg.Set(dbConn, "accounts", targetEmail, string(b)); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "progress": acct["progress"]})
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

func AdminListUsersHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		requester, err := GetEmailFromRequest(dbConn, r)
		if err != nil || requester == "" || admins == nil || !admins.IsAdmin(requester) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		users := map[string]map[string]interface{}{}
		if accs, err := dbpkg.GetAll(dbConn, "accounts"); err == nil {
			for email, raw := range accs {
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &obj); err == nil {
					users[email] = obj
				}
			}
		}
		if regs, err := dbpkg.GetAll(dbConn, "registration"); err == nil {
			for email, raw := range regs {
				if _, ok := users[email]; ok {
					continue
				}
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &obj); err == nil {
					users[email] = obj
				}
			}
		}
		if lbs, err := dbpkg.GetAll(dbConn, "leaderboard"); err == nil {
			for email, raw := range lbs {
				if _, ok := users[email]; ok {
					continue
				}
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &obj); err == nil {
					users[email] = obj
				}
			}
		}

		out := []map[string]interface{}{}
		for email, obj := range users {
			name := ""
			if n, ok := obj["name"].(string); ok {
				name = n
			}
			cryptic := 0
			ctf := 0
			if lm, ok := obj["levels"].(map[string]interface{}); ok {
				if v, ok2 := lm["cryptic"]; ok2 {
					if vf, ok3 := v.(float64); ok3 {
						cryptic = int(vf)
					}
				}
				if v, ok2 := lm["ctf"]; ok2 {
					if vf, ok3 := v.(float64); ok3 {
						ctf = int(vf)
					}
				}
			}
			if prog, ok := obj["progress"].(map[string]interface{}); ok {
				if p, ok2 := prog["cryptic"].([]interface{}); ok2 && len(p) > 0 {
					if s, ok3 := p[0].(string); ok3 {
						parts := strings.SplitN(s, "-", 2)
						if len(parts) == 2 {
							if n, err := strconv.Atoi(parts[1]); err == nil {
								cryptic = n
							}
						}
					}
				}
				if p, ok2 := prog["ctf"].([]interface{}); ok2 && len(p) > 0 {
					if s, ok3 := p[0].(string); ok3 {
						parts := strings.SplitN(s, "-", 2)
						if len(parts) == 2 {
							if n, err := strconv.Atoi(parts[1]); err == nil {
								ctf = n
							}
						}
					}
				}
			}
			out = append(out, map[string]interface{}{"email": email, "name": name, "cryptic": cryptic, "ctf": ctf})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	}
}

func AdminUserActionHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		requester, err := GetEmailFromRequest(dbConn, r)
		if err != nil || requester == "" || admins == nil || !admins.IsAdmin(requester) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]string
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad payload", http.StatusBadRequest)
			return
		}
		email := payload["email"]
		action := payload["action"]
		if email == "" || action == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		acctRaw, _ := dbpkg.Get(dbConn, "accounts", email)
		var acct map[string]interface{}
		if acctRaw != "" {
			json.Unmarshal([]byte(acctRaw), &acct)
		} else {
			acct = map[string]interface{}{"levels": map[string]float64{"cryptic": 0, "ctf": 0}}
		}
		switch action {
		case "reset_cryptic":
			if lm, ok := acct["levels"].(map[string]interface{}); ok {
				lm["cryptic"] = float64(0)
				acct["levels"] = lm
			} else {
				acct["levels"] = map[string]float64{"cryptic": 0, "ctf": 0}
			}
			if prog, ok := acct["progress"].(map[string]interface{}); ok {
				delete(prog, "cryptic")
				acct["progress"] = prog
			}
			b, _ := json.Marshal(acct)
			if err := dbpkg.Set(dbConn, "accounts", email, string(b)); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		case "reset_ctf":
			if lm, ok := acct["levels"].(map[string]interface{}); ok {
				lm["ctf"] = float64(0)
				acct["levels"] = lm
			} else {
				acct["levels"] = map[string]float64{"cryptic": 0, "ctf": 0}
			}
			if prog, ok := acct["progress"].(map[string]interface{}); ok {
				delete(prog, "ctf")
				acct["progress"] = prog
			}
			b, _ := json.Marshal(acct)
			if err := dbpkg.Set(dbConn, "accounts", email, string(b)); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		case "delete":
			if err := dbpkg.Delete(dbConn, "accounts", email); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			_ = dbpkg.Delete(dbConn, "leaderboard", email)
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
