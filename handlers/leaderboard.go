package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	dbpkg "sudocrypt25/db"
)

type leaderboard struct {
	Email  string  `json:"email"`
	Name   string  `json:"name"`
	Points int     `json:"points"`
	Time   float64 `json:"time"`
}

func ProcessLeaderboard(dbConn *sql.DB) error {
	sample := leaderboard{
		Email:  "exun@dpsrkp.net",
		Name:   "Exun",
		Points: 9999,
		Time:   float64(time.Now().Unix()),
	}
	b, _ := json.Marshal(sample)
	if err := dbpkg.Set(dbConn, "leaderboard", sample.Email, string(b)); err != nil {
		return err
	}

	data, err := dbpkg.GetAll(dbConn, "leaderboard")
	if err != nil {
		return err
	}
	entries := []leaderboard{}
	for _, v := range data {
		var e leaderboard
		if err := json.Unmarshal([]byte(v), &e); err != nil {
			log.Println("skip", err)
			continue
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Points == entries[j].Points {
			return entries[i].Time < entries[j].Time
		}
		return entries[i].Points > entries[j].Points
	})
	fmt.Println("leaderboard:")
	for i, e := range entries {
		fmt.Printf("%d. %s (%s) - %d points\n", i+1, e.Name, e.Email, e.Points)
	}
	return nil
}

func GenerateLeaderboardHTML(dbConn *sql.DB, admins *Admins) (string, error) {
	data, err := dbpkg.GetAll(dbConn, "leaderboard")
	if err != nil {
		return "", err
	}
	entries := []leaderboard{}
	for _, v := range data {
		var e leaderboard
		if err := json.Unmarshal([]byte(v), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Points == entries[j].Points {
			return entries[i].Time < entries[j].Time
		}
		return entries[i].Points > entries[j].Points
	})

	cardBytes, err := os.ReadFile("components/leaderboard/card.html")
	if err != nil {
		return "", err
	}
	cardTpl := string(cardBytes)
	var sb strings.Builder
	rankCounter := 1
	for _, e := range entries {
		if admins != nil && admins.IsAdmin(e.Email) {
			continue
		}
		rank := fmt.Sprintf("%d", rankCounter)
		level := fmt.Sprintf("%d", e.Points)
		emailText := ""
		item := cardTpl
		item = strings.ReplaceAll(item, "{rank}", rank)
		item = strings.ReplaceAll(item, "{name}", template.HTMLEscapeString(e.Name))
		item = strings.ReplaceAll(item, "{level}", level)
		item = strings.ReplaceAll(item, "{email_text}", emailText)
		emailDisplay := template.HTMLEscapeString(e.Email)
		item = strings.ReplaceAll(item, "{email}", emailDisplay)
		sb.WriteString(item)
		rankCounter++
	}
	return sb.String(), nil
}

func LeaderboardAPIHandler(dbConn *sql.DB, admins *Admins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		sortBy := q.Get("sort")
		order := strings.ToLower(q.Get("order"))
		if order != "asc" && order != "desc" {
			order = "desc"
		}

		data, err := dbpkg.GetAll(dbConn, "leaderboard")
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		entries := []leaderboard{}
		for _, v := range data {
			var e leaderboard
			if err := json.Unmarshal([]byte(v), &e); err != nil {
				continue
			}
			if admins != nil && admins.IsAdmin(e.Email) {
				continue
			}
			entries = append(entries, e)
		}

		cmp := func(i, j int) bool {
			switch sortBy {
			case "time":
				if entries[i].Time == entries[j].Time {
					return entries[i].Points > entries[j].Points
				}
				if order == "asc" {
					return entries[i].Time < entries[j].Time
				}
				return entries[i].Time > entries[j].Time
			case "user":
				if order == "asc" {
					return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
				}
				return strings.ToLower(entries[i].Name) > strings.ToLower(entries[j].Name)
			default:
				if entries[i].Points == entries[j].Points {
					if order == "asc" {
						return entries[i].Time < entries[j].Time
					}
					return entries[i].Time > entries[j].Time
				}
				if order == "asc" {
					return entries[i].Points < entries[j].Points
				}
				return entries[i].Points > entries[j].Points
			}
		}
		sort.Slice(entries, cmp)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}
}
