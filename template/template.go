package template

import (
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

var templates *template.Template

type TemplateData struct {
	PageTitle         string
	CurrentPath       string
	User              map[string]interface{}
	TimeGateStart     string
	TimeGateEnd       string
	IsAuthenticated   bool
	IsEventOver       bool
	IsBeforeStart     bool
	ShowAnnouncements bool
	LeaderboardHTML   template.HTML
	LevelsHTML        template.HTML
	LevelsData        template.JS
	LevelNum          string
	LevelAnswerHash   string
	UserEmail         string
	SrcHint           template.HTML
	Sponsors          []Sponsor
}

type Sponsor struct {
	ImageURL string
	Link     string
	Alt      string
	Height   string
}

func InitTemplates() error {
	var files []string
	err := filepath.WalkDir("components", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	t, err := template.ParseFiles(files...)
	if err != nil {
		return err
	}
	templates = t
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
