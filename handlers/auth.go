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
		return nil
	}
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	msg := "From: " + smtpUser + "\nTo: " + to + "\nSubject: " + subject + "\n\n" + body
	addr := smtpHost + ":" + smtpPort
	return smtp.SendMail(addr, auth, smtpUser, []string{to}, []byte(msg))
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
		email := r.URL.Query().Get("email")
		if !isValidEmail(email) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid email"})
			return
		}
		otp := getOTP(email)
		body := fmt.Sprintf("Your verification code is: %06d", otp)
		go SendMail(email, "Your OTP", body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"otp": "success"})
	}
}

func ApiAuthHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		method := q.Get("method")
		email := q.Get("email")
		password := q.Get("password")
		ph := q.Get("phonenumber")
		name := q.Get("name")
		otp := q.Get("otp")
		if method == "signup" {
			if email == "" || password == "" || name == "" || otp == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "missing fields"})
				return
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
			_, err := db.Get(dbConn, "accounts", email)
			if err == nil {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "account exists"})
				return
			}
			user := map[string]interface{}{"email": email, "name": name, "phonenumber": ph, "password": hashHex(password), "created_at": time.Now().Unix()}
			b, _ := json.Marshal(user)
			err = db.Set(dbConn, "accounts", email, string(b))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "db error"})
				return
			}
			sid := hashHex(fmt.Sprintf("%s:%d", email, time.Now().UnixNano()))
			db.Set(dbConn, "sessions", sid, string(b))
			cookie := &http.Cookie{Name: "session_id", Value: sid, Path: "/", HttpOnly: true}
			http.SetCookie(w, cookie)
			db.Set(dbConn, "emails", email, fmt.Sprintf("%d", time.Now().Unix()))
			leaderboard := map[string]interface{}{"email": email, "score": 0}
			lb, _ := json.Marshal(leaderboard)
			db.Set(dbConn, "leaderboard", email, string(lb))
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}
		if method == "login" {
			if email == "" || password == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "missing fields"})
				return
			}
			v, err := db.Get(dbConn, "accounts", email)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "no account"})
				return
			}
			var account map[string]interface{}
			json.Unmarshal([]byte(v), &account)
			stored, _ := account["password"].(string)
			if stored != hashHex(password) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "incorrect password"})
				return
			}
			sid := hashHex(fmt.Sprintf("%s:%d", email, time.Now().UnixNano()))
			db.Set(dbConn, "sessions", sid, v)
			cookie := &http.Cookie{Name: "session_id", Value: sid, Path: "/", HttpOnly: true}
			http.SetCookie(w, cookie)
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "unknown method"})
	}
}
