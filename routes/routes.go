package routes

import (
	"database/sql"
	"fmt"
	htmltmpl "html/template"
	"net/http"
	"os"
	"time"

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
		td := template.TemplateData{PageTitle: "Time Gate", CurrentPath: r.URL.Path, IsAuthenticated: auth}
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
		td := template.TemplateData{PageTitle: "Play", CurrentPath: r.URL.Path, IsAuthenticated: auth}
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
}
