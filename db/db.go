package db

import (
	"database/sql"
	"fmt"
	"time"
)

func InitDB(d *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
	email TEXT PRIMARY KEY,
	data TEXT,
	created_at INTEGER
);
CREATE TABLE IF NOT EXISTS pending_signups (
	email TEXT PRIMARY KEY,
	data TEXT,
	created_at INTEGER
);
CREATE TABLE IF NOT EXISTS emails (
	email TEXT PRIMARY KEY,
	created_at INTEGER
);
CREATE TABLE IF NOT EXISTS leaderboard (
	email TEXT PRIMARY KEY,
	data TEXT,
	created_at INTEGER
);

`
	_, err := d.Exec(schema)
	return err
}

func Set(d *sql.DB, namespace, key, value string) error {
	now := time.Now().Unix()
	switch namespace {
	case "accounts", "registration", "users":
		_, err := d.Exec(`INSERT OR REPLACE INTO users(email, data, created_at) VALUES(?,?,?)`, key, value, now)
		return err
	case "pending_signup":
		_, err := d.Exec(`INSERT OR REPLACE INTO pending_signups(email, data, created_at) VALUES(?,?,?)`, key, value, now)
		return err
	case "emails":
		_, err := d.Exec(`INSERT OR REPLACE INTO emails(email, created_at) VALUES(?,?)`, key, now)
		return err
	case "leaderboard":
		_, err := d.Exec(`INSERT OR REPLACE INTO leaderboard(email, data, created_at) VALUES(?,?,?)`, key, value, now)
		return err
	default:
		_, err := d.Exec(`INSERT OR REPLACE INTO users(email, data, created_at) VALUES(?,?,?)`, key, value, now)
		return err
	}
}

func Get(d *sql.DB, namespace, key string) (string, error) {
	var query string
	switch namespace {
	case "accounts", "registration", "users":
		query = `SELECT data FROM users WHERE email = ?`
	case "pending_signup":
		query = `SELECT data FROM pending_signups WHERE email = ?`
	case "emails":
		query = `SELECT created_at FROM emails WHERE email = ?`
	case "leaderboard":
		query = `SELECT data FROM leaderboard WHERE email = ?`
	default:
		query = `SELECT data FROM users WHERE email = ?`
	}
	row := d.QueryRow(query, key)
	var out string
	switch namespace {
	case "emails":
		var createdAt sql.NullInt64
		if err := row.Scan(&createdAt); err != nil {
			return "", err
		}
		if createdAt.Valid {
			out = fmt.Sprintf("%d", createdAt.Int64)
		} else {
			out = ""
		}
		return out, nil
	default:
		var data sql.NullString
		if err := row.Scan(&data); err != nil {
			return "", err
		}
		if data.Valid {
			return data.String, nil
		}
		return "", fmt.Errorf("no rows in result set")
	}
}

func Delete(d *sql.DB, namespace, key string) error {
	switch namespace {
	case "accounts", "registration", "users":
		_, err := d.Exec(`DELETE FROM users WHERE email = ?`, key)
		return err
	case "pending_signup":
		_, err := d.Exec(`DELETE FROM pending_signups WHERE email = ?`, key)
		return err
	case "emails":
		_, err := d.Exec(`DELETE FROM emails WHERE email = ?`, key)
		return err
	case "leaderboard":
		_, err := d.Exec(`DELETE FROM leaderboard WHERE email = ?`, key)
		return err
	default:
		_, err := d.Exec(`DELETE FROM users WHERE email = ?`, key)
		return err
	}
}

func GetAll(d *sql.DB, namespace string) (map[string]string, error) {
	res := map[string]string{}
	var rows *sql.Rows
	var err error
	switch namespace {
	case "accounts", "registration", "users":
		rows, err = d.Query(`SELECT email, data FROM users`)

	case "pending_signup":
		rows, err = d.Query(`SELECT email, data FROM pending_signups`)
	case "emails":
		rows, err = d.Query(`SELECT email, created_at FROM emails`)
	case "leaderboard":
		rows, err = d.Query(`SELECT email, data FROM leaderboard`)
	default:
		return res, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var v sql.NullString
		if namespace == "emails" {
			var createdAt sql.NullInt64
			if err := rows.Scan(&k, &createdAt); err != nil {
				return nil, err
			}
			if createdAt.Valid {
				res[k] = fmt.Sprintf("%d", createdAt.Int64)
			} else {
				res[k] = ""
			}
		} else {
			if err := rows.Scan(&k, &v); err != nil {
				return nil, err
			}
			if v.Valid {
				res[k] = v.String
			} else {
				res[k] = ""
			}
		}
	}
	return res, nil
}
