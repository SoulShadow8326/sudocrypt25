package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	dbpkg "sudocrypt25/db"
)

func AdminUpdateUserProgressHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("email")
		if err != nil || c.Value == "" || admins == nil || !admins.IsAdmin(c.Value) {
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
