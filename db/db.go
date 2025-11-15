package db

import (
	"database/sql"
	"encoding/json"
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
CREATE TABLE IF NOT EXISTS attempt_logs ( 
	email TEXT PRIMARY KEY,
	logs TEXT
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
CREATE TABLE IF NOT EXISTS levels (
    id TEXT PRIMARY KEY,
    data TEXT,
    created_at INTEGER
);
CREATE TABLE IF NOT EXISTS announcements (
	id TEXT PRIMARY KEY,
	data TEXT,
	created_at INTEGER
);
CREATE TABLE IF NOT EXISTS hints (
	level_id TEXT,
	hint_id TEXT,
	data TEXT,
	created_at INTEGER,
	PRIMARY KEY (level_id, hint_id)
);
CREATE TABLE IF NOT EXISTS messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	data TEXT,
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

CREATE TABLE IF NOT EXISTS sessions (
    session_id TEXT PRIMARY KEY,
    email TEXT,
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
		var lb map[string]interface{}
		if err := json.Unmarshal([]byte(value), &lb); err != nil {
			return err
		}
		var existing sql.NullString
		if err := d.QueryRow(`SELECT data FROM users WHERE email = ?`, key).Scan(&existing); err != nil && err != sql.ErrNoRows {
			return err
		}
		var user map[string]interface{}
		if existing.Valid {
			json.Unmarshal([]byte(existing.String), &user)
		} else {
			user = map[string]interface{}{"email": key}
		}
		if name, ok := lb["name"].(string); ok && name != "" {
			user["name"] = name
		}
		if points, ok := lb["points"]; ok {
			user["points"] = points
		}
		if t, ok := lb["time"]; ok {
			user["time"] = t
		}
		ub, _ := json.Marshal(user)
		_, err := d.Exec(`INSERT OR REPLACE INTO users(email, data, created_at) VALUES(?,?,?)`, key, string(ub), now)
		return err
	case "levels":
		_, err := d.Exec(`INSERT OR REPLACE INTO levels(id, data, created_at) VALUES(?,?,?)`, key, value, now)
		return err
	case "sessions":
		_, err := d.Exec(`INSERT OR REPLACE INTO sessions(session_id, email, created_at) VALUES(?,?,?)`, key, value, now)
		return err
	case "announcements":
		_, err := d.Exec(`INSERT OR REPLACE INTO announcements(id, data, created_at) VALUES(?,?,?)`, key, value, now)
		return err
	case "attempt_logs":
		_, err := d.Exec(`INSERT OR REPLACE INTO attempt_logs(email, logs) VALUES(?,?)`, key, value)
		return err
	case "hints":
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid hints key")
		}
		levelID := parts[0]
		hintID := parts[1]
		_, err := d.Exec(`INSERT OR REPLACE INTO hints(level_id, hint_id, data, created_at) VALUES(?,?,?,?)`, levelID, hintID, value, now)
		return err
	case "logs":
		parts := strings.SplitN(value, "|", 3)
		ns := ""
		ev := ""
		dat := ""
		if len(parts) > 0 {
			ns = parts[0]
		}
		if len(parts) > 1 {
			ev = parts[1]
		}
		if len(parts) > 2 {
			dat = parts[2]
		}
		_, err := d.Exec(`INSERT INTO logs(namespace, key, event, data, created_at) VALUES(?,?,?,?,?)`, ns, key, ev, dat, now)
		return err
	case "messages":
		v := strings.TrimSpace(value)
		var dataStr string
		if strings.HasPrefix(v, "{") || strings.HasPrefix(v, "[") {
			dataStr = v
		} else {
			parts := strings.SplitN(value, "|", 5)
			obj := map[string]interface{}{"from": "", "to": "", "level_id": "", "type": "", "content": ""}
			if len(parts) > 0 {
				obj["from"] = parts[0]
			}
			if len(parts) > 1 {
				obj["to"] = parts[1]
			}
			if len(parts) > 2 {
				obj["level_id"] = parts[2]
			}
			if len(parts) > 3 {
				obj["type"] = parts[3]
			}
			if len(parts) > 4 {
				obj["content"] = parts[4]
			}
			b, _ := json.Marshal(obj)
			dataStr = string(b)
		}
		_, err := d.Exec(`INSERT INTO messages(data, created_at, read) VALUES(?,?,?)`, dataStr, now, 0)
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
		query = `SELECT data FROM users WHERE email = ?`
	case "levels":
		query = `SELECT data FROM levels WHERE id = ?`
	case "sessions":
		query = `SELECT email FROM sessions WHERE session_id = ?`
	case "announcements":
		query = `SELECT data FROM announcements WHERE id = ?`
	case "attempt_logs":
		query = `SELECT logs FROM attempt_logs WHERE email = ?`
	case "hints":
		rows, err := d.Query(`SELECT hint_id, data FROM hints WHERE level_id = ? ORDER BY created_at ASC`, key)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		m := map[string]string{}
		for rows.Next() {
			var id string
			var data sql.NullString
			if err := rows.Scan(&id, &data); err != nil {
				return "", err
			}
			if data.Valid {
				m[id] = data.String
			} else {
				m[id] = ""
			}
		}
		b, err := json.Marshal(m)
		if err != nil {
			return "", err
		}
		return string(b), nil
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
	if strings.HasPrefix(namespace, "messages/") {
		email := strings.TrimPrefix(namespace, "messages/")
		_, err := d.Exec(`DELETE FROM messages WHERE (json_extract(data, '$.from') = ? OR json_extract(data, '$.to') = ?) AND json_extract(data, '$.type') = ?`, email, email, key)
		return err
	}
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
		var existing sql.NullString
		if err := d.QueryRow(`SELECT data FROM users WHERE email = ?`, key).Scan(&existing); err != nil {
			return err
		}
		if existing.Valid {
			var user map[string]interface{}
			json.Unmarshal([]byte(existing.String), &user)
			delete(user, "points")
			delete(user, "time")
			ub, _ := json.Marshal(user)
			_, err := d.Exec(`INSERT OR REPLACE INTO users(email, data, created_at) VALUES(?,?,?)`, key, string(ub), time.Now().Unix())
			return err
		}
		return nil
	case "levels":
		_, err := d.Exec(`DELETE FROM levels WHERE id = ?`, key)
		return err
	case "sessions":
		_, err := d.Exec(`DELETE FROM sessions WHERE session_id = ?`, key)
		return err
	case "announcements":
		_, err := d.Exec(`DELETE FROM announcements WHERE id = ?`, key)
		return err
	case "attempt_logs":
		_, err := d.Exec(`DELETE FROM attempt_logs WHERE email = ?`, key)
		return err
	case "hints":
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid hints key")
		}
		levelID := parts[0]
		hintID := parts[1]
		_, err := d.Exec(`DELETE FROM hints WHERE level_id = ? AND hint_id = ?`, levelID, hintID)
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
		rows, err = d.Query(`SELECT email, data FROM users`)
	case "levels":
		rows, err = d.Query(`SELECT id, data FROM levels`)
	case "announcements":
		rows, err = d.Query(`SELECT id, data FROM announcements`)
	case "messages":
		rows, err = d.Query(`SELECT id, data, created_at, read FROM messages ORDER BY created_at ASC`)
	case "logs":
		rows, err = d.Query(`SELECT id, namespace, key, event, data, created_at FROM logs ORDER BY created_at ASC`)
	case "hints":
		rows, err = d.Query(`SELECT level_id || '/' || hint_id as key, data FROM hints ORDER BY created_at ASC`)
	case "attempt_logs":
		rows, err = d.Query(`SELECT email, logs FROM attempt_logs`)
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
		switch namespace {
		case "emails":
			var createdAt sql.NullInt64
			if err := rows.Scan(&k, &createdAt); err != nil {
				return nil, err
			}
			if createdAt.Valid {
				res[k] = fmt.Sprintf("%d", createdAt.Int64)
			} else {
				res[k] = ""
			}
		case "messages":
			var id int
			var data sql.NullString
			var createdAt sql.NullInt64
			var read sql.NullInt64
			if err := rows.Scan(&id, &data, &createdAt, &read); err != nil {
				return nil, err
			}
			key := fmt.Sprintf("%d", id)
			payload := map[string]interface{}{}
			if data.Valid {
				if err := json.Unmarshal([]byte(data.String), &payload); err != nil {
					payload = map[string]interface{}{"content": data.String}
				}
			}
			payload["id"] = id
			payload["created_at"] = createdAt.Int64
			payload["read"] = read.Int64
			b, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			res[key] = string(b)
		case "logs":
			var id int
			var ns, keyCol, event, data sql.NullString
			var createdAt sql.NullInt64
			if err := rows.Scan(&id, &ns, &keyCol, &event, &data, &createdAt); err != nil {
				return nil, err
			}
			keyLogs := fmt.Sprintf("%d", id)
			entry := struct {
				ID        int    `json:"id"`
				Namespace string `json:"namespace"`
				Key       string `json:"key"`
				Event     string `json:"event"`
				Data      string `json:"data"`
				CreatedAt int64  `json:"created_at"`
			}{
				ID: id,
				Namespace: func() string {
					if ns.Valid {
						return ns.String
					}
					return ""
				}(),
				Key: func() string {
					if keyCol.Valid {
						return keyCol.String
					}
					return ""
				}(),
				Event: func() string {
					if event.Valid {
						return event.String
					}
					return ""
				}(),
				Data: func() string {
					if data.Valid {
						return data.String
					}
					return ""
				}(),
				CreatedAt: createdAt.Int64,
			}
			b, err := json.Marshal(entry)
			if err != nil {
				return nil, err
			}
			res[keyLogs] = string(b)
		default:
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
