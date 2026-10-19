# Learning Levels and Progress Reset Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace "3 correct answers = secure" with learning levels 0–5 (hit +1, mistake −2, learned at 5, persistent red flag), count unambiguous wrong inputs as mistakes, and add a per-script progress reset to the settings.

**Architecture:** Levels are absolute state in a new `progress` table (like `memory`), loaded into `GameState` maps with dirty sets that `mergeSessionStats` saves. All "is it learned?" checks go through `learned(level)`. The reset deletes a script's kana rows in one store transaction, rewinds its learning path and restarts the round through a helper extracted from Play Again.

**Tech Stack:** Go 1.25, Fyne v2.7.3, SQLite via modernc.org/sqlite.

**Spec:** `docs/superpowers/specs/2026-10-19-learning-levels-design.md`. Its "Review resolutions" section takes precedence over the sections above it.

---

## Conventions for every task

- Branch: `feat/learning-levels` (already checked out). Never switch branches.
- Run tests with `go test ./...` from the repo root (Fyne needs CGO; the first build is slow). `go vet ./...` must stay clean.
- All code that touches `GameState` fields runs under `gs.mu` ("Must be called under lock" in the doc comment), exactly like the surrounding code.
- Match the surrounding style: short doc comments in plain English on every function, no new dependencies.
- **Do not commit.** Leave the changes in the working tree; the controller reviews and commits. Each task ends with the commit message the controller will use.
- Field names: the spec says `levels`/`wordLevels`; use **`kanaProgress`/`wordProgress`** instead, because `GameState.wordLevel` (JLPT level), `GameState.level` (difficulty) and the package var `levels` (difficulty table) already exist.

## File map

| File | Change |
| --- | --- |
| `store/store.go` | `Progress` type, `ensureProgress` migration, `Progress`, `SaveProgress`, (Part B) `ResetKana` |
| `store/store_test.go` | migration, round trip, reset tests |
| `fyne/learning.go` (new) | level constants, `learned`, `hit`, `mistake`, GameState level helpers, `saveProgress` |
| `fyne/learning_test.go` (new) | level model tests and the test helpers `learnKana`, `setKanaLevel`, `setWordLevel` |
| `fyne/game.go` | fields, load, merge, `recordCorrect`/`recordMiss` via helpers, rules on levels, wrong input, `ResetScript` |
| `fyne/wordgame.go` | word hit/miss via helpers, `unitKnown`, counts, remove `secureCorrect`/`wordCorrect`/known boost |
| `fyne/packages.go` | `newestPackageDone`, `markKnown` on levels, remove `knownDirty` helpers |
| `fyne/progress.go` | remove `totalCorrect` |
| `fyne/statswidgets.go`, `fyne/difficulty.go` | `stateFor`/`charFallFactor` take a level; cell flag |
| `fyne/stats.go`, `fyne/words.go` | snapshot `Levels`/`Flagged`, legend and words labels |
| `fyne/app.go` | `restartRound` |
| `fyne/settings.go` | reset buttons and confirmation |
| tests in `fyne/` | fixture rewrite as listed per task |

---

# Part A — Learning levels

### Task A1: Store — `progress` table, migration, read/write

**Files:**
- Modify: `store/store.go`
- Test: `store/store_test.go`

- [ ] **Step 1: Write the failing tests** in `store/store_test.go` (follow `TestMemorySeededOnceFromKanaStats` and `TestIntroducedSeededOnceFromWordProgress`, which build an old database and reopen it):

```go
func TestProgressRoundTrip(t *testing.T) {
	st := openTemp(t)
	if got, err := st.Progress("kana:"); err != nil || len(got) != 0 {
		t.Fatalf("empty store: %v, %v", got, err)
	}
	_ = st.SaveProgress("kana:か", Progress{Level: 3, Flagged: true})
	_ = st.SaveProgress("kana:か", Progress{Level: 4, Flagged: true})
	_ = st.SaveProgress("word:ねこ", Progress{Level: 5})
	kana, err := st.Progress("kana:")
	if err != nil || len(kana) != 1 || kana["か"] != (Progress{Level: 4, Flagged: true}) {
		t.Fatalf("kana = %v, %v", kana, err)
	}
	if words, _ := st.Progress("word:"); words["ねこ"] != (Progress{Level: 5}) || len(words) != 1 {
		t.Fatalf("words = %v", words)
	}
}

func TestProgressSeededOnceFromStats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// A database from before learning levels: stats, but no progress table.
	_ = st.SaveKanaStats("か", 3, 4, 0)
	_ = st.SaveKanaStats("き", 2, 1, 0)
	_ = st.SaveKanaStats("く", 0, 2, 0)
	_ = st.SaveKanaStats("け", 9, 0, 9)
	_ = st.SaveWordStats("ねこ", 4, 1)
	_ = st.SaveWordStats("いぬ", 1, 3)
	if _, err := st.db.Exec(`DROP TABLE progress`); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	kana, _ := st.Progress("kana:")
	want := map[string]Progress{"か": {Level: 5}, "き": {Level: 2}, "け": {Level: 5}}
	if !reflect.DeepEqual(kana, want) {
		t.Fatalf("kana = %v, want %v (no flags, nothing for 0 correct)", kana, want)
	}
	words, _ := st.Progress("word:")
	if want := map[string]Progress{"ねこ": {Level: 5}, "いぬ": {Level: 1}}; !reflect.DeepEqual(words, want) {
		t.Fatalf("words = %v, want %v", words, want)
	}
	_ = st.SaveKanaStats("こ", 5, 0, 5)
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if kana, _ := st.Progress("kana:"); len(kana) != 3 {
		t.Fatalf("re-open must not seed again: %v", kana)
	}
}
```

- [ ] **Step 2: Run** `go test ./store/ -run 'Progress'` — expected: compile failure (`Progress` undefined).

- [ ] **Step 3: Implement** in `store/store.go`.

Type, next to `WordStats`:

```go
// Progress is an item's learning level (0–5, learned at 5) and whether it
// slipped since it was last learned.
type Progress struct {
	Level   int
	Flagged bool
}
```

In `migrate`, replace `return ensureIntroduced(ctx, db, now)` with:

```go
	if err := ensureIntroduced(ctx, db, now); err != nil {
		return err
	}
	return ensureProgress(ctx, db)
```

The migration, after `ensureIntroduced`:

```go
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
```

Read and write, after `SaveMemoryState`:

```go
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
```

(modernc sqlite scans INTEGER 0/1 into `bool` and binds `bool` as 0/1; if the scan fails in the test, scan into an `int` and compare `!= 0`.)

- [ ] **Step 4: Run** `go test ./store/` — expected: PASS.

- [ ] **Step 5: Commit message:** `feat(store): add learning progress table seeded from stats`

---

### Task A2: Level model in the game — record hits and mistakes, persist them

No rule changes yet: playability, mastery and the panel still use the correct counts after this task. This task only keeps levels and flags up to date and saves them.

**Files:**
- Create: `fyne/learning.go`, `fyne/learning_test.go`
- Modify: `fyne/game.go` (struct, `NewGameState`, `recordCorrect`, `recordMiss`, `mergeSessionStats`), `fyne/wordgame.go` (`recordWordCorrect`, `recordWordMiss`), `fyne/game_test.go` (`newTestState`)

- [ ] **Step 1: Add the fields** to `GameState` after the `memory*` fields:

```go
	kanaProgress           map[string]store.Progress // learning level and flag per character
	kanaProgressDirty      map[string]bool           // characters whose progress is not saved yet
	kanaProgressLoadFailed bool                      // stored progress unreadable: never overwrite it
```

and after the `wordMemory*` fields:

```go
	wordProgress           map[string]store.Progress // learning level and flag per reading
	wordProgressDirty      map[string]bool           // readings whose progress is not saved yet
	wordProgressLoadFailed bool                      // stored word progress unreadable: never overwrite it
```

Initialise the four maps in `NewGameState`'s literal and in `newTestState` (`fyne/game_test.go`) with `map[string]store.Progress{}` / `map[string]bool{}`. In `NewGameState`, next to the memory loads:

```go
		if p, err := st.Progress("kana:"); err == nil {
			gs.kanaProgress = p
		} else {
			gs.kanaProgressLoadFailed = true
		}
		if p, err := st.Progress("word:"); err == nil {
			gs.wordProgress = p
		} else {
			gs.wordProgressLoadFailed = true
		}
```

`Reset` does not reload progress (like memory, it is absolute state kept in memory).

- [ ] **Step 2: Write the failing tests** in `fyne/learning_test.go`:

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/store"
)

// learnKana puts each character at the learned level.
func learnKana(gs *GameState, chars ...string) {
	for _, c := range chars {
		gs.kanaProgress[c] = store.Progress{Level: learnedLevel}
	}
	gs.invalidatePlayable()
}

// setKanaLevel puts one character at level n, not flagged.
func setKanaLevel(gs *GameState, char string, n int) {
	gs.kanaProgress[char] = store.Progress{Level: n}
	gs.invalidatePlayable()
}

// setWordLevel puts one reading at level n, not flagged.
func setWordLevel(gs *GameState, kana string, n int) {
	gs.wordProgress[kana] = store.Progress{Level: n}
}

func TestHitAndMistake(t *testing.T) {
	p := store.Progress{}
	for want := 1; want <= 5; want++ {
		if p = hit(p); p.Level != want {
			t.Fatalf("hit: level %d, want %d", p.Level, want)
		}
	}
	if p = hit(p); p.Level != learnedLevel {
		t.Fatalf("level capped at 5, got %d", p.Level)
	}
	if p = mistake(p); p != (store.Progress{Level: 3, Flagged: true}) {
		t.Fatalf("mistake from 5 = %+v, want level 3 flagged", p)
	}
	if p = mistake(mistake(p)); p != (store.Progress{Level: 0, Flagged: true}) {
		t.Fatalf("level floored at 0, got %+v", p)
	}
	p = store.Progress{Level: 4, Flagged: true}
	if p = hit(p); p != (store.Progress{Level: 5}) {
		t.Fatalf("reaching 5 clears the flag, got %+v", p)
	}
	if !learned(5) || learned(4) {
		t.Fatal("learned means level 5")
	}
}

func TestKanaHitAndMissChangeTheLevel(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	setKanaLevel(gs, "か", 4)
	gs.recordCorrect("か")
	if p := gs.kanaProgress["か"]; p.Level != 5 || !gs.kanaProgressDirty["か"] {
		t.Fatalf("after a hit: %+v dirty %v", p, gs.kanaProgressDirty["か"])
	}
	gs.recordMiss("か")
	if p := gs.kanaProgress["か"]; p != (store.Progress{Level: 3, Flagged: true}) {
		t.Fatalf("after a miss: %+v", p)
	}
}

func TestWordHitAndMissChangeTheLevel(t *testing.T) {
	gs := newTestState()
	gs.recordWordCorrect("あい")
	gs.recordWordCorrect("あい")
	if p := gs.wordProgress["あい"]; p.Level != 2 || !gs.wordProgressDirty["あい"] {
		t.Fatalf("after two hits: %+v", p)
	}
	gs.recordWordMiss("あい")
	if p := gs.wordProgress["あい"]; p != (store.Progress{Level: 0, Flagged: true}) {
		t.Fatalf("after a miss: %+v", p)
	}
	if gs.kanaProgress["あ"].Level != 0 {
		t.Fatal("a word miss must not touch its kana")
	}
}

func TestLevelRisesBeforeAutoProgression(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels")
	learnKana(gs, "あ", "い", "う")
	setKanaLevel(gs, "え", 4)
	gs.recordCorrect("え") // the hit that reaches 5 completes 80 % of the row
	// After Task A3 isRowMastered reads levels; until then this test only
	// checks the level. Task A3 extends it to expect the unlock.
	if gs.kanaProgress["え"].Level != 5 {
		t.Fatal("level must rise in recordCorrect")
	}
}

func TestProgressSavedAndReloaded(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.recordCorrect("か")
	gs.recordMiss("き")
	gs.recordWordCorrect("あい")
	gs.mergeSessionStats()
	dirty := len(gs.kanaProgressDirty) + len(gs.wordProgressDirty)
	gs.mu.Unlock()
	if dirty != 0 {
		t.Fatalf("%d progress entries still dirty after the merge", dirty)
	}
	again := NewGameState(st)
	if again.kanaProgress["か"].Level != 1 || again.kanaProgress["き"] != (store.Progress{Flagged: true}) || again.wordProgress["あい"].Level != 1 {
		t.Fatalf("reloaded kana %v words %v", again.kanaProgress, again.wordProgress)
	}
}

func TestProgressNotWrittenWhenLoadFailed(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveProgress("kana:か", store.Progress{Level: 5})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.kanaProgressLoadFailed = true
	gs.kanaProgress = map[string]store.Progress{}
	gs.recordCorrect("か")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if p, _ := st.Progress("kana:"); p["か"].Level != 5 {
		t.Fatalf("stored progress overwritten: %+v", p["か"])
	}
}
```

(No `kanacore` import in this file unless a test uses it.)

- [ ] **Step 3: Run** `go test ./fyne/ -run 'HitAndMistake|ChangeTheLevel|LevelRises|ProgressSaved|ProgressNotWritten'` — expected: compile failure.

- [ ] **Step 4: Implement `fyne/learning.go`:**

```go
package main

import "kana/store"

// A kana or word is learned at learnedLevel. A hit raises its level by one,
// a mistake lowers it by mistakeDrop and flags it until it is learned again.
const (
	learnedLevel = 5
	mistakeDrop  = 2
)

// learned reports whether level counts as learned.
func learned(level int) bool {
	return level >= learnedLevel
}

// hit returns p after a hit: one level up, at most learnedLevel. Reaching
// learnedLevel clears the flag.
func hit(p store.Progress) store.Progress {
	p.Level = min(p.Level+1, learnedLevel)
	if learned(p.Level) {
		p.Flagged = false
	}
	return p
}

// mistake returns p after a mistake: mistakeDrop levels down, at least 0,
// and flagged.
func mistake(p store.Progress) store.Progress {
	p.Level = max(p.Level-mistakeDrop, 0)
	p.Flagged = true
	return p
}

// kanaLevel is a character's learning level. Must be called under lock.
func (gs *GameState) kanaLevel(char string) int {
	return gs.kanaProgress[char].Level
}

// wordLearned reports whether a reading is learned. Must be called under
// lock.
func (gs *GameState) wordLearned(kana string) bool {
	return learned(gs.wordProgress[kana].Level)
}

// raiseKana records a hit on a character and reports whether it became
// learned with it. Must be called under lock.
func (gs *GameState) raiseKana(char string) bool {
	before := gs.kanaProgress[char]
	gs.kanaProgress[char] = hit(before)
	gs.kanaProgressDirty[char] = true
	return !learned(before.Level) && learned(gs.kanaProgress[char].Level)
}

// lowerKana records a mistake on a character and reports whether it was
// learned before. Must be called under lock.
func (gs *GameState) lowerKana(char string) bool {
	before := gs.kanaProgress[char]
	gs.kanaProgress[char] = mistake(before)
	gs.kanaProgressDirty[char] = true
	return learned(before.Level)
}

// raiseWord records a hit on a reading. Must be called under lock.
func (gs *GameState) raiseWord(kana string) {
	gs.wordProgress[kana] = hit(gs.wordProgress[kana])
	gs.wordProgressDirty[kana] = true
}

// lowerWord records a mistake on a reading. Must be called under lock.
func (gs *GameState) lowerWord(kana string) {
	gs.wordProgress[kana] = mistake(gs.wordProgress[kana])
	gs.wordProgressDirty[kana] = true
}

// saveProgress writes changed kana and word progress. A failed save stays
// dirty for the next merge. If loading a kind failed, nothing of that kind
// is written, so the stored levels are not replaced by fresh ones. Must be
// called under lock.
func (gs *GameState) saveProgress() {
	if gs.store == nil {
		return
	}
	if !gs.kanaProgressLoadFailed {
		for char := range gs.kanaProgressDirty {
			if err := gs.store.SaveProgress("kana:"+char, gs.kanaProgress[char]); err == nil {
				delete(gs.kanaProgressDirty, char)
			}
		}
	}
	if !gs.wordProgressLoadFailed {
		for kana := range gs.wordProgressDirty {
			if err := gs.store.SaveProgress("word:"+kana, gs.wordProgress[kana]); err == nil {
				delete(gs.wordProgressDirty, kana)
			}
		}
	}
}
```

- [ ] **Step 5: Wire it in.**

`fyne/game.go` — `recordCorrect`: after the memory update and before `gs.raiseTempo()` add (keep the existing `secureCorrect` check for now; Task A3 removes it):

```go
	if gs.raiseKana(char) {
		gs.invalidatePlayable() // the kana just became learned
	}
```

Replace `recordMiss` with the shared helper plus the tempo:

```go
// applyKanaMistake records a mistake on a character: session miss, streak
// reset, memory miss (if known), level down and flag. A character that was
// learned may make words unplayable. Must be called under lock.
func (gs *GameState) applyKanaMistake(char string) {
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
	if gs.lowerKana(char) {
		gs.invalidatePlayable()
	}
}

// recordMiss records a character that fell through: a mistake plus the
// tempo. Must be called under lock.
func (gs *GameState) recordMiss(char string) {
	gs.applyKanaMistake(char)
	gs.lowerTempo()
}
```

`mergeSessionStats`: call `gs.saveProgress()` right after `gs.saveMemory()`; update its doc comment ("Memory and progress are saved first …").

`fyne/wordgame.go` — `recordWordCorrect`: add `gs.raiseWord(kana)` after the memory update. Replace `recordWordMiss` with:

```go
// applyWordMistake records a mistake on a word: session miss, word memory
// miss (if known), level down and flag. Its kana are not touched. Must be
// called under lock.
func (gs *GameState) applyWordMistake(kana string) {
	stat := gs.wordSessionStats[kana]
	stat.Kana = kana
	stat.MissCount++
	gs.wordSessionStats[kana] = stat
	gs.wordSessionDirty = true

	if prev, known := gs.wordMemory[kana]; known {
		gs.wordMemory[kana] = memory.Miss(prev, gs.now())
		gs.wordMemoryDirty[kana] = true
	}
	gs.lowerWord(kana)
}

// recordWordMiss records a word that fell through: a mistake plus the
// tempo. Must be called under lock.
func (gs *GameState) recordWordMiss(kana string) {
	gs.applyWordMistake(kana)
	gs.lowerTempo()
}
```

- [ ] **Step 6: Run** `go test ./...` — expected: PASS (all old tests untouched and green).

- [ ] **Step 7: Commit message:** `feat(fyne): track learning levels for kana and words`

---

### Task A3: Kana rules on levels, with the kana fixtures

**Files:**
- Modify: `fyne/game.go` (`recordCorrect`, `isRowMastered`, `spawnKana`), `fyne/wordgame.go` (`unitKnown`), `fyne/statswidgets.go` (`stateFor`), `fyne/difficulty.go` (`charFallFactor`)
- Modify tests: `fyne/game_test.go`, `fyne/game_lifecycle_test.go`, `fyne/wordgame_test.go`, `fyne/packages_test.go`, `fyne/difficulty_test.go`, `fyne/learning_test.go`

- [ ] **Step 1: Rewrite the kana fixtures first** (they define "learned" by correct counts and will fail once the rules switch). Rule: a fixture that sets `CorrectCount: 3` (or more) on `overallStats`/`sessionStats` *to mean learned* now calls `learnKana`; one that sets 1–2 *to mean learning* uses `setKanaLevel(gs, char, n)` with n = the old total (it stays below 5) — **except** when the test then hits once to make the kana learned: use `learnedLevel-1` (4) there; `CorrectCount: 0` fixtures that mean "not learned" become `setKanaLevel(gs, char, 0)` or are deleted. Fixtures that test statistics themselves (counts after a hit, merge arithmetic) stay as they are. Sites:
  - `fyne/game_test.go`: `masterRows` (body → `learnKana(gs, row.Characters()...)`), `TestIsRowMastered` (line ~208), the loop at ~219.
  - `fyne/game_lifecycle_test.go`: `TestCheckAutoProgressionNoUnlockWhenNotAllMastered` (~172); the store-based setups at ~227 (`st.SaveKanaStats(char, 3, 0, 3)` → add `_ = st.SaveProgress("kana:"+char, store.Progress{Level: 5})`) and ~254 (`SaveKanaStats("あ", 1, 0, 1)` → add level 1).
  - `fyne/wordgame_test.go`: helper `secure` (body → `learnKana(gs, chars...)`; keep the name `secure` or rename it to `learnKana` callers — prefer deleting `secure` and replacing its calls with `learnKana`), `TestUnitKnown` (き level 2 → not known; く "2 stored + 1 session" → `setKanaLevel(gs, "く", 5)`), ~267 (え level 2), ~470 (`TestPlayableRefreshesWhenKanaBecomesSecure`: い level 4, one hit learns it).
  - `fyne/packages_test.go`: ~116 (え level 0), ~171 and ~336 (store-based: add `SaveProgress(..., Level: 5)`), ~363–366 (`TestRecheckWhenAKanaBecomesSecure`: あ level 4, one hit learns it; い/う/え level 0).
  - `fyne/difficulty_test.go` `TestCharFallFactor`: `map[int]float32{0: 1.2, 1: 1.0, 4: 1.0, 5: 0.7}`.
  - `fyne/stats_test.go` `TestPanelCellStatesAndMissed` (~49): `TotalCorrect["く"] = 5` (Task A4 rewrites this test; until then `stateFor` reads the value as a level).
  - Anything else `go test` flags: apply the same rule.

- [ ] **Step 2: Extend `TestLevelRisesBeforeAutoProgression`** in `fyne/learning_test.go`: after `gs.recordCorrect("え")` expect `gs.selectedRows["k"]` to be true (the k row unlocked by that very hit) and `gs.pendingIntro` to name it. Add:

```go
func TestRowsStayUnlockedAfterALearnedKanaDrops(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels", "k")
	learnKana(gs, "あ", "い", "う", "え", "お")
	gs.recordMiss("あ")
	if !gs.selectedRows["k"] {
		t.Fatal("unlocked rows stay when a kana drops below 5")
	}
}

func TestWordPausesWhileItsKanaIsBelowLearned(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	learnKana(gs, "あ", "い")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable = %v", got)
	}
	gs.recordMiss("あ")
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("あ dropped to 3: playable = %v, want none", got)
	}
	gs.recordCorrect("あ")
	gs.recordCorrect("あ")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("あ learned again: playable = %v", got)
	}
}

func TestFallingWordKeepsFallingWhenItsKanaDrops(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	learnKana(gs, "あ", "い")
	word := wordTileAt(gs, "あい", 100)
	gs.tiles = []*KanaTile{word}
	gs.recordMiss("あ")
	gs.tick()
	if len(gs.tiles) != 1 || gs.tiles[0] != word {
		t.Fatal("a word already falling keeps falling; only new spawns stop")
	}
}
```

- [ ] **Step 3: Run** `go test ./fyne/` — expected: the new tests and the rewritten fixtures FAIL (rules still on counts).

- [ ] **Step 4: Switch the rules.**
  - `recordCorrect`: delete the `secureCorrect` block; keep `if gs.raiseKana(char) { gs.invalidatePlayable() }` and make sure it runs before `checkAutoProgression`.
  - `isRowMastered`: count `learned(gs.kanaLevel(char))`; doc: "at least 80 % of the row's characters are learned (level 5)".
  - `unitKnown`: `return learned(gs.kanaLevel(unit))`; doc "the kana is learned".
  - `spawnKana`: `tile.fallSeconds = base * charFallFactor(gs.kanaLevel(char)) * tempoFactor(gs.tempoStep)`; delete the `correct :=` line.
  - `stateFor(level int)`: `case learned(level): cellMastered; case level > 0: cellLearning; default: cellNew`. Update the `cellState` comments: new = level 0, learning = 1–4, mastered = level 5 (learned).
  - `charFallFactor(level int)`: same switch through `stateFor`; doc "new characters get more time, learned ones less".
  - `mergeKanaStats`'s closing comment about "(un)secure": say "learned" no longer depends on stats; keep `invalidatePlayable` only if still needed — it is not (playability reads levels); remove that call and its comment.
  - `stats.go` still calls `stateFor(snap.TotalCorrect[char])` — leave it for Task A4 (it compiles; the panel shows counts mapped as levels for one task).

- [ ] **Step 5: Run** `go test ./...` — expected: PASS. Then `grep -n "CorrectCount: 3\|>= 3\|secureCorrect" fyne/*.go` and check every remaining hit is word-related (Task A5) or a pure statistics test.

- [ ] **Step 6: Commit message:** `feat(fyne): decide kana mastery and playable words by learning level`

---

### Task A4: Status panel from levels and flags

**Files:**
- Modify: `fyne/stats.go` (`StatsSnapshot`, `Update`, legend), `fyne/statswidgets.go` (`kanaCell`), `fyne/game.go` (`snapshot`), `fyne/progress.go` (remove `totalCorrect`)
- Modify tests: `fyne/stats_test.go`, `fyne/progress_test.go`, `fyne/learning_test.go`

- [ ] **Step 1: Write the failing tests.**
  - In `fyne/stats_test.go`: `panelSnap` builds `Levels: map[string]int{}` and `Flagged: map[string]bool{}` instead of `TotalCorrect`. Rewrite `TestPanelCellStatesAndMissed` as `TestPanelCellStatesAndFlags`: `snap.Levels["き"] = 2`, `snap.Levels["く"] = 5`, `snap.Levels["け"] = 4`, `snap.Flagged["ぬ"] = true`; expect か new, き and け learning, く mastered; ぬ flagged, な not; and a cell in `snap.MissedKanas` but not flagged (`snap.MissedKanas = []kanacore.Kana{{Char: "な", Romaji: "na"}}`) is **not** framed.
  - A legend test: the panel's legend texts are `new`, `learning`, `learned`, `slipped` (walk `test.LaidOutObjects(p)` for `*canvas.Text` or reuse an existing helper in `stats_test.go`).
  - In `fyne/progress_test.go`: replace `TestSnapshotTotalCorrectAddsStoredAndSession` with `TestSnapshotLevelsAndFlags`: `gs.kanaProgress["か"] = store.Progress{Level: 3, Flagged: true}`; expect `snap.Levels["か"] == 3 && snap.Flagged["か"]`, and the snapshot maps are copies (changing `gs.kanaProgress` afterwards does not change them).
  - In `fyne/learning_test.go`, the flag survives a restart:

```go
func TestFlagShownAfterRestart(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.recordMiss("か")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	again := NewGameState(st)
	again.mu.Lock()
	snap := again.snapshot()
	again.mu.Unlock()
	if !snap.Flagged["か"] {
		t.Fatal("the red frame must survive a restart")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/` — expected: compile failure (`Levels`, `Flagged` unknown).

- [ ] **Step 3: Implement.**
  - `StatsSnapshot`: replace `TotalCorrect map[string]int` with

```go
	Levels  map[string]int  // learning level per character
	Flagged map[string]bool // characters that slipped since they were last learned
```

  - `snapshot()`: build both maps from `gs.kanaProgress` (only entries with `Level > 0` or `Flagged`), drop `TotalCorrect`.
  - `progress.go`: delete `totalCorrect` (no other users).
  - `kanaCell`: rename the field `missed` to `flagged` and the `set` parameter likewise; doc: "applies a learning state and the red frame of a character that slipped".
  - `StatsPanel.Update`: delete the `missed` map built from `snap.MissedKanas`; `cell.set(stateFor(snap.Levels[char]), snap.Flagged[char])`.
  - Legend items: `{"new", cellNew, false}, {"learning", cellLearning, false}, {"learned", cellMastered, false}, {"slipped", cellNew, true}`.
  - The game-over dialog keeps using `snap.MissedKanas` (unchanged).

- [ ] **Step 4: Run** `go test ./...` — expected: PASS.

- [ ] **Step 5: Commit message:** `feat(fyne): colour the kana grid by level and keep the slipped frame`

---

### Task A5: Word rules on levels, "already known", labels

**Files:**
- Modify: `fyne/wordgame.go`, `fyne/packages.go`, `fyne/game.go` (struct, `NewGameState`, `Reset`, `mergeSessionStats`), `fyne/words.go`
- Modify tests: `fyne/packages_test.go`, `fyne/wordgame_test.go`, `fyne/settings_test.go`, `fyne/stats_test.go`, `fyne/words_test.go`, `fyne/game_test.go` (`newTestState`)

- [ ] **Step 1: Rewrite the word fixtures.** Same rule as Task A3, for `wordOverallStats`/`wordSessionStats` with `CorrectCount: 3` meaning learned → `setWordLevel(gs, kana, learnedLevel)`. Sites: `packages_test.go` ~80, ~86 (session 3 → level 5), ~122, ~430; `settings_test.go` ~183; `wordgame_test.go` ~429, ~927–928 (2 stored + 1 session → level 5 if the test means learned, else keep the stats test as is — read it). Expected strings: `"Introduced: 1 · learned: 1 · left in source: 1"` (`settings_test.go` ~216), `"Words: … learned · …"` (`stats_test.go` ~262–300, `words_test.go` ~83–84, `wordgame_test.go` ~432), and `WordsStatus{…, Learned: n}` instead of `Secure: n`.

- [ ] **Step 2: Rewrite the "already known" tests.**
  - `TestFinishPackage` (`packages_test.go` ~185): the stored word stats stay honest — あう has **no** stats row (or 0/0), あお keeps 5/2; instead expect `gs.wordProgress["あう"] == store.Progress{Level: 5}` and `gs.wordProgress["あお"].Level == 5`, both saved at once (`st.Progress("word:")`). Memory still `memory.Seed(3, now)`. Replace the `wordCorrect` assertion. Give あお a flag before (`gs.wordProgress["あお"] = store.Progress{Level: 2, Flagged: true}`) and expect it cleared.
  - The failed-save test (~540–587): hide the `progress` table instead of `word_stats` (`ALTER TABLE progress RENAME TO hidden_progress`), expect `gs.wordProgressDirty["あう"]` to stay set across a failing merge and be saved after renaming back and `gs.Reset()` (Reset merges first); `gs.wordProgress["あう"].Level` stays 5 throughout. Keep the newest-package part unchanged.

- [ ] **Step 3: Run** `go test ./fyne/` — expected: FAIL.

- [ ] **Step 4: Implement.**
  - `newestPackageDone`: count `gs.wordLearned(kana)`; rename `secured` to `learnedN`; doc "learned".
  - `introducedCounts() (introduced, learnedN int)`: count `gs.wordLearned(kana)`.
  - `wordsSummary`: `"Introduced: %d · learned: %d · left in source: %d"`; doc updated.
  - `WordsStatus.Secure` → `Learned int // introduced readings at level 5`; `formatWordsLine`: `"Words: %d introduced · %d learned · %s"`; `wordsStatus` sets `ws.Learned`. Update the `stats.go` comment on `wordsText`.
  - `markKnown`:

```go
// knownSeed is the memory a word ticked as "already known" starts with: the
// state of three correct answers.
const knownSeed = 3

// markKnown records a word the learner already knows: it becomes learned
// (level 5, not flagged), saved at once, and gets the memory state of
// knownSeed correct answers. Its stats keep their real counts. No points,
// no tempo change. A failed save stays dirty and mergeSessionStats retries
// it. Must be called under lock.
func (gs *GameState) markKnown(kana string, now time.Time) {
	gs.wordProgress[kana] = store.Progress{Level: learnedLevel}
	gs.wordProgressDirty[kana] = true
	gs.saveProgress()
	gs.wordMemory[kana] = memory.Seed(knownSeed, now)
	gs.wordMemoryDirty[kana] = true
}
```

  - Delete `boostKnownDirty`, `saveKnownDirty`, the `knownDirty` field and its initialisers (`NewGameState`, `newTestState`), the `gs.boostKnownDirty()` call in `Reset`, the `gs.saveKnownDirty()` call in `mergeSessionStats`, and the `knownDirty` lines in `mergeWordStats`.
  - Delete `const secureCorrect` and `wordCorrect` (no users left; check with `grep -n "wordCorrect(\|secureCorrect" fyne/`).
  - `FinishPackage` doc: "the ones in known count as learned".
  - `invalidatePlayable` doc: "whenever a kana may have become learned or stopped being learned, …".

- [ ] **Step 5: Run** `go test ./...` and `go vet ./...` — expected: PASS, clean. `grep -rn "secure" fyne/*.go` should leave no rule or label that means the old "3 correct" (prose like "secure vowels" in test comments becomes "learned").

- [ ] **Step 6: Commit message:** `feat(fyne): decide learned words by level and mark known words learned`

---

### Task A6: Wrong input with one tile on the field

**Files:**
- Modify: `fyne/game.go` (struct, `checkAnswer`, `tick`, `Reset`)
- Test: `fyne/learning_test.go`

- [ ] **Step 1: Write the failing tests:**

```go
func TestWrongInputWithOneKanaTile(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	setKanaLevel(gs, "か", 4)
	tile := tileAt("か", "ka", 100)
	gs.tiles = []*KanaTile{tile}
	gs.tempoStep, gs.answerStreak = 3, 2
	gs.checkAnswer("ki")
	if len(gs.tiles) != 1 || gs.tiles[0] != tile {
		t.Fatal("the tile stays on the field")
	}
	if p := gs.kanaProgress["か"]; p != (store.Progress{Level: 2, Flagged: true}) {
		t.Fatalf("progress = %+v, want level 2 flagged", p)
	}
	if gs.sessionStats["か"].MissCount != 1 || gs.currentStreak["か"] != 0 {
		t.Fatal("a wrong input is a session miss and resets the character's streak")
	}
	if gs.missed != 0 || gs.tempoStep != 3 || gs.answerStreak != 2 || len(gs.missedKanas) != 0 {
		t.Fatal("a wrong input does not count towards the miss limit or the tempo")
	}
}

func TestWrongInputWithOneWordTile(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	learnKana(gs, "あ", "い")
	setWordLevel(gs, "あい", 3)
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 100)}
	gs.checkAnswer("ao")
	if p := gs.wordProgress["あい"]; p != (store.Progress{Level: 1, Flagged: true}) {
		t.Fatalf("word progress = %+v", p)
	}
	if gs.kanaProgress["あ"].Level != 5 || gs.kanaProgress["い"].Level != 5 {
		t.Fatal("a word mistake does not touch its kana")
	}
}

func TestWrongInputIgnored(t *testing.T) {
	test.NewApp()
	cases := map[string]func(gs *GameState) string{
		"two tiles": func(gs *GameState) string {
			gs.tiles = []*KanaTile{tileAt("か", "ka", 100), tileAt("き", "ki", 50)}
			return "ku"
		},
		"blank input": func(gs *GameState) string {
			gs.tiles = []*KanaTile{tileAt("か", "ka", 100)}
			return "  "
		},
		"paused": func(gs *GameState) string {
			gs.tiles = []*KanaTile{tileAt("か", "ka", 100)}
			gs.paused = true
			return "ki"
		},
		"over": func(gs *GameState) string {
			gs.tiles = []*KanaTile{tileAt("か", "ka", 100)}
			gs.over = true
			return "ki"
		},
	}
	for name, setup := range cases {
		gs := newTestState()
		gs.checkAnswer(setup(gs))
		if len(gs.kanaProgress) != 0 || len(gs.sessionStats) != 0 {
			t.Errorf("%s: progress %v, stats %v; want no mistake", name, gs.kanaProgress, gs.sessionStats)
		}
	}
}

func TestLateEnterForAFallenTileIsIgnored(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	gs.now = func() time.Time { return now }
	fallen := tileAt("か", "ka", gs.canvasH-1)
	fallen.fallSeconds = 0.1 // falls through on the next tick
	other := tileAt("き", "ki", 10)
	gs.tiles = []*KanaTile{other, fallen}
	gs.tick()
	if len(gs.tiles) != 1 {
		t.Fatalf("want か fallen through, tiles = %d", len(gs.tiles))
	}
	now = now.Add(time.Second)
	gs.checkAnswer("ka") // meant for か, too late
	if p := gs.kanaProgress["き"]; p.Level != 0 || p.Flagged {
		t.Fatalf("late Enter punished き: %+v", p)
	}
	now = now.Add(time.Second) // 2 s after the fall
	gs.checkAnswer("ka")
	if !gs.kanaProgress["き"].Flagged {
		t.Fatal("after 1.5 s a wrong input counts again")
	}
}
```

(`tileAt` and `wordTileAt` exist in `game_test.go`/`wordgame_test.go`; check their signatures and adapt. Add `"time"` to the imports. If `tick` needs `gs.canvas`/`onStatsChanged`, they may stay nil, as in the existing tick tests.)

- [ ] **Step 2: Run** `go test ./fyne/ -run 'WrongInput|LateEnter'` — expected: FAIL.

- [ ] **Step 3: Implement.**

Fields, next to `missedWords`:

```go
	lastFallen   *KanaTile // the tile that fell through last, for a late Enter
	lastFallenAt time.Time
```

Constant near `ticksPerSecond`:

```go
// lateAnswerWindow is how long after a tile fell through a wrong input that
// answers it is taken as a late Enter for it, not as a mistake.
const lateAnswerWindow = 1500 * time.Millisecond
```

In `tick`, inside `if tile.pos.Y > gs.canvasH {`: `gs.lastFallen, gs.lastFallenAt = tile, gs.now()`.

In `Reset`, next to `gs.tiles = nil`: `gs.lastFallen = nil`.

In `checkAnswer`, after the `if matched { … }` block (still under the lock):

```go
	if !matched {
		gs.wrongInput(input)
	}
```

and the helper:

```go
// wrongInput counts an input that answers no tile as a mistake on the only
// tile on the field. With several tiles it is unclear which one was meant,
// and an input answering a tile that just fell through is a late Enter;
// both are ignored, as is blank input. The tile stays, the miss limit and
// the tempo are untouched, and no feedback is shown. Must be called under
// lock.
func (gs *GameState) wrongInput(input string) {
	if strings.TrimSpace(input) == "" || len(gs.tiles) != 1 {
		return
	}
	if gs.lastFallen != nil && gs.now().Sub(gs.lastFallenAt) < lateAnswerWindow && gs.answers(gs.lastFallen, input) {
		return
	}
	tile := gs.tiles[0]
	if tile.word != nil {
		gs.applyWordMistake(tile.word.Kana)
	} else {
		gs.applyKanaMistake(tile.kana.Char)
	}
}
```

Add `"strings"` to the imports. Update `checkAnswer`'s doc comment: "… An input that answers no tile may count as a mistake (see wrongInput)." The input bar already refreshes the panel after every submit, so the frame shows at once.

- [ ] **Step 4: Run** `go test ./...` — expected: PASS.

- [ ] **Step 5: Commit message:** `feat(fyne): count a wrong answer on the only tile as a mistake`

---

# Part B — Reset per script

### Task B1: Store — `ResetKana`

**Files:**
- Modify: `store/store.go`
- Test: `store/store_test.go`

- [ ] **Step 1: Write the failing test:**

```go
func TestResetKana(t *testing.T) {
	st := openTemp(t)
	now := time.Now()
	for _, c := range []string{"か", "カ"} {
		_ = st.SaveKanaStats(c, 3, 1, 2)
		_ = st.SaveMemoryState("kana:"+c, memory.State{Stability: 1, LastSeen: now})
		_ = st.SaveProgress("kana:"+c, Progress{Level: 5})
	}
	_ = st.SaveWordStats("か", 1, 0)
	_ = st.SaveMemoryState("word:か", memory.State{Stability: 1, LastSeen: now})
	_ = st.SaveProgress("word:か", Progress{Level: 1})

	if err := st.ResetKana([]string{"か", "き"}); err != nil {
		t.Fatal(err)
	}
	stats, _ := st.KanaStatistics()
	kmem, _ := st.MemoryStates("kana:")
	kprog, _ := st.Progress("kana:")
	if _, ok := stats["か"]; ok || len(stats) != 1 || len(kmem) != 1 || len(kprog) != 1 {
		t.Fatalf("か must be gone, カ kept: stats %v memory %v progress %v", stats, kmem, kprog)
	}
	words, _ := st.WordStatistics()
	wmem, _ := st.MemoryStates("word:")
	wprog, _ := st.Progress("word:")
	if len(words) != 1 || len(wmem) != 1 || len(wprog) != 1 {
		t.Fatal("the word か is not the character か: word data stays")
	}
}
```

- [ ] **Step 2: Run** `go test ./store/ -run ResetKana` — expected: compile failure.

- [ ] **Step 3: Implement:**

```go
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
```

- [ ] **Step 4: Run** `go test ./store/` — expected: PASS.

- [ ] **Step 5: Commit message:** `feat(store): delete a set of characters' progress`

---

### Task B2: `GameState.ResetScript`

**Files:**
- Modify: `fyne/game.go` (or a new `fyne/reset.go` — preferred, it keeps `game.go` from growing)
- Test: `fyne/reset_test.go` (new)

- [ ] **Step 1: Write the failing tests** in `fyne/reset_test.go`. Use a real store (`openTestStore`) so the store side is checked too:

```go
package main

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/memory"
	"kana/store"
)

// resetGame returns a game on a store with both scripts active, a few rows
// of each selected and every kana of them learned, flagged on か and カ.
func resetGame(t *testing.T) (*GameState, *store.Store) {
	t.Helper()
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveActiveScripts([]string{"hiragana", "katakana"})
	_ = st.SaveSelectedRows([]string{"vowels", "k", "kata:vowels", "kata:k"})
	_ = st.SaveKatakanaOffered(true)
	for _, id := range []string{"vowels", "k", "kata:vowels", "kata:k"} {
		row, _ := kanacore.RowByID(id)
		for _, c := range row.Characters() {
			_ = st.SaveKanaStats(c, 6, 1, 0)
			_ = st.SaveMemoryState("kana:"+c, memory.State{Stability: 2})
			_ = st.SaveProgress("kana:"+c, store.Progress{Level: 5})
		}
	}
	_ = st.SaveWordStats("あか", 2, 0)
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.kanaProgress["か"] = store.Progress{Level: 3, Flagged: true}
	gs.recordCorrect("き") // unsaved session data
	// The bundled JLPT list makes a word package due at start; clear it so
	// the reset is not refused as busy.
	gs.pendingPackage = nil
	gs.pendingLevelOffer = false
	gs.paused = false
	gs.mu.Unlock()
	if p := gs.PendingDialogs(); len(p.Intro) > 0 || p.KatakanaOffer || len(p.Package) > 0 || p.LevelOffer != 0 {
		t.Fatalf("fixture must start without a pending dialog: %+v", p)
	}
	return gs, st
}

func TestResetScriptActive(t *testing.T) {
	gs, st := resetGame(t)
	if err := gs.ResetScript(kanacore.ScriptHiragana); err != nil {
		t.Fatal(err)
	}
	gs.mu.Lock()
	defer gs.mu.Unlock()
	for _, c := range []string{"あ", "か", "き"} {
		if _, ok := gs.overallStats[c]; ok {
			t.Errorf("%s still has stats", c)
		}
		if _, ok := gs.sessionStats[c]; ok {
			t.Errorf("%s still has session stats", c)
		}
		if _, ok := gs.memory[c]; ok || gs.kanaProgress[c] != (store.Progress{}) || gs.kanaProgressDirty[c] || gs.memoryDirty[c] {
			t.Errorf("%s keeps memory or progress", c)
		}
	}
	if gs.kanaProgress["カ"].Level != 5 || gs.overallStats["カ"].CorrectCount != 6 {
		t.Fatal("katakana must be untouched")
	}
	stored, _ := st.KanaStatistics()
	if _, ok := stored["き"]; ok || stored["カ"].CorrectCount != 6 {
		t.Fatalf("store after reset: %v", stored)
	}
	if !equalIDs(selectedIDs(gs), []string{"vowels", "kata:vowels", "kata:k"}) || !gs.autoProgress {
		t.Fatalf("rows = %v, auto %v; want hiragana back at its first step", selectedIDs(gs), gs.autoProgress)
	}
	if !equalIDs(gs.pendingIntro, []string{"vowels"}) || !gs.activeScripts[kanacore.ScriptHiragana] {
		t.Fatalf("pending intro %v; want the first step's intro, hiragana still active", gs.pendingIntro)
	}
	if words, _ := st.WordStatistics(); words["あか"].CorrectCount != 2 || !gs.katakanaOffered {
		t.Fatal("word data and the katakana offer stay")
	}
}

func TestResetScriptPausesItsWords(t *testing.T) {
	gs, _ := resetGame(t)
	gs.mu.Lock()
	withWords(gs, bw("あい", 5, "love"))
	before := len(gs.playableWords())
	gs.mu.Unlock()
	if err := gs.ResetScript(kanacore.ScriptHiragana); err != nil {
		t.Fatal(err)
	}
	gs.mu.Lock()
	after := len(gs.playableWords())
	gs.mu.Unlock()
	if before != 1 || after != 0 {
		t.Fatalf("playable before %d, after %d; want あい paused by the reset", before, after)
	}
}

func TestResetScriptInactive(t *testing.T) {
	gs, _ := resetGame(t)
	gs.mu.Lock()
	gs.activeScripts = map[kanacore.Script]bool{kanacore.ScriptHiragana: true}
	gs.mu.Unlock()
	if err := gs.ResetScript(kanacore.ScriptKatakana); err != nil {
		t.Fatal(err)
	}
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.hasSelectedRowOf(kanacore.ScriptKatakana) || len(gs.pendingIntro) != 0 || gs.kanaProgress["カ"].Level != 0 {
		t.Fatalf("inactive katakana: rows %v, intro %v", selectedIDs(gs), gs.pendingIntro)
	}
	gs.enableKatakanaLocked()
	if !equalIDs(gs.pendingIntro, []string{"kata:vowels"}) {
		t.Fatalf("turned on later: intro %v, want its first step", gs.pendingIntro)
	}
}

func TestResetScriptRefusedWhileBusy(t *testing.T) {
	for name, busy := range map[string]func(gs *GameState){
		"intro":    func(gs *GameState) { gs.pendingIntro = []string{"s"} },
		"offer":    func(gs *GameState) { gs.pendingOffer = true },
		"package":  func(gs *GameState) { gs.pendingPackage = []string{"あい"} },
		"level":    func(gs *GameState) { gs.pendingLevelOffer = true },
		"gameover": func(gs *GameState) { gs.over = true },
	} {
		gs, st := resetGame(t)
		gs.mu.Lock()
		busy(gs)
		gs.mu.Unlock()
		if err := gs.ResetScript(kanacore.ScriptHiragana); !errors.Is(err, errResetBusy) {
			t.Errorf("%s: err = %v, want errResetBusy", name, err)
		}
		if stored, _ := st.KanaStatistics(); stored["あ"].CorrectCount != 6 {
			t.Errorf("%s: store changed", name)
		}
	}
}

func TestResetScriptStoreFailureChangesNothing(t *testing.T) {
	gs, st := resetGame(t)
	_ = st.Close() // every store call fails now
	if err := gs.ResetScript(kanacore.ScriptHiragana); err == nil || errors.Is(err, errResetBusy) {
		t.Fatalf("err = %v, want the store error", err)
	}
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.kanaProgress["あ"].Level != 5 || !gs.selectedRows["k"] || len(gs.pendingIntro) != 0 {
		t.Fatal("a failed reset must leave the game as it was")
	}
}
```

`selectedIDs(gs)` returns the selected row IDs in `kanacore.AllKanaRows` order — add it to `reset_test.go` if no such helper exists. If `store.Close` makes `mergeSessionStats` panic instead of erroring, close a second handle instead or rename the `kana_stats` table via `sql.Open` as `packages_test.go` does; the point is that `ResetKana` fails.

- [ ] **Step 2: Run** `go test ./fyne/ -run ResetScript` — expected: compile failure.

- [ ] **Step 3: Implement** in `fyne/reset.go`:

```go
package main

import (
	"errors"

	"kana/kanacore"
)

// errResetBusy refuses a reset while a dialog is pending or the game is
// over: Reset drops pending offers, and the katakana offer, already saved
// as made, would be lost.
var errResetBusy = errors.New("Finish the open dialog first.") //nolint:staticcheck // shown to the learner as is

// scriptChars returns every character of the script's rows.
func scriptChars(s kanacore.Script) []string {
	var chars []string
	for _, row := range kanacore.RowsFor(s) {
		chars = append(chars, row.Characters()...)
	}
	return chars
}

// ResetScript deletes the stats, memory and learning progress of every
// character of the script and starts its learning path again: all its rows
// are deselected and, if the script is active, its first step is selected
// with its intro queued and auto-progression on. The other script, the
// katakana offer and all word data stay. Nothing changes on error. The
// caller restarts the round.
func (gs *GameState) ResetScript(s kanacore.Script) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.over || len(gs.pendingIntro) > 0 || gs.pendingOffer || len(gs.pendingPackage) > 0 || gs.pendingLevelOffer {
		return errResetBusy
	}
	// Merge first, so a later merge of older session data cannot bring
	// the deleted counts back.
	gs.mergeSessionStats()
	chars := scriptChars(s)
	if gs.store != nil {
		if err := gs.store.ResetKana(chars); err != nil {
			return err
		}
	}
	for _, c := range chars {
		delete(gs.overallStats, c)
		delete(gs.sessionStats, c)
		delete(gs.currentStreak, c)
		delete(gs.memory, c)
		delete(gs.memoryDirty, c)
		delete(gs.kanaProgress, c)
		delete(gs.kanaProgressDirty, c)
	}
	gs.invalidatePlayable()

	var rows []string
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] && row.Script != s {
			rows = append(rows, row.ID)
		}
	}
	first := kanacore.ProgressionStepsFor(s)[0]
	active := gs.activeScripts[s]
	if active {
		rows = append(rows, first...)
	}
	gs.setSelectedRowsLocked(rows)
	gs.autoProgress = true
	if gs.store != nil {
		_ = gs.store.SaveAutoProgress(true)
	}
	if active {
		gs.pendingIntro = append(gs.pendingIntro, first...)
	}
	return nil
}
```

(The error text is shown to the learner as is, hence the capital and the full stop.)

- [ ] **Step 4: Run** `go test ./...` — expected: PASS.

- [ ] **Step 5: Commit message:** `feat(fyne): reset one script's progress and learning path`

---

### Task B3: Restart helper and the reset buttons in the settings

**Files:**
- Modify: `fyne/app.go` (`restartRound`), `fyne/settings.go` (`newSettingsForm`, `newKanaTab`, `showSettingsDialog`)
- Test: `fyne/settings_test.go`

- [ ] **Step 1: Extract `restartRound`** in `fyne/app.go` from the Play Again branch of `showGameOverDialog` (lines `gs.Reset()` … `showPendingDialogs(...)`), unchanged in behaviour:

```go
// restartRound starts a new round: Reset, new game goroutines and event
// watcher, refreshed panel and input bar, then whatever dialog is pending.
// Play Again and the progress reset use it.
func restartRound(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, w fyne.Window) {
	gs.Reset()           // closes old stopCh and eventCh; old watcher drains and exits on stale events
	gs.Start(gameCanvas) // launches new tick/spawn goroutines
	gs.mu.Lock()
	ch := gs.eventCh
	gs.mu.Unlock()
	go watchEvents(ch, gs, statsPanel, gameCanvas, inputBar, w) // new watcher tied to the new eventCh

	gs.mu.Lock()
	snap := gs.snapshot()
	gs.mu.Unlock()
	statsPanel.Update(snap)
	inputBar.Update(snap)
	inputBar.SetPaused(false) // Reset cleared the user pause
	gameCanvas.Refresh()
	showPendingDialogs(gs, statsPanel, inputBar, w)
}
```

The Play Again branch becomes `restartRound(gs, statsPanel, gameCanvas, inputBar, w)`. Run `go test ./fyne/` — PASS (pure refactor).

- [ ] **Step 2: Write the failing tests** in `fyne/settings_test.go`:

```go
func TestKanaTabHasResetButtons(t *testing.T) {
	test.NewApp()
	var asked []kanacore.Script
	f := newSettingsForm(settingsGame().settingsState(), func(s kanacore.Script) { asked = append(asked, s) })
	for _, label := range []string{"Reset hiragana progress…", "Reset katakana progress…"} {
		b := buttonIn(f.kanaTab.Content, label)
		if b == nil {
			t.Fatalf("no %q button", label)
		}
		test.Tap(b)
	}
	if len(asked) != 2 || asked[0] != kanacore.ScriptHiragana || asked[1] != kanacore.ScriptKatakana {
		t.Fatalf("onReset called with %v", asked)
	}
}

func TestSettingsResetConfirmation(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	gs := settingsGame()
	learnKana(gs, "あ", "い")
	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, statsPanel, gameCanvas, w)
	defer gs.Stop()

	showSettingsDialog(gs, statsPanel, gameCanvas, ib, w)
	settings := w.Canvas().Overlays().Top()
	test.Tap(buttonIn(settings, "Reset hiragana progress…"))
	confirm := w.Canvas().Overlays().Top()
	if confirm == settings || buttonIn(confirm, "Cancel") == nil {
		t.Fatal("no confirmation dialog")
	}
	test.Tap(buttonIn(confirm, "Cancel"))
	if gs.kanaProgress["あ"].Level != 5 || w.Canvas().Overlays().Top() != settings {
		t.Fatal("Cancel changes nothing and keeps the settings open")
	}

	test.Tap(buttonIn(settings, "Reset hiragana progress…"))
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Reset"))
	gs.mu.Lock()
	level, open, over := gs.kanaProgress["あ"].Level, gs.settingsOpen, gs.over
	intro := append([]string(nil), gs.pendingIntro...)
	gs.mu.Unlock()
	if level != 0 || open || over || !equalIDs(intro, []string{"vowels"}) {
		t.Fatalf("after Reset: level %d, settings open %v, over %v, intro %v", level, open, over, intro)
	}
	// The settings dialog is gone; the vowels intro (shown by restartRound) may be on top.
	for _, o := range w.Canvas().Overlays().List() {
		if o == settings {
			t.Fatal("the settings dialog must close after a reset")
		}
	}
}

func TestSettingsResetBusyShowsTheReason(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	gs := settingsGame()
	gs.pendingOffer = true
	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, statsPanel, gameCanvas, w)

	showSettingsDialog(gs, statsPanel, gameCanvas, ib, w)
	settings := w.Canvas().Overlays().Top()
	test.Tap(buttonIn(settings, "Reset hiragana progress…"))
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Reset"))
	top := w.Canvas().Overlays().Top()
	if len(w.Canvas().Overlays().List()) != 2 || !gs.settingsOpen {
		t.Fatal("want the settings still open with an error dialog on top")
	}
	shown := false
	for _, o := range test.LaidOutObjects(top) {
		if l, ok := o.(*widget.Label); ok && strings.Contains(l.Text, "Finish the open dialog first.") {
			shown = true
		}
	}
	if !shown {
		t.Fatal("the error dialog must name the reason")
	}
}
```

Update every other `newSettingsForm(st)` call in the tests (`settings_test.go`, `fonts_test.go` ~156 and ~175) to `newSettingsForm(st, nil)`. The button handler must tolerate a nil `onReset` (do nothing).

- [ ] **Step 3: Run** `go test ./fyne/ -run 'Reset'` — expected: compile failure.

- [ ] **Step 4: Implement.**
  - `settingsForm` gets a field `onReset func(kanacore.Script)`; `newSettingsForm(st settingsState, onReset func(kanacore.Script))` stores it before building the tabs. (This replaces the spec's `func(kanacore.Script) error`: the confirmation and the error dialog need the window, which only `showSettingsDialog` has, so the form only reports the click.)
  - `newKanaTab`: after `f.autoCheck`, one button per script:

```go
	resets := container.NewHBox()
	for _, info := range kanacore.Scripts() {
		s := info.Script
		resets.Add(widget.NewButton("Reset "+strings.ToLower(info.Label)+" progress…", func() {
			if f.onReset != nil {
				f.onReset(s)
			}
		}))
	}
	return container.NewVBox(scriptRow, rowTabs, widget.NewSeparator(), f.autoCheck, resets)
```

  - `showSettingsDialog`: declare `var d *dialog.CustomDialog` before building the form, then:

```go
	var d *dialog.CustomDialog
	onReset := func(s kanacore.Script) {
		name := strings.ToLower(scriptLabel(s))
		c := dialog.NewConfirm("Reset "+name+" progress",
			"Delete the statistics and learning state of all "+name+"? This cannot be undone.",
			func(ok bool) {
				if !ok {
					return
				}
				if err := gs.ResetScript(s); err != nil {
					dialog.ShowError(err, win)
					return
				}
				// Reset clears settingsOpen, so closed() is not called;
				// unsaved changes in the other tabs are dropped, as with Cancel.
				d.Hide()
				restartRound(gs, statsPanel, gameCanvas, inputBar, win)
			}, win)
		c.SetConfirmText("Reset")
		c.SetDismissText("Cancel")
		c.Show()
	}
	form := newSettingsForm(gs.settingsState(), onReset)
	...
	d = dialog.NewCustomWithoutButtons("Settings", form.tabs, win) // was `d := …`
```

  Add to `settings.go` (not to `kanacore`):

```go
// scriptLabel is the display name of a script ("Hiragana").
func scriptLabel(s kanacore.Script) string {
	for _, info := range kanacore.Scripts() {
		if info.Script == s {
			return info.Label
		}
	}
	return string(s)
}
```

- [ ] **Step 5: Run** `go test ./...` and `go vet ./...` — expected: PASS, clean. Then build: `go build -o /tmp/kana-desktop ./fyne/` succeeds.

- [ ] **Step 6: Commit message:** `feat(fyne): reset a script's progress from the settings`

---

## Final check (controller)

- [ ] `go test ./...`, `go vet ./...`, `gofmt -l .` (empty).
- [ ] `grep -rn "secureCorrect\|knownDirty\|TotalCorrect\|totalCorrect\|wordCorrect(" fyne/ store/` — no hits.
- [ ] Final code review of the whole branch against the spec, then push for a manual test on the user's machine: levels rise and fall, red frame persists across restart, wrong input on a single tile, reset of each script with its intro, migration of the real `~/.local/share/kana/kana.db` (back it up first: `cp ~/.local/share/kana/kana.db ~/.local/share/kana/kana.db.bak`).
