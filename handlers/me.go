package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	dbpkg "sudocrypt25/db"
)

func MeHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email, err := GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		acctRaw, _ := dbpkg.Get(dbConn, "accounts", email)
		var acct map[string]interface{}
		if acctRaw != "" {
			json.Unmarshal([]byte(acctRaw), &acct)
		}
		name := ""
		if n, ok := acct["name"].(string); ok {
			name = n
		}
		isAdmin := false
		if admins != nil && admins.IsAdmin(email) {
			isAdmin = true
		}
		if adm, _ := acct["admin"].(bool); adm {
			isAdmin = true
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"email": email, "name": name, "admin": isAdmin})
	}
}
