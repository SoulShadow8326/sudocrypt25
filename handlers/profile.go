package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	htmltmpl "html/template"
	"net/http"
	"net/url"
	"strings"

	dbpkg "sudocrypt25/db"
)

func UpdateBioHandler(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		email, err := GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Bio       string `json:"bio"`
			BioPublic bool   `json:"bio_public"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		acctRaw, err := dbpkg.Get(dbConn, "users", email)
		if err != nil {
			http.Error(w, "Account not found", http.StatusNotFound)
			return
		}

		var acct map[string]interface{}
		if err := json.Unmarshal([]byte(acctRaw), &acct); err != nil {
			http.Error(w, "Error loading account", http.StatusInternalServerError)
			return
		}

		acct["bio"] = req.Bio
		acct["bio_public"] = req.BioPublic

		acctBytes, _ := json.Marshal(acct)
		if err := dbpkg.Set(dbConn, "users", email, string(acctBytes)); err != nil {
			http.Error(w, "Failed to update", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Bio updated successfully",
		})
	}
}

func UserProfileHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from="+r.URL.Path, http.StatusFound)
			return
		}

		ident := strings.TrimPrefix(r.URL.Path, "/profile/")
		ident = strings.TrimSpace(ident)
		email, _ := url.PathUnescape(ident)

		acctRaw, err := dbpkg.Get(dbConn, "users", email)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		var acct map[string]interface{}
		if err := json.Unmarshal([]byte(acctRaw), &acct); err != nil {
			http.Error(w, "Error loading user", http.StatusInternalServerError)
			return
		}

		currentUserEmail := ""
		if ce, err := GetEmailFromRequest(dbConn, r); err == nil {
			currentUserEmail = ce
		}

		isOwnProfile := email == currentUserEmail

		userBio := ""
		if bio, ok := acct["bio"].(string); ok {
			userBio = bio
		}

		bioPublic := false
		if bp, ok := acct["bio_public"].(bool); ok {
			bioPublic = bp
		}

		showBio := isOwnProfile || bioPublic
		displayName := ""
		if n, ok := acct["name"].(string); ok {
			displayName = n
		}
		userImg := ""
		if displayName != "" {
			userImg = fmt.Sprintf("https://api.dicebear.com/9.x/big-smile/svg?seed=%s", displayName)
		} else {
			userImg = fmt.Sprintf("https://api.dicebear.com/9.x/big-smile/svg?seed=%s", email)
		}

		levelsCryptic := 0.0
		levelsCTF := 0.0
		if lv, ok := acct["levels"].(map[string]interface{}); ok {
			if v, ok2 := lv["cryptic"].(float64); ok2 {
				levelsCryptic = v
			}
			if v, ok2 := lv["ctf"].(float64); ok2 {
				levelsCTF = v
			}
		}

		viewerIsAdmin := false
		if currentUserEmail != "" {
			if admins != nil && admins.IsAdmin(currentUserEmail) {
				viewerIsAdmin = true
			} else {
				if acctRawV, err := dbpkg.Get(dbConn, "accounts", currentUserEmail); err == nil && acctRawV != "" {
					var acctV map[string]interface{}
					if json.Unmarshal([]byte(acctRawV), &acctV) == nil {
						if adm, _ := acctV["admin"].(bool); adm {
							viewerIsAdmin = true
						}
					}
				}
			}
		}

		data := map[string]interface{}{
			"Name":            displayName,
			"Email":           email,
			"Bio":             userBio,
			"Img":             userImg,
			"IsOwnProfile":    isOwnProfile,
			"BioPublic":       bioPublic,
			"ShowBio":         showBio,
			"LevelsCryptic":   levelsCryptic,
			"LevelsCTF":       levelsCTF,
			"ScoreTimes":      nil,
			"ScorePoints":     nil,
			"PageTitle":       fmt.Sprintf("%s - Profile", displayName),
			"IsAuthenticated": true,
			"Admin":           viewerIsAdmin,
		}

		logsMap, _ := dbpkg.GetAll(dbConn, "logs")
		times := []int64{}
		points := []int{}
		cum := 0
		for _, v := range logsMap {
			var entry struct {
				ID        int    `json:"id"`
				Namespace string `json:"namespace"`
				Key       string `json:"key"`
				Event     string `json:"event"`
				Data      string `json:"data"`
				CreatedAt int64  `json:"created_at"`
			}
			if err := json.Unmarshal([]byte(v), &entry); err != nil {
				continue
			}
			if entry.Key != email {
				continue
			}
			if entry.Namespace != "submit" {
				continue
			}
			if strings.Contains(entry.Data, "correct") {
				cum += 1
				times = append(times, entry.CreatedAt)
				points = append(points, cum)
			}
		}
		if len(times) > 0 {
			tb, _ := json.Marshal(times)
			pb, _ := json.Marshal(points)
			data["ScoreTimesJSON"] = string(tb)
			data["ScorePointsJSON"] = string(pb)
		} else {
			data["ScoreTimesJSON"] = "[]"
			data["ScorePointsJSON"] = "[]"
		}

		tmpl, err := htmltmpl.ParseFiles("components/profile/profile.html", "components/header/header.html", "components/footer/footer.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		if err := tmpl.ExecuteTemplate(w, "profile", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
	}
}
