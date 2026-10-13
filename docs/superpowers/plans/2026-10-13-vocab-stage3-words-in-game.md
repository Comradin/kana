# Vocabulary Trainer Stage 3: Words in the Game Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Words fall among the kana tiles (mixed mode). The learner types the reading of a word made only of kana they have secured; a hit shows the meaning, a miss shows reading and meaning. Word progress (stats and spaced-repetition memory) is kept per reading, separate from the kana.

**Architecture:** `store` gains a `word_stats` table and the `word_level` / `meaning_lang` settings. `GameState` loads the bundled list plus the private lists once into a `vocab.Index`, keeps word stats and word memory next to (never mixed with) the kana ones, and caches the playable entries. A `KanaTile` carries either a kana or a word (`word *vocab.Entry`); spawn, answer, miss and the tile cleanup branch on it. Feedback labels are `canvas.Text` objects owned by `GameState`, aged by `tick()` and published in the tile snapshot. The status panel shows a words line and the data attribution.

**Tech Stack:** Go 1.25, Fyne v2.7.3 (`fyne.io/fyne/v2/lang` for the system locale), modernc SQLite, standard `testing`.

**Spec:** `docs/superpowers/specs/2026-10-12-vocab-trainer-design.md`, section "Stage 3: Words in the game (mixed mode)", plus "Decisions" (playable rule, っ/ッ/ー) and the Stage 3 testing list.

**Conventions:** `gofmt -l .` prints nothing, `go vet ./...` is clean and `go test -count=1 ./...` passes before each commit. Commit messages: `type(scope): summary` plus a short prose paragraph; no Co-Authored-By, no AI/assistant/session mentions. Run every command from the repository root.

---

## Notes on the spec

Places where the spec does not match the real code, and how this plan resolves them:

1. **`meaning_lang` default.** The spec puts the default (system language if one of the nine JMdict languages, else `en`) on the settings key. `store` cannot import Fyne, so `Store.MeaningLang()` returns `""` when unset, meaning "follow the system". `fyne/words.go` resolves it with `lang.SystemLocale()` (`resolveMeaningLang`). `""` is never replaced by a concrete language in the database, so a learner who changes the system language gets the new language until they pick one in stage 4.
2. **`missReason`.** The existing `TestFormatStatusAndMissReason` expects `"5 kana slipped through."`. The spec changes the text to `"N tiles slipped through."`; Task 5 updates the test.
3. **Tile type.** The spec says "a tile carries either a kana or a word (`tile.word *vocab.Entry`)". The type stays `KanaTile` (renaming it would touch every test); a word tile has `word != nil` and a zero `kana`. Every reader of `tile.kana` branches on `tile.word` first.
4. **Feedback labels.** The spec lists text, position, kind and `ticksLeft`. A label holds a `*canvas.Text` (message and position) created once in `addFeedback`, so `buildSnapshot`, which runs every tick, does not allocate canvas objects. The labels are appended to the tile snapshot, so the existing `GameCanvas` renders them with no change to `canvas.go`, and they are hidden exactly when tiles are hidden. They also stay visible after game over (tick stops when over) until `Reset` clears them.
5. **Attribution line.** "Word data: JMdict (EDRDG, CC BY-SA 4.0), JLPT levels by J. Waller" does not fit the 292 px panel at any readable size as one `canvas.Text`. It is shown as two lines below the legend.
6. **"17 secure".** The spec does not say secure out of what. This plan counts the **playable** words with 3 or more correct answers (stored + session), so `secure ≤ playable`.
7. **Words-only wording of the status line** ("names the source instead of the level") belongs to stage 4; stage 3 always shows the level.
8. **Bundled error with private lists.** With a broken bundled list the spec shows "Words unavailable: <error>". Private lists still load and play; the line still shows the error, because the learner's main word source is missing.
9. **Private list problems and unreadable folders** are ignored in stage 3 (`vocab.LoadDir` errors give no lists). Stage 4 shows them in the Words tab.
10. **Words directory in tests.** `wordsDir()` follows `store.DefaultPath()`, as the spec says (also when `main` falls back to `./kana.db`). A `TestMain` in `fyne` points `XDG_DATA_HOME` at a temporary directory, so no test ever reads the developer's real `~/.local/share/kana/words/`.
11. **Session line.** The panel's "correct" and "accuracy" now count word hits too. The spec is silent, but `Missed` already counts word misses, so leaving hits out would make the accuracy wrong.
12. **Bundled load cost.** `vocab.Bundled()` is wrapped in `sync.OnceValues`, so the ~1 MB list is decoded once per process (the app builds one `GameState`, tests build many).
13. **Game-over list layout.** "せんせい (sensei)" entries are grouped three per line, so ten missed words do not make the dialog wider than the window.
14. **Word hits and the katakana offer.** A word hit does not call `recordCorrect`, so it never unlocks rows; `maybeOfferKatakana` is still called after it, as after any hit, and is a no-op because no kana stats changed.

## Lock discipline

- Every new `GameState` method is "Must be called under lock" (`gs.mu`), except `loadWords`, which runs inside `NewGameState` before the game starts (like `startLearningPath`).
- No new goroutines. Tiles and feedback labels are created under the lock in the spawn/tick goroutines or in `checkAnswer`, exactly as tiles already are; they reach the renderer only through the atomic `objectSnapshot`.
- UI work stays where it is: `tick` already refreshes the canvas and calls `onStatsChanged` via `fyne.Do`; `checkAnswer` runs on the UI goroutine (`OnSubmitted`) and refreshes directly. Nothing new needs `fyne.Do`.
- Tests call `GameState` methods without the lock, as the existing tests do (no goroutines are running), except where a test uses a `NewGameState` with a store and calls lock-requiring methods; those tests lock as `TestMergePersistsMemory` does.

---

## File Structure

- Modify `store/store.go`: `word_stats` table in `migrate`, `WordStats`, `WordStatistics`, `SaveWordStats`, `DefaultWordLevel`, `WordLevel`/`SaveWordLevel`, `MeaningLang`/`SaveMeaningLang`.
- Modify `store/store_test.go`.
- Create `fyne/words.go`: pure word helpers and package-level hooks — `meaningLangs`, `bundledWords`, `systemLanguage`, `wordsDir`, `resolveMeaningLang`, `missedWordsText`, `truncateRunes`, `meaningSnippet`, `hitFeedbackText`, `missFeedbackText`, `WordsStatus`, `formatWordsLine`, `wordDataSource`.
- Create `fyne/wordgame.go`: `GameState` word methods — `recordWordCorrect`, `recordWordMiss`, `wordCorrect`, `mergeWordStats`, `loadWords`, `unitKnown`, `playableWords`, `invalidatePlayable`, `wordCandidates`, `pickWord`, `wordsStatus`.
- Create `fyne/feedback.go`: `feedbackLabel`, `addFeedback`, `ageFeedback`.
- Modify `fyne/game.go`: fields, `NewGameState`, `Reset`, `tick`, `spawnKana`, `checkAnswer`, `recordCorrect`/`recordMiss` (tempo helpers), `mergeSessionStats` split, `saveMemory`, `spawnCandidates`, `applySettings`, `dropInactiveTiles`, `enableKatakanaLocked`, `buildSnapshot`, `snapshot`.
- Modify `fyne/tile.go`: `word` field, `newWordTile`, word tile colours.
- Modify `fyne/difficulty.go`: `missReason` text, `wordShare`, `wordFallFactor`.
- Modify `fyne/stats.go`: snapshot fields, words line, attribution lines, word hits in the session line.
- Modify `fyne/app.go`: missed words in the game-over dialog.
- Create `fyne/main_test.go` (`TestMain`), `fyne/wordgame_test.go`, `fyne/words_test.go`.
- Modify `fyne/game_test.go` (`newTestState`), `fyne/tile_test.go`, `fyne/difficulty_test.go`, `fyne/stats_test.go`.
- Modify `README.md`, `PLAN.md`.

Line numbers below refer to the files as they are at the start of this plan; they shift as tasks add code, so locate by function name.

---

### Task 1: Store — word stats and word settings

**Files:** Modify `store/store.go` (constants lines 20–33, `KanaStats` area lines 40–46, `migrate` lines 383–404), `store/store_test.go`.

- [ ] **Step 1: Write the failing tests** — append to `store/store_test.go`:

```go
func TestWordStatsRoundTrip(t *testing.T) {
	st := openTemp(t)
	got, err := st.WordStatistics()
	if err != nil || len(got) != 0 {
		t.Fatalf("empty WordStatistics = %v, %v", got, err)
	}
	if err := st.SaveWordStats("せんせい", 3, 1); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordStats("せんせい", 4, 1); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordStats("ねこ", -2, -1); err != nil {
		t.Fatal(err)
	}
	got, err = st.WordStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if w := got["せんせい"]; w != (WordStats{Kana: "せんせい", CorrectCount: 4, MissCount: 1}) {
		t.Fatalf("せんせい = %+v, want absolute values 4/1", w)
	}
	if w := got["ねこ"]; w.CorrectCount != 0 || w.MissCount != 0 {
		t.Fatalf("negative counts must be stored as 0, got %+v", w)
	}
	if err := st.SaveWordStats("", 1, 0); err == nil {
		t.Fatal("an empty reading must be rejected")
	}
}

func TestWordStatsSeparateFromKanaStats(t *testing.T) {
	st := openTemp(t)
	if err := st.SaveKanaStats("か", 5, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordStats("か", 1, 0); err != nil {
		t.Fatal(err)
	}
	kana, _ := st.KanaStatistics()
	words, _ := st.WordStatistics()
	if kana["か"].CorrectCount != 5 || words["か"].CorrectCount != 1 {
		t.Fatalf("kana か = %+v, word か = %+v; the mosquito must not share the character's stats", kana["か"], words["か"])
	}
}

func TestWordStatsSurviveReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordStats("ねこ", 2, 0); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	st, err = Open(path) // migrate runs again on an existing database
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if got, _ := st.WordStatistics(); got["ねこ"].CorrectCount != 2 {
		t.Fatalf("after reopen = %+v", got)
	}
}

func TestWordLevelAndMeaningLang(t *testing.T) {
	st := openTemp(t)
	if l, err := st.WordLevel(); err != nil || l != DefaultWordLevel || DefaultWordLevel != 5 {
		t.Fatalf("unset WordLevel = %d, %v; want 5", l, err)
	}
	if m, err := st.MeaningLang(); err != nil || m != "" {
		t.Fatalf("unset MeaningLang = %q, %v; want \"\" (follow the system)", m, err)
	}
	if err := st.SaveWordLevel(3); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordLevel(0); err == nil {
		t.Fatal("level 0 must be rejected")
	}
	if err := st.SaveWordLevel(6); err == nil {
		t.Fatal("level 6 must be rejected")
	}
	if l, _ := st.WordLevel(); l != 3 {
		t.Fatalf("WordLevel = %d, want 3", l)
	}
	if err := st.SaveMeaningLang("de"); err != nil {
		t.Fatal(err)
	}
	if m, _ := st.MeaningLang(); m != "de" {
		t.Fatalf("MeaningLang = %q, want de", m)
	}
	if err := st.setSetting(wordLevelKey, "N9"); err != nil {
		t.Fatal(err)
	}
	if l, err := st.WordLevel(); err != nil || l != DefaultWordLevel {
		t.Fatalf("invalid stored level = %d, %v; want the default", l, err)
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 ./store/` — expected: compile errors (`WordStatistics`, `SaveWordStats`, `WordStats`, `DefaultWordLevel`, `WordLevel`, `SaveWordLevel`, `MeaningLang`, `SaveMeaningLang`, `wordLevelKey` undefined).

- [ ] **Step 3: Implement** in `store/store.go`.

Add to the `const (...)` block with the other keys (after `pointsFactorKey`):

```go
	wordLevelKey       = "word_level"
	meaningLangKey     = "meaning_lang"
```

Below `const DefaultScoreLimit = 1000` add:

```go
// DefaultWordLevel is the JLPT level words are played up to when unset (N5).
const DefaultWordLevel = 5
```

Below the `KanaStats` type add:

```go
// WordStats is the aggregated statistics for one word reading. It is kept
// apart from KanaStats: the word か (mosquito) is not the character か.
type WordStats struct {
	Kana         string
	CorrectCount int
	MissCount    int
}
```

After `SaveScoreLimit` add:

```go
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
```

After `KanaStatistics` add:

```go
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
```

In `migrate`, append a statement to `stmts` after the `kana_stats` one:

```go
		`CREATE TABLE IF NOT EXISTS word_stats (
			kana TEXT PRIMARY KEY,
			correct_count INTEGER NOT NULL DEFAULT 0,
			miss_count INTEGER NOT NULL DEFAULT 0
		);`,
```

- [ ] **Step 4: Run** `go test -count=1 ./store/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...` — all clean.

- [ ] **Step 5: Commit**

```bash
git add store/store.go store/store_test.go
git commit -m "feat(store): add word stats and word settings

Adds the word_stats table (per reading, separate from kana_stats) with
WordStatistics and SaveWordStats, and the word_level (default N5) and
meaning_lang (unset means follow the system language) settings."
```

---

### Task 2: GameState — word progress (stats, memory, merge)

**Files:** Modify `fyne/game.go` (struct lines 29–88, `NewGameState` lines 91–162, `Reset` lines 274–331, `recordCorrect` lines 556–581, `recordMiss` lines 584–604, `mergeSessionStats` lines 607–645, `saveMemory` lines 651–660), `fyne/game_test.go` (`newTestState` lines 15–38). Create `fyne/wordgame.go`, `fyne/wordgame_test.go`.

- [ ] **Step 1: Write the failing tests** — create `fyne/wordgame_test.go`:

```go
package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/memory"
	"kana/store"
)

func TestRecordWordCorrectTouchesOnlyWordProgress(t *testing.T) {
	gs := newTestState()
	gs.recordWordCorrect("あい")
	if got := gs.wordSessionStats["あい"]; got.CorrectCount != 1 || got.Kana != "あい" || !gs.wordSessionDirty {
		t.Fatalf("word session = %+v, dirty %v", got, gs.wordSessionDirty)
	}
	if s, ok := gs.wordMemory["あい"]; !ok || s.Stability != memory.MinStability || !gs.wordMemoryDirty["あい"] {
		t.Fatalf("word memory = %+v, %v", s, ok)
	}
	if len(gs.sessionStats) != 0 || len(gs.memory) != 0 || gs.sessionDirty {
		t.Fatal("a word hit must not touch kana stats or kana memory")
	}
	if gs.answerStreak != 1 {
		t.Fatalf("answerStreak = %d, want 1", gs.answerStreak)
	}
}

func TestRecordWordMiss(t *testing.T) {
	gs := newTestState()
	gs.tempoStep, gs.answerStreak = 3, 4
	gs.recordWordMiss("あい")
	if got := gs.wordSessionStats["あい"]; got.MissCount != 1 || !gs.wordSessionDirty {
		t.Fatalf("word session = %+v", got)
	}
	if _, ok := gs.wordMemory["あい"]; ok {
		t.Fatal("a miss on a new word must not create memory")
	}
	if gs.tempoStep != 1 || gs.answerStreak != 0 {
		t.Fatalf("tempoStep=%d streak=%d, want 1/0", gs.tempoStep, gs.answerStreak)
	}
	gs.wordMemory["あお"] = memory.State{Stability: 1, LastSeen: time.Now()}
	gs.recordWordMiss("あお")
	if s := gs.wordMemory["あお"].Stability; s < 0.29 || s > 0.31 || !gs.wordMemoryDirty["あお"] {
		t.Fatalf("miss on known word → %v, want 0.3 and dirty", s)
	}
}

func TestMergePersistsWordProgress(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)

	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.recordWordCorrect("あい")
	gs.recordWordMiss("あい")
	gs.mergeSessionStats() // only words changed: the kana merge returns early
	sessionLeft, dirty, memDirty := len(gs.wordSessionStats), gs.wordSessionDirty, len(gs.wordMemoryDirty)
	overall := gs.wordOverallStats["あい"]
	gs.mu.Unlock()

	if sessionLeft != 0 || dirty || memDirty != 0 {
		t.Fatalf("after merge: session %d, dirty %v, memory dirty %d", sessionLeft, dirty, memDirty)
	}
	if overall.CorrectCount != 2 || overall.MissCount != 1 {
		t.Fatalf("overall = %+v, want 2/1", overall)
	}
	if words, _ := st.WordStatistics(); words["あい"].CorrectCount != 2 || words["あい"].MissCount != 1 {
		t.Fatalf("persisted = %+v", words["あい"])
	}
	if mem, _ := st.MemoryStates("word:"); len(mem) != 1 {
		t.Fatalf("word memory = %v, want あい only", mem)
	}
	if kana, _ := st.MemoryStates("kana:"); len(kana) != 0 {
		t.Fatalf("word memory leaked into kana memory: %v", kana)
	}

	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if words, _ := st.WordStatistics(); words["あい"].CorrectCount != 3 {
		t.Fatalf("after second merge = %+v, want 3 (double-count?)", words["あい"])
	}

	reloaded := NewGameState(st)
	if reloaded.wordOverallStats["あい"].CorrectCount != 3 {
		t.Fatalf("reloaded overall = %+v", reloaded.wordOverallStats["あい"])
	}
	if _, ok := reloaded.wordMemory["あい"]; !ok {
		t.Fatal("word memory must load on start")
	}
}

// TestMergesAreIndependent breaks one stats table through a second
// connection: the other kind must still be merged.
func TestMergesAreIndependent(t *testing.T) {
	for _, broken := range []string{"kana_stats", "word_stats"} {
		t.Run(broken, func(t *testing.T) {
			test.NewApp()
			path := filepath.Join(t.TempDir(), "test.db")
			st, err := store.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = st.Close() })
			_ = st.SaveSelectedRows([]string{"vowels"})
			gs := NewGameState(st)

			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec("DROP TABLE " + broken); err != nil {
				t.Fatal(err)
			}

			gs.mu.Lock()
			gs.recordCorrect("か")
			gs.recordWordCorrect("あい")
			gs.mergeSessionStats()
			kanaDirty, wordDirty := gs.sessionDirty, gs.wordSessionDirty
			gs.mu.Unlock()

			if broken == "kana_stats" {
				if !kanaDirty || wordDirty {
					t.Fatalf("kanaDirty=%v wordDirty=%v, want true/false", kanaDirty, wordDirty)
				}
				if words, err := st.WordStatistics(); err != nil || words["あい"].CorrectCount != 1 {
					t.Fatalf("words = %v, %v", words, err)
				}
			} else {
				if kanaDirty || !wordDirty {
					t.Fatalf("kanaDirty=%v wordDirty=%v, want false/true", kanaDirty, wordDirty)
				}
				if kana, err := st.KanaStatistics(); err != nil || kana["か"].CorrectCount != 1 {
					t.Fatalf("kana = %v, %v", kana, err)
				}
			}
		})
	}
}

func TestResetReloadsWordStatsAndKeepsMemory(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordStats("あい", 2, 0)
	gs := NewGameState(st)
	if gs.wordOverallStats["あい"].CorrectCount != 2 {
		t.Fatalf("loaded = %+v", gs.wordOverallStats["あい"])
	}
	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.mu.Unlock()
	gs.Reset()
	if len(gs.wordSessionStats) != 0 || gs.wordSessionDirty {
		t.Fatal("Reset must clear the word session")
	}
	if gs.wordOverallStats["あい"].CorrectCount != 3 {
		t.Fatalf("after Reset overall = %+v, want 3", gs.wordOverallStats["あい"])
	}
	if _, ok := gs.wordMemory["あい"]; !ok {
		t.Fatal("Reset must keep word memory")
	}
}

func TestFailedWordMemoryLoadSkipsSave(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.wordMemoryLoadFailed = true
	gs.recordWordCorrect("あい")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if got, _ := st.MemoryStates("word:"); len(got) != 0 {
		t.Fatalf("failed load must not write word memory, got %v", got)
	}
}

func TestStoredWordLevelIsLoaded(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if l := NewGameState(st).wordLevel; l != 5 {
		t.Fatalf("default wordLevel = %d, want 5", l)
	}
	_ = st.SaveWordLevel(3)
	if l := NewGameState(st).wordLevel; l != 3 {
		t.Fatalf("wordLevel = %d, want 3", l)
	}
}
```

Note: `sql.Open("sqlite", …)` works because `kana/store` imports `modernc.org/sqlite`, which registers the driver.

- [ ] **Step 2: Run** `go test -count=1 ./fyne/` — expected: compile errors (`recordWordCorrect`, `wordSessionStats`, `wordMemory`, … undefined).

- [ ] **Step 3: Implement.**

In `fyne/game.go`, add to the `GameState` struct after the `rnd` field:

```go

	// Words (vocabulary trainer, mixed mode): progress per reading, kept
	// apart from the kana stats and memory.
	wordOverallStats     map[string]store.WordStats
	wordSessionStats     map[string]store.WordStats
	wordSessionDirty     bool
	wordMemory           map[string]memory.State // spaced-repetition state per reading
	wordMemoryDirty      map[string]bool         // readings whose state is not saved yet
	wordMemoryLoadFailed bool                    // stored word states unreadable: never overwrite them
	wordLevel            int                     // words are played up to this JLPT level, 5 = N5
```

In `NewGameState`, add to the struct literal (after `rnd: rand.Float64,`):

```go
		wordOverallStats: make(map[string]store.WordStats),
		wordSessionStats: make(map[string]store.WordStats),
		wordMemory:       map[string]memory.State{},
		wordMemoryDirty:  map[string]bool{},
		wordLevel:        store.DefaultWordLevel,
```

and inside `if st != nil {`, right after the existing `MemoryStates("kana:")` block:

```go
		if ws, err := st.WordStatistics(); err == nil {
			gs.wordOverallStats = ws
		}
		if ms, err := st.MemoryStates("word:"); err == nil {
			gs.wordMemory = ms
		} else {
			gs.wordMemoryLoadFailed = true
		}
		if level, err := st.WordLevel(); err == nil {
			gs.wordLevel = level
		}
```

In `fyne/game_test.go`, add to the `newTestState` struct literal (after `rnd: rand.Float64,`):

```go
		wordOverallStats: make(map[string]store.WordStats),
		wordSessionStats: make(map[string]store.WordStats),
		wordMemory:       map[string]memory.State{},
		wordMemoryDirty:  map[string]bool{},
		wordLevel:        5,
```

Replace `recordCorrect` and `recordMiss` in `fyne/game.go` with (behaviour unchanged; the tempo code moves into two helpers that word answers share):

```go
// recordCorrect updates session stats only. Must be called under lock.
// Reports whether it announced newly unlocked rows.
func (gs *GameState) recordCorrect(char string) bool {
	streak := gs.currentStreak[char] + 1
	gs.currentStreak[char] = streak

	stat := gs.sessionStats[char]
	stat.Char = char
	stat.CorrectCount++
	stat.Streak = streak
	gs.sessionStats[char] = stat
	gs.sessionDirty = true

	prev, known := gs.memory[char]
	gs.memory[char] = memory.Correct(prev, known, gs.now())
	gs.memoryDirty[char] = true

	gs.raiseTempo()

	if unlocked := gs.checkAutoProgression(); len(unlocked) > 0 {
		gs.announceRows(unlocked)
		return true
	}
	return false
}

// recordMiss updates session stats + resets streak. Must be called under lock.
func (gs *GameState) recordMiss(char string) {
	gs.currentStreak[char] = 0

	stat := gs.sessionStats[char]
	stat.Char = char
	stat.MissCount++
	stat.Streak = 0
	gs.sessionStats[char] = stat
	gs.sessionDirty = true

	if prev, known := gs.memory[char]; known {
		gs.memory[char] = memory.Miss(prev, gs.now())
		gs.memoryDirty[char] = true
	}

	gs.lowerTempo()
}

// raiseTempo counts a correct answer (kana or word) towards the session
// tempo. Must be called under lock.
func (gs *GameState) raiseTempo() {
	gs.answerStreak++
	if gs.answerStreak%streakPerStep == 0 && gs.tempoStep < maxTempoStep {
		gs.tempoStep++
	}
}

// lowerTempo resets the answer streak and takes tempo steps back after a
// miss (kana or word). Must be called under lock.
func (gs *GameState) lowerTempo() {
	gs.answerStreak = 0
	gs.tempoStep -= missTempoSteps
	if gs.tempoStep < 0 {
		gs.tempoStep = 0
	}
}
```

Replace `mergeSessionStats` and `saveMemory` with:

```go
// mergeSessionStats writes (baseline + session) to store for kana and words.
// Memory is saved first, so a failing stats read cannot lose it. The kana
// and word merges are independent: each has its own baseline read and early
// return, so a skipped or failed one never skips the other. Must be called
// under lock.
func (gs *GameState) mergeSessionStats() {
	gs.saveMemory()
	gs.mergeWordStats()
	gs.mergeKanaStats()
}

// mergeKanaStats writes (baseline + session) kana stats. A failed baseline
// read keeps sessionDirty so the next merge retries. Must be called under
// lock.
func (gs *GameState) mergeKanaStats() {
	if !gs.sessionDirty {
		return
	}

	var baseline map[string]store.KanaStats
	if gs.store != nil {
		stats, err := gs.store.KanaStatistics()
		if err != nil {
			// Leave sessionDirty=true so next merge retries.
			return
		}
		baseline = stats
	}
	if baseline == nil {
		baseline = make(map[string]store.KanaStats)
	}

	for char, session := range gs.sessionStats {
		if session.CorrectCount == 0 && session.MissCount == 0 && session.Streak == 0 && gs.currentStreak[char] == 0 {
			continue
		}
		base := baseline[char]
		base.Char = char
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		base.Streak = gs.currentStreak[char]
		if gs.store != nil {
			if err := gs.store.SaveKanaStats(char, base.CorrectCount, base.MissCount, base.Streak); err != nil {
				continue
			}
		}
		gs.overallStats[char] = base
		delete(gs.sessionStats, char)
	}

	gs.sessionDirty = false
}

// saveMemory writes changed kana and word memory states. If loading a kind
// failed, nothing of that kind is written, so the stored states are not
// replaced by fresh ones. Must be called under lock.
func (gs *GameState) saveMemory() {
	if gs.store == nil {
		return
	}
	if !gs.memoryLoadFailed {
		for char := range gs.memoryDirty {
			if err := gs.store.SaveMemoryState("kana:"+char, gs.memory[char]); err == nil {
				delete(gs.memoryDirty, char)
			}
		}
	}
	if !gs.wordMemoryLoadFailed {
		for kana := range gs.wordMemoryDirty {
			if err := gs.store.SaveMemoryState("word:"+kana, gs.wordMemory[kana]); err == nil {
				delete(gs.wordMemoryDirty, kana)
			}
		}
	}
}
```

In `Reset`, after `gs.sessionDirty = false` add:

```go
	gs.wordSessionStats = make(map[string]store.WordStats)
	gs.wordSessionDirty = false
```

and after the `// reload overall stats` block (before `gs.buildSnapshot()`) add:

```go
	gs.wordOverallStats = make(map[string]store.WordStats)
	if gs.store != nil {
		if ws, err := gs.store.WordStatistics(); err == nil {
			gs.wordOverallStats = ws
		}
	}
```

Create `fyne/wordgame.go`:

```go
package main

import (
	"kana/memory"
	"kana/store"
)

// recordWordCorrect records a hit on a word: word session stats, word memory
// and the answer streak. Kana stats are not touched. Must be called under
// lock.
func (gs *GameState) recordWordCorrect(kana string) {
	stat := gs.wordSessionStats[kana]
	stat.Kana = kana
	stat.CorrectCount++
	gs.wordSessionStats[kana] = stat
	gs.wordSessionDirty = true

	prev, known := gs.wordMemory[kana]
	gs.wordMemory[kana] = memory.Correct(prev, known, gs.now())
	gs.wordMemoryDirty[kana] = true

	gs.raiseTempo()
}

// recordWordMiss records a word that fell through: word session stats, word
// memory (only if the word has a state) and the tempo. Must be called under
// lock.
func (gs *GameState) recordWordMiss(kana string) {
	stat := gs.wordSessionStats[kana]
	stat.Kana = kana
	stat.MissCount++
	gs.wordSessionStats[kana] = stat
	gs.wordSessionDirty = true

	if prev, known := gs.wordMemory[kana]; known {
		gs.wordMemory[kana] = memory.Miss(prev, gs.now())
		gs.wordMemoryDirty[kana] = true
	}

	gs.lowerTempo()
}

// wordCorrect is a reading's stored plus session correct count. Must be
// called under lock.
func (gs *GameState) wordCorrect(kana string) int {
	return gs.wordOverallStats[kana].CorrectCount + gs.wordSessionStats[kana].CorrectCount
}

// mergeWordStats writes (baseline + session) word stats, independently of
// the kana stats. A failed baseline read keeps wordSessionDirty so the next
// merge retries. Must be called under lock.
func (gs *GameState) mergeWordStats() {
	if !gs.wordSessionDirty {
		return
	}

	var baseline map[string]store.WordStats
	if gs.store != nil {
		stats, err := gs.store.WordStatistics()
		if err != nil {
			return
		}
		baseline = stats
	}
	if baseline == nil {
		baseline = make(map[string]store.WordStats)
	}

	for kana, session := range gs.wordSessionStats {
		if session.CorrectCount == 0 && session.MissCount == 0 {
			continue
		}
		base := baseline[kana]
		base.Kana = kana
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		if gs.store != nil {
			if err := gs.store.SaveWordStats(kana, base.CorrectCount, base.MissCount); err != nil {
				continue
			}
		}
		gs.wordOverallStats[kana] = base
		delete(gs.wordSessionStats, kana)
	}

	gs.wordSessionDirty = false
}
```

- [ ] **Step 4: Run** `gofmt -w fyne/ && go test -count=1 ./fyne/` — expected: PASS (all existing tests unchanged, including `TestTempoRisesWithStreakAndDropsOnMiss`, `TestMergeSessionStatsRoundTrip`, `TestFailedMemoryLoadSkipsSave`). Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/game_test.go fyne/wordgame.go fyne/wordgame_test.go
git commit -m "feat(fyne): keep word progress per reading

GameState records word hits and misses in their own session stats and
memory, merges them into word_stats and word: memory items independently
of the kana merge, and reloads them on Reset. The tempo code is shared by
kana and word answers."
```

---

### Task 3: Word index, meaning language and the known rule

**Files:** Create `fyne/words.go`, `fyne/words_test.go`, `fyne/main_test.go`. Modify `fyne/game.go` (struct, `NewGameState`, `recordCorrect`, `mergeKanaStats`, `Reset`, `applySettings` lines 737–774, `enableKatakanaLocked` lines 1016–1033), `fyne/game_test.go` (`newTestState`), `fyne/wordgame.go`, `fyne/wordgame_test.go`.

- [ ] **Step 1: Write the failing tests.**

Create `fyne/main_test.go`:

```go
package main

import (
	"fmt"
	"os"
	"testing"
)

// TestMain points XDG_DATA_HOME at an empty directory, so NewGameState never
// reads the developer's own private word lists.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "kana-fyne-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
```

Create `fyne/words_test.go`:

```go
package main

import "testing"

func TestResolveMeaningLang(t *testing.T) {
	cases := []struct{ stored, system, want string }{
		{"", "de-DE", "de"},
		{"", "sl", "sl"},
		{"", "pt_BR", "en"},
		{"", "ja-JP", "en"},
		{"", "", "en"},
		{"", "SV-se", "sv"},
		{"fr", "de-DE", "fr"},
	}
	for _, c := range cases {
		if got := resolveMeaningLang(c.stored, c.system); got != c.want {
			t.Errorf("resolveMeaningLang(%q, %q) = %q, want %q", c.stored, c.system, got, c.want)
		}
	}
}
```

Append to `fyne/wordgame_test.go` and make its import block exactly:

```go
import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/memory"
	"kana/store"
	"kana/vocab"
)
```

```go
// bw is a bundled word with one English meaning.
func bw(kana string, level int, en string) vocab.Word {
	return vocab.Word{Kana: kana, JLPT: level, Source: vocab.BundledSource, Meanings: vocab.Meanings{"en": {en}}}
}

// withWords replaces the word index with the given bundled words.
func withWords(gs *GameState, words ...vocab.Word) {
	gs.words = vocab.NewIndex(words, nil)
	gs.invalidatePlayable()
}

// secure gives each character three stored correct answers.
func secure(gs *GameState, chars ...string) {
	for _, c := range chars {
		gs.overallStats[c] = store.KanaStats{Char: c, CorrectCount: 3}
	}
	gs.invalidatePlayable()
}

func playableKana(gs *GameState) []string {
	var out []string
	for _, e := range gs.playableWords() {
		out = append(out, e.Kana)
	}
	return out
}

func TestUnitKnown(t *testing.T) {
	gs := newTestState() // hiragana only
	secure(gs, "か", "カ")
	gs.overallStats["き"] = store.KanaStats{Char: "き", CorrectCount: 2}
	gs.overallStats["く"] = store.KanaStats{Char: "く", CorrectCount: 2}
	gs.sessionStats["く"] = store.KanaStats{Char: "く", CorrectCount: 1}
	cases := map[string]bool{"か": true, "き": false, "く": true, "カ": false, "っ": true, "ッ": false, "ー": true, "け": false}
	for unit, want := range cases {
		if got := gs.unitKnown(unit); got != want {
			t.Errorf("hiragana only: unitKnown(%s) = %v, want %v", unit, got, want)
		}
	}
	gs.activeScripts = map[kanacore.Script]bool{kanacore.ScriptKatakana: true}
	cases = map[string]bool{"か": false, "カ": true, "っ": false, "ッ": true, "ー": true}
	for unit, want := range cases {
		if got := gs.unitKnown(unit); got != want {
			t.Errorf("katakana only: unitKnown(%s) = %v, want %v", unit, got, want)
		}
	}
}

func TestPlayableWordsFollowLevelAndKnownKana(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"), bw("あお", 4, "blue"), bw("いえ", 5, "house"))
	secure(gs, "あ", "い", "お")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable at N5 = %v, want [あい] (あお is N4, いえ needs え)", got)
	}
	gs.wordLevel = 4
	gs.invalidatePlayable()
	if got := playableKana(gs); !equalIDs(got, []string{"あい", "あお"}) {
		t.Fatalf("playable at N4 = %v", got)
	}
}

func TestPrivateListWordsIgnoreLevel(t *testing.T) {
	gs := newTestState()
	gs.words = vocab.NewIndex(nil, []vocab.List{{Name: "mine", Words: []vocab.Word{{Kana: "あお", Source: "mine", Meanings: vocab.Meanings{}}}}})
	secure(gs, "あ", "お")
	if got := playableKana(gs); !equalIDs(got, []string{"あお"}) {
		t.Fatalf("playable = %v, want [あお]", got)
	}
}

func TestNoWordIndexMeansNoWords(t *testing.T) {
	gs := newTestState()
	if got := gs.playableWords(); got != nil {
		t.Fatalf("playable without an index = %v", got)
	}
}

func TestPlayableRefreshesWhenKanaBecomesSecure(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	secure(gs, "あ")
	gs.overallStats["い"] = store.KanaStats{Char: "い", CorrectCount: 2}
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable before い is secure = %v", got)
	}
	gs.recordCorrect("い")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable after い is secure = %v", got)
	}
}

func TestKatakanaWordStopsWhenKatakanaTurnsOff(t *testing.T) {
	gs := newTestState()
	gs.activeScripts[kanacore.ScriptKatakana] = true
	withWords(gs, bw("アイ", 5, "eye"))
	secure(gs, "ア", "イ")
	if got := playableKana(gs); !equalIDs(got, []string{"アイ"}) {
		t.Fatalf("playable with katakana = %v", got)
	}
	gs.applySettings([]string{"vowels", "kata:vowels"}, []kanacore.Script{kanacore.ScriptHiragana}, false, 0, LevelNormal, pointsAuto)
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable after katakana off = %v", got)
	}
}

func TestNewGameStateLoadsBundledWords(t *testing.T) {
	test.NewApp()
	gs := NewGameState(openTestStore(t))
	if gs.words == nil || gs.wordsErr != "" || len(gs.words.Entries()) < 5000 {
		t.Fatalf("words = %v entries, err %q", gs.words != nil, gs.wordsErr)
	}
}

func TestPrivateListsLoadWhenBundledFails(t *testing.T) {
	test.NewApp()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	dir := filepath.Join(data, "kana", "words")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mine.csv"), []byte("kana,meaning\nあお,blue\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := bundledWords
	bundledWords = func() ([]vocab.Word, error) { return nil, errors.New("broken data") }
	t.Cleanup(func() { bundledWords = orig })

	gs := NewGameState(nil)
	if gs.wordsErr != "broken data" {
		t.Fatalf("wordsErr = %q", gs.wordsErr)
	}
	if entries := gs.words.Entries(); len(entries) != 1 || entries[0].Kana != "あお" {
		t.Fatalf("entries = %v, want the private あお only", entries)
	}
}

func TestMeaningLangFollowsSystemUnlessStored(t *testing.T) {
	test.NewApp()
	orig := systemLanguage
	systemLanguage = func() string { return "de-AT" }
	t.Cleanup(func() { systemLanguage = orig })
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if l := NewGameState(st).meaningLang; l != "de" {
		t.Fatalf("meaningLang = %q, want de from the system", l)
	}
	_ = st.SaveMeaningLang("sv")
	if l := NewGameState(st).meaningLang; l != "sv" {
		t.Fatalf("meaningLang = %q, want the stored sv", l)
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 ./fyne/` — expected: compile errors (`resolveMeaningLang`, `gs.words`, `invalidatePlayable`, `playableWords`, `unitKnown`, `bundledWords`, `systemLanguage`, `wordsErr`, `meaningLang` undefined).

- [ ] **Step 3: Implement.**

Create `fyne/words.go`:

```go
package main

import (
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2/lang"
	"kana/store"
	"kana/vocab"
)

// meaningLangs are the languages the bundled word list carries.
var meaningLangs = []string{"en", "de", "fr", "ru", "es", "hu", "nl", "sv", "sl"}

// bundledWords loads the bundled word list once per process. Tests replace
// it to simulate a broken list.
var bundledWords = sync.OnceValues(vocab.Bundled)

// systemLanguage returns the system locale, e.g. "de-DE". Tests replace it.
var systemLanguage = func() string { return string(lang.SystemLocale()) }

// wordsDir is the private word-list folder next to the database.
func wordsDir() (string, error) {
	p, err := store.DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), "words"), nil
}

// resolveMeaningLang returns the stored meaning language, or, when none is
// stored, the system language if the word list has it, else English.
func resolveMeaningLang(stored, system string) string {
	if stored != "" {
		return stored
	}
	base := strings.ToLower(system)
	if i := strings.IndexAny(base, "-_."); i >= 0 {
		base = base[:i]
	}
	for _, l := range meaningLangs {
		if l == base {
			return l
		}
	}
	return "en"
}
```

In `fyne/game.go`, add `"kana/vocab"` to the import block, and add to the `GameState` struct after `wordLevel`:

```go
	meaningLang          string        // language meanings are shown in
	words                *vocab.Index  // every word reading; nil = no words
	wordsErr             string        // bundled list failed to load
	playable             []vocab.Entry // cached playableWords result
	playableOK           bool          // playable is up to date
```

Replace `NewGameState` with (the struct literal and the `st != nil` block are the current code plus Task 2's additions; new are `storedLang`, the `MeaningLang` read, and the two lines before `return`):

```go
// NewGameState constructs a new GameState, loading persisted state if store is non-nil.
func NewGameState(st *store.Store) *GameState {
	gs := &GameState{
		sessionStats:     make(map[string]store.KanaStats),
		overallStats:     make(map[string]store.KanaStats),
		currentStreak:    make(map[string]int),
		selectedRows:     make(map[string]bool),
		activeScripts:    map[kanacore.Script]bool{kanacore.ScriptHiragana: true},
		eventCh:          make(chan gameEvent, 4),
		stopCh:           make(chan struct{}),
		scoreLimit:       store.DefaultScoreLimit,
		charSet:          kanacore.AllKana(),
		store:            st,
		canvasW:          400,
		canvasH:          600,
		level:            LevelByName(""),
		memory:           map[string]memory.State{},
		memoryDirty:      map[string]bool{},
		now:              time.Now,
		rnd:              rand.Float64,
		wordOverallStats: make(map[string]store.WordStats),
		wordSessionStats: make(map[string]store.WordStats),
		wordMemory:       map[string]memory.State{},
		wordMemoryDirty:  map[string]bool{},
		wordLevel:        store.DefaultWordLevel,
	}

	for _, id := range kanacore.DefaultRowIDs() {
		gs.selectedRows[id] = true
	}

	storedLang := ""
	if st != nil {
		if names, err := st.ActiveScripts(); err == nil && len(names) > 0 {
			gs.activeScripts = scriptSet(parseScripts(names))
		}
		if offered, err := st.KatakanaOffered(); err == nil {
			gs.katakanaOffered = offered
		}
		if stats, err := st.KanaStatistics(); err == nil {
			for _, stat := range stats {
				copied := stat
				gs.overallStats[stat.Char] = copied
				gs.currentStreak[stat.Char] = stat.Streak
			}
		}
		if ms, err := st.MemoryStates("kana:"); err == nil {
			gs.memory = ms
		} else {
			gs.memoryLoadFailed = true
		}
		if ws, err := st.WordStatistics(); err == nil {
			gs.wordOverallStats = ws
		}
		if ms, err := st.MemoryStates("word:"); err == nil {
			gs.wordMemory = ms
		} else {
			gs.wordMemoryLoadFailed = true
		}
		if level, err := st.WordLevel(); err == nil {
			gs.wordLevel = level
		}
		if l, err := st.MeaningLang(); err == nil {
			storedLang = l
		}
		if auto, err := st.AutoProgress(); err == nil {
			gs.autoProgress = auto
		}
		if name, err := st.Difficulty(); err == nil {
			gs.level = LevelByName(name)
		}
		if p, err := st.PointsFactor(); err == nil {
			gs.pointsSetting = validPoints(p)
		}
		if limit, err := st.ScoreLimit(); err == nil {
			if limit < 0 {
				limit = 0
			}
			gs.scoreLimit = limit
		}
		rows, err := st.SelectedRows()
		switch {
		case err != nil:
			// Keep the basic defaults; never overwrite what we could not read.
		case len(rows) > 0:
			gs.applySelectedRows(rows)
		default:
			gs.startLearningPath()
		}
	}

	gs.meaningLang = resolveMeaningLang(storedLang, systemLanguage())
	gs.loadWords()
	return gs
}
```

In `fyne/game_test.go` `newTestState`, add `meaningLang: "en",` to the struct literal (after `wordLevel: 5,`). `words` stays nil: existing tests play kana only.

Invalidation points in `fyne/game.go`:

- `recordCorrect`: right after `gs.sessionDirty = true` insert

  ```go
	if gs.overallStats[char].CorrectCount+stat.CorrectCount == secureCorrect {
		gs.invalidatePlayable() // the kana just became secure
	}
  ```

- `mergeKanaStats`: before the final `gs.sessionDirty = false` insert `gs.invalidatePlayable() // the stored baseline may differ`.
- `Reset`: after the word-stats reload block from Task 2 insert `gs.invalidatePlayable()`.
- `applySettings`: insert `gs.invalidatePlayable()` directly before `gs.dropInactiveTiles()`.
- `enableKatakanaLocked`: after `gs.activeScripts[kanacore.ScriptKatakana] = true` insert `gs.invalidatePlayable()`.

Append to `fyne/wordgame.go` (and add `"kana/kanacore"` and `"kana/vocab"` to its import block):

```go
// secureCorrect is the correct count at which a kana or word counts as
// secure (the panel's "mastered").
const secureCorrect = 3

// loadWords builds the word index from the bundled list and the private
// lists. A bundled load error is kept for the status panel; kana play is
// unaffected. An unreadable list folder gives no private lists (stage 4
// shows list problems). Called from NewGameState before the game starts, so
// no lock is needed.
func (gs *GameState) loadWords() {
	bundled, err := bundledWords()
	if err != nil {
		gs.wordsErr = err.Error()
		bundled = nil
	}
	var lists []vocab.List
	if dir, err := wordsDir(); err == nil {
		lists, _ = vocab.LoadDir(dir)
	}
	gs.words = vocab.NewIndex(bundled, lists)
	gs.invalidatePlayable()
}

// unitKnown reports whether a word unit is known in mixed mode: the kana is
// secure and its script is active. っ and ッ need their script, ー either
// script. Must be called under lock.
func (gs *GameState) unitKnown(unit string) bool {
	switch unit {
	case "っ":
		return gs.activeScripts[kanacore.ScriptHiragana]
	case "ッ":
		return gs.activeScripts[kanacore.ScriptKatakana]
	case "ー":
		return gs.activeScripts[kanacore.ScriptHiragana] || gs.activeScripts[kanacore.ScriptKatakana]
	}
	rowID, ok := kanacore.CharToRow[unit]
	if !ok || !gs.activeScripts[kanacore.ScriptOf(rowID)] {
		return false
	}
	return gs.overallStats[unit].CorrectCount+gs.sessionStats[unit].CorrectCount >= secureCorrect
}

// playableWords returns the entries playable in mixed mode: bundled words up
// to wordLevel plus every private word, all of whose units are known. The
// result is cached until invalidatePlayable; do not modify it. Must be
// called under lock.
func (gs *GameState) playableWords() []vocab.Entry {
	if gs.words == nil {
		return nil
	}
	if !gs.playableOK {
		gs.playable = vocab.Playable(gs.words.Entries(), gs.unitKnown, vocab.Filter{MaxLevel: gs.wordLevel})
		gs.playableOK = true
	}
	return gs.playable
}

// invalidatePlayable marks the playable cache stale. Call it whenever a kana
// may have become secure, the active scripts, the level or the index
// changed. Must be called under lock.
func (gs *GameState) invalidatePlayable() {
	gs.playableOK = false
}
```

- [ ] **Step 4: Run** `gofmt -w fyne/ && go test -count=1 ./fyne/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 5: Commit**

```bash
git add fyne/words.go fyne/words_test.go fyne/main_test.go fyne/wordgame.go fyne/wordgame_test.go fyne/game.go fyne/game_test.go
git commit -m "feat(fyne): load words and decide which are playable

NewGameState loads the bundled list and the private lists into one word
index and resolves the meaning language from the setting or the system
locale. A word is playable when every unit is a secure kana of an active
script (っ/ッ need their script, ー either); the result is cached and
invalidated when kana become secure or settings change."
```

---

### Task 4: Word tiles

**Files:** Modify `fyne/tile.go` (whole file, 78 lines), `fyne/tile_test.go`, `fyne/game.go` (`spawnCandidates` lines 689–704, `dropInactiveTiles` lines 778–788), `fyne/wordgame_test.go`.

- [ ] **Step 1: Write the failing tests.**

Append to `fyne/tile_test.go` and add `"kana/vocab"` to its imports:

```go
func TestWordTileLooksDifferent(t *testing.T) {
	test.NewApp()
	kana := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	word := newWordTile(vocab.Entry{Kana: "か", Units: []string{"か"}}, 400)
	if word.word == nil || word.word.Kana != "か" {
		t.Fatalf("word = %+v", word.word)
	}
	if kana.word != nil {
		t.Fatal("a kana tile carries no word")
	}
	if word.face.FillColor == kana.face.FillColor || word.face.StrokeWidth == 0 {
		t.Fatal("a one-unit word must not look like a kana tile")
	}
	if len(word.Objects()) != 3 {
		t.Fatalf("objects = %d, want 3", len(word.Objects()))
	}
}

func TestWordTileWidthFollowsText(t *testing.T) {
	test.NewApp()
	short := newWordTile(vocab.Entry{Kana: "ねこ"}, 600)
	long := newWordTile(vocab.Entry{Kana: "かんきょう"}, 600)
	if long.Width() <= short.Width() {
		t.Fatalf("long %.0f should be wider than short %.0f", long.Width(), short.Width())
	}
	if long.text.TextSize != wordTextSize {
		t.Fatalf("a word that fits keeps the full size, got %v", long.text.TextSize)
	}
	long.Move(fyne.NewPos(40, 20))
	left := long.text.Position().X - 40
	right := 40 + long.Width() - (long.text.Position().X + long.text.Size().Width)
	if diff := left - right; left < 0 || right < 0 || diff > 1 || diff < -1 {
		t.Fatalf("text not centred (left %.1f, right %.1f)", left, right)
	}
}

func TestWordTileShrinksToFitField(t *testing.T) {
	test.NewApp()
	tile := newWordTile(vocab.Entry{Kana: "コンピューター"}, 150)
	if tile.Width() > 150 {
		t.Fatalf("width %.1f exceeds the 150 px field", tile.Width())
	}
	if tile.text.TextSize >= wordTextSize {
		t.Fatalf("font not shrunk: %v", tile.text.TextSize)
	}
}
```

Append to `fyne/wordgame_test.go` and add `"fyne.io/fyne/v2"` to its imports:

```go
// wordTileAt is a word tile at (10, y) for a reading in gs.words.
func wordTileAt(gs *GameState, kana string, y float32) *KanaTile {
	e, ok := gs.words.Lookup(kana)
	if !ok {
		panic("no word " + kana)
	}
	tile := newWordTile(e, gs.canvasW)
	tile.fallSeconds = 20
	tile.Move(fyne.NewPos(10, y))
	return tile
}

func TestSpawnCandidatesIgnoreWordTiles(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("い", 5, "stomach"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "い", 0), tileAt("あ", "a", 0)}
	if got := gs.spawnCandidates([]string{"あ", "い"}); !equalIDs(got, []string{"い"}) {
		t.Fatalf("candidates = %v, want [い]: a falling word い must not block the kana い", got)
	}
}

func TestDropInactiveTilesRemovesUnplayableWords(t *testing.T) {
	gs := newTestState()
	gs.activeScripts[kanacore.ScriptKatakana] = true
	withWords(gs, bw("アイ", 5, "eye"), bw("あい", 5, "love"))
	secure(gs, "ア", "イ", "あ", "い")
	gs.tiles = []*KanaTile{wordTileAt(gs, "アイ", 0), wordTileAt(gs, "あい", 0), tileAt("あ", "a", 0)}
	gs.applySettings([]string{"vowels", "kata:vowels"}, []kanacore.Script{kanacore.ScriptHiragana}, false, 0, LevelNormal, pointsAuto)
	if len(gs.tiles) != 2 || gs.tiles[0].word == nil || gs.tiles[0].word.Kana != "あい" || gs.tiles[1].word != nil {
		t.Fatalf("expected あい (word) and あ (kana) to remain, got %d tiles", len(gs.tiles))
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 ./fyne/` — expected: compile errors (`newWordTile`, `wordTextSize`, `word` field undefined).

- [ ] **Step 3: Implement.**

Replace `fyne/tile.go` with:

```go
package main

import (
	"image/color"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"kana/kanacore"
	"kana/vocab"
)

const (
	tileW     float32 = 52 // single glyph
	tileWideW float32 = 84 // two glyphs (yōon)
	tileH     float32 = 60

	wordTextSize    float32 = 32
	minWordTextSize float32 = 10
	wordTilePad     float32 = 12 // space left and right of a word
)

var (
	tileFaceColor   = color.RGBA{R: 0xee, G: 0xdf, B: 0xc0, A: 0xff}
	tileShadowColor = color.RGBA{R: 0xb8, G: 0x95, B: 0x6a, A: 0xff}
	kanaTextColor   = color.RGBA{R: 0x2c, G: 0x1a, B: 0x0e, A: 0xff}

	wordTileFaceColor   = color.RGBA{R: 0xdc, G: 0xe6, B: 0xd2, A: 0xff}
	wordTileBorderColor = color.RGBA{R: 0x5e, G: 0x7f, B: 0x52, A: 0xff}
)

// tileWidthFor returns the tile width for a kana, wider for two-glyph kana.
func tileWidthFor(char string) float32 {
	if utf8.RuneCountInString(char) > 1 {
		return tileWideW
	}
	return tileW
}

// KanaTile is a falling card rendered as three canvas objects. It carries
// either a kana or, when word is set, a word; a word tile's kana is unset.
type KanaTile struct {
	kana        kanacore.Kana
	word        *vocab.Entry // nil for a kana tile
	pos         fyne.Position
	width       float32
	shadow      *canvas.Rectangle
	face        *canvas.Rectangle
	text        *canvas.Text
	fallSeconds float32 // time to cross the canvas; see fallStep
}

func newKanaTile(k kanacore.Kana) *KanaTile {
	w := tileWidthFor(k.Char)

	shadow := canvas.NewRectangle(tileShadowColor)
	shadow.Resize(fyne.NewSize(w, tileH))

	face := canvas.NewRectangle(tileFaceColor)
	face.Resize(fyne.NewSize(w, tileH))

	text := canvas.NewText(k.Char, kanaTextColor)
	text.TextSize = 32
	text.Resize(fyne.MeasureText(text.Text, text.TextSize, text.TextStyle))

	t := &KanaTile{kana: k, width: w, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// newWordTile builds a word tile: a sage face with a border, as wide as the
// word plus padding, so even a one-unit word (か) does not look like a kana
// tile. A word wider than maxW (the field width) gets a smaller font, down
// to minWordTextSize, so the tile fits.
func newWordTile(e vocab.Entry, maxW float32) *KanaTile {
	text := canvas.NewText(e.Kana, kanaTextColor)
	text.TextSize = wordTextSize
	size := fyne.MeasureText(text.Text, text.TextSize, text.TextStyle)
	avail := maxW - 2*wordTilePad
	for size.Width > avail && text.TextSize > minWordTextSize {
		text.TextSize--
		size = fyne.MeasureText(text.Text, text.TextSize, text.TextStyle)
	}
	text.Resize(size)
	w := size.Width + 2*wordTilePad

	shadow := canvas.NewRectangle(tileShadowColor)
	shadow.Resize(fyne.NewSize(w, tileH))

	face := canvas.NewRectangle(wordTileFaceColor)
	face.StrokeColor = wordTileBorderColor
	face.StrokeWidth = 2
	face.Resize(fyne.NewSize(w, tileH))

	word := e
	t := &KanaTile{word: &word, width: w, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// Width returns the tile's width in device-independent pixels.
func (t *KanaTile) Width() float32 {
	return t.width
}

// Move updates the positions of all three canvas objects atomically.
func (t *KanaTile) Move(pos fyne.Position) {
	t.pos = pos
	t.shadow.Move(fyne.NewPos(pos.X+3, pos.Y+3))
	t.face.Move(pos)
	ts := t.text.Size()
	t.text.Move(fyne.NewPos(pos.X+(t.width-ts.Width)/2, pos.Y+(tileH-ts.Height)/2))
}

// Objects returns the canvas objects for the renderer, shadow first so face renders on top.
func (t *KanaTile) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{t.shadow, t.face, t.text}
}
```

In `fyne/game.go`, in `spawnCandidates` replace

```go
	for _, t := range gs.tiles {
		falling[t.kana.Char] = true
	}
```

with

```go
	for _, t := range gs.tiles {
		if t.word == nil { // falling words do not block a kana
			falling[t.kana.Char] = true
		}
	}
```

and update its doc comment to "spawnCandidates drops kana that are already falling as kana tiles, unless that would leave none. Must be called under lock."

Replace `dropInactiveTiles` with:

```go
// dropInactiveTiles removes kana tiles whose row is deselected or whose
// script is inactive, and word tiles whose word is no longer playable. Must
// be called under lock.
func (gs *GameState) dropInactiveTiles() {
	var playable map[string]bool // built on the first word tile
	kept := gs.tiles[:0]
	for _, t := range gs.tiles {
		if t.word != nil {
			if playable == nil {
				playable = make(map[string]bool)
				for _, e := range gs.playableWords() {
					playable[e.Kana] = true
				}
			}
			if playable[t.word.Kana] {
				kept = append(kept, t)
			}
			continue
		}
		rowID, ok := kanacore.CharToRow[t.kana.Char]
		if ok && (!gs.selectedRows[rowID] || !gs.activeScripts[kanacore.ScriptOf(rowID)]) {
			continue
		}
		kept = append(kept, t)
	}
	gs.tiles = kept
}
```

- [ ] **Step 4: Run** `gofmt -w fyne/ && go test -count=1 ./fyne/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 5: Commit**

```bash
git add fyne/tile.go fyne/tile_test.go fyne/game.go fyne/wordgame_test.go
git commit -m "feat(fyne): add word tiles

A tile can carry a word: a bordered sage face as wide as the word, with a
smaller font when the word is wider than the field. Falling words do not
block kana spawns, and word tiles are dropped when their word stops being
playable."
```

---

### Task 5: Answers and misses for words

**Files:** Modify `fyne/game.go` (struct, `Reset`, `tick` lines 402–438, `checkAnswer` lines 484–519, `snapshot` lines 1101–1134), `fyne/stats.go` (`StatsSnapshot` lines 35–49), `fyne/difficulty.go` (`missReason` lines 274–277), `fyne/difficulty_test.go` (line 61), `fyne/app.go` (`showGameOverDialog` lines 158–214), `fyne/words.go`, `fyne/words_test.go`, `fyne/wordgame_test.go`.

- [ ] **Step 1: Write the failing tests.**

In `fyne/difficulty_test.go` change the `missReason` check to:

```go
	if got := missReason(5); got != "5 tiles slipped through." {
		t.Errorf("missReason = %q", got)
	}
```

Append to `fyne/words_test.go`:

```go
func TestMissedWordsText(t *testing.T) {
	if got := missedWordsText(nil); got != "" {
		t.Fatalf("no words = %q", got)
	}
	got := missedWordsText([]string{"せんせい", "きって", "せんせい", "コーヒー", "まっちゃ"})
	want := "きって (kitte), せんせい (sensei), まっちゃ (matcha)\nコーヒー (koohii)"
	if got != want {
		t.Fatalf("missedWordsText = %q, want %q", got, want)
	}
}
```

Append to `fyne/wordgame_test.go`:

```go
func TestWordHitScoresPerUnit(t *testing.T) {
	cases := []struct {
		level        string
		points, want int
	}{
		{"normal", pointsAuto, 30},
		{"pro", pointsAuto, 90},
		{"normal", 2, 60},
	}
	for _, c := range cases {
		gs := newTestState()
		gs.level = LevelByName(c.level)
		gs.pointsSetting = c.points
		withWords(gs, bw("あおい", 5, "blue"))
		gs.tiles = []*KanaTile{wordTileAt(gs, "あおい", 0)}
		gs.checkAnswer("aoi")
		if len(gs.tiles) != 0 || gs.score != c.want {
			t.Errorf("%s/%d: tiles %d, score %d, want %d", c.level, c.points, len(gs.tiles), gs.score, c.want)
		}
	}
}

func TestWordHitRecordsWordProgressOnly(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("しお", 5, "salt"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "しお", 0)}
	gs.checkAnswer("sio") // any valid reading
	if len(gs.tiles) != 0 {
		t.Fatal("しお not answered by 'sio'")
	}
	if gs.wordSessionStats["しお"].CorrectCount != 1 || gs.answerStreak != 1 {
		t.Fatalf("word stats %+v, streak %d", gs.wordSessionStats["しお"], gs.answerStreak)
	}
	if _, ok := gs.wordMemory["しお"]; !ok {
		t.Fatal("a word hit must create word memory")
	}
	if len(gs.sessionStats) != 0 || len(gs.memory) != 0 {
		t.Fatal("a word hit must not change any kana's stats")
	}
}

func TestLowerTileWinsAcrossKinds(t *testing.T) {
	for _, wordLower := range []bool{true, false} {
		gs := newTestState()
		withWords(gs, bw("え", 5, "picture"))
		wy, ky := float32(100), float32(300)
		if wordLower {
			wy, ky = 300, 100
		}
		word := wordTileAt(gs, "え", wy)
		kana := tileAt("え", "e", ky)
		gs.tiles = []*KanaTile{word, kana}
		gs.checkAnswer("e")
		if len(gs.tiles) != 1 {
			t.Fatalf("wordLower=%v: %d tiles left", wordLower, len(gs.tiles))
		}
		if wordLower && (gs.tiles[0] != kana || gs.wordSessionStats["え"].CorrectCount != 1 || len(gs.sessionStats) != 0) {
			t.Fatal("the lower word tile should have been answered")
		}
		if !wordLower && (gs.tiles[0] != word || gs.sessionStats["え"].CorrectCount != 1 || len(gs.wordSessionStats) != 0) {
			t.Fatal("the lower kana tile should have been answered")
		}
	}
}

func TestWordMiss(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	gs.wordMemory["あい"] = memory.State{Stability: 1, LastSeen: time.Now()}
	gs.tempoStep = 3
	gs.canvasH = 600
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 599)}
	gs.tick()
	if len(gs.tiles) != 0 || gs.missed != 1 {
		t.Fatalf("tiles %d, missed %d", len(gs.tiles), gs.missed)
	}
	if gs.wordSessionStats["あい"].MissCount != 1 {
		t.Fatalf("word stats %+v", gs.wordSessionStats["あい"])
	}
	if s := gs.wordMemory["あい"].Stability; s < 0.29 || s > 0.31 {
		t.Fatalf("word memory after miss = %v, want 0.3", s)
	}
	if gs.tempoStep != 1 {
		t.Fatalf("tempoStep = %d, want 1", gs.tempoStep)
	}
	if !equalIDs(gs.missedWords, []string{"あい"}) || len(gs.missedKanas) != 0 || len(gs.sessionStats) != 0 {
		t.Fatalf("missedWords %v, missedKanas %v", gs.missedWords, gs.missedKanas)
	}
}

func TestWordMissesCountTowardsLimit(t *testing.T) {
	gs := newTestState()
	gs.level = LevelByName("pro")
	gs.missed = 4
	gs.canvasH = 600
	withWords(gs, bw("あい", 5, "love"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 599)}
	gs.tick()
	if !gs.over || gs.overReason != "misses" {
		t.Fatalf("over=%v reason=%q", gs.over, gs.overReason)
	}
}

func TestSnapshotCarriesMissedWordsAndResetClears(t *testing.T) {
	gs := newTestState()
	gs.missedWords = []string{"あい"}
	if snap := gs.snapshot(); !equalIDs(snap.MissedWords, []string{"あい"}) {
		t.Fatalf("snapshot MissedWords = %v", snap.MissedWords)
	}
	gs.Reset()
	if len(gs.missedWords) != 0 {
		t.Fatal("Reset must clear missed words")
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 ./fyne/` — expected: compile errors (`missedWordsText`, `missedWords`, `MissedWords` undefined). After fixing compilation the `missReason` test fails until the text changes.

- [ ] **Step 3: Implement.**

In `fyne/difficulty.go`:

```go
// missReason is the game-over reason after too many misses (kana and words).
func missReason(missed int) string {
	return fmt.Sprintf("%d tiles slipped through.", missed)
}
```

In `fyne/game.go`, add to the struct after `missedKanas []kanacore.Kana`:

```go
	missedWords []string // readings of missed words, in order
```

In `Reset`, after `gs.missedKanas = nil` add `gs.missedWords = nil`.

In `tick`, replace the miss block

```go
		if tile.pos.Y > gs.canvasH {
			missedNow = true
			gs.recordMiss(tile.kana.Char)
			gs.missedKanas = append(gs.missedKanas, tile.kana)
			gs.tiles = append(gs.tiles[:i], gs.tiles[i+1:]...)
```

with

```go
		if tile.pos.Y > gs.canvasH {
			missedNow = true
			if tile.word != nil {
				gs.recordWordMiss(tile.word.Kana)
				gs.missedWords = append(gs.missedWords, tile.word.Kana)
			} else {
				gs.recordMiss(tile.kana.Char)
				gs.missedKanas = append(gs.missedKanas, tile.kana)
			}
			gs.tiles = append(gs.tiles[:i], gs.tiles[i+1:]...)
```

(the rest of the block — `gs.missed++`, `checkMissedLimit`, `break` — is unchanged).

Replace `checkAnswer` with:

```go
// checkAnswer removes the lowest tile that input answers: a kana tile by its
// canonical or alternative romaji, a word tile by any valid reading of the
// whole word. Acquires the lock itself; releases before triggering
// canvas.Refresh() so the UI updates immediately on correct answers.
func (gs *GameState) checkAnswer(input string) {
	gs.mu.Lock()
	if gs.paused || gs.over {
		gs.mu.Unlock()
		return
	}
	best := -1
	for i, tile := range gs.tiles {
		if !gs.answers(tile, input) {
			continue
		}
		if best < 0 || tile.pos.Y > gs.tiles[best].pos.Y {
			best = i
		}
	}
	matched := best >= 0
	if matched {
		tile := gs.tiles[best]
		gs.tiles = append(gs.tiles[:best], gs.tiles[best+1:]...)
		announced := false
		if tile.word != nil {
			gs.score += 10 * len(tile.word.Units) * gs.pointsFactor()
			gs.recordWordCorrect(tile.word.Kana)
		} else {
			gs.score += 10 * gs.pointsFactor()
			announced = gs.recordCorrect(tile.kana.Char)
		}
		if gs.scoreLimit > 0 && gs.score >= gs.scoreLimit {
			gs.endGame("score")
		}
		if !announced {
			gs.maybeOfferKatakana()
		}
		gs.buildSnapshot()
	}
	canvas := gs.canvas
	gs.mu.Unlock()

	if matched && canvas != nil {
		canvas.Refresh()
	}
}

// answers reports whether input answers the tile. Must be called under lock.
func (gs *GameState) answers(tile *KanaTile, input string) bool {
	if tile.word != nil {
		return kanacore.ReadingMatches(tile.word.Kana, input)
	}
	return gs.charSet.Matches(tile.kana.Char, input)
}
```

In `snapshot`, add to the returned literal after `MissedKanas: …`:

```go
		MissedWords:   append([]string{}, gs.missedWords...),
```

In `fyne/stats.go`, add to `StatsSnapshot` after `MissedKanas []kanacore.Kana`:

```go
	MissedWords   []string // readings of missed words, in order
```

Append to `fyne/words.go` and make its import block exactly:

```go
import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"fyne.io/fyne/v2/lang"
	"kana/kanacore"
	"kana/store"
	"kana/vocab"
)
```

```go
// missedWordsText lists each missed word once, sorted, as
// "せんせい (sensei)", three per line so the game-over dialog stays narrow.
// "" when no word was missed.
func missedWordsText(words []string) string {
	seen := map[string]bool{}
	var uniq []string
	for _, w := range words {
		if !seen[w] {
			seen[w] = true
			uniq = append(uniq, w)
		}
	}
	sort.Strings(uniq)
	var lines []string
	for i := 0; i < len(uniq); i += 3 {
		end := min(i+3, len(uniq))
		parts := make([]string, 0, 3)
		for _, w := range uniq[i:end] {
			parts = append(parts, fmt.Sprintf("%s (%s)", w, kanacore.Romaji(w)))
		}
		lines = append(lines, strings.Join(parts, ", "))
	}
	return strings.Join(lines, "\n")
}
```

In `fyne/app.go` `showGameOverDialog`, directly after the `content := container.NewVBox(...)` statement (before `dialog.ShowCustomConfirm`) add:

```go
	if words := missedWordsText(snap.MissedWords); words != "" {
		content.Add(widget.NewLabel("Words missed:"))
		content.Add(widget.NewLabel(words))
	}
```

- [ ] **Step 4: Run** `gofmt -w fyne/ && go test -count=1 ./fyne/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/stats.go fyne/difficulty.go fyne/difficulty_test.go fyne/app.go fyne/words.go fyne/words_test.go fyne/wordgame_test.go
git commit -m "feat(fyne): answer and miss word tiles

A word tile is answered by any valid reading of the whole word and scores
10 points per kana unit; the lowest matching tile wins across kana and
words. A missed word counts towards the miss limit, lowers the tempo and is
listed with its romaji in the game-over dialog."
```

---

### Task 6: Spawning words

**Files:** Modify `fyne/difficulty.go`, `fyne/difficulty_test.go`, `fyne/game.go` (`spawnKana` lines 440–479), `fyne/wordgame.go`, `fyne/wordgame_test.go`.

- [ ] **Step 1: Write the failing tests.**

Append to `fyne/difficulty_test.go`:

```go
func TestWordShare(t *testing.T) {
	cases := map[int]float64{0: 0, 10: 0.1, 30: 0.3, 500: 0.3}
	for n, want := range cases {
		if got := wordShare(n); math.Abs(got-want) > 1e-9 {
			t.Errorf("wordShare(%d) = %v, want %v", n, got, want)
		}
	}
}

func TestWordFallFactor(t *testing.T) {
	cases := map[int]float32{1: 1, 2: 1.2, 3: 1.4, 6: 2, 9: 2}
	for units, want := range cases {
		if got := wordFallFactor(units); got-want > 1e-5 || want-got > 1e-5 {
			t.Errorf("wordFallFactor(%d) = %v, want %v", units, got, want)
		}
	}
}
```

and add `"math"` to `fyne/difficulty_test.go`'s imports.

Append to `fyne/wordgame_test.go` and add `"math/rand"` to its imports:

```go
// vowelWords returns n distinct words written with vowels only (at most 150).
func vowelWords(n int) []vocab.Word {
	vowels := []string{"あ", "い", "う", "え", "お"}
	var out []vocab.Word
	for _, a := range vowels {
		for _, b := range vowels {
			for _, c := range append([]string{""}, vowels...) {
				if len(out) == n {
					return out
				}
				out = append(out, bw(a+b+c, 5, "word"))
			}
		}
	}
	return out
}

func countWordSpawns(gs *GameState, spawns int) int {
	words := 0
	for i := 0; i < spawns; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if gs.tiles[0].word != nil {
			words++
		}
	}
	return words
}

func TestWordShareOfSpawns(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.rnd = rand.New(rand.NewSource(7)).Float64
	withWords(gs, vowelWords(60)...)
	if n := countWordSpawns(gs, 200); n != 0 {
		t.Fatalf("%d words spawned while no vowel is secure", n)
	}
	secure(gs, "あ", "い", "う", "え", "お")
	if n := countWordSpawns(gs, 1000); n < 250 || n > 350 {
		t.Fatalf("%d/1000 word spawns with 60 playable, want about 300", n)
	}
}

func TestFewPlayableWordsSpawnRarely(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.rnd = rand.New(rand.NewSource(3)).Float64
	withWords(gs, vowelWords(10)...)
	secure(gs, "あ", "い", "う", "え", "お")
	if n := countWordSpawns(gs, 1000); n < 60 || n > 140 {
		t.Fatalf("%d/1000 word spawns with 10 playable, want about 100", n)
	}
}

func TestWordSpawnFallTimeAndPlacement(t *testing.T) {
	gs := newTestState() // Normal: 16–24 s
	selectRows(gs, "vowels")
	gs.rnd = func() float64 { return 0 } // always a word, always the first new candidate
	gs.canvasW = 150
	withWords(gs, bw("あおい", 5, "blue"))
	secure(gs, "あ", "お", "い")
	for i := 0; i < 30; i++ {
		gs.tiles = nil
		gs.spawnKana()
		tile := gs.tiles[0]
		if tile.word == nil || tile.word.Kana != "あおい" {
			t.Fatalf("spawned %+v, want the word あおい", tile.kana)
		}
		if tile.fallSeconds < 16*1.4-0.01 || tile.fallSeconds > 24*1.4+0.01 {
			t.Fatalf("fallSeconds = %v, want [22.4, 33.6]", tile.fallSeconds)
		}
		if tile.pos.X < 0 || tile.pos.X+tile.Width() > gs.canvasW+0.5 {
			t.Fatalf("tile at x=%.1f width %.1f leaves the %v px field", tile.pos.X, tile.Width(), gs.canvasW)
		}
	}
}

func TestWordSpawnAvoidsFallingWords(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.rnd = func() float64 { return 0 }
	withWords(gs, bw("あい", 5, "love"), bw("あお", 5, "blue"))
	secure(gs, "あ", "い", "お")
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 0)}
	gs.spawnKana()
	if got := gs.tiles[1].word; got == nil || got.Kana != "あお" {
		t.Fatal("expected あお while あい is falling")
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 0), wordTileAt(gs, "あお", 0)}
	gs.spawnKana()
	if got := gs.tiles[2].word; got == nil || got.Kana != "あい" {
		t.Fatal("with every word falling, the exclusion must be dropped")
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 ./fyne/` — expected: compile errors (`wordShare`, `wordFallFactor` undefined). With stubs, `TestWordShareOfSpawns` and the spawn tests fail (no words spawn).

- [ ] **Step 3: Implement.**

Append to `fyne/difficulty.go`:

```go
// wordShare is the chance that a spawn is a word: one percent per playable
// word, at most 30 %.
func wordShare(playable int) float64 {
	return math.Min(0.3, float64(playable)/100)
}

// wordFallFactor gives longer words more time: 20 % per kana unit after the
// first, at most double. It replaces the per-character factor for words.
func wordFallFactor(units int) float32 {
	f := 1 + 0.2*float32(units-1)
	if f > 2 {
		return 2
	}
	if f < 1 {
		return 1
	}
	return f
}
```

Append to `fyne/wordgame.go`:

```go
// wordCandidates returns the playable readings without those already
// falling as word tiles, unless that would leave none. Must be called under
// lock.
func (gs *GameState) wordCandidates(playable []vocab.Entry) []string {
	falling := make(map[string]bool)
	for _, t := range gs.tiles {
		if t.word != nil {
			falling[t.word.Kana] = true
		}
	}
	all := make([]string, 0, len(playable))
	free := make([]string, 0, len(playable))
	for _, e := range playable {
		all = append(all, e.Kana)
		if !falling[e.Kana] {
			free = append(free, e.Kana)
		}
	}
	if len(free) == 0 {
		return all
	}
	return free
}

// pickWord decides with gs.rnd whether this spawn is a word (wordShare) and,
// if so, picks one with memory.Pick over the word memory. With no playable
// word it returns false without drawing from gs.rnd. Must be called under
// lock.
func (gs *GameState) pickWord() (vocab.Entry, bool) {
	playable := gs.playableWords()
	if len(playable) == 0 || gs.rnd() >= wordShare(len(playable)) {
		return vocab.Entry{}, false
	}
	kana := memory.Pick(gs.wordCandidates(playable), gs.wordMemory, gs.now(), gs.rnd)
	return gs.words.Lookup(kana)
}
```

Replace `spawnKana` in `fyne/game.go` with:

```go
// spawnKana adds one falling tile: a word with probability wordShare when
// words are playable, otherwise a kana.
func (gs *GameState) spawnKana() {
	gs.mu.Lock()
	if gs.over || gs.paused {
		gs.mu.Unlock()
		return
	}

	var tile *KanaTile
	base := gs.level.MinFall + rand.Float32()*(gs.level.MaxFall-gs.level.MinFall)
	if entry, ok := gs.pickWord(); ok {
		tile = newWordTile(entry, gs.canvasW)
		tile.fallSeconds = base * wordFallFactor(len(entry.Units)) * tempoFactor(gs.tempoStep)
	} else {
		chars := gs.availableCharacters()
		if len(chars) == 0 {
			gs.mu.Unlock()
			return
		}
		char := memory.Pick(gs.spawnCandidates(chars), gs.memory, gs.now(), gs.rnd)
		romaji, _ := gs.charSet.GetRomaji(char)
		tile = newKanaTile(kanacore.Kana{Char: char, Romaji: romaji})
		correct := gs.overallStats[char].CorrectCount + gs.sessionStats[char].CorrectCount
		tile.fallSeconds = base * charFallFactor(correct) * tempoFactor(gs.tempoStep)
	}

	maxX := gs.canvasW - tile.Width()
	if maxX < 0 {
		maxX = 0
	}
	x := rand.Float32() * maxX
	tile.Move(fyne.NewPos(x, 0))
	gs.tiles = append(gs.tiles, tile)

	gs.buildSnapshot()
	canvas := gs.canvas
	gs.mu.Unlock()

	if canvas != nil {
		fyne.Do(func() { canvas.Refresh() })
	}
}
```

`TestSpawnPrefersWeakCharacter` and `TestSpawnAvoidsFallingDuplicates` keep passing: `newTestState` has no word index, so `pickWord` returns before drawing from `gs.rnd` and the seeded sequence is unchanged.

- [ ] **Step 4: Run** `gofmt -w fyne/ && go test -count=1 ./fyne/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 5: Commit**

```bash
git add fyne/difficulty.go fyne/difficulty_test.go fyne/game.go fyne/wordgame.go fyne/wordgame_test.go
git commit -m "feat(fyne): spawn words among the kana

A spawn is a word with probability min(0.3, playable/100), chosen with the
word memory and skipping words already falling. Words fall for the level's
time scaled by 20 % per extra kana unit (at most double) and the tempo."
```

---

### Task 7: Feedback labels

**Files:** Create `fyne/feedback.go`. Modify `fyne/game.go` (struct, `Reset`, `tick`, `checkAnswer`, `buildSnapshot` lines 1086–1098), `fyne/words.go`, `fyne/words_test.go`, `fyne/wordgame_test.go`.

- [ ] **Step 1: Write the failing tests.**

Append to `fyne/words_test.go` and make its import block exactly:

```go
import (
	"strings"
	"testing"

	"kana/vocab"
)
```

```go
func TestMeaningSnippet(t *testing.T) {
	m := vocab.Meanings{"en": {"yes", "that is correct", "right", "indeed"}, "de": {"ja"}}
	if got := meaningSnippet(m, "en"); got != "yes; that is correct; right" {
		t.Fatalf("en = %q, want the first three items", got)
	}
	if got := meaningSnippet(m, "de"); got != "ja" {
		t.Fatalf("de = %q", got)
	}
	if got := meaningSnippet(m, "fr"); got != "yes; that is correct; right" {
		t.Fatalf("fr = %q, want the English fallback", got)
	}
	long := vocab.Meanings{"en": {strings.Repeat("a", 40), strings.Repeat("b", 40)}}
	got := meaningSnippet(long, "en")
	if r := []rune(got); len(r) != maxMeaningRunes || !strings.HasSuffix(got, "…") {
		t.Fatalf("long = %q (%d runes), want %d runes ending in …", got, len(r), maxMeaningRunes)
	}
	if got := meaningSnippet(nil, "en"); got != "" {
		t.Fatalf("no meanings = %q", got)
	}
}

func TestFeedbackTexts(t *testing.T) {
	hai := vocab.Entry{Kana: "はい", Meanings: vocab.Meanings{"en": {"yes"}}}
	if got := hitFeedbackText(hai, "en"); got != "はい — yes" {
		t.Errorf("hit = %q", got)
	}
	if got := missFeedbackText(hai, "en"); got != "はい · hai — yes" {
		t.Errorf("miss = %q", got)
	}
	bare := vocab.Entry{Kana: "はい"}
	if got := hitFeedbackText(bare, "en"); got != "はい" {
		t.Errorf("hit without meanings = %q", got)
	}
	if got := missFeedbackText(bare, "en"); got != "はい · hai" {
		t.Errorf("miss without meanings = %q", got)
	}
}
```

Append to `fyne/wordgame_test.go`:

```go
func TestWordHitShowsFeedbackAtTile(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 200), tileAt("か", "ka", 0)}
	gs.checkAnswer("ka")
	if len(gs.feedback) != 0 {
		t.Fatal("a kana hit shows no feedback")
	}
	gs.checkAnswer("ai")
	if len(gs.feedback) != 1 {
		t.Fatalf("feedback = %d labels, want 1", len(gs.feedback))
	}
	f := gs.feedback[0]
	if f.kind != feedbackHit || f.text.Text != "あい — love" || f.ticksLeft != feedbackTicks {
		t.Fatalf("label = %v %q %d", f.kind, f.text.Text, f.ticksLeft)
	}
	if p := f.text.Position(); p.X != 10 || p.Y != 200 {
		t.Fatalf("label at %v, want the tile's last position (10, 200)", p)
	}
}

func TestWordMissShowsFeedbackAtBottom(t *testing.T) {
	gs := newTestState()
	gs.canvasH = 600
	withWords(gs, bw("あい", 5, "love"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 599)}
	gs.tick()
	if len(gs.feedback) != 1 {
		t.Fatalf("feedback = %d labels, want 1", len(gs.feedback))
	}
	f := gs.feedback[0]
	if f.kind != feedbackMiss || f.text.Text != "あい · ai — love" || f.text.Color != cellMissedColor {
		t.Fatalf("label = %v %q %v", f.kind, f.text.Text, f.text.Color)
	}
	if p := f.text.Position(); p.Y < 600-60 || p.Y+f.text.Size().Height > 600 {
		t.Fatalf("miss label at y=%v, want at the bottom of the 600 px field", p.Y)
	}
}

func TestFeedbackExpiresAfter20Ticks(t *testing.T) {
	gs := newTestState()
	gs.addFeedback(feedbackHit, "あい — love", fyne.NewPos(10, 10))
	for i := 0; i < feedbackTicks-1; i++ {
		gs.tick()
	}
	if len(gs.feedback) != 1 {
		t.Fatal("label gone before 20 ticks")
	}
	gs.tick()
	if len(gs.feedback) != 0 {
		t.Fatal("label still shown after 20 ticks")
	}
}

func TestFeedbackFreezesAndHidesWhilePaused(t *testing.T) {
	gs := newTestState()
	gs.addFeedback(feedbackHit, "あい — love", fyne.NewPos(10, 10))
	gs.buildSnapshot()
	if n := len(tileObjects(gs)); n != 1 {
		t.Fatalf("snapshot objects = %d, want the label", n)
	}
	gs.TogglePause()
	if n := len(tileObjects(gs)); n != 0 {
		t.Fatalf("label visible while paused: %d objects", n)
	}
	for i := 0; i < 30; i++ {
		gs.tick()
	}
	if len(gs.feedback) != 1 || gs.feedback[0].ticksLeft != feedbackTicks {
		t.Fatal("labels must not count down while paused")
	}
	gs.TogglePause()
	if n := len(tileObjects(gs)); n != 1 {
		t.Fatalf("label hidden after resuming: %d objects", n)
	}
}

func TestResetClearsFeedback(t *testing.T) {
	gs := newTestState()
	gs.addFeedback(feedbackMiss, "あい · ai — love", fyne.NewPos(10, 10))
	gs.Reset()
	if len(gs.feedback) != 0 || len(tileObjects(gs)) != 0 {
		t.Fatal("Reset must clear feedback labels")
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 ./fyne/` — expected: compile errors (`meaningSnippet`, `maxMeaningRunes`, `hitFeedbackText`, `missFeedbackText`, `feedback`, `feedbackHit`, `feedbackTicks`, `addFeedback` undefined).

- [ ] **Step 3: Implement.**

Append to `fyne/words.go` (imports unchanged from Task 5):

```go
const (
	maxMeaningItems = 3  // meanings shown after a hit or miss
	maxMeaningRunes = 60 // longer meaning text ends in an ellipsis
)

// truncateRunes shortens s to at most n runes, ending in "…" when cut.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimRight(string(r[:n-1]), " ;,") + "…"
}

// meaningSnippet is the meaning text for feedback: the first three meanings
// in lang (with vocab's fallback to English and any language), cut to about
// 60 characters. "" when the word has no meanings.
func meaningSnippet(m vocab.Meanings, lang string) string {
	items := m.For(lang)
	if len(items) > maxMeaningItems {
		items = items[:maxMeaningItems]
	}
	return truncateRunes(strings.Join(items, "; "), maxMeaningRunes)
}

// hitFeedbackText is shown after a word hit: "はい — yes".
func hitFeedbackText(e vocab.Entry, lang string) string {
	if m := meaningSnippet(e.Meanings, lang); m != "" {
		return e.Kana + " — " + m
	}
	return e.Kana
}

// missFeedbackText is shown after a missed word, with the solution:
// "はい · hai — yes".
func missFeedbackText(e vocab.Entry, lang string) string {
	s := e.Kana + " · " + kanacore.Romaji(e.Kana)
	if m := meaningSnippet(e.Meanings, lang); m != "" {
		s += " — " + m
	}
	return s
}
```

Create `fyne/feedback.go`:

```go
package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type feedbackKind int

const (
	feedbackHit feedbackKind = iota
	feedbackMiss
)

const (
	feedbackTicks             = 20 // 2 s at 100 ms ticks
	feedbackTextSize  float32 = 18
	feedbackBottomGap float32 = 8
)

var feedbackHitColor = color.RGBA{R: 0x3d, G: 0x6b, B: 0x35, A: 0xff}

// feedbackLabel is a short text shown after a word hit or miss. The text
// object carries the message and its position; it is created once and
// published in the tile snapshot.
type feedbackLabel struct {
	kind      feedbackKind
	text      *canvas.Text
	ticksLeft int // tick() counts down; the label goes at 0
}

// addFeedback shows msg for feedbackTicks ticks. A hit label sits at pos
// (the tile's last position), a miss label at the bottom of the field in the
// error colour; both are kept inside the field. Must be called under lock.
func (gs *GameState) addFeedback(kind feedbackKind, msg string, pos fyne.Position) {
	var c color.Color = feedbackHitColor
	if kind == feedbackMiss {
		c = cellMissedColor
	}
	txt := canvas.NewText(msg, c)
	txt.TextSize = feedbackTextSize
	txt.TextStyle = fyne.TextStyle{Bold: true}
	size := txt.MinSize()
	txt.Resize(size)
	y := pos.Y
	if kind == feedbackMiss {
		y = gs.canvasH - size.Height - feedbackBottomGap
	}
	txt.Move(fyne.NewPos(clampTo(pos.X, gs.canvasW-size.Width), clampTo(y, gs.canvasH-size.Height)))
	gs.feedback = append(gs.feedback, &feedbackLabel{kind: kind, text: txt, ticksLeft: feedbackTicks})
}

// clampTo keeps v within [0, hi]; a negative hi gives 0.
func clampTo(v, hi float32) float32 {
	if v > hi {
		v = hi
	}
	if v < 0 {
		v = 0
	}
	return v
}

// ageFeedback counts every label down by one tick and drops expired ones.
// Only tick() calls it, so labels freeze while the game is paused. Must be
// called under lock.
func (gs *GameState) ageFeedback() {
	kept := gs.feedback[:0]
	for _, f := range gs.feedback {
		f.ticksLeft--
		if f.ticksLeft > 0 {
			kept = append(kept, f)
		}
	}
	gs.feedback = kept
}
```

In `fyne/game.go`:

- Struct, after `missedWords`: `feedback []*feedbackLabel // word hit/miss labels, oldest first`.
- `Reset`: after `gs.missedWords = nil` add `gs.feedback = nil`.
- `tick`: directly after the `if gs.over || gs.paused { … return }` block, before `missedNow := false`, add `gs.ageFeedback()`.
- `tick` word miss branch (from Task 5) becomes:

  ```go
			if tile.word != nil {
				gs.recordWordMiss(tile.word.Kana)
				gs.missedWords = append(gs.missedWords, tile.word.Kana)
				gs.addFeedback(feedbackMiss, missFeedbackText(*tile.word, gs.meaningLang), tile.pos)
			} else {
  ```

- `checkAnswer` word branch (from Task 5) becomes:

  ```go
		if tile.word != nil {
			gs.score += 10 * len(tile.word.Units) * gs.pointsFactor()
			gs.recordWordCorrect(tile.word.Kana)
			gs.addFeedback(feedbackHit, hitFeedbackText(*tile.word, gs.meaningLang), tile.pos)
		} else {
  ```

- Replace `buildSnapshot` with:

  ```go
// buildSnapshot rebuilds the atomic snapshot of canvas objects: tiles, then
// feedback labels on top. While the learner has paused or the settings
// dialog is open, nothing is published, so the pause cannot be used to
// study the tiles; the canvas shows the hint for a user pause only. Must be
// called under lock.
func (gs *GameState) buildSnapshot() {
	hide := gs.userPaused || gs.settingsOpen
	gs.showPauseHint.Store(gs.userPaused)
	if hide {
		gs.objectSnapshot.Store([]fyne.CanvasObject{})
		return
	}
	objs := make([]fyne.CanvasObject, 0, len(gs.tiles)*3+len(gs.feedback))
	for _, tile := range gs.tiles {
		objs = append(objs, tile.Objects()...)
	}
	for _, f := range gs.feedback {
		objs = append(objs, f.text)
	}
	gs.objectSnapshot.Store(objs)
}
  ```

`canvas.go` needs no change: `gameCanvasRenderer.Objects` already renders whatever the snapshot holds, above the background.

- [ ] **Step 4: Run** `gofmt -w fyne/ && go test -count=1 ./fyne/` — expected: PASS (the pause tests that count 3 objects per tile still pass: they have no labels). Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 5: Commit**

```bash
git add fyne/feedback.go fyne/game.go fyne/words.go fyne/words_test.go fyne/wordgame_test.go
git commit -m "feat(fyne): show the meaning after a word hit or miss

A hit shows the word and its meaning where the tile was; a miss shows
word, romaji and meaning at the bottom in the error colour. Labels last 20
ticks, freeze and hide while paused, and are cleared by Reset. Meanings
follow meaning_lang and are cut to three items and about 60 characters."
```

---

### Task 8: Status panel — words line and attribution

**Files:** Modify `fyne/stats.go` (`StatsSnapshot`, `StatsPanel` struct lines 106–123, `newStatsPanel` lines 136–280, `Update` lines 317–328), `fyne/game.go` (`snapshot`), `fyne/words.go`, `fyne/words_test.go`, `fyne/wordgame.go`, `fyne/wordgame_test.go`, `fyne/stats_test.go`.

- [ ] **Step 1: Write the failing tests.**

Append to `fyne/words_test.go`:

```go
func TestFormatWordsLine(t *testing.T) {
	if got := formatWordsLine(WordsStatus{Playable: 42, Secure: 17, Level: 5}); got != "Words: 42 playable · 17 secure · up to N5" {
		t.Errorf("line = %q", got)
	}
	if got := formatWordsLine(WordsStatus{Err: "broken data", Playable: 3, Level: 5}); got != "Words unavailable: broken data" {
		t.Errorf("error line = %q", got)
	}
	long := formatWordsLine(WordsStatus{Err: strings.Repeat("x", 100)})
	if r := []rune(long); len(r) != maxWordsLineRunes || !strings.HasSuffix(long, "…") {
		t.Errorf("long error = %q (%d runes)", long, len(r))
	}
}
```

Append to `fyne/wordgame_test.go`:

```go
func TestSnapshotWordsStatus(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"), bw("あお", 5, "blue"), bw("いえ", 5, "house"))
	secure(gs, "あ", "い", "お")
	gs.wordOverallStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 2}
	gs.wordSessionStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 1}
	snap := gs.snapshot()
	if snap.Words != (WordsStatus{Playable: 2, Secure: 1, Level: 5}) {
		t.Fatalf("Words = %+v, want 2 playable, 1 secure, N5", snap.Words)
	}
	if snap.WordCorrect != 1 {
		t.Fatalf("WordCorrect = %d, want the session's 1", snap.WordCorrect)
	}
}

func TestSnapshotShowsBundledError(t *testing.T) {
	test.NewApp()
	orig := bundledWords
	bundledWords = func() ([]vocab.Word, error) { return nil, errors.New("broken data") }
	t.Cleanup(func() { bundledWords = orig })
	gs := NewGameState(nil)
	gs.mu.Lock()
	snap := gs.snapshot()
	gs.mu.Unlock()
	if got := formatWordsLine(snap.Words); got != "Words unavailable: broken data" {
		t.Fatalf("line = %q", got)
	}
}
```

Append to `fyne/stats_test.go`:

```go
func TestPanelShowsWordsLineAndSource(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
	snap.Words = WordsStatus{Playable: 42, Secure: 17, Level: 5}
	p.Update(snap)
	if p.wordsText.Text != "Words: 42 playable · 17 secure · up to N5" {
		t.Fatalf("words line = %q", p.wordsText.Text)
	}
	var lines []string
	for _, txt := range p.sourceTexts {
		lines = append(lines, txt.Text)
	}
	if got := strings.Join(lines, " "); got != "Word data: JMdict (EDRDG, CC BY-SA 4.0), JLPT levels by J. Waller" {
		t.Fatalf("attribution = %q", got)
	}
}

func TestPanelCountsWordHits(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
	snap.SessionStats["あ"] = store.KanaStats{Char: "あ", CorrectCount: 1}
	snap.WordCorrect = 1
	snap.Missed = 2
	p.Update(snap)
	if p.correctText.Text != "2" || p.accuracyText.Text != "50 %" {
		t.Fatalf("correct %q, accuracy %q; want 2 and 50 %%", p.correctText.Text, p.accuracyText.Text)
	}
}
```

and add `"strings"` to `fyne/stats_test.go`'s imports.

- [ ] **Step 2: Run** `go test -count=1 ./fyne/` — expected: compile errors (`WordsStatus`, `formatWordsLine`, `maxWordsLineRunes`, `snap.Words`, `snap.WordCorrect`, `p.wordsText`, `p.sourceTexts` undefined).

- [ ] **Step 3: Implement.**

Append to `fyne/words.go`:

```go
// maxWordsLineRunes keeps the words line inside the 292 px status panel.
const maxWordsLineRunes = 48

// wordDataSource attributes the bundled word data; two lines so it fits the
// status panel. Stage 4 moves it to the settings' Sources tab.
var wordDataSource = []string{"Word data: JMdict (EDRDG, CC BY-SA 4.0),", "JLPT levels by J. Waller"}

// WordsStatus is the status panel's words line.
type WordsStatus struct {
	Err      string // the bundled list failed to load
	Playable int    // words playable now
	Secure   int    // playable words with 3 or more correct answers
	Level    int    // words are played up to this JLPT level
}

// formatWordsLine is the status panel's words line.
func formatWordsLine(ws WordsStatus) string {
	if ws.Err != "" {
		return truncateRunes("Words unavailable: "+ws.Err, maxWordsLineRunes)
	}
	return fmt.Sprintf("Words: %d playable · %d secure · up to N%d", ws.Playable, ws.Secure, ws.Level)
}
```

Append to `fyne/wordgame.go`:

```go
// wordsStatus summarises the words for the status panel. Must be called
// under lock.
func (gs *GameState) wordsStatus() WordsStatus {
	ws := WordsStatus{Err: gs.wordsErr, Level: gs.wordLevel}
	for _, e := range gs.playableWords() {
		ws.Playable++
		if gs.wordCorrect(e.Kana) >= secureCorrect {
			ws.Secure++
		}
	}
	return ws
}

// sessionWordCorrect is the number of word hits this session. Must be
// called under lock.
func (gs *GameState) sessionWordCorrect() int {
	n := 0
	for _, st := range gs.wordSessionStats {
		n += st.CorrectCount
	}
	return n
}
```

In `fyne/game.go` `snapshot`, add to the returned literal after `MissLimit: …`:

```go
		WordCorrect:   gs.sessionWordCorrect(),
		Words:         gs.wordsStatus(),
```

In `fyne/stats.go`:

Add to `StatsSnapshot` after `MissLimit int`:

```go

	WordCorrect int         // word hits this session
	Words       WordsStatus // the words line
```

Add to the `StatsPanel` struct after `correctText, missedText, accuracyText *canvas.Text`:

```go

	wordsText   *canvas.Text   // "Words: … playable · … secure · up to N5"
	sourceTexts []*canvas.Text // word data attribution
```

In `newStatsPanel`, replace

```go
	top := container.NewVBox(session)
```

with

```go
	p.wordsText = smallText("", 11)
	p.wordsText.Color = panelSubColor
	top := container.NewVBox(session, p.wordsText)
```

and replace the final `p.content = container.NewBorder(top, legend, nil, nil, container.NewVScroll(padded))` with

```go
	source := container.NewVBox()
	for _, line := range wordDataSource {
		txt := smallText(line, 10)
		txt.Color = panelSubColor
		p.sourceTexts = append(p.sourceTexts, txt)
		source.Add(txt)
	}
	p.content = container.NewBorder(top, container.NewVBox(legend, source), nil, nil, container.NewVScroll(padded))
```

In `Update`, replace

```go
	correct := 0
	for _, st := range snap.SessionStats {
```

with

```go
	correct := snap.WordCorrect
	for _, st := range snap.SessionStats {
```

and after the `accuracyText` if/else add:

```go
	p.wordsText.Text = formatWordsLine(snap.Words)
```

- [ ] **Step 4: Run** `gofmt -w fyne/ && go test -count=1 ./fyne/` — expected: PASS (`TestPanelHidesEmptyGroupsAndKeepsWidth` still sees a 288 px grid). Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 5: Manual check.** `go run ./fyne/` with a learner who has secured the vowels: the panel shows the words line under the session tiles and the two attribution lines under the legend, neither wider than the panel; words fall among the kana with a green border; typing a word shows its meaning; letting one fall shows reading and meaning at the bottom. Stop the app with the window's close button.

- [ ] **Step 6: Commit**

```bash
git add fyne/stats.go fyne/stats_test.go fyne/game.go fyne/words.go fyne/words_test.go fyne/wordgame.go fyne/wordgame_test.go
git commit -m "feat(fyne): show words and their source in the status panel

The panel shows how many words are playable and secure up to the level, or
why words are unavailable, and attributes the word data to JMdict and
Waller's JLPT lists. Word hits count in the session's correct and
accuracy."
```

---

### Task 9: Docs

**Files:** Modify `README.md`, `PLAN.md`.

- [ ] **Step 1: README.md**

1. Under `### Core Gameplay`, after the **Romaji Input** bullet, add:

   ```markdown
   - **Words**: once you can read them, Japanese words fall among the kana tiles (green border).
     A word appears when every kana in it is secure (answered correctly three times) in an
     active script: bundled JLPT words up to your level (N5 by default) and every word from your
     own lists. Type the whole reading (`sensei`, `kitte`, `koohii`, `ko-hi-`); a hit shows the
     meaning, a missed word shows its romaji and meaning. One spawn in a hundred is a word per
     playable word, at most 30 %; longer words fall more slowly (up to twice the time).
   ```

2. In `### Desktop App (Fyne)`, change the stats-panel bullet to end with "…, a words line (playable, secure, level) and the word data attribution", and the game-over bullet to "Game-over dialog with missed character and word review and Play Again".
3. In `### Scoring`, change the first bullet to: "**+10 points** per correct kana and per kana unit of a correct word, times the points factor (Auto uses the level's factor: ×1 Relaxed/Normal, ×2 Fast, ×3 Pro)". In `## How to Play` step 4, change "Each correct answer scores 10 points" to "Each correct kana scores 10 points (a word 10 per kana unit)".
4. In `### Character Data (kanacore/)`, change "for the upcoming vocabulary trainer" to "for the vocabulary trainer". In `### Word Lists (vocab/)`, change "Word data for the upcoming vocabulary trainer" to "Word data for the vocabulary trainer".
5. In `### Desktop App (fyne/)`, add after the `tile.go` line (and change it to "`KanaTile` — shadow + face + text canvas objects, for a kana or a word"):

   ```markdown
   - `words.go`: word helpers — meaning language, missed-word and feedback texts, the words line
   - `wordgame.go`: word progress (stats, memory, merge), the playable-word rule and cache, word spawning
   - `feedback.go`: the labels shown after a word hit or miss
   ```

6. In `### Persistence (store/)`, extend the sentence with ", per-word statistics and the word settings (`word_level`, `meaning_lang`)".
7. In `## Data Persistence`, add to the "It holds:" list:

   ```markdown
   - Per-word statistics (`word_stats`: correct and miss count per reading) and per-word
     spaced-repetition memory (`word:` items in the `memory` table)
   - The JLPT level words are played up to (`word_level`, default N5) and the meaning language
     (`meaning_lang`; unset means the system language if the word list has it, else English)
   ```

8. In `## Word data`, after the paragraph ending with `ATTRIBUTION.md`, add: "The game names these sources in the status panel."
9. In `### Private word lists`, replace "The game uses them from the next stage of the vocabulary trainer on." with "Their words fall in the game like bundled words, at any JLPT level; the folder is read when the app starts."

- [ ] **Step 2: PLAN.md** — in `## Vocabulary Trainer`:

   - `- [ ] Show words made only of kana the learner already knows (e.g. は + い → はい), in hiragana and katakana` → `- [x] Show words made only of kana the learner already knows (e.g. は + い → はい), in hiragana and katakana (mixed mode)`
   - In the two following items, replace "; the game uses it from stage 3" and "; the game uses them from stage 3" with "; used by the game".

- [ ] **Step 3: Commit**

```bash
git add README.md PLAN.md
git commit -m "docs: describe words in the game

Documents mixed-mode words, their scoring, feedback and status line, the
new fyne files, and the word stats and settings in the database."
```

---

## Spec coverage (stage 3 testing list)

| Spec test | Where |
|---|---|
| Word share with seeded `rnd`: 0 with no playable word, about 0.3 with ≥ 30 | Task 6 `TestWordShareOfSpawns`, `TestFewPlayableWordsSpawnRarely` |
| Known rule: katakana word stops when katakana is off | Task 3 `TestKatakanaWordStopsWhenKatakanaTurnsOff`, Task 4 `TestDropInactiveTilesRemovesUnplayableWords` |
| Word tile fall time, points, hit and miss (stats, memory, miss limit, tempo, `missedWords`) | Task 6 `TestWordSpawnFallTimeAndPlacement`; Task 5 `TestWordHitScoresPerUnit`, `TestWordHitRecordsWordProgressOnly`, `TestWordMiss`, `TestWordMissesCountTowardsLimit` |
| Merge persists word stats and memory, also when only words changed or the kana read fails | Task 2 `TestMergePersistsWordProgress`, `TestMergesAreIndependent` |
| Kana and word tile both matching: the lower wins | Task 5 `TestLowerTileWinsAcrossKinds` |
| Feedback expires after 20 ticks, freezes while paused, hidden with tiles, cleared by `Reset` | Task 7 |
| Status line text, including "Words unavailable" | Task 8 |
| store: `word_stats` and new settings round-trip and default | Task 1 |
