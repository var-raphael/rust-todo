package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type CrawlRecord struct {
	URL           string
	Status        string
	LayerUsed     string
	CrawledAt     time.Time
	FailureReason string
	ParentURL     string
	Cleaned       bool
	CleanedAt     *time.Time
}

func Init() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find home directory: %w", err)
	}

	dir := filepath.Join(home, ".phantomcrawl")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("could not create .phantomcrawl directory: %w", err)
	}

	dbPath := filepath.Join(dir, "state.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS crawled (
			url            TEXT PRIMARY KEY,
			status         TEXT,
			layer_used     TEXT,
			crawled_at     TIMESTAMP,
			failure_reason TEXT,
			parent_url     TEXT,
			cleaned        INTEGER DEFAULT 0,
			cleaned_at     TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Add columns to existing DBs that don't have them yet
	db.conn.Exec(`ALTER TABLE crawled ADD COLUMN cleaned INTEGER DEFAULT 0`)
	db.conn.Exec(`ALTER TABLE crawled ADD COLUMN cleaned_at TIMESTAMP`)

	return nil
}

func (db *DB) IsCrawled(url string) bool {
	var count int
	db.conn.QueryRow(
		"SELECT COUNT(*) FROM crawled WHERE url = ? AND status = 'crawled'",
		url,
	).Scan(&count)
	return count > 0
}

func (db *DB) MarkCrawled(url, layer string) error {
	_, err := db.conn.Exec(`
		INSERT OR REPLACE INTO crawled 
		(url, status, layer_used, crawled_at)
		VALUES (?, 'crawled', ?, ?)`,
		url, layer, time.Now(),
	)
	return err
}

func (db *DB) MarkFailed(url, reason, layer string) error {
	_, err := db.conn.Exec(`
		INSERT OR REPLACE INTO crawled
		(url, status, layer_used, crawled_at, failure_reason)
		VALUES (?, 'failed', ?, ?, ?)`,
		url, layer, time.Now(), reason,
	)
	return err
}

func (db *DB) MarkCleaned(url string) error {
	_, err := db.conn.Exec(`
		UPDATE crawled SET cleaned = 1, cleaned_at = ? WHERE url = ?`,
		time.Now(), url,
	)
	return err
}

func (db *DB) IsCleaned(url string) bool {
	var count int
	db.conn.QueryRow(
		"SELECT COUNT(*) FROM crawled WHERE url = ? AND cleaned = 1",
		url,
	).Scan(&count)
	return count > 0
}

func (db *DB) GetUncleaned() ([]string, error) {
	rows, err := db.conn.Query(
		"SELECT url FROM crawled WHERE status = 'crawled' AND cleaned = 0",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url string
		rows.Scan(&url)
		urls = append(urls, url)
	}
	return urls, nil
}

func (db *DB) GetFailed() ([]string, error) {
	rows, err := db.conn.Query(
		"SELECT url FROM crawled WHERE status = 'failed'",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url string
		rows.Scan(&url)
		urls = append(urls, url)
	}
	return urls, nil
}

func (db *DB) GetAllRecords() ([]CrawlRecord, error) {
	rows, err := db.conn.Query(`
		SELECT url, status, layer_used, crawled_at, 
		       COALESCE(failure_reason, ''), COALESCE(parent_url, ''),
		       cleaned, cleaned_at
		FROM crawled
		ORDER BY crawled_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CrawlRecord
	for rows.Next() {
		var r CrawlRecord
		var cleanedAt sql.NullTime
		var cleanedInt int
		err := rows.Scan(
			&r.URL, &r.Status, &r.LayerUsed, &r.CrawledAt,
			&r.FailureReason, &r.ParentURL,
			&cleanedInt, &cleanedAt,
		)
		if err != nil {
			continue
		}
		r.Cleaned = cleanedInt == 1
		if cleanedAt.Valid {
			r.CleanedAt = &cleanedAt.Time
		}
		records = append(records, r)
	}
	return records, nil
}

func (db *DB) GetStats() (total int, failed int, err error) {
	db.conn.QueryRow(
		"SELECT COUNT(*) FROM crawled WHERE status = 'crawled'",
	).Scan(&total)
	db.conn.QueryRow(
		"SELECT COUNT(*) FROM crawled WHERE status = 'failed'",
	).Scan(&failed)
	return total, failed, nil
}

func (db *DB) GetCleanStats() (cleaned int, pending int, err error) {
	db.conn.QueryRow(
		"SELECT COUNT(*) FROM crawled WHERE status = 'crawled' AND cleaned = 1",
	).Scan(&cleaned)
	db.conn.QueryRow(
		"SELECT COUNT(*) FROM crawled WHERE status = 'crawled' AND cleaned = 0",
	).Scan(&pending)
	return cleaned, pending, nil
}

func (db *DB) Reset() error {
	_, err := db.conn.Exec("DELETE FROM crawled")
	return err
}

func (db *DB) Close() {
	db.conn.Close()
}
