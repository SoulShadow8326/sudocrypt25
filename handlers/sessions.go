package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"

	"sudocrypt25/db"
)

func genSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func CreateSession(dbConn *sql.DB, email string) (string, error) {
	sid, err := genSessionID()
	if err != nil {
		return "", err
	}
	if err := db.Set(dbConn, "sessions", sid, email); err != nil {
		return "", err
	}
	return sid, nil
}

func GetEmailFromRequest(dbConn *sql.DB, r *http.Request) (string, error) {
	c, err := r.Cookie("session_id")
	if err != nil || c.Value == "" {
		return "", err
	}
	email, err := db.Get(dbConn, "sessions", c.Value)
	if err != nil {
		return "", err
	}
	return email, nil
}

func DeleteSession(dbConn *sql.DB, r *http.Request) error {
	c, err := r.Cookie("session_id")
	if err != nil || c.Value == "" {
		return err
	}
	return db.Delete(dbConn, "sessions", c.Value)
}

func SetSessionCookie(w http.ResponseWriter, sid string) {
	cookie := &http.Cookie{Name: "session_id", Value: sid, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400}
	http.SetCookie(w, cookie)
}
