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
		emailC, err := r.Cookie("email")
		if err != nil || emailC.Value == "" {
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

		userEmail := strings.ToLower(emailC.Value)
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

				userEmail := strings.ToLower(emailC.Value)
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
