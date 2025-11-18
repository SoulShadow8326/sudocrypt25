package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	htmltmpl "html/template"
	"net/http"
	"os"
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
		type sRaw struct {
			ImageUrl string `json:"imageUrl"`
			Alt      string `json:"alt"`
			Link     string `json:"link"`
			Height   string `json:"height"`
		}
		if b, err := os.ReadFile("components/assets/sponsors.json"); err == nil {
			var arr []sRaw
			if json.Unmarshal(b, &arr) == nil {
				var out []template.Sponsor
				for _, it := range arr {
					out = append(out, template.Sponsor{ImageURL: it.ImageUrl, Link: it.Link, Alt: it.Alt, Height: it.Height})
				}
				td.Sponsors = out
			}
		}
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
		phase := handlers.EventPhase()
		isOver := phase == 1
		isBefore := phase == -1
		td := template.TemplateData{PageTitle: "Time Gate", CurrentPath: r.URL.Path, TimeGateStart: os.Getenv("TIMEGATE_START"), TimeGateEnd: os.Getenv("TIMEGATE_END"), IsAuthenticated: auth, IsEventOver: isOver, IsBeforeStart: isBefore}
		if err := template.RenderFile(w, "components/timegate.html", td); err != nil {
			http.ServeFile(w, r, "components/timegate.html")
		}
	})
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		_ = handlers.DeleteSession(dbConn, r)
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
			email, err := handlers.GetEmailFromRequest(dbConn, r)
			if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
				http.Redirect(w, r, "/timegate?toast=1&from=/play", http.StatusFound)
				return
			}
		}
		td := template.TemplateData{PageTitle: "Play", CurrentPath: r.URL.Path, IsAuthenticated: auth, ShowAnnouncements: true}
		if email, err := handlers.GetEmailFromRequest(dbConn, r); err == nil && email != "" {
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
						td.LevelAnswerHash = lvl.PublicHash
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
			email, err := handlers.GetEmailFromRequest(dbConn, r)
			if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
				http.Redirect(w, r, "/timegate?toast=1&from=/leaderboard", http.StatusFound)
				return
			}
		}
		td := template.TemplateData{PageTitle: "Leaderboard", CurrentPath: r.URL.Path, IsAuthenticated: auth}
		if html, err := handlers.GenerateLeaderboardHTML(dbConn, admins); err == nil {
			td.LeaderboardHTML = htmltmpl.HTML(html)
		}
		if err := template.RenderTemplate(w, "leaderboard", td); err != nil {
			http.ServeFile(w, r, "components/leaderboard/leaderboard.html")
		}
	})
	http.HandleFunc("/set_level", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		email, err := handlers.GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handlers.SetLevelHandler(dbConn)(w, r)
	})
	http.HandleFunc("/delete_level", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		email, err := handlers.GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handlers.DeleteLevelHandler(dbConn)(w, r)
	})
	http.HandleFunc("/submit", handlers.SubmitHandler(dbConn))
	http.HandleFunc("/api/play/current", handlers.CurrentLevelHandler(dbConn))
	http.HandleFunc("/api/leaderboard", handlers.LeaderboardAPIHandler(dbConn, admins))
	http.HandleFunc("/api/levels", handlers.LevelsListHandler(dbConn))
	http.HandleFunc("/api/admin/announcements/set", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		email, err := handlers.GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handlers.SetAnnouncementHandler(dbConn, admins)(w, r)
	})
	http.HandleFunc("/api/admin/announcements/delete", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		email, err := handlers.GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handlers.DeleteAnnouncementHandler(dbConn, admins)(w, r)
	})
	http.HandleFunc("/api/admin/user/progress", handlers.AdminUpdateUserProgressHandler(dbConn, admins))
	http.HandleFunc("/api/admin/users", handlers.AdminListUsersHandler(dbConn, admins))
	http.HandleFunc("/api/admin/user", handlers.AdminUserActionHandler(dbConn, admins))
	http.HandleFunc("/api/messages", handlers.ListMessagesHandler(dbConn, admins))
	http.HandleFunc("/api/admin/messages/mark_read", handlers.MarkMessagesReadHandler(dbConn, admins))
	http.HandleFunc("/api/message/send", func(w http.ResponseWriter, r *http.Request) {
		handlers.SendMessageHandler(dbConn, admins)(w, r)
	})
	http.HandleFunc("/api/me", handlers.MeHandler(dbConn, admins))
	http.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		email, err := handlers.GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handlers.LogsHandler(dbConn, admins)(w, r)
	})
	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/auth?toast=1&from=/admin", http.StatusFound)
			return
		}
		auth := true
		if email, err := handlers.GetEmailFromRequest(dbConn, r); err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
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
		if email, err := handlers.GetEmailFromRequest(dbConn, r); err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
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
	http.HandleFunc("/api/admin/ai_leads", handlers.ToggleAILeadsHandler(dbConn, admins))

	http.HandleFunc("/api/user/update_bio", handlers.UpdateBioHandler(dbConn))
	http.HandleFunc("/profile/", handlers.UserProfileHandler(dbConn, admins))

	http.HandleFunc("/api/attempt_logs", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil || c.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		email, err := handlers.GetEmailFromRequest(dbConn, r)
		if err != nil || email == "" || admins == nil || !admins.IsAdmin(email) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handlers.AttemptLog(dbConn, admins)(w, r)
	})
}
