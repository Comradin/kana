package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kana/memory"

	_ "modernc.org/sqlite"
)

const (
	selectedRowsKey    = "selected_rows"
	autoProgressKey    = "auto_progress"
	scoreLimitKey      = "score_limit"
	activeScriptsKey   = "active_scripts"
	katakanaOfferedKey = "katakana_offered"
	difficultyKey      = "difficulty"
	pointsFactorKey    = "points_factor"
	wordLevelKey       = "word_level"
	meaningLangKey     = "meaning_lang"
	bundledWordsKey    = "bundled_words"
	databaseFilePerm   = 0o644
	databaseDirPerm    = 0o755
	defaultOpenTimout  = 5 * time.Second
)

const DefaultScoreLimit = 1000

// DefaultWordLevel is the JLPT level words are played up to when unset (N5).
const DefaultWordLevel = 5

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

// WordStats is the aggregated statistics for one word reading. It is kept
// apart from KanaStats: the word か (mosquito) is not the character か.
type WordStats struct {
	Kana         string
	CorrectCount int
	MissCount    int
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

// ActiveScripts loads the stored script names. Returns nil if unset; the
// caller decides the default.
func (s *Store) ActiveScripts() ([]string, error) {
	value, err := s.getSetting(activeScriptsKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var scripts []string
	if err := json.Unmarshal([]byte(value), &scripts); err != nil {
		return nil, fmt.Errorf("store: decode active scripts: %w", err)
	}
	return scripts, nil
}

// SaveActiveScripts persists the active script names.
func (s *Store) SaveActiveScripts(scripts []string) error {
	payload, err := json.Marshal(scripts)
	if err != nil {
		return fmt.Errorf("store: encode active scripts: %w", err)
	}
	return s.setSetting(activeScriptsKey, string(payload))
}

// KatakanaOffered reports whether the learner was already offered katakana.
func (s *Store) KatakanaOffered() (bool, error) {
	value, err := s.getSetting(katakanaOfferedKey)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == "1", nil
}

// SaveKatakanaOffered persists whether katakana was offered.
func (s *Store) SaveKatakanaOffered(offered bool) error {
	value := "0"
	if offered {
		value = "1"
	}
	return s.setSetting(katakanaOfferedKey, value)
}

// Difficulty returns the stored level name, or "" when unset.
func (s *Store) Difficulty() (string, error) {
	value, err := s.getSetting(difficultyKey)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SaveDifficulty persists the level name.
func (s *Store) SaveDifficulty(name string) error {
	return s.setSetting(difficultyKey, name)
}

// PointsFactor returns the stored points setting, or 0 (Auto) when unset or
// unreadable.
func (s *Store) PointsFactor() (int, error) {
	value, err := s.getSetting(pointsFactorKey)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n, convErr := strconv.Atoi(value)
	if convErr != nil {
		return 0, nil
	}
	return n, nil
}

// SavePointsFactor persists the points setting.
func (s *Store) SavePointsFactor(p int) error {
	return s.setSetting(pointsFactorKey, strconv.Itoa(p))
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

// BundledWords reports whether the bundled word list is played in mixed
// mode. It is off unless saved as on.
func (s *Store) BundledWords() (bool, error) {
	value, err := s.getSetting(bundledWordsKey)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == "1", nil
}

// SaveBundledWords turns the bundled word list on or off.
func (s *Store) SaveBundledWords(enabled bool) error {
	if enabled {
		return s.setSetting(bundledWordsKey, "1")
	}
	return s.setSetting(bundledWordsKey, "0")
}

// ScoreLimit returns the configured score threshold for ending a session.
func (s *Store) ScoreLimit() (int, error) {
	value, err := s.getSetting(scoreLimitKey)
	if errors.Is(err, sql.ErrNoRows) || strings.TrimSpace(value) == "" {
		return DefaultScoreLimit, nil
	}
	if err != nil {
		return 0, err
	}
	limit, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return DefaultScoreLimit, nil
	}
	if limit < 0 {
		limit = 0
	}
	return limit, nil
}

// SaveScoreLimit persists the score limit. Values less than zero become endless (0).
func (s *Store) SaveScoreLimit(limit int) error {
	if limit < 0 {
		limit = 0
	}
	return s.setSetting(scoreLimitKey, strconv.Itoa(limit))
}

// WordLevel returns the JLPT level words are played up to (5 = N5 only).
// Unset or invalid values give DefaultWordLevel.
func (s *Store) WordLevel() (int, error) {
	value, err := s.getSetting(wordLevelKey)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultWordLevel, nil
	}
	if err != nil {
		return DefaultWordLevel, err
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(value))
	if convErr != nil || n < 1 || n > 5 {
		return DefaultWordLevel, nil
	}
	return n, nil
}

// SaveWordLevel persists the JLPT level (1–5).
func (s *Store) SaveWordLevel(level int) error {
	if level < 1 || level > 5 {
		return fmt.Errorf("store: word level %d out of range 1–5", level)
	}
	return s.setSetting(wordLevelKey, strconv.Itoa(level))
}

// MeaningLang returns the language meanings are shown in, or "" when unset:
// the app then follows the system language.
func (s *Store) MeaningLang() (string, error) {
	value, err := s.getSetting(meaningLangKey)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SaveMeaningLang persists the meaning language; "" means follow the system.
func (s *Store) SaveMeaningLang(lang string) error {
	return s.setSetting(meaningLangKey, lang)
}

// SaveKanaStats upserts the aggregated statistics for the provided kana.
func (s *Store) SaveKanaStats(char string, correctCount, missCount, streak int) error {
	if char == "" {
		return errors.New("store: char is required")
	}
	if correctCount < 0 {
		correctCount = 0
	}
	if missCount < 0 {
		missCount = 0
	}
	if streak < 0 {
		streak = 0
	}
	_, err := s.db.Exec(`
		INSERT INTO kana_stats (char, correct_count, miss_count, streak)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(char) DO UPDATE SET
			correct_count = excluded.correct_count,
			miss_count = excluded.miss_count,
			streak = excluded.streak
	`, char, correctCount, missCount, streak)
	if err != nil {
		return fmt.Errorf("store: save kana stats %s: %w", char, err)
	}
	return nil
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

// SaveWordStats upserts the aggregated statistics for one word reading
// (absolute values, like SaveKanaStats).
func (s *Store) SaveWordStats(kana string, correctCount, missCount int) error {
	if kana == "" {
		return errors.New("store: word reading is required")
	}
	if correctCount < 0 {
		correctCount = 0
	}
	if missCount < 0 {
		missCount = 0
	}
	_, err := s.db.Exec(`
		INSERT INTO word_stats (kana, correct_count, miss_count)
		VALUES (?, ?, ?)
		ON CONFLICT(kana) DO UPDATE SET
			correct_count = excluded.correct_count,
			miss_count = excluded.miss_count
	`, kana, correctCount, missCount)
	if err != nil {
		return fmt.Errorf("store: save word stats %s: %w", kana, err)
	}
	return nil
}

// WordStatistics returns the stats for all tracked word readings.
func (s *Store) WordStatistics() (map[string]WordStats, error) {
	rows, err := s.db.Query(`SELECT kana, correct_count, miss_count FROM word_stats`)
	if err != nil {
		return nil, fmt.Errorf("store: query word stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]WordStats)
	for rows.Next() {
		var ws WordStats
		if err := rows.Scan(&ws.Kana, &ws.CorrectCount, &ws.MissCount); err != nil {
			return nil, fmt.Errorf("store: scan word stats: %w", err)
		}
		stats[ws.Kana] = ws
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate word stats: %w", err)
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
		`CREATE TABLE IF NOT EXISTS word_stats (
			kana TEXT PRIMARY KEY,
			correct_count INTEGER NOT NULL DEFAULT 0,
			miss_count INTEGER NOT NULL DEFAULT 0
		);`,
	}

	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate statement failed: %w", err)
		}
	}
	return ensureMemory(ctx, db, time.Now())
}

// ensureMemory creates the memory table if it is missing and, in the same
// transaction, seeds it from kana_stats so earlier progress counts as known.
// If anything fails nothing is committed and the next open tries again.
func ensureMemory(ctx context.Context, db *sql.DB, now time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin memory migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var name string
	err = tx.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='memory'`).Scan(&name)
	if err == nil {
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("store: check memory table: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `CREATE TABLE memory (
		item TEXT PRIMARY KEY,
		stability REAL NOT NULL,
		last_seen INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("store: create memory table: %w", err)
	}

	type seed struct {
		char    string
		correct int
	}
	rows, err := tx.QueryContext(ctx, `SELECT char, correct_count FROM kana_stats WHERE correct_count > 0`)
	if err != nil {
		return fmt.Errorf("store: read kana stats for seeding: %w", err)
	}
	var seeds []seed
	for rows.Next() {
		var s seed
		if err := rows.Scan(&s.char, &s.correct); err != nil {
			rows.Close()
			return fmt.Errorf("store: scan kana stats for seeding: %w", err)
		}
		seeds = append(seeds, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: iterate kana stats for seeding: %w", err)
	}

	for _, s := range seeds {
		st := memory.Seed(s.correct, now)
		if _, err := tx.ExecContext(ctx, `INSERT INTO memory (item, stability, last_seen) VALUES (?, ?, ?)`,
			"kana:"+s.char, st.Stability, st.LastSeen.Unix()); err != nil {
			return fmt.Errorf("store: seed memory: %w", err)
		}
	}
	return tx.Commit()
}

// MemoryStates returns the memory states of all items starting with prefix,
// keyed by the rest of the item ID.
func (s *Store) MemoryStates(prefix string) (map[string]memory.State, error) {
	rows, err := s.db.Query(`SELECT item, stability, last_seen FROM memory WHERE substr(item, 1, length(?)) = ?`, prefix, prefix)
	if err != nil {
		return nil, fmt.Errorf("store: query memory: %w", err)
	}
	defer rows.Close()
	states := make(map[string]memory.State)
	for rows.Next() {
		var item string
		var stability float64
		var lastSeen int64
		if err := rows.Scan(&item, &stability, &lastSeen); err != nil {
			return nil, fmt.Errorf("store: scan memory: %w", err)
		}
		states[strings.TrimPrefix(item, prefix)] = memory.State{Stability: stability, LastSeen: time.Unix(lastSeen, 0)}
	}
	return states, rows.Err()
}

// SaveMemoryState stores one item's memory state.
func (s *Store) SaveMemoryState(item string, st memory.State) error {
	_, err := s.db.Exec(`
		INSERT INTO memory (item, stability, last_seen) VALUES (?, ?, ?)
		ON CONFLICT(item) DO UPDATE SET stability = excluded.stability, last_seen = excluded.last_seen
	`, item, st.Stability, st.LastSeen.Unix())
	if err != nil {
		return fmt.Errorf("store: save memory %s: %w", item, err)
	}
	return nil
}
