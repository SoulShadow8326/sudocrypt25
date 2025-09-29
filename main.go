package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	dbpkg "sudocrypt25/db"
	routes "sudocrypt25/routes"

	_ "github.com/mattn/go-sqlite3"
)

var dbConn *sql.DB

func main() {
	var err error
	dbConn, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	err = dbpkg.InitDB(dbConn)
	if err != nil {
		log.Fatal(err)
	}
	routes.InitRoutes(dbConn)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
