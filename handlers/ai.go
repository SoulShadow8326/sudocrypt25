package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	dbpkg "sudocrypt25/db"
	"sync"
	"time"

	"google.golang.org/genai"
)

var geminiKeys []string
var geminiIdx int
var geminiMu sync.Mutex
var botPrefix string
var botLoaded bool
var botMu sync.Mutex

func loadBotJSON() {
	botMu.Lock()
	defer botMu.Unlock()
	if botLoaded {
		return
	}
	path := os.Getenv("BOT_JSON_PATH")
	if path == "" {
		path = "./bot.json"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		botLoaded = true
		return
	}
	var m map[string]interface{}
	if json.Unmarshal(b, &m) != nil {
		botLoaded = true
		return
	}
	keys := []string{"system", "prompt", "instructions", "bot", "description", "intro"}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
				botPrefix = s
				break
			}
		}
	}
	botLoaded = true
}

func loadGeminiKeys() {
	if len(geminiKeys) > 0 {
		return
	}
	for i := 1; i <= 15; i++ {
		k := os.Getenv("GEMINI_API_KEY_" + itoa(i))
		if k != "" {
			geminiKeys = append(geminiKeys, k)
		}
	}
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }

func pickGeminiKey() string {
	geminiMu.Lock()
	defer geminiMu.Unlock()
	if len(geminiKeys) == 0 {
		loadGeminiKeys()
	}
	if len(geminiKeys) == 0 {
		return ""
	}
	k := geminiKeys[geminiIdx%len(geminiKeys)]
	geminiIdx++
	return k
}

func AILeadHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		emailC, err := GetEmailFromRequest(dbConn, r)
		if err != nil || emailC == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		var payload map[string]string
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			defer r.Body.Close()
			json.NewDecoder(r.Body).Decode(&payload)
		} else {
			r.ParseForm()
			payload = map[string]string{"level": r.FormValue("level"), "question": r.FormValue("question")}
		}
		// check global ai leads setting
		if v, err := dbpkg.Get(dbConn, "settings", "ai_leads"); err == nil {
			if strings.TrimSpace(strings.ToLower(v)) == "0" || strings.TrimSpace(strings.ToLower(v)) == "false" {
				http.Error(w, "ai leads disabled", http.StatusForbidden)
				return
			}
		}

		lvlID := strings.TrimSpace(payload["level"])
		if lvlID == "" {
			http.Error(w, "missing level", http.StatusBadRequest)
			return
		}
		lvl, err := GetLevel(dbConn, lvlID)
		if err != nil || lvl == nil {
			http.Error(w, "no level", http.StatusNotFound)
			return
		}
		if strings.TrimSpace(lvl.Walkthrough) == "" {
			http.Error(w, "no walkthrough", http.StatusNotFound)
			return
		}
		loadBotJSON()
		var promptText string
		var arr []string
		if json.Unmarshal([]byte(lvl.Walkthrough), &arr) == nil {
			promptText = strings.Join(arr, "\n\n")
		} else {
			promptText = lvl.Walkthrough
		}
		if q, ok := payload["question"]; ok && strings.TrimSpace(q) != "" {
			promptText = promptText + "\n\nUser question: " + q
		}
		promptText = "You are given the following walkthrough. Answer the user's question if any. Reply with ONLY the word true or false (lowercase) indicating whether the statement/question is valid based on the walkthrough. No other text." + "\n\nWalkthrough:\n" + promptText
		if botPrefix != "" {
			promptText = botPrefix + "\n\n" + promptText
		}

		userQuestion := strings.TrimSpace(payload["question"])
		if userQuestion != "" {
			userEmail := strings.ToLower(emailC)
			msgsMap, _ := dbpkg.GetAll(dbConn, "messages")
			history := make([]struct {
				ts      int64
				from    string
				to      string
				content string
			}, 0)
			for _, v := range msgsMap {
				var mm map[string]interface{}
				if json.Unmarshal([]byte(v), &mm) != nil {
					continue
				}
				lid := ""
				if x, ok := mm["level_id"].(string); ok {
					lid = x
				}
				if lid == "" {
					if x, ok := mm["level"].(string); ok {
						lid = x
					}
				}
				if lid == "" {
					if x, ok := mm["LevelID"].(string); ok {
						lid = x
					}
				}
				if lid != lvlID {
					continue
				}
				from := ""
				if x, ok := mm["from"].(string); ok {
					from = x
				}
				to := ""
				if x, ok := mm["to"].(string); ok {
					to = x
				}
				content := ""
				if x, ok := mm["content"].(string); ok {
					content = x
				}
				var ts int64
				if x, ok := mm["created_at"]; ok {
					switch tv := x.(type) {
					case float64:
						ts = int64(tv)
					case int64:
						ts = tv
					case int:
						ts = int64(tv)
					case string:
						if v, err := strconv.ParseInt(tv, 10, 64); err == nil {
							ts = v
						}
					}
				}
				lowFrom := strings.ToLower(from)
				lowTo := strings.ToLower(to)
				if lowFrom != userEmail && lowTo != userEmail && lowFrom != "admin@sudocrypt.com" && lowTo != "admin@sudocrypt.com" {
					continue
				}
				history = append(history, struct {
					ts      int64
					from    string
					to      string
					content string
				}{ts, from, to, content})
			}
			if len(history) > 0 {
				sort.Slice(history, func(i, j int) bool { return history[i].ts < history[j].ts })
				var b strings.Builder
				b.WriteString("\n\nConversation history:\n")
				for _, h := range history {
					who := "User"
					if strings.EqualFold(h.from, "admin@sudocrypt.com") {
						who = "Admin"
					}
					if strings.EqualFold(h.from, userEmail) {
						who = "User"
					}
					b.WriteString(fmt.Sprintf("%s: %s\n", who, h.content))
				}
				promptText = promptText + b.String()
			}

			var textOut string
			var lastErr string
			for attempt := 0; attempt < 2; attempt++ {
				key := pickGeminiKey()
				if key == "" {
					lastErr = "no api keys"
					break
				}

				var cli *genai.Client
				var err error
				geminiMu.Lock()
				prev := os.Getenv("GEMINI_API_KEY")
				os.Setenv("GEMINI_API_KEY", key)
				cli, err = genai.NewClient(context.Background(), nil)
				os.Setenv("GEMINI_API_KEY", prev)
				geminiMu.Unlock()
				if err != nil {
					lastErr = "llm client error: " + err.Error()
					fmt.Println("ai: genai.NewClient error:", err)
					continue
				}

				ctx2, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				res, err := cli.Models.GenerateContent(ctx2, "gemini-2.5-flash", genai.Text(promptText), nil)
				cancel()
				if err != nil {
					lastErr = "llm error: " + err.Error()
					fmt.Println("ai: GenerateContent error:", err)
					continue
				}
				if res == nil {
					lastErr = "empty response"
					fmt.Println("ai: empty res")
					continue
				}
				textOut = strings.TrimSpace(res.Text())
				if textOut == "" {
					lastErr = "empty response"
					fmt.Println("ai: empty text")
					continue
				}
				re := regexp.MustCompile(`(?i)\b(true|false)\b`)
				m := re.FindStringSubmatch(textOut)
				if len(m) >= 2 {
					val := strings.ToLower(m[1]) == "true"

					userEmail := strings.ToLower(emailC)
					userQuestion := strings.TrimSpace(payload["question"])
					if userQuestion != "" {
						userVal := strings.Join([]string{userEmail, "admin@sudocrypt.com", lvlID, "lead", userQuestion}, "|")
						_ = dbpkg.Set(dbConn, "messages", userEmail, userVal)
					}
					aiContent := "false"
					if val {
						aiContent = "true"
					}
					aiVal := strings.Join([]string{"admin@sudocrypt.com", userEmail, lvlID, "lead", aiContent}, "|")
					_ = dbpkg.Set(dbConn, "messages", userEmail, aiVal)

					if val {
						acctRaw, err := dbpkg.Get(dbConn, "accounts", userEmail)
						var acct map[string]interface{}
						if err == nil {
							json.Unmarshal([]byte(acctRaw), &acct)
						} else {
							acct = map[string]interface{}{"levels": map[string]float64{"cryptic": 0, "ctf": 0}}
						}
						progMap := map[string][]interface{}{}
						if pm, ok := acct["progress"].(map[string]interface{}); ok {
							for k, v := range pm {
								if arr2, ok2 := v.([]interface{}); ok2 && len(arr2) >= 2 {
									progMap[k] = []interface{}{arr2[0], arr2[1]}
								}
							}
						} else if p, ok := acct["progress"].([]interface{}); ok && len(p) >= 2 {
							progMap["cryptic"] = []interface{}{p[0], p[1]}
						}
						levelsMap := map[string]float64{}
						if lm, ok := acct["levels"].(map[string]interface{}); ok {
							for k, v := range lm {
								if vf, ok2 := v.(float64); ok2 {
									levelsMap[k] = vf
								}
							}
						}
						partsArr := arr
						partsLower := make([]string, 0)
						for _, p := range partsArr {
							partsLower = append(partsLower, strings.ToLower(p))
						}
						partsTok := regexp.MustCompile(`[A-Za-z0-9\.]+`).FindAllString(strings.ToLower(userQuestion), -1)
						matchedIdx := -1
						if len(partsLower) > 0 && len(partsTok) > 0 {
							qstr := strings.ToLower(userQuestion)
							for i, p := range partsLower {
								if p == "" {
									continue
								}
								if strings.Contains(p, qstr) || strings.Contains(qstr, p) {
									matchedIdx = i
									break
								}
								for _, tok := range partsTok {
									if len(tok) < 2 {
										continue
									}
									if strings.Contains(p, tok) {
										matchedIdx = i
										break
									}
								}
								if matchedIdx != -1 {
									break
								}
							}
						}
						partsIdx := matchedIdx
						partsCount := len(partsLower)
						partsIdxValid := partsIdx >= 0 && partsIdx < partsCount
						parts := strings.SplitN(lvlID, "-", 2)
						typ := "cryptic"
						if len(parts) == 2 {
							typ = parts[0]
						}
						curr := 0
						if v, ok := levelsMap[typ]; ok {
							curr = int(v)
						}
						expectedLevel := fmt.Sprintf("%s-%d", typ, curr)
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
							progLevel = expectedLevel
							progCheckpoint = 0
						}
						if progLevel != expectedLevel {
							progLevel = expectedLevel
							progCheckpoint = 0
						}
						if partsIdxValid {
							nextCheckpoint := int(progCheckpoint) + 1
							if partsIdx == nextCheckpoint {
								progCheckpoint = float64(partsIdx)
								if progCheckpoint > 9 {
									progCheckpoint = 9
								}
								progMap[typ] = []interface{}{progLevel, progCheckpoint}
								acct["progress"] = progMap
								b, _ := json.Marshal(acct)
								_ = dbpkg.Set(dbConn, "accounts", userEmail, string(b))
							}
						}
					}

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]bool{"result": val})
					return
				}
				lastErr = "invalid response: " + textOut
				fmt.Println("ai: invalid response, textOut=", textOut)
			}

			if lastErr == "no api keys" {
				http.Error(w, lastErr, http.StatusInternalServerError)
				return
			}
			http.Error(w, lastErr, http.StatusInternalServerError)
		}
	}
}

func ToggleAILeadsHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		email, err := GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var payload map[string]interface{}
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			defer r.Body.Close()
			json.NewDecoder(r.Body).Decode(&payload)
		} else {
			r.ParseForm()
			payload = map[string]interface{}{"enabled": r.FormValue("enabled")}
		}
		enabled := true
		if v, ok := payload["enabled"]; ok && v != nil {
			switch tv := v.(type) {
			case bool:
				enabled = tv
			case string:
				s := strings.TrimSpace(strings.ToLower(tv))
				if s == "0" || s == "false" || s == "off" {
					enabled = false
				} else if s == "1" || s == "true" || s == "on" {
					enabled = true
				}
			case float64:
				enabled = int(tv) != 0
			}
		}
		val := "1"
		if !enabled {
			val = "0"
		}
		if err := dbpkg.Set(dbConn, "settings", "ai_leads", val); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "enabled": enabled})
	}
}
