package db

import (
	"database/sql"
	"fmt"
	"strings"
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
CREATE TABLE IF NOT EXISTS levels (
    id TEXT PRIMARY KEY,
    data TEXT,
    created_at INTEGER
);
CREATE TABLE IF NOT EXISTS user_levels (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email TEXT,
	type TEXT,
	level INTEGER,
	advanced_at INTEGER
);
CREATE TABLE IF NOT EXISTS messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	from_email TEXT,
	to_email TEXT,
	level_id TEXT,
	type TEXT,
	content TEXT,
	created_at INTEGER,
	read INTEGER DEFAULT 0
);
CREATE TABLE IF NOT EXISTS logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	namespace TEXT,
	key TEXT,
	event TEXT,
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
	case "levels":
		_, err := d.Exec(`INSERT OR REPLACE INTO levels(id, data, created_at) VALUES(?,?,?)`, key, value, now)
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
	case "levels":
		query = `SELECT data FROM levels WHERE id = ?`
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
	case "levels":
		_, err := d.Exec(`DELETE FROM levels WHERE id = ?`, key)
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
	case "levels":
		rows, err = d.Query(`SELECT id, data FROM levels`)
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

func Log(d *sql.DB, namespace, key, event, data string) error {
	now := time.Now().Unix()
	_, err := d.Exec(`INSERT INTO logs(namespace, key, event, data, created_at) VALUES(?,?,?,?,?)`, namespace, key, event, data, now)
	return err
}

func AddMessage(d *sql.DB, fromEmail, toEmail, levelID, mtype, content string) error {
	now := time.Now().Unix()
	_, err := d.Exec(`INSERT INTO messages(from_email, to_email, level_id, type, content, created_at, read) VALUES(?,?,?,?,?,?,?)`, fromEmail, toEmail, levelID, mtype, content, now, 0)
	return err
}

func GetMessages(d *sql.DB, email string) (map[string]string, error) {
	res := map[string]string{}
	rows, err := d.Query(`SELECT id, from_email, to_email, level_id, type, content, created_at, read FROM messages WHERE from_email = ? OR to_email = ? ORDER BY created_at ASC`, email, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var from, to, levelID, mtype, content sql.NullString
		var createdAt sql.NullInt64
		var read sql.NullInt64
		if err := rows.Scan(&id, &from, &to, &levelID, &mtype, &content, &createdAt, &read); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%d", id)
		val := fmt.Sprintf(`{"id":%d,"from":"%s","to":"%s","level_id":"%s","type":"%s","content":"%s","created_at":%d,"read":%d}`,
			id,
			escapeStringNull(from),
			escapeStringNull(to),
			escapeStringNull(levelID),
			escapeStringNull(mtype),
			escapeStringNull(content),
			createdAt.Int64,
			read.Int64,
		)
		res[key] = val
	}
	return res, nil
}

func AddUserLevel(d *sql.DB, email, typ string, level int, ts int64) error {
	if ts == 0 {
		ts = time.Now().Unix()
	}
	_, err := d.Exec(`INSERT INTO user_levels(email, type, level, advanced_at) VALUES(?,?,?,?)`, email, typ, level, ts)
	return err
}

// helper used above to safely read sql.NullString
func escapeStringNull(ns sql.NullString) string {
	if ns.Valid {
		s := strings.ReplaceAll(ns.String, "\\", "\\\\")
		s = strings.ReplaceAll(s, "\"", "\\\"")
		return s
	}
	return ""
}
