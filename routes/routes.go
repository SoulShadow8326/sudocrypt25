package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	htmltmpl "html/template"
	"net/http"
	"os"
	"strings"
	"time"

	dbpkg "sudocrypt25/db"
	"sudocrypt25/handlers"
	"sudocrypt25/template"
)

func InitRoutes(dbConn *sql.DB, admins *handlers.Admins) {
	handlers.InitHandlers()
	template.InitTemplates()
	http.Handle("/components/", http.StripPrefix("/components/", http.FileServer(http.Dir("components"))))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("components/assets"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Redirect(w, r, "/404", http.StatusFound)
			return
		}

		_, err := r.Cookie("session_id")
		auth := err == nil
		td := template.TemplateData{PageTitle: "Home", CurrentPath: r.URL.Path, TimeGateStart: os.Getenv("TIMEGATE_START"), IsAuthenticated: auth}
		if err := template.RenderTemplate(w, "landing", td); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("session_id")
		auth := err == nil
		td := template.TemplateData{PageTitle: "Auth", CurrentPath: r.URL.Path, TimeGateStart: os.Getenv("TIMEGATE_START"), IsAuthenticated: auth}
		if err := template.RenderTemplate(w, "auth", td); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/auth/", func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("session_id")
		auth := err == nil
		td := template.TemplateData{PageTitle: "Auth", CurrentPath: r.URL.Path, TimeGateStart: os.Getenv("TIMEGATE_START"), IsAuthenticated: auth}
		if err := template.RenderTemplate(w, "auth", td); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("session_id")
		auth := err == nil
		td := template.TemplateData{PageTitle: "Not Found", CurrentPath: r.URL.Path, IsAuthenticated: auth}
		if err := template.RenderFile(w, "components/404.html", td); err != nil {
			fmt.Printf("render /404 failed: %v\n", err)
			http.ServeFile(w, r, "components/404.html")
		}
	})
	http.HandleFunc("/404/", func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("session_id")
		auth := err == nil
		td := template.TemplateData{PageTitle: "Not Found", CurrentPath: r.URL.Path, IsAuthenticated: auth}
		if err := template.RenderFile(w, "components/404.html", td); err != nil {
			fmt.Printf("render /404 failed: %v\n", err)
			http.ServeFile(w, r, "components/404.html")
		}
	})
	http.HandleFunc("/send_otp", handlers.SendOtpHandler(dbConn))
	http.HandleFunc("/api/auth", handlers.ApiAuthHandler(dbConn))
	http.HandleFunc("/api/announcements", handlers.AnnouncementsHandler(dbConn))
	http.HandleFunc("/timegate", func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("session_id")
		auth := err == nil
		td := template.TemplateData{PageTitle: "Time Gate", CurrentPath: r.URL.Path, TimeGateStart: os.Getenv("TIMEGATE_START"), IsAuthenticated: auth}
		if err := template.RenderFile(w, "components/timegate.html", td); err != nil {
			http.ServeFile(w, r, "components/timegate.html")
		}
	})
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		cookie := &http.Cookie{Name: "session_id", Value: "", Path: "/", HttpOnly: true, Expires: time.Unix(0, 0), MaxAge: -1}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusFound)
	})
	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/play", http.StatusFound)
			return
		}
		auth := true
		if !handlers.IsTimeGateOpen() {
			c, err := r.Cookie("email")
			if err != nil || c.Value == "" || admins == nil || !admins.IsAdmin(c.Value) {
				http.Redirect(w, r, "/timegate?toast=1&from=/play", http.StatusFound)
				return
			}
		}
		td := template.TemplateData{PageTitle: "Play", CurrentPath: r.URL.Path, IsAuthenticated: auth, ShowAnnouncements: true}
		if c2, err := r.Cookie("email"); err == nil && c2.Value != "" {
			email := c2.Value
			td.UserEmail = email
			acctRaw, err := dbpkg.Get(dbConn, "accounts", email)
			if err == nil {
				var acct map[string]interface{}
				if err := json.Unmarshal([]byte(acctRaw), &acct); err == nil {
					typ := r.URL.Query().Get("type")
					if typ == "" {
						typ = "cryptic"
					}
					curr := 0
					if lm, ok := acct["levels"].(map[string]interface{}); ok {
						if v, ok := lm[typ].(float64); ok {
							curr = int(v)
						}
					}
					td.LevelNum = fmt.Sprintf("%d", curr)

					levelID := fmt.Sprintf("%s-%d", typ, curr)
					if lvl, err := handlers.GetLevel(dbConn, levelID); err == nil && lvl != nil {
						if lvl.SourceHint != "" {
							td.SrcHint = htmltmpl.HTML("<!--" + lvl.SourceHint + "-->")
						} else {
							td.SrcHint = htmltmpl.HTML("")
						}
					}
				}
			}
		}
		if err := template.RenderTemplate(w, "play", td); err != nil {
			http.ServeFile(w, r, "components/play/play.html")
		}
	})
	http.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/leaderboard", http.StatusFound)
			return
		}
		auth := true
		if !handlers.IsTimeGateOpen() {
			c, err := r.Cookie("email")
			if err != nil || c.Value == "" || admins == nil || !admins.IsAdmin(c.Value) {
				http.Redirect(w, r, "/timegate?toast=1&from=/leaderboard", http.StatusFound)
				return
			}
		}
		td := template.TemplateData{PageTitle: "Leaderboard", CurrentPath: r.URL.Path, IsAuthenticated: auth}
		if html, err := handlers.GenerateLeaderboardHTML(dbConn); err == nil {
			td.LeaderboardHTML = htmltmpl.HTML(html)
		}
		if err := template.RenderTemplate(w, "leaderboard", td); err != nil {
			http.ServeFile(w, r, "components/leaderboard/leaderboard.html")
		}
	})
	http.HandleFunc("/set_level", handlers.SetLevelHandler(dbConn))
	http.HandleFunc("/delete_level", handlers.DeleteLevelHandler(dbConn))
	http.HandleFunc("/submit", handlers.SubmitHandler(dbConn))
	http.HandleFunc("/api/play/current", handlers.CurrentLevelHandler(dbConn))
	http.HandleFunc("/api/leaderboard", handlers.LeaderboardAPIHandler(dbConn))
	http.HandleFunc("/api/levels", handlers.LevelsListHandler(dbConn))
	http.HandleFunc("/api/admin/announcements/set", handlers.SetAnnouncementHandler(dbConn, admins))
	http.HandleFunc("/api/admin/announcements/delete", handlers.DeleteAnnouncementHandler(dbConn, admins))
	http.HandleFunc("/api/admin/user/progress", handlers.AdminUpdateUserProgressHandler(dbConn, admins))
	// messages APIs
	http.HandleFunc("/api/messages", handlers.ListMessagesHandler(dbConn, admins))
	http.HandleFunc("/api/message/send", func(w http.ResponseWriter, r *http.Request) {
		handlers.SendMessageHandler(dbConn, admins)(w, r)
	})
	// logs API
	http.HandleFunc("/api/logs", handlers.LogsHandler(dbConn, admins))
	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/admin", http.StatusFound)
			return
		}
		auth := true
		c2, err := r.Cookie("email")
		if err != nil || c2.Value == "" || admins == nil || !admins.IsAdmin(c2.Value) {
			http.Redirect(w, r, "/timegate?toast=1&from=/admin", http.StatusFound)
			return
		}
		td := template.TemplateData{PageTitle: "Admin", CurrentPath: r.URL.Path, IsAuthenticated: auth}
		if html, js, err := handlers.GenerateAdminLevelsHTML(dbConn); err == nil {
			td.LevelsHTML = htmltmpl.HTML(html)
			td.LevelsData = htmltmpl.JS(js)
		}
		if err := template.RenderTemplate(w, "admin", td); err != nil {
			http.ServeFile(w, r, "components/admin/admin.html")
		}
	})

	http.HandleFunc("/admin/announcement/create", handlers.AdminCreateAnnouncementFormHandler(dbConn, admins))
	http.HandleFunc("/admin/announcement/delete", handlers.AdminDeleteAnnouncementFormHandler(dbConn, admins))
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/dashboard", http.StatusFound)
			return
		}
		auth := true
		c2, err := r.Cookie("email")
		if err != nil || c2.Value == "" || admins == nil || !admins.IsAdmin(c2.Value) {
			http.Redirect(w, r, "/timegate?toast=1&from=/dashboard", http.StatusFound)
			return
		}
		td := template.TemplateData{PageTitle: "Dashboard", CurrentPath: r.URL.Path, IsAuthenticated: auth}
		if html, js, err := handlers.GenerateAdminLevelsHTML(dbConn); err == nil {
			td.LevelsHTML = htmltmpl.HTML(html)
			td.LevelsData = htmltmpl.JS(js)
		}
		if err := template.RenderTemplate(w, "dashboard", td); err != nil {
			http.ServeFile(w, r, "components/dashboard/dashboard.html")
		}
	})
	http.HandleFunc("/api/hints", handlers.HintsHandler(dbConn))
	http.HandleFunc("/api/admin/hints", handlers.AdminHintsHandler(dbConn, admins))
	http.HandleFunc("/api/admin/levels/leads", handlers.AdminLevelLeadsHandler(dbConn, admins))
	http.HandleFunc("/api/ai/lead", handlers.AILeadHandler(dbConn))

	http.HandleFunc("/api/user/update_bio", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		emailCookie, err := r.Cookie("email")
		if err != nil || emailCookie.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		email := emailCookie.Value

		var req struct {
			Bio       string `json:"bio"`
			BioPublic bool   `json:"bio_public"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		acctRaw, err := dbpkg.Get(dbConn, "accounts", email)
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
		if err := dbpkg.Set(dbConn, "accounts", email, string(acctBytes)); err != nil {
			http.Error(w, "Failed to update", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Bio updated successfully",
		})
	})
	http.HandleFunc("/user/", func(w http.ResponseWriter, r *http.Request) {
		println("User requeested")
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			println("redirect from /user/")
			http.Redirect(w, r, "/auth?toast=1&from="+r.URL.Path, http.StatusFound)
			return
		}

		userName := strings.TrimPrefix(r.URL.Path, "/user/")
		userName = strings.TrimSpace(userName)
    
		allAccounts, err := dbpkg.GetAll(dbConn, "accounts")
		if err != nil {
			http.Error(w, "Error loading users", http.StatusInternalServerError)
			return
		}

		var email string
		for userEmail, acctData := range allAccounts {
			var tempAcct map[string]interface{}
			if err := json.Unmarshal([]byte(acctData), &tempAcct); err != nil {
				continue
			}
			if name, ok := tempAcct["name"].(string); ok && name == userName {
				email = userEmail
				break
			}
		}

		if email == "" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		acctRaw, err := dbpkg.Get(dbConn, "accounts", email)
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
		if emailCookie, err := r.Cookie("email"); err == nil {
			currentUserEmail = emailCookie.Value
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
		userImg := ""
		if userName != "" {
			userImg = fmt.Sprintf("https://api.dicebear.com/9.x/big-smile/svg?seed=%s", userName)
		} else {
			// Fallback 2 mail
			userImg = fmt.Sprintf("https://api.dicebear.com/9.x/big-smile/svg?seed=%s", email)
		}

		data := map[string]interface{}{
			"Name":            userName,
			"Email":           email,
			"Bio":             userBio,
			"Img":             userImg,
			"IsOwnProfile":    isOwnProfile,
			"BioPublic":       bioPublic,
			"ShowBio":         showBio,
			"PageTitle":       fmt.Sprintf("%s - Profile", userName),
			"IsAuthenticated": true,
		}

		tmpl, err := htmltmpl.ParseFiles("components/user/profile.html", "components/header/header.html", "components/footer/footer.html")
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "user_profile", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})
}
