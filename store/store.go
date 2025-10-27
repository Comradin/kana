package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	selectedRowsKey   = "selected_rows"
	autoProgressKey   = "auto_progress"
	databaseFilePerm  = 0o644
	databaseDirPerm   = 0o755
	defaultOpenTimout = 5 * time.Second
)

// Store provides persisted access to user settings and kana statistics.
type Store struct {
	db *sql.DB
}

// KanaStats represents the aggregated statistics for a single kana character.
type KanaStats struct {
	Char         string
	CorrectCount int
	MissCount    int
	Streak       int
}

// Open initialises the SQLite database located at path and applies migrations.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store: database path is required")
	}

	if err := ensureDir(path); err != nil {
		return nil, fmt.Errorf("store: ensure directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open database: %w", err)
	}

	// Apply a short timeout so migrations don't hang if the DB is locked.
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpenTimout)
	defer cancel()

	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close releases the underlying database resources.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SelectedRows loads the stored row identifiers. Returns nil slice if unset.
func (s *Store) SelectedRows() ([]string, error) {
	value, err := s.getSetting(selectedRowsKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []string
	if err := json.Unmarshal([]byte(value), &rows); err != nil {
		return nil, fmt.Errorf("store: decode selected rows: %w", err)
	}
	return rows, nil
}

// SaveSelectedRows persists the provided selection. Passing nil clears the value.
func (s *Store) SaveSelectedRows(rows []string) error {
	if rows == nil {
		return s.deleteSetting(selectedRowsKey)
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("store: encode selected rows: %w", err)
	}
	return s.setSetting(selectedRowsKey, string(payload))
}

// AutoProgress returns the persisted auto progression flag.
func (s *Store) AutoProgress() (bool, error) {
	value, err := s.getSetting(autoProgressKey)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch value {
	case "1", "true", "TRUE":
		return true, nil
	default:
		return false, nil
	}
}

// SaveAutoProgress toggles the auto progression flag.
func (s *Store) SaveAutoProgress(enabled bool) error {
	if enabled {
		return s.setSetting(autoProgressKey, "1")
	}
	return s.setSetting(autoProgressKey, "0")
}

// IncrementCorrect increments the correct counter and streak for the given kana.
func (s *Store) IncrementCorrect(char string) error {
	_, err := s.db.Exec(`
		INSERT INTO kana_stats (char, correct_count, miss_count, streak)
		VALUES (?, 1, 0, 1)
		ON CONFLICT(char) DO UPDATE SET
			correct_count = correct_count + 1,
			streak = streak + 1
	`, char)
	if err != nil {
		return fmt.Errorf("store: increment correct: %w", err)
	}
	return nil
}

// IncrementMiss increments the miss counter and resets the streak for the given kana.
func (s *Store) IncrementMiss(char string) error {
	_, err := s.db.Exec(`
		INSERT INTO kana_stats (char, correct_count, miss_count, streak)
		VALUES (?, 0, 1, 0)
		ON CONFLICT(char) DO UPDATE SET
			miss_count = miss_count + 1,
			streak = 0
	`, char)
	if err != nil {
		return fmt.Errorf("store: increment miss: %w", err)
	}
	return nil
}

// SetStreak updates the streak value for the given kana without affecting counters.
func (s *Store) SetStreak(char string, streak int) error {
	if streak < 0 {
		streak = 0
	}
	_, err := s.db.Exec(`
		INSERT INTO kana_stats (char, correct_count, miss_count, streak)
		VALUES (?, 0, 0, ?)
		ON CONFLICT(char) DO UPDATE SET
			streak = excluded.streak
	`, char, streak)
	if err != nil {
		return fmt.Errorf("store: set streak: %w", err)
	}
	return nil
}

// KanaStatistics returns the stats for all tracked characters.
func (s *Store) KanaStatistics() (map[string]KanaStats, error) {
	rows, err := s.db.Query(`
		SELECT char, correct_count, miss_count, streak
		FROM kana_stats
	`)
	if err != nil {
		return nil, fmt.Errorf("store: query kana stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]KanaStats)
	for rows.Next() {
		var ks KanaStats
		if err := rows.Scan(&ks.Char, &ks.CorrectCount, &ks.MissCount, &ks.Streak); err != nil {
			return nil, fmt.Errorf("store: scan kana stats: %w", err)
		}
		stats[ks.Char] = ks
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate kana stats: %w", err)
	}
	return stats, nil
}

func (s *Store) getSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *Store) setSetting(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("store: set setting %s: %w", key, err)
	}
	return nil
}

func (s *Store) deleteSetting(key string) error {
	_, err := s.db.Exec(`DELETE FROM settings WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("store: delete setting %s: %w", key, err)
	}
	return nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, databaseDirPerm)
}

func migrate(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`PRAGMA journal_mode = WAL;`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS kana_stats (
			char TEXT PRIMARY KEY,
			correct_count INTEGER NOT NULL DEFAULT 0,
			miss_count INTEGER NOT NULL DEFAULT 0,
			streak INTEGER NOT NULL DEFAULT 0
		);`,
	}

	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate statement failed: %w", err)
		}
	}
	return nil
}
