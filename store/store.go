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
	packageSizeKey     = "package_size"
	wordSourceKey      = "word_source"
	kanaFontKey        = "kana_font"
	newestPackageKey   = "newest_package"
	databaseFilePerm   = 0o644
	databaseDirPerm    = 0o755
	defaultOpenTimout  = 5 * time.Second
)

const DefaultScoreLimit = 1000

// DefaultWordLevel is the JLPT level words are played up to when unset (N5).
const DefaultWordLevel = 5

// DefaultPackageSize is the number of words in a word package when unset.
const DefaultPackageSize = 5

// DefaultKanaFont is the kana font when unset (Noto Sans JP).
const DefaultKanaFont = "noto"

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

// Progress is an item's learning level (0–5, learned at 5) and whether it
// slipped since it was last learned.
type Progress struct {
	Level   int
	Flagged bool
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

// validPackageSize reports whether n is a package size the app offers.
func validPackageSize(n int) bool {
	return n == 3 || n == 5 || n == 10
}

// PackageSize returns the number of words per word package: 3, 5 or 10.
// Unset or invalid values give DefaultPackageSize.
func (s *Store) PackageSize() (int, error) {
	value, err := s.getSetting(packageSizeKey)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultPackageSize, nil
	}
	if err != nil {
		return DefaultPackageSize, err
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(value))
	if convErr != nil || !validPackageSize(n) {
		return DefaultPackageSize, nil
	}
	return n, nil
}

// SavePackageSize persists the package size (3, 5 or 10).
func (s *Store) SavePackageSize(n int) error {
	if !validPackageSize(n) {
		return fmt.Errorf("store: package size %d is not 3, 5 or 10", n)
	}
	return s.setSetting(packageSizeKey, strconv.Itoa(n))
}

// WordSource returns where new words come from ("jlpt" or "list:<name>"),
// or "" when unset: the app then picks a default and saves it.
func (s *Store) WordSource() (string, error) {
	value, err := s.getSetting(wordSourceKey)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SaveWordSource persists where new words come from.
func (s *Store) SaveWordSource(source string) error {
	return s.setSetting(wordSourceKey, source)
}

// KanaFont returns the kana font setting ("noto", "mplus" or "system"), or
// DefaultKanaFont when unset. The app checks the value.
func (s *Store) KanaFont() (string, error) {
	value, err := s.getSetting(kanaFontKey)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultKanaFont, nil
	}
	if err != nil {
		return DefaultKanaFont, err
	}
	return value, nil
}

// SaveKanaFont persists the kana font setting.
func (s *Store) SaveKanaFont(name string) error {
	return s.setSetting(kanaFontKey, name)
}

// NewestPackage returns the readings of the newest word package, nil when
// unset or empty.
func (s *Store) NewestPackage() ([]string, error) {
	value, err := s.getSetting(newestPackageKey)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && value == "") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Split(value, "\n"), nil
}

// SaveNewestPackage persists the readings of the newest word package,
// joined with "\n"; nil or empty stores an empty package.
func (s *Store) SaveNewestPackage(readings []string) error {
	return s.setSetting(newestPackageKey, strings.Join(readings, "\n"))
}

// IntroducedWords returns the readings that were shown in a word package
// intro (or counted as introduced when the table was created).
func (s *Store) IntroducedWords() (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT kana FROM introduced_words`)
	if err != nil {
		return nil, fmt.Errorf("store: query introduced words: %w", err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var kana string
		if err := rows.Scan(&kana); err != nil {
			return nil, fmt.Errorf("store: scan introduced words: %w", err)
		}
		out[kana] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate introduced words: %w", err)
	}
	return out, nil
}

// SaveIntroduced records a reading as introduced at the given time. A
// reading that is already introduced keeps its first time.
func (s *Store) SaveIntroduced(kana string, at time.Time) error {
	if kana == "" {
		return errors.New("store: word reading is required")
	}
	_, err := s.db.Exec(`
		INSERT INTO introduced_words (kana, introduced_at) VALUES (?, ?)
		ON CONFLICT(kana) DO NOTHING
	`, kana, at.Unix())
	if err != nil {
		return fmt.Errorf("store: save introduced word %s: %w", kana, err)
	}
	return nil
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
	now := time.Now()
	if err := ensureMemory(ctx, db, now); err != nil {
		return err
	}
	if err := ensureIntroduced(ctx, db, now); err != nil {
		return err
	}
	return ensureProgress(ctx, db)
}

// ensureIntroduced creates the introduced_words table if it is missing and,
// in the same transaction, seeds it with every reading that has word stats
// or a word memory state, so words learned before word packages keep
// falling. If anything fails nothing is committed and the next open tries
// again.
func ensureIntroduced(ctx context.Context, db *sql.DB, now time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin introduced migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var name string
	err = tx.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='introduced_words'`).Scan(&name)
	if err == nil {
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("store: check introduced_words table: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `CREATE TABLE introduced_words (
		kana TEXT PRIMARY KEY,
		introduced_at INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("store: create introduced_words table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO introduced_words (kana, introduced_at)
		SELECT kana, ? FROM word_stats
		UNION
		SELECT substr(item, 6), ? FROM memory WHERE substr(item, 1, 5) = 'word:'
	`, now.Unix(), now.Unix()); err != nil {
		return fmt.Errorf("store: seed introduced words: %w", err)
	}
	return tx.Commit()
}

// ensureProgress creates the progress table if it is missing and, in the
// same transaction, seeds it from the stats: three or more correct answers
// (the old "secure") give level 5, fewer give their count; nothing starts
// flagged. If anything fails nothing is committed and the next open tries
// again.
func ensureProgress(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin progress migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var name string
	err = tx.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='progress'`).Scan(&name)
	if err == nil {
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("store: check progress table: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `CREATE TABLE progress (
		item TEXT PRIMARY KEY,
		level INTEGER NOT NULL,
		flagged INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("store: create progress table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO progress (item, level, flagged)
		SELECT 'kana:' || char, CASE WHEN correct_count >= 3 THEN 5 ELSE correct_count END, 0
		FROM kana_stats WHERE correct_count > 0
		UNION ALL
		SELECT 'word:' || kana, CASE WHEN correct_count >= 3 THEN 5 ELSE correct_count END, 0
		FROM word_stats WHERE correct_count > 0
	`); err != nil {
		return fmt.Errorf("store: seed progress: %w", err)
	}
	return tx.Commit()
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

// Progress returns the learning progress of all items starting with prefix,
// keyed by the rest of the item ID. Items without a row are at level 0.
func (s *Store) Progress(prefix string) (map[string]Progress, error) {
	rows, err := s.db.Query(`SELECT item, level, flagged FROM progress WHERE substr(item, 1, length(?)) = ?`, prefix, prefix)
	if err != nil {
		return nil, fmt.Errorf("store: query progress: %w", err)
	}
	defer rows.Close()
	out := make(map[string]Progress)
	for rows.Next() {
		var item string
		var p Progress
		if err := rows.Scan(&item, &p.Level, &p.Flagged); err != nil {
			return nil, fmt.Errorf("store: scan progress: %w", err)
		}
		out[strings.TrimPrefix(item, prefix)] = p
	}
	return out, rows.Err()
}

// SaveProgress stores one item's learning progress.
func (s *Store) SaveProgress(item string, p Progress) error {
	_, err := s.db.Exec(`
		INSERT INTO progress (item, level, flagged) VALUES (?, ?, ?)
		ON CONFLICT(item) DO UPDATE SET level = excluded.level, flagged = excluded.flagged
	`, item, p.Level, p.Flagged)
	if err != nil {
		return fmt.Errorf("store: save progress %s: %w", item, err)
	}
	return nil
}

// ResetKana deletes the stats, memory and learning progress of the given
// characters in one transaction; on error nothing is deleted. Word data is
// not touched.
func (s *Store) ResetKana(chars []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin kana reset: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, c := range chars {
		for _, q := range []string{
			`DELETE FROM kana_stats WHERE char = ?`,
			`DELETE FROM memory WHERE item = 'kana:' || ?`,
			`DELETE FROM progress WHERE item = 'kana:' || ?`,
		} {
			if _, err := tx.Exec(q, c); err != nil {
				return fmt.Errorf("store: reset kana %s: %w", c, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit kana reset: %w", err)
	}
	return nil
}
