package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"time"

	"sudocrypt25/db"
	tplt "sudocrypt25/template"
)

func InitHandlers() {
	tplt.InitTemplates()
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func hashHex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func SendMail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	if smtpHost == "" || smtpPort == "" {
		fmt.Println("SendMail: SMTP_HOST or SMTP_PORT not set, skipping send")
		return fmt.Errorf("smtp config missing")
	}
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	headers := make(map[string]string)
	headers["From"] = smtpUser
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""
	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(v)
		sb.WriteString("\r\n")
	}
	sb.WriteString("\r\n")
	sb.WriteString(body)
	addr := smtpHost + ":" + smtpPort
	if err := smtp.SendMail(addr, auth, smtpUser, []string{to}, []byte(sb.String())); err != nil {
		fmt.Println("SendMail error:", err)
		return err
	}
	fmt.Println("SendMail: sent email to", to)
	return nil
}

func getOTP(email string) int {
	salt := os.Getenv("AUTH_SALT")
	if salt == "" {
		salt = "default_salt"
	}
	h := sha256.Sum256([]byte(email + salt))
	n := int64(0)
	for i := 0; i < 8; i++ {
		n = (n << 8) + int64(h[i])
	}
	if n < 0 {
		n = -n
	}
	return int(n % 1000000)
}

func SendOtpHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if r.Method == http.MethodPost {
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
				return
			}
		} else {
			payload = map[string]string{"email": r.URL.Query().Get("email")}
		}
		email := strings.TrimSpace(strings.ToLower(payload["email"]))
		if !isValidEmail(email) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid email"})
			return
		}

		regVal, regErr := db.Get(dbConn, "registration", email)
		fmt.Println("SendOtpHandler: checking registration for", email, "err=", regErr, "valLen=", len(regVal))
		if regErr == nil {
			fmt.Println("SendOtpHandler: registration exists for", email)
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "email already exists as a user"})
			return
		}

		if _, err := db.Get(dbConn, "emails", email); err == nil {
			fmt.Println("SendOtpHandler: found email record for", email)
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "email already exists as a user"})
			return
		}

		otp := getOTP(email)
		s := fmt.Sprintf("%06d", otp)
		digits := make([]string, 6)
		for i := 0; i < 6; i++ {
			digits[i] = string(s[i])
		}
		b, err := os.ReadFile("components/otp.html")
		if err == nil {
			html := string(b)
			html = strings.ReplaceAll(html, "{digit1}", digits[0])
			html = strings.ReplaceAll(html, "{digit2}", digits[1])
			html = strings.ReplaceAll(html, "{digit3}", digits[2])
			html = strings.ReplaceAll(html, "{digit4}", digits[3])
			html = strings.ReplaceAll(html, "{digit5}", digits[4])
			html = strings.ReplaceAll(html, "{digit6}", digits[5])
			go SendMail(email, "Sudocrypt OTP", html)
		} else {
			go SendMail(email, "Your OTP", fmt.Sprintf("Your verification code is: %06d", otp))
		}
		if name, ok := payload["name"]; ok {
			pending := map[string]string{"name": name, "phonenumber": payload["phonenumber"], "email": email, "password": payload["password"]}
			pb, _ := json.Marshal(pending)
			db.Set(dbConn, "pending_signup", email, string(pb))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"otp": "success"})
	}
}

func ApiAuthHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		method := q.Get("method")
		emailRaw := q.Get("email")
		email := strings.TrimSpace(strings.ToLower(emailRaw))
		password := q.Get("password")
		ph := q.Get("phonenumber")
		name := q.Get("name")

		otp := q.Get("otp")
		if method == "signup" {
			if email == "" || otp == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "missing fields"})
				return
			}
			if name == "" || password == "" || ph == "" {
				v, err := db.Get(dbConn, "pending_signup", email)
				if err == nil {
					var pending map[string]string
					json.Unmarshal([]byte(v), &pending)
					if name == "" {
						name = pending["name"]
					}
					if ph == "" {
						ph = pending["phonenumber"]
					}
					if password == "" {
						password = pending["password"]
					}
				}
				if name == "" || password == "" || ph == "" {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "missing fields"})
					return
				}
			}
			if !isValidEmail(email) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid email"})
				return
			}
			if len(ph) != 10 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid phone"})
				return
			}
			expected := fmt.Sprintf("%06d", getOTP(email))
			if otp != expected {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "incorrect otp"})
				return
			}
			_, err := db.Get(dbConn, "registration", email)
			if err == nil {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "account exists"})
				return
			}
			user := map[string]interface{}{"email": email, "name": name, "phonenumber": ph, "password": hashHex(password), "created_at": time.Now().Unix()}
			b, _ := json.Marshal(user)
			err = db.Set(dbConn, "registration", email, string(b))

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "db error"})
				return
			}
			sid, err := CreateSession(dbConn, email)
			if err == nil {
				SetSessionCookie(w, sid)
			}
			db.Set(dbConn, "emails", email, fmt.Sprintf("%d", time.Now().Unix()))
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			lb := map[string]interface{}{"email": email, "time": float64(time.Now().Unix()), "points": 0, "name": name}
			lbB, _ := json.Marshal(lb)
			db.Set(dbConn, "leaderboard", email, string(lbB))
			if pend, err := db.GetAll(dbConn, "pending_signup"); err == nil {
				for k := range pend {
					if k == email || strings.Contains(k, email) {
						db.Delete(dbConn, "pending_signup", k)
					}
				}
			}
			return
		}
		if method == "login" {
			if email == "" || password == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "missing fields"})
				return
			}
			v, err := db.Get(dbConn, "registration", email)
			fmt.Println("Login attempt for", email, "dbGetErr=", err, "valLen=", len(v))
			if err != nil {
				fmt.Println("Login failed: no account for", email)
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "no account"})
				return
			}
			var account map[string]interface{}
			json.Unmarshal([]byte(v), &account)
			stored, _ := account["password"].(string)
			if stored != hashHex(password) {
				fmt.Println("Login failed: password mismatch for", email)
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "incorrect password"})
				return
			}
			sid, err := CreateSession(dbConn, email)
			if err == nil {
				SetSessionCookie(w, sid)
			}
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "unknown method"})
	}
}
