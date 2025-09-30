package template

import (
	"html/template"
	"net/http"
	"os"
)

var templates *template.Template

type TemplateData struct {
	PageTitle       string
	CurrentPath     string
	User            map[string]interface{}
	TimeGateStart   string
	IsAuthenticated bool
}

func InitTemplates() error {
	var err error
	templates, err = template.ParseGlob("components/*/*.html")
	if err != nil {
		return err
	}
	extra, err := template.ParseGlob("components/*.html")
	if err == nil {
		for _, t := range extra.Templates() {
			templates.AddParseTree(t.Name(), t.Tree)
		}
	}
	return nil
}

func RenderTemplate(w http.ResponseWriter, name string, data TemplateData) error {
	if templates == nil {
		if err := InitTemplates(); err != nil {
			return err
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return templates.ExecuteTemplate(w, name, data)
}

func RenderFile(w http.ResponseWriter, filePath string, data TemplateData) error {
	if templates == nil {
		if err := InitTemplates(); err != nil {
			return err
		}
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	t, err := template.ParseGlob("components/*/*.html")
	if err != nil {
		return err
	}
	extra, err := template.ParseGlob("components/*.html")
	if err == nil {
		for _, tt := range extra.Templates() {
			t.AddParseTree(tt.Name(), tt.Tree)
		}
	}
	if _, err := t.New("__file__").Parse(string(b)); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.ExecuteTemplate(w, "__file__", data)
}
