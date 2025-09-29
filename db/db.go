package db

import (
	"database/sql"
	"time"
)

func InitDB(d *sql.DB) error {
	schema := `CREATE TABLE IF NOT EXISTS kv (namespace TEXT, key TEXT PRIMARY KEY, value TEXT, created_at INTEGER);`
	_, err := d.Exec(schema)
	return err
}

func Set(d *sql.DB, namespace, key, value string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT OR REPLACE INTO kv(namespace, key, value, created_at) VALUES(?,?,?,?)`, namespace, key, value, time.Now().Unix())
	if err != nil {
		return err
	}
	return tx.Commit()
}

func Get(d *sql.DB, namespace, key string) (string, error) {
	row := d.QueryRow(`SELECT value FROM kv WHERE namespace=? AND key=?`, namespace, key)
	var v string
	err := row.Scan(&v)
	if err != nil {
		return "", err
	}
	return v, nil
}

func Delete(d *sql.DB, namespace, key string) error {
	_, err := d.Exec(`DELETE FROM kv WHERE namespace=? AND key=?`, namespace, key)
	return err
}

func GetAll(d *sql.DB, namespace string) (map[string]string, error) {
	rows, err := d.Query(`SELECT key, value FROM kv WHERE namespace=?`, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		out[k] = v
	}
	return out, nil
}
