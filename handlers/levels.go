package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	dbpkg "sudocrypt25/db"
)

type Level struct {
	ID         string `json:"id"`
	Answer     string `json:"answer"`
	Markup     string `json:"markup"`
	SourceHint string `json:"sourcehint"`
	PublicHash string `json:"public_hash,omitempty"`
}

func isValidLevelID(id string) bool {
	re := regexp.MustCompile(`^(ctf|cryptic)-([0-9]+)$`)
	return re.MatchString(id)
}

func SetLevelHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		levelid := q.Get("levelid")
		answer := q.Get("answer")
		markup := q.Get("markup")
		source := q.Get("source")
		if !isValidLevelID(levelid) {
			http.Error(w, "invalid level id", http.StatusBadRequest)
			return
		}
		lvl := Level{ID: levelid, Answer: answer, Markup: markup, SourceHint: source, PublicHash: ComputePublicHash(answer)}
		b, _ := json.Marshal(lvl)
		if err := dbpkg.Set(dbConn, "levels", levelid, string(b)); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		go func(id string) {
			_ = id
		}(levelid)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func DeleteLevelHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		level := q.Get("level")
		if !isValidLevelID(level) {
			http.Error(w, "invalid level id", http.StatusBadRequest)
			return
		}
		if err := dbpkg.Delete(dbConn, "levels", level); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func GetLevel(dbConn *sql.DB, id string) (*Level, error) {
	if !isValidLevelID(id) {
		return nil, fmt.Errorf("invalid id")
	}
	s, err := dbpkg.Get(dbConn, "levels", id)
	if err != nil {
		return nil, err
	}
	var lvl Level
	if err := json.Unmarshal([]byte(s), &lvl); err != nil {
		return nil, err
	}
	return &lvl, nil
}

func GetAllLevels(dbConn *sql.DB) (map[string]Level, error) {
	out := map[string]Level{}
	data, err := dbpkg.GetAll(dbConn, "levels")
	if err != nil {
		return nil, err
	}
	for k, v := range data {
		var lvl Level
		if err := json.Unmarshal([]byte(v), &lvl); err == nil {
			out[k] = lvl
		}
	}
	return out, nil
}

func GenerateAdminLevelsHTML(dbConn *sql.DB) (string, string, error) {
	levels, err := GetAllLevels(dbConn)
	if err != nil {
		return "", "", err
	}
	tplBytes, err := os.ReadFile("components/admin/level.html")
	if err != nil {
		return "", "", err
	}
	tpl := string(tplBytes)
	type kv struct {
		id  string
		lvl Level
		n   int
		t   string
	}
	var cryptic []kv
	var ctf []kv
	for id, lvl := range levels {
		parts := strings.SplitN(id, "-", 2)
		if len(parts) != 2 {
			continue
		}
		typ := parts[0]
		num, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		e := kv{id: id, lvl: lvl, n: num, t: typ}
		if typ == "cryptic" {
			cryptic = append(cryptic, e)
		} else if typ == "ctf" {
			ctf = append(ctf, e)
		}
	}
	sort.Slice(cryptic, func(i, j int) bool { return cryptic[i].n < cryptic[j].n })
	sort.Slice(ctf, func(i, j int) bool { return ctf[i].n < ctf[j].n })

	var sb strings.Builder
	dataMap := map[string]Level{}
	render := func(item kv) {
		s := tpl
		s = strings.ReplaceAll(s, "{{define \"admin_level\"}}", "")
		s = strings.ReplaceAll(s, "{{end}}", "")
		s = strings.ReplaceAll(s, "{{.ID}}", item.lvl.ID)
		s = strings.ReplaceAll(s, "{{.SourceHint}}", item.lvl.SourceHint)
		s = strings.ReplaceAll(s, "{{.Answer}}", item.lvl.Answer)
		sb.WriteString(s)
		dataMap[item.id] = item.lvl
	}

	if len(cryptic) > 0 {
		sb.WriteString("<h1 class=\"levels-heading\"><span style=\"color: #c7110d\">Cryptic</span> Levels</h1>\n")
		for _, it := range cryptic {
			render(it)
		}
	}
	if len(ctf) > 0 {
		sb.WriteString("<h1 class=\"levels-heading\"><span style=\"color: #c7110d\">CTF</span> Levels</h1>\n")
		for _, it := range ctf {
			render(it)
		}
	}

	jsb, _ := json.Marshal(dataMap)
	return sb.String(), string(jsb), nil
}

func ComputePublicHash(answer string) string {
	salt := os.Getenv("AUTH_SALT")
	if salt == "" {
		salt = "public_salt_to_prevent_rainbow_tables"
	}
	h := sha256.Sum256([]byte(salt + answer))
	return hex.EncodeToString(h[:])
}

func SubmitHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		answer := q.Get("answer")
		typ := q.Get("type")
		if typ == "" {
			typ = "cryptic"
		}
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

		acctRaw, err := dbpkg.Get(dbConn, "accounts", email)
		var acct map[string]interface{}
		if err == nil {
			json.Unmarshal([]byte(acctRaw), &acct)
		} else {
			acct = map[string]interface{}{"levels": map[string]float64{"cryptic": 0, "ctf": 0}}
		}
		now := time.Now().Unix()
		if dq, ok := acct["disqualified"].(bool); ok && dq {
			http.Error(w, "disqualified", http.StatusForbidden)
			return
		}
		if admin, _ := acct["admin"].(bool); !admin {
			if os.Getenv("EVENT_ACTIVE") == "0" {
				http.Error(w, "event not active", http.StatusForbidden)
				return
			}
		}
		if last, ok := acct["last_submit"].(float64); ok {
			if now-int64(last) < 1 {
				json.NewEncoder(w).Encode(map[string]bool{"success": false})
				return
			}
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
		lvl, err := GetLevel(dbConn, levelID)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]bool{"success": false})
			return
		}

		correct := strings.TrimSpace(lvl.Answer) == strings.TrimSpace(answer)
		if correct {
			levelsMap[typ] = float64(curr + 1)
			acct["levels"] = levelsMap
			acct["last_submit"] = float64(now)
			b, _ := json.Marshal(acct)
			dbpkg.Set(dbConn, "accounts", email, string(b))

			total := 0
			if v, ok := levelsMap["cryptic"]; ok {
				total += int(v)
			}
			if v, ok := levelsMap["ctf"]; ok {
				total += int(v)
			}
			name := email
			if n, ok := acct["name"].(string); ok && n != "" {
				name = n
			}
			lb := map[string]interface{}{"email": email, "name": name, "points": total, "time": float64(now)}
			lbB, _ := json.Marshal(lb)
			dbpkg.Set(dbConn, "leaderboard", email, string(lbB))

			dbpkg.Delete(dbConn, "messages/"+email, typ)

			lval := fmt.Sprintf("submit|%s|%s|correct", typ, strings.TrimSpace(answer))
			dbpkg.Set(dbConn, "logs", email, lval)

			nextCurr := curr + 1
			nextLevelID := fmt.Sprintf("%s-%d", typ, nextCurr)
			nextLvl, _ := GetLevel(dbConn, nextLevelID)
			if nextLvl != nil {
				nextLvl.Answer = ""
			}
			resp := map[string]interface{}{"success": true, "next_level": nextLvl}
			json.NewEncoder(w).Encode(resp)
			return
		}
		acct["last_submit"] = float64(now)
		b, _ := json.Marshal(acct)
		dbpkg.Set(dbConn, "accounts", email, string(b))

		lval := fmt.Sprintf("submit|%s|%s|incorrect", typ, strings.TrimSpace(answer))
		dbpkg.Set(dbConn, "logs", email, lval)
		json.NewEncoder(w).Encode(map[string]bool{"success": false})
	}
}

func LevelsListHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		levels, err := GetAllLevels(dbConn)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		ids := make([]string, 0, len(levels))
		for id := range levels {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ids)
	}
}

func CurrentLevelHandler(dbConn *sql.DB) http.HandlerFunc {
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

		acctRaw, err := dbpkg.Get(dbConn, "accounts", email)
		var acct map[string]interface{}
		if err == nil {
			json.Unmarshal([]byte(acctRaw), &acct)
		} else {
			acct = map[string]interface{}{"levels": map[string]float64{"cryptic": 0, "ctf": 0}}
		}

		typ := r.URL.Query().Get("type")
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
		lvl, err := GetLevel(dbConn, levelID)
		if err != nil {
			placeholder := &Level{ID: "", Markup: "<p>No level available currently. Please check back later.</p>"}
			placeholder.Answer = ""
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(placeholder)
			return
		}
		lvl.Answer = ""
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lvl)
	}
}
