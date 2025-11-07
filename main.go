package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	dbpkg "sudocrypt25/db"
	handlers "sudocrypt25/handlers"
	routes "sudocrypt25/routes"

	_ "github.com/mattn/go-sqlite3"
)

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
	if err != nil {
		log.Fatal(err)
	}
	err = dbpkg.InitDB(dbConn)
	if err != nil {
		log.Fatal(err)
	}
	admins := handlers.NewAdmins(os.Getenv("ADMIN_EMAILS"))
	routes.InitRoutes(dbConn, admins)
	socketPath := os.Getenv("SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/tmp/sudocrypt25.sock"
	}
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		ln.Close()
		os.Remove(socketPath)
	}()
	os.Chmod(socketPath, 0666)
	log.Println("listening on unix socket:" + socketPath)
	log.Fatal(http.Serve(ln, nil))
}
