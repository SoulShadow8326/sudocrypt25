package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	dbpkg "sudocrypt25/db"
	"sync"
	"time"

	"google.golang.org/genai"
)

var geminiKeys []string
var geminiIdx int
var geminiMu sync.Mutex

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
		promptText := lvl.Walkthrough
		if q, ok := payload["question"]; ok && strings.TrimSpace(q) != "" {
			promptText = promptText + "\n\nUser question: " + q
		}
		promptText = "You are given the following walkthrough. Answer the user's question if any. Reply with ONLY the word true or false (lowercase) indicating whether the statement/question is valid based on the walkthrough. No other text." + "\n\nWalkthrough:\n" + promptText

		var textOut string
		var lastErr string
		for attempt := 0; attempt < 2; attempt++ {
			key := pickGeminiKey()
			if key == "" {
				lastErr = "no api keys"
				break
			}

			prev := os.Getenv("GEMINI_API_KEY")
			os.Setenv("GEMINI_API_KEY", key)
			cli, err := genai.NewClient(context.Background(), nil)
			if err != nil {
				lastErr = "llm client error: " + err.Error()
				fmt.Println("ai: genai.NewClient error:", err)
				os.Setenv("GEMINI_API_KEY", prev)
				continue
			}

			ctx2, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			res, err := cli.Models.GenerateContent(ctx2, "gemini-2.5-flash", genai.Text(promptText), nil)
			cancel()
			os.Setenv("GEMINI_API_KEY", prev)
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
