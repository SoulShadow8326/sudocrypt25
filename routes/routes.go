package routes

import (
	"database/sql"
	"fmt"
	"net/http"

	"sudocrypt25/handlers"
	"sudocrypt25/template"
)

func InitRoutes(dbConn *sql.DB) {
	handlers.InitHandlers()
	template.InitTemplates()
	http.Handle("/components/", http.StripPrefix("/components/", http.FileServer(http.Dir("components"))))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("components/assets"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		td := template.TemplateData{PageTitle: "Home", CurrentPath: r.URL.Path}
		if err := template.RenderTemplate(w, "landing", td); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		td := template.TemplateData{PageTitle: "Auth", CurrentPath: r.URL.Path}
		if err := template.RenderTemplate(w, "auth", td); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/auth/", func(w http.ResponseWriter, r *http.Request) {
		td := template.TemplateData{PageTitle: "Auth", CurrentPath: r.URL.Path}
		if err := template.RenderTemplate(w, "auth", td); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		td := template.TemplateData{PageTitle: "Not Found", CurrentPath: r.URL.Path}
		if err := template.RenderFile(w, "components/404.html", td); err != nil {
			fmt.Printf("render /404 failed: %v\n", err)
			http.ServeFile(w, r, "components/404.html")
		}
	})
		http.HandleFunc("/404/", func(w http.ResponseWriter, r *http.Request) {
		td := template.TemplateData{PageTitle: "Not Found", CurrentPath: r.URL.Path}
		if err := template.RenderFile(w, "components/404.html", td); err != nil {
			fmt.Printf("render /404 failed: %v\n", err)
			http.ServeFile(w, r, "components/404.html")
		}
	})
	http.HandleFunc("/send_otp", handlers.SendOtpHandler(dbConn))
	http.HandleFunc("/api/auth", handlers.ApiAuthHandler(dbConn))
}
