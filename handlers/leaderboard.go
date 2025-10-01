package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
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
