package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"sudocrypt25/db"
	dbpkg "sudocrypt25/db"
	handlers "sudocrypt25/handlers"
	routes "sudocrypt25/routes"

	_ "github.com/mattn/go-sqlite3"
)

type lbe struct {
	Email  string  `json:"email"`
	Name   string  `json:"name"`
	Points int     `json:"points"`
	Time   float64 `json:"time"`
}

var dbConn *sql.DB

func main() {
	if _, err := os.Stat(".env"); err == nil {
		f, err := os.Open(".env")
		if err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					continue
				}
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
					v = v[1 : len(v)-1]
				}
				os.Setenv(k, v)
			}
			if err := scanner.Err(); err != nil {
				fmt.Println("warning: error reading .env:", err)
			}
			f.Close()
		}
	}
	dbConn, err := sql.Open("sqlite3", "./data.db")
	sample := lbe{
		Email:  "fuck@me.com",
		Name:   "ad",
		Points: 5,
		Time:   float64(time.Now().Unix()),
	}
	b, _ := json.Marshal(sample)
	if err := db.Set(dbConn, "leaderboard", sample.Email, string(b)); err != nil {
		log.Fatal((err))
	}
	data, err := db.GetAll(dbConn, "leaderboard")
	if err != nil {
		log.Fatal((err))
	}
	entries := []lbe{}
	for _, v := range data {
		var e lbe
		if err := json.Unmarshal([]byte(v), &e); err != nil {
			log.Println("skip", err)
			continue
		}
		entries = append(entries, e)
	} //sortering
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
	if err != nil {
		log.Fatal(err)
	}
	err = dbpkg.InitDB(dbConn)
	if err != nil {
		log.Fatal(err)
	}
	admins := handlers.NewAdmins(os.Getenv("ADMIN_EMAILS"))
	routes.InitRoutes(dbConn, admins)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
