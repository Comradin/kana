# Meaning Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A saved setting "Words ask for: Reading / Meaning" makes word tiles ask for the word's meaning only, and every word keeps a reading level and a meaning level of its own, each with its own slipped flag, memory and hit/miss counts.

**Architecture:** `vocab.Meanings.Answerable` tells whether a word has a meaning to type. `store` gets a `meaning_stats` table and a `words_ask` setting; meaning progress and memory use the existing generic `Progress`/`MemoryStates` API under the prefix `meaning:`. In `fyne` a `wordKind` (`readingKind`, `meaningKind`) selects one of two parallel sets of word fields through `gs.track(kind)`; every word tile carries the kind it asks for (`KanaTile.ask`), a meaning tile carries a small "?" marker, answers count for the kind they match, a typed reading on a meaning tile shows the hint "type the meaning", and selection, weak items, the package rhythm and the status line follow `gs.wordsAsk`.

**Tech Stack:** Go 1.25, Fyne v2.7.3, SQLite (modernc.org/sqlite). No new dependencies.

Spec: `docs/superpowers/specs/2026-10-23-meaning-mode-design.md`. Terms: `GLOSSARY.md` (meaning mode, learning level, reading level, meaning level, learned, slipped, weak item, waiting word, words-only mode). The GLOSSARY already has every term; no GLOSSARY task.

**Conventions:** comments like the surrounding code (short doc comment on every function, "Must be called under lock." where it applies). Every `GameState` field is read and written under `gs.mu`; the `fyne` tests touch fields directly without the lock, as the existing tests do. **Test data is synthetic only** (invented words such as かさ/umbrella, あい/love); every new `fyne` test that calls `NewGameState` calls `isolateWordsDir(t)` first, and nothing may read anything under `~/.local/share/kana/` (private lists live there). Quick checks per task: `go build ./... && go vet ./...`, `gofmt -l .` (empty; run `gofmt -w` on the listed files when it is not, e.g. after adding fields to an aligned struct literal) and the named `-run` tests; `go test ./fyne/` takes about 3.5 minutes, so the whole package runs only where a step says so. Commit messages: conventional subject of at most 72 characters plus one short body paragraph (given as a second `-m`), no Co-Authored-By line, no mention of AI. Implementers may leave the commit to the controller, who reviews and commits with the given message. `CLAUDE.md` is not in the repository and is not touched.

---

## Decisions made in the plan

The spec leaves these open or calls them indicative; check them before Task 1.

1. **Store layout for progress and memory.** Meaning levels and slipped flags are rows `meaning:<reading>` in the existing `progress` table, meaning memory rows `meaning:<reading>` in the `memory` table, beside the reading's `word:<reading>`. The store API is already generic (`Progress(prefix)`, `SaveProgress(item)`, `MemoryStates(prefix)`, `SaveMemoryState(item)`), so it needs no new functions for them; existing `word:` rows keep meaning the reading and nothing is migrated.
2. **Store layout for hit/miss counts:** a new table `meaning_stats (kana TEXT PRIMARY KEY, correct_count, miss_count)`, created with `CREATE TABLE IF NOT EXISTS` like `word_stats`, with `SaveMeaningStats`/`MeaningStatistics` mirroring `SaveWordStats`/`WordStatistics` (both through one private helper). Not new columns in `word_stats` and not `meaning:` keys in it: `word_stats` rows are read by `WordStatistics` and were used to seed `introduced_words` and `progress`, so mixing keys there would leak into every reader. Existing `word_stats` rows stay the reading's.
3. **The setting:** key `words_ask`, values `store.WordsAskReading` ("reading", also for unset or unknown values) and `store.WordsAskMeaning` ("meaning"); `WordsAsk()`/`SaveWordsAsk()` (saving another value is an error). A string rather than a flag, so the reverse direction (own backlog issue) can add a value.
4. **How `GameState` holds the mode:** `wordsAsk wordKind`, with `type wordKind int` and `readingKind` (zero value) / `meaningKind`. `wordKind.prefix()` gives `word:` / `meaning:`, `wordKind.setting()` and `kindFromSetting` convert to the stored value. The settings dialog passes it as `wordSettings.ask`; `applyWordSettings` saves it.
5. **Two parallel sets of word fields.** The existing `word…` fields stay the reading's (so the many tests that use them keep working); new `meaning…` fields mirror them (`meaningOverallStats`, `meaningSessionStats`, `meaningSessionDirty`, `meaningMemory`, `meaningMemoryDirty`, `meaningMemoryLoadFailed`, `meaningProgress`, `meaningProgressDirty`, `meaningProgressLoadFailed`). `gs.track(kind)` returns a `wordTrack` view (the maps by reference, the flags as pointers), so every function that records, saves, loads or merges takes a `wordKind` instead of being written twice. Maps are cleared and filled in place (`clear`, `maps.Copy`) instead of replaced, so a view never points at a stale map.
6. **Tiles keep their question:** `KanaTile.ask wordKind` is set when the tile is built (`newWordTile` → reading, new `newMeaningTile` → meaning). Answers, the hint, hits and misses go by the tile's `ask`, not by the current setting, so a change of the setting applies only to tiles spawned afterwards. `dropInactiveTiles` and the playable cache do not depend on the mode, so switching the mode drops no falling tile.
7. **Which answer counts for which level** (`answeredKind`): a meaning tile counts every hit for the meaning; a reading tile counts a hit for the reading when the input is one of its readings, else (answered by meaning) for the meaning. So in reading mode an input that is both a reading and a meaning counts for the reading (as today), and in meaning mode only meanings are tried, which is "meanings checked first": an input that is both is a meaning hit.
8. **Hint:** a new feedback kind `feedbackHint`, text `"type the meaning"` (`meaningHintText`), placed at the tile like a hit label, in the Warm Paper accent brown `#8b5e3c` (`feedbackHintColor`), not the hit green or the miss red. It is shown when no tile is answered and the input is the reading of a falling meaning tile (the lowest such tile); it is no mistake, the tile keeps falling, no points, no tempo change. A new hint replaces an earlier hint label still on the field, so repeated Enters do not stack labels. A reading typed for a meaning tile that just fell through counts as a late Enter (no mistake on another tile), like any other late answer.
9. **Mistakes** lower the level, memory and miss count of the tile's question only (`recordWordMiss(kana, tile.ask)` in `tick`, `applyWordMistake(kana, tile.ask)` in `wrongInput`).
10. **Eligible words in meaning mode:** `vocab.Meanings.Answerable(lang)` is true when one of the meanings `MeaningMatches` compares (`For(lang)` and English) keeps some text after normalising. Because `For` falls back to the language-less meanings, English and then any language, this is in effect "has a typeable meaning in any language" — exactly the set of words `MeaningMatches` can ever accept, so eligibility follows matching "as today" (see open question A). `gs.askable(e)` applies it in meaning mode only; `gs.askableWords()` filters `playableWords()` at pick time (no cache: the meaning language may change). Words without a meaning stay playable and introduced; they only do not spawn in meaning mode.
11. **Selection:** `pickWord` draws from `askableWords()` (so the word share, too, counts only askable words), `pickReading` uses `gs.track(gs.wordsAsk)`'s progress for the focus pick and its memory for `memory.Pick`.
12. **Package rhythm and counts:** `newestPackageDone` counts only words that can fall now *and* are askable, and judges them by the asked kind's level. `introducedCounts` (status line, Words tab summary) counts in meaning mode only askable introduced words, learned by their meaning level. Which words a new package brings is unchanged (packages are shared); a package word without a meaning is introduced as usual and simply never falls in meaning mode.
13. **Status line:** `WordsStatus.Meaning`; only the two count lines switch their label: "Meanings: N introduced · N learned · …" (also with "N waiting for kana"). The hint lines ("Words: learn more kana for new words", "Words: JLPT N5 finished", …) keep "Words:", since they are about words and packages, not levels.
14. **Words tab summary** follows the same counts and says "Introduced: 2 · meanings learned: 1 · left in source: 1" in meaning mode (the spec names only the status line; this keeps the two from contradicting each other).
15. **Words-only plus meaning mode with no askable word:** kana fall (as for any words-only fallback) and the line says "Words only: no word has a meaning to ask for" (`WordsStatus.NoMeanings`).
16. **Tile marker:** a bold "?" (13 pt) in `wordTileBorderColor`, in the top-right corner of a meaning tile, 3 px in from the edges. The meaning tile is wider than a reading tile by the marker's width, the word stays centred, so the marker sits in the right-hand gap and never covers the kana (pinned by a test). It is drawn on the tile's own fixed sage face, not on the window background, so it reads whatever the theme (the app's Warm Paper theme ignores the OS variant anyway). The shown form, font shrinking for long words and the border are unchanged.
17. **"Already known" in the package intro** marks only the reading as learned (`markKnown` unchanged): the GLOSSARY says the meaning level "starts at 0 and grows only from meanings typed".
18. **Slipped frame:** no word tile has a red frame today — the red frame exists only on the kana cells of the status panel (`kanaCell.set`). There is nothing to switch, so this plan adds none; the slipped flag is kept per kind, so a later frame (or the level-dots issue) can read `gs.track(tile.ask).progress`. See open question B.
19. **Statistics display:** per-word hit/miss counts are not shown anywhere today, so "show the chosen mode's counts" has no site yet; they are only stored apart. The session line's "Correct" counts every hit, kana and words of both kinds (`sessionWordCorrect` sums both tracks), because it is a session tally of answered tiles next to "Missed".
20. **Settings dialog:** a bold label "Words ask for" and a horizontal required radio group "Reading" / "Meaning" in the Words tab directly under "Tiles".
21. **Task order:** storage first (Tasks 1–3, no behaviour change), then the tile, the answers, the mode, selection, packages and status line, the dialog, and the README last.

## Open questions (choices made, check them)

- **A. Eligibility vs. the fallback in `For`.** The spec says "a word with no meaning in the meaning language or English does not fall in meaning mode", and also "meaning matching is as today". Today `MeaningMatches` accepts `For(lang)`, which falls back to any language, so a private word with only Hungarian meanings is answerable in German mode by its Hungarian meaning. The plan keeps matching as today and makes eligibility follow it (Decision 10). If only the meaning language and English should count, both `Answerable` and `MeaningMatches` would need `m[lang]`/`m[""]` instead of `For(lang)` — a change to today's matching.
- **B. Slipped frame.** See Decision 18: the spec's "a word tile's red frame follows the level of the chosen mode" has no existing frame to follow. If a red frame on slipped word tiles is wanted, it is a small addition to `newWordTile`/`newMeaningTile` (stroke colour `cellMissedColor` when `gs.track(kind).progress[kana].Flagged`), not in this plan.

---

## File structure

- Modify `vocab/meaning.go`, `vocab/meaning_test.go`: `Meanings.Answerable`.
- Modify `store/store.go`, `store/store_test.go`: `meaning_stats` table, `SaveMeaningStats`, `MeaningStatistics`, private `saveCounts`/`counts`, `WordsAskReading`/`WordsAskMeaning`, `WordsAsk`, `SaveWordsAsk`.
- Create `fyne/wordkind.go`, `fyne/wordkind_test.go`: `wordKind`, `wordKinds`, `prefix`, `setting`, `kindFromSetting`, `wordTrack`, `gs.track`, `gs.loadTrack`, `gs.loadWordStats`, `gs.saveWordStats`, `answeredKind`, `typedReading`, `gs.readingTyped`, `gs.askable`, `gs.askableWords`.
- Modify `fyne/game.go`: meaning fields, `wordsAsk`, `NewGameState`, `Reset`, `tick`, `spawnKana`, `checkAnswer`, `wrongInput`, `answers`, `saveMemory`.
- Modify `fyne/learning.go`: `wordLearned`, `raiseWord`, `lowerWord`, `saveProgress`.
- Modify `fyne/wordgame.go`: `recordWordCorrect`, `applyWordMistake`, `recordWordMiss`, `mergeWordStats`, `mergeWordStatsOf`, `pickWord`, `introducedCounts`, `wordsSummary`, `wordsStatus`, `sessionWordCorrect`.
- Modify `fyne/focus.go`: `pickReading`.
- Modify `fyne/packages.go`: `wordSettings.ask`, `newestPackageDone`, `applyWordSettings`.
- Modify `fyne/tile.go`: `KanaTile.ask`, `KanaTile.marker`, `newMeaningTile`, `buildWordTile`, `Move`, `Objects`.
- Modify `fyne/feedback.go`: `feedbackHint`, `feedbackHintColor`, `meaningHintText`, `showMeaningHint`.
- Modify `fyne/words.go`: `WordsStatus.Meaning`, `WordsStatus.NoMeanings`, `formatWordsLine`, `wordsOnlyReason`.
- Modify `fyne/settings.go`: `settingsState.wordsAsk`, `settingsForm.wordsAsk`, `askReadingLabel`/`askMeaningLabel`, `newWordsTab`, `askedKind`, `words`.
- Create `fyne/meaningmode_test.go`; modify tests `fyne/game_test.go` (`newTestState`), `fyne/learning_test.go`, `fyne/wordgame_test.go`, `fyne/tile_test.go`, `fyne/words_test.go`, `fyne/settings_test.go`.
- Modify `README.md`.

---

### Task 1: `vocab.Meanings.Answerable`

**Files:**
- Modify: `vocab/meaning.go`
- Test: `vocab/meaning_test.go`

- [ ] **Step 1: Write the failing test.** Append to `vocab/meaning_test.go`:

```go
func TestAnswerable(t *testing.T) {
	cases := []struct {
		name string
		m    Meanings
		lang string
		want bool
	}{
		{"nil", nil, "de", false},
		{"empty", Meanings{}, "de", false},
		{"meaning language", Meanings{"de": {"Regenschirm"}}, "de", true},
		{"English only", Meanings{"en": {"umbrella"}}, "de", true},
		{"language-less", Meanings{"": {"die Person dort drüben"}}, "fr", true},
		{"only brackets", Meanings{"en": {"(polite)", "[obsolete]"}}, "en", false},
		{"another language, as For falls back to it", Meanings{"hu": {"esernyő"}}, "de", true},
	}
	for _, c := range cases {
		if got := c.m.Answerable(c.lang); got != c.want {
			t.Errorf("%s: Answerable(%q) = %v, want %v", c.name, c.lang, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run** `go test ./vocab/ -run TestAnswerable` → build failure: `c.m.Answerable undefined`.

- [ ] **Step 3: Implement.** In `vocab/meaning.go`, after `MeaningMatches`, add:

```go
// Answerable reports whether some meaning can be typed for lang: one of the
// meanings MeaningMatches compares (For(lang) and English) keeps some text
// once case, brackets, punctuation and a leading article are ignored.
func (m Meanings) Answerable(lang string) bool {
	for _, items := range [][]string{m.For(lang), m["en"]} {
		for _, it := range items {
			if normalizeMeaning(it) != "" {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 4: Run** `go test ./vocab/ && go build ./... && go vet ./... && gofmt -l .` → PASS, no output from gofmt.

- [ ] **Step 5: Commit.**

```bash
git add vocab/meaning.go vocab/meaning_test.go
git commit -m "feat(vocab): tell whether a word has a meaning to type" -m "Meaning mode asks only about words whose meaning can be typed. Answerable checks the same meanings MeaningMatches compares, so eligibility follows matching."
```

---

### Task 2: Store — meaning stats and the words-ask setting

**Files:**
- Modify: `store/store.go`
- Test: `store/store_test.go`

- [ ] **Step 1: Write the failing tests.** Append to `store/store_test.go`:

```go
func TestMeaningStatsApartFromWordStats(t *testing.T) {
	st := openTemp(t)
	if err := st.SaveWordStats("かさ", 4, 1); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveMeaningStats("かさ", 2, 3); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveMeaningStats("いぬ", -1, -2); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveMeaningStats("", 1, 0); err == nil {
		t.Fatal("an empty reading must be refused")
	}
	words, err := st.WordStatistics()
	if err != nil || !reflect.DeepEqual(words, map[string]WordStats{"かさ": {Kana: "かさ", CorrectCount: 4, MissCount: 1}}) {
		t.Fatalf("word stats %v, %v; the meaning counts must not mix in", words, err)
	}
	meanings, err := st.MeaningStatistics()
	want := map[string]WordStats{
		"かさ": {Kana: "かさ", CorrectCount: 2, MissCount: 3},
		"いぬ": {Kana: "いぬ"}, // negative counts are stored as 0
	}
	if err != nil || !reflect.DeepEqual(meanings, want) {
		t.Fatalf("meaning stats %v, %v; want %v", meanings, err, want)
	}
	if err := st.SaveMeaningStats("かさ", 5, 3); err != nil {
		t.Fatal(err)
	}
	if m, _ := st.MeaningStatistics(); m["かさ"].CorrectCount != 5 {
		t.Fatalf("upsert: %+v", m["かさ"])
	}
}

func TestMeaningProgressAndMemoryApartFromReading(t *testing.T) {
	st := openTemp(t)
	now := time.Unix(1_700_000_000, 0)
	for _, err := range []error{
		st.SaveProgress("word:かさ", Progress{Level: 5}),
		st.SaveProgress("meaning:かさ", Progress{Level: 1, Flagged: true}),
		st.SaveMemoryState("word:かさ", memory.State{Stability: 4, LastSeen: now}),
		st.SaveMemoryState("meaning:かさ", memory.State{Stability: 0.5, LastSeen: now}),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	words, _ := st.Progress("word:")
	meanings, _ := st.Progress("meaning:")
	if !reflect.DeepEqual(words, map[string]Progress{"かさ": {Level: 5}}) ||
		!reflect.DeepEqual(meanings, map[string]Progress{"かさ": {Level: 1, Flagged: true}}) {
		t.Fatalf("progress: word %v, meaning %v", words, meanings)
	}
	wm, _ := st.MemoryStates("word:")
	mm, _ := st.MemoryStates("meaning:")
	if len(wm) != 1 || wm["かさ"].Stability != 4 || len(mm) != 1 || mm["かさ"].Stability != 0.5 {
		t.Fatalf("memory: word %v, meaning %v", wm, mm)
	}
}

func TestWordsAsk(t *testing.T) {
	st := openTemp(t)
	if a, err := st.WordsAsk(); err != nil || a != WordsAskReading {
		t.Fatalf("unset WordsAsk = %q, %v; want %q", a, err, WordsAskReading)
	}
	if err := st.SaveWordsAsk(WordsAskMeaning); err != nil {
		t.Fatal(err)
	}
	if a, _ := st.WordsAsk(); a != WordsAskMeaning {
		t.Fatalf("WordsAsk = %q, want %q", a, WordsAskMeaning)
	}
	if err := st.SaveWordsAsk("kanji"); err == nil {
		t.Fatal("an unknown value must be refused")
	}
	if err := st.setSetting(wordsAskKey, "kanji"); err != nil {
		t.Fatal(err)
	}
	if a, _ := st.WordsAsk(); a != WordsAskReading {
		t.Fatalf("unknown stored value gives %q, want %q", a, WordsAskReading)
	}
}
```

- [ ] **Step 2: Run** `go test ./store/ -run 'TestMeaningStats|TestMeaningProgress|TestWordsAsk'` → build failure: `st.SaveMeaningStats undefined`, `undefined: WordsAskReading`, `undefined: wordsAskKey`.

- [ ] **Step 3: Implement.** In `store/store.go`:

In the `const (` block with the keys, add after `focusWeakKey       = "focus_weak"`:

```go
	wordsAskKey        = "words_ask"
```

After `const DefaultKanaFont = "noto"` add:

```go
// What word tiles ask for, as stored in the words_ask setting.
const (
	WordsAskReading = "reading" // the default
	WordsAskMeaning = "meaning"
)

// The tables of per-reading hit and miss counts.
const (
	wordStatsTable    = "word_stats"    // typing the reading
	meaningStatsTable = "meaning_stats" // typing the meaning
)
```

Change the `WordStats` doc comment to:

```go
// WordStats is the aggregated statistics for one word reading: of typing the
// reading (word_stats) or the meaning (meaning_stats). It is kept apart from
// KanaStats: the word か (mosquito) is not the character か.
```

After `SaveFocusWeak` add:

```go
// WordsAsk returns what word tiles ask for: WordsAskMeaning, or
// WordsAskReading when stored so, unset or unknown.
func (s *Store) WordsAsk() (string, error) {
	value, err := s.getSetting(wordsAskKey)
	if errors.Is(err, sql.ErrNoRows) {
		return WordsAskReading, nil
	}
	if err != nil {
		return WordsAskReading, err
	}
	if value == WordsAskMeaning {
		return value, nil
	}
	return WordsAskReading, nil
}

// SaveWordsAsk persists what word tiles ask for (WordsAskReading or
// WordsAskMeaning).
func (s *Store) SaveWordsAsk(ask string) error {
	if ask != WordsAskReading && ask != WordsAskMeaning {
		return fmt.Errorf("store: words ask %q is not %q or %q", ask, WordsAskReading, WordsAskMeaning)
	}
	return s.setSetting(wordsAskKey, ask)
}
```

Replace `SaveWordStats` and `WordStatistics` (the whole two functions) with:

```go
// SaveWordStats upserts the aggregated statistics for one word reading
// (absolute values, like SaveKanaStats).
func (s *Store) SaveWordStats(kana string, correctCount, missCount int) error {
	return s.saveCounts(wordStatsTable, "word stats", kana, correctCount, missCount)
}

// WordStatistics returns the stats for all tracked word readings.
func (s *Store) WordStatistics() (map[string]WordStats, error) {
	return s.counts(wordStatsTable, "word stats")
}

// SaveMeaningStats upserts the hit and miss counts of typing one word's
// meaning, keyed by its reading (absolute values, like SaveWordStats).
func (s *Store) SaveMeaningStats(kana string, correctCount, missCount int) error {
	return s.saveCounts(meaningStatsTable, "meaning stats", kana, correctCount, missCount)
}

// MeaningStatistics returns the meaning stats of all tracked readings.
func (s *Store) MeaningStatistics() (map[string]WordStats, error) {
	return s.counts(meaningStatsTable, "meaning stats")
}

// saveCounts upserts one reading's counts in table (wordStatsTable or
// meaningStatsTable); what names them in errors. Negative counts become 0.
func (s *Store) saveCounts(table, what, kana string, correctCount, missCount int) error {
	if kana == "" {
		return errors.New("store: word reading is required")
	}
	correctCount = max(correctCount, 0)
	missCount = max(missCount, 0)
	_, err := s.db.Exec(`
		INSERT INTO `+table+` (kana, correct_count, miss_count)
		VALUES (?, ?, ?)
		ON CONFLICT(kana) DO UPDATE SET
			correct_count = excluded.correct_count,
			miss_count = excluded.miss_count
	`, kana, correctCount, missCount)
	if err != nil {
		return fmt.Errorf("store: save %s %s: %w", what, kana, err)
	}
	return nil
}

// counts returns every reading's counts in table; what names them in
// errors.
func (s *Store) counts(table, what string) (map[string]WordStats, error) {
	rows, err := s.db.Query(`SELECT kana, correct_count, miss_count FROM ` + table)
	if err != nil {
		return nil, fmt.Errorf("store: query %s: %w", what, err)
	}
	defer rows.Close()

	stats := make(map[string]WordStats)
	for rows.Next() {
		var ws WordStats
		if err := rows.Scan(&ws.Kana, &ws.CorrectCount, &ws.MissCount); err != nil {
			return nil, fmt.Errorf("store: scan %s: %w", what, err)
		}
		stats[ws.Kana] = ws
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate %s: %w", what, err)
	}
	return stats, nil
}
```

In `migrate`, add to `stmts` after the `word_stats` statement:

```go
		`CREATE TABLE IF NOT EXISTS meaning_stats (
			kana TEXT PRIMARY KEY,
			correct_count INTEGER NOT NULL DEFAULT 0,
			miss_count INTEGER NOT NULL DEFAULT 0
		);`,
```

- [ ] **Step 4: Run** `go test ./store/ && go build ./... && go vet ./... && gofmt -l .` → PASS (the existing `TestWordStats…` tests still pass), gofmt silent.

- [ ] **Step 5: Commit.**

```bash
git add store/store.go store/store_test.go
git commit -m "feat(store): keep meaning stats and what word tiles ask for" -m "A meaning_stats table holds hit and miss counts per reading apart from word_stats, so existing counts keep meaning the reading. The words_ask setting stores reading or meaning, reading by default."
```

---

### Task 3: `fyne` — two tracks per word (no behaviour change)

**Files:**
- Create: `fyne/wordkind.go`, `fyne/wordkind_test.go`
- Modify: `fyne/game.go` (fields, `NewGameState`, `Reset`, `tick`, `checkAnswer`, `wrongInput`, `saveMemory`), `fyne/learning.go` (`wordLearned`, `raiseWord`, `lowerWord`, `saveProgress`), `fyne/wordgame.go` (`recordWordCorrect`, `applyWordMistake`, `recordWordMiss`, `mergeWordStats`, `introducedCounts`, `sessionWordCorrect`), `fyne/packages.go` (`newestPackageDone`)
- Test: `fyne/game_test.go` (`newTestState`), `fyne/learning_test.go`, `fyne/wordgame_test.go`

- [ ] **Step 1: Write the failing tests.** Create `fyne/wordkind_test.go`:

```go
package main

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/memory"
	"kana/store"
)

func TestWordKindPrefixAndSetting(t *testing.T) {
	if readingKind.prefix() != "word:" || meaningKind.prefix() != "meaning:" {
		t.Fatalf("prefixes %q, %q", readingKind.prefix(), meaningKind.prefix())
	}
	if readingKind.setting() != store.WordsAskReading || meaningKind.setting() != store.WordsAskMeaning {
		t.Fatalf("settings %q, %q", readingKind.setting(), meaningKind.setting())
	}
	if kindFromSetting(store.WordsAskMeaning) != meaningKind || kindFromSetting(store.WordsAskReading) != readingKind || kindFromSetting("x") != readingKind {
		t.Fatal("kindFromSetting: meaning is meaningKind, anything else readingKind")
	}
}

func TestMeaningTrackIsKeptApart(t *testing.T) {
	gs := newTestState()
	gs.recordWordCorrect("かさ", meaningKind)
	gs.recordWordCorrect("かさ", meaningKind)
	gs.recordWordMiss("かさ", meaningKind)
	if p := gs.meaningProgress["かさ"]; p != (store.Progress{Level: 0, Flagged: true}) || !gs.meaningProgressDirty["かさ"] {
		t.Fatalf("meaning progress %+v, dirty %v", p, gs.meaningProgressDirty["かさ"])
	}
	if s := gs.meaningSessionStats["かさ"]; s.CorrectCount != 2 || s.MissCount != 1 || s.Kana != "かさ" || !gs.meaningSessionDirty {
		t.Fatalf("meaning session %+v, dirty %v", s, gs.meaningSessionDirty)
	}
	if _, ok := gs.meaningMemory["かさ"]; !ok || !gs.meaningMemoryDirty["かさ"] {
		t.Fatal("a meaning hit must create meaning memory")
	}
	if len(gs.wordProgress) != 0 || len(gs.wordSessionStats) != 0 || len(gs.wordMemory) != 0 || gs.wordSessionDirty {
		t.Fatal("the reading track must stay untouched")
	}
	if gs.sessionWordCorrect() != 2 {
		t.Fatalf("session word hits %d, want 2 (both kinds count)", gs.sessionWordCorrect())
	}
	gs.meaningProgress["かさ"] = store.Progress{Level: learnedLevel}
	if !gs.wordLearned("かさ", meaningKind) || gs.wordLearned("かさ", readingKind) {
		t.Fatal("wordLearned goes by the kind's level")
	}
}

func TestMeaningTrackSavedAndReloaded(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.recordWordCorrect("かさ", meaningKind)
	gs.recordWordCorrect("いぬ", readingKind)
	gs.mergeSessionStats()
	gs.mu.Unlock()

	meanings, _ := st.Progress("meaning:")
	readings, _ := st.Progress("word:")
	if meanings["かさ"].Level != 1 || readings["かさ"].Level != 0 || readings["いぬ"].Level != 1 || meanings["いぬ"].Level != 0 {
		t.Fatalf("stored progress: meaning %v, reading %v", meanings, readings)
	}
	if ms, _ := st.MemoryStates("meaning:"); len(ms) != 1 {
		t.Fatalf("stored meaning memory %v, want かさ only", ms)
	}
	if s, _ := st.MeaningStatistics(); s["かさ"].CorrectCount != 1 || s["いぬ"].CorrectCount != 0 {
		t.Fatalf("stored meaning stats %v", s)
	}
	if s, _ := st.WordStatistics(); s["いぬ"].CorrectCount != 1 || s["かさ"].CorrectCount != 0 {
		t.Fatalf("stored word stats %v", s)
	}

	gs.mu.Lock()
	gs.recordWordMiss("かさ", meaningKind)
	gs.mu.Unlock()
	gs.Reset() // merges, then reloads the overall stats
	if len(gs.meaningSessionStats) != 0 || gs.meaningSessionDirty || gs.meaningOverallStats["かさ"] != (store.WordStats{Kana: "かさ", CorrectCount: 1, MissCount: 1}) {
		t.Fatalf("after Reset: session %v, overall %v", gs.meaningSessionStats, gs.meaningOverallStats)
	}

	again := NewGameState(st)
	if again.meaningProgress["かさ"] != (store.Progress{Flagged: true}) || again.wordProgress["かさ"].Level != 0 || again.meaningOverallStats["かさ"].MissCount != 1 {
		t.Fatalf("reloaded: meaning %+v, reading %+v, stats %+v", again.meaningProgress["かさ"], again.wordProgress["かさ"], again.meaningOverallStats["かさ"])
	}
	if _, ok := again.meaningMemory["かさ"]; !ok {
		t.Fatal("meaning memory not reloaded")
	}
}

func TestExistingWordProgressLoadsAsReading(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveProgress("word:かさ", store.Progress{Level: 3, Flagged: true})
	_ = st.SaveMemoryState("word:かさ", memory.State{Stability: 2, LastSeen: time.Now()})
	_ = st.SaveWordStats("かさ", 7, 2)
	gs := NewGameState(st)
	if gs.wordProgress["かさ"] != (store.Progress{Level: 3, Flagged: true}) || gs.wordOverallStats["かさ"].CorrectCount != 7 {
		t.Fatalf("reading: %+v, stats %+v", gs.wordProgress["かさ"], gs.wordOverallStats["かさ"])
	}
	if _, ok := gs.wordMemory["かさ"]; !ok {
		t.Fatal("reading memory not loaded")
	}
	if len(gs.meaningProgress) != 0 || len(gs.meaningMemory) != 0 || len(gs.meaningOverallStats) != 0 {
		t.Fatalf("meaning must start empty: %v %v %v", gs.meaningProgress, gs.meaningMemory, gs.meaningOverallStats)
	}
}
```

Then update the existing test call sites to the new signatures:

```bash
sed -i -E 's/gs\.(recordWordCorrect|recordWordMiss)\("([^"]+)"\)/gs.\1("\2", readingKind)/g' fyne/learning_test.go fyne/wordgame_test.go
```

In `fyne/game_test.go`, `newTestState`, add after `wordProgressDirty: map[string]bool{},`:

```go
		meaningOverallStats:  make(map[string]store.WordStats),
		meaningSessionStats:  make(map[string]store.WordStats),
		meaningMemory:        map[string]memory.State{},
		meaningMemoryDirty:   map[string]bool{},
		meaningProgress:      map[string]store.Progress{},
		meaningProgressDirty: map[string]bool{},
```

- [ ] **Step 2: Run** `go vet ./fyne/` → build failure: `undefined: readingKind`, `unknown field meaningOverallStats`, `too many arguments in call to gs.recordWordCorrect`.

- [ ] **Step 3: Create `fyne/wordkind.go`:**

```go
package main

import (
	"maps"

	"kana/memory"
	"kana/store"
)

// wordKind is what a word tile asks for and what a word answer counts for:
// the word's reading or its meaning. Each kind has its own learning level,
// slipped flag, memory and hit/miss counts per reading.
type wordKind int

const (
	readingKind wordKind = iota // the zero value: tiles asked for the reading before meanings had levels
	meaningKind
)

// wordKinds lists every kind, reading first.
var wordKinds = []wordKind{readingKind, meaningKind}

// prefix is the kind's item prefix in the store's progress and memory
// tables: "word:" for the reading (the items as they were before meaning
// levels), "meaning:" for the meaning.
func (k wordKind) prefix() string {
	if k == meaningKind {
		return "meaning:"
	}
	return "word:"
}

// setting is the kind as stored in the words_ask setting.
func (k wordKind) setting() string {
	if k == meaningKind {
		return store.WordsAskMeaning
	}
	return store.WordsAskReading
}

// kindFromSetting reads the words_ask setting: store.WordsAskMeaning is the
// meaning, anything else the reading.
func kindFromSetting(s string) wordKind {
	if s == store.WordsAskMeaning {
		return meaningKind
	}
	return readingKind
}

// wordTrack is one kind's per-reading data in GameState. The maps are
// GameState's own, so writes through the view land there; the flags point
// at GameState's fields. Build it with track for each use; do not keep it.
type wordTrack struct {
	overallStats       map[string]store.WordStats
	sessionStats       map[string]store.WordStats
	sessionDirty       *bool
	memory             map[string]memory.State
	memoryDirty        map[string]bool
	memoryLoadFailed   *bool
	progress           map[string]store.Progress
	progressDirty      map[string]bool
	progressLoadFailed *bool
}

// track returns kind k's data. Must be called under lock.
func (gs *GameState) track(k wordKind) wordTrack {
	if k == meaningKind {
		return wordTrack{
			overallStats: gs.meaningOverallStats, sessionStats: gs.meaningSessionStats, sessionDirty: &gs.meaningSessionDirty,
			memory: gs.meaningMemory, memoryDirty: gs.meaningMemoryDirty, memoryLoadFailed: &gs.meaningMemoryLoadFailed,
			progress: gs.meaningProgress, progressDirty: gs.meaningProgressDirty, progressLoadFailed: &gs.meaningProgressLoadFailed,
		}
	}
	return wordTrack{
		overallStats: gs.wordOverallStats, sessionStats: gs.wordSessionStats, sessionDirty: &gs.wordSessionDirty,
		memory: gs.wordMemory, memoryDirty: gs.wordMemoryDirty, memoryLoadFailed: &gs.wordMemoryLoadFailed,
		progress: gs.wordProgress, progressDirty: gs.wordProgressDirty, progressLoadFailed: &gs.wordProgressLoadFailed,
	}
}

// loadWordStats reads kind k's stored hit and miss counts. gs.store must
// not be nil. Must be called under lock (or before the game starts).
func (gs *GameState) loadWordStats(k wordKind) (map[string]store.WordStats, error) {
	if k == meaningKind {
		return gs.store.MeaningStatistics()
	}
	return gs.store.WordStatistics()
}

// saveWordStats stores one reading's counts of kind k. gs.store must not be
// nil. Must be called under lock.
func (gs *GameState) saveWordStats(k wordKind, kana string, correct, miss int) error {
	if k == meaningKind {
		return gs.store.SaveMeaningStats(kana, correct, miss)
	}
	return gs.store.SaveWordStats(kana, correct, miss)
}

// loadTrack loads kind k's stats, memory and progress from gs.store into
// its (empty) maps. Unreadable memory or progress sets its load-failed
// flag, so the stored values are never overwritten. Called from
// NewGameState before the game starts, so no lock is needed.
func (gs *GameState) loadTrack(k wordKind) {
	t := gs.track(k)
	if ws, err := gs.loadWordStats(k); err == nil {
		maps.Copy(t.overallStats, ws)
	}
	if ms, err := gs.store.MemoryStates(k.prefix()); err == nil {
		maps.Copy(t.memory, ms)
	} else {
		*t.memoryLoadFailed = true
	}
	if p, err := gs.store.Progress(k.prefix()); err == nil {
		maps.Copy(t.progress, p)
	} else {
		*t.progressLoadFailed = true
	}
}
```

- [ ] **Step 4: Fields and loading in `fyne/game.go`.** In `GameState`, after the line `wordProgressLoadFailed bool // stored word progress unreadable: never overwrite it`, add (gofmt realigns the block):

```go
	// The same per meaning, keyed by reading (see wordKind and track).
	meaningOverallStats       map[string]store.WordStats
	meaningSessionStats       map[string]store.WordStats
	meaningSessionDirty       bool
	meaningMemory             map[string]memory.State
	meaningMemoryDirty        map[string]bool
	meaningMemoryLoadFailed   bool
	meaningProgress           map[string]store.Progress
	meaningProgressDirty      map[string]bool
	meaningProgressLoadFailed bool
```

In `NewGameState`'s literal, after `wordProgressDirty: map[string]bool{},` add the same six map initialisations as in `newTestState` (Step 1).

In `NewGameState`, replace

```go
		if ws, err := st.WordStatistics(); err == nil {
			gs.wordOverallStats = ws
		}
		if ms, err := st.MemoryStates("word:"); err == nil {
			gs.wordMemory = ms
		} else {
			gs.wordMemoryLoadFailed = true
		}
```

with

```go
		for _, k := range wordKinds {
			gs.loadTrack(k)
		}
```

and delete

```go
		if p, err := st.Progress("word:"); err == nil {
			gs.wordProgress = p
		} else {
			gs.wordProgressLoadFailed = true
		}
```

In `Reset`, delete the two lines

```go
	gs.wordSessionStats = make(map[string]store.WordStats)
	gs.wordSessionDirty = false
```

and replace

```go
	gs.wordOverallStats = make(map[string]store.WordStats)
	if gs.store != nil {
		if ws, err := gs.store.WordStatistics(); err == nil {
			gs.wordOverallStats = ws
		}
	}
```

with

```go
	for _, k := range wordKinds {
		t := gs.track(k)
		clear(t.sessionStats)
		*t.sessionDirty = false
		clear(t.overallStats)
		if gs.store != nil {
			if ws, err := gs.loadWordStats(k); err == nil {
				maps.Copy(t.overallStats, ws)
			}
		}
	}
```

In `saveMemory`, replace the block `if !gs.wordMemoryLoadFailed { … }` with:

```go
	for _, k := range wordKinds {
		t := gs.track(k)
		if *t.memoryLoadFailed {
			continue
		}
		for kana := range t.memoryDirty {
			if err := gs.store.SaveMemoryState(k.prefix()+kana, t.memory[kana]); err == nil {
				delete(t.memoryDirty, kana)
			}
		}
	}
```

and change its doc comment's first line to `// saveMemory writes changed kana memory states and both kinds of word memory.`.

Keep today's behaviour at the three call sites (Task 5 changes them): in `tick` `gs.recordWordMiss(tile.word.Kana, readingKind)`, in `checkAnswer` `gs.recordWordCorrect(tile.word.Kana, readingKind)`, in `wrongInput` `gs.applyWordMistake(tile.word.Kana, readingKind)`.

- [ ] **Step 5: `fyne/learning.go`.** Replace `wordLearned`, `raiseWord` and `lowerWord` with:

```go
// wordLearned reports whether a reading is learned in kind k. Must be
// called under lock.
func (gs *GameState) wordLearned(kana string, k wordKind) bool {
	return learned(gs.track(k).progress[kana].Level)
}
```

```go
// raiseWord records a hit on a reading in kind k. Must be called under
// lock.
func (gs *GameState) raiseWord(kana string, k wordKind) {
	t := gs.track(k)
	t.progress[kana] = hit(t.progress[kana])
	t.progressDirty[kana] = true
}

// lowerWord records a mistake on a reading in kind k. Must be called under
// lock.
func (gs *GameState) lowerWord(kana string, k wordKind) {
	t := gs.track(k)
	t.progress[kana] = mistake(t.progress[kana])
	t.progressDirty[kana] = true
}
```

In `saveProgress` replace the block `if !gs.wordProgressLoadFailed { … }` with:

```go
	for _, k := range wordKinds {
		t := gs.track(k)
		if *t.progressLoadFailed {
			continue
		}
		for kana := range t.progressDirty {
			if err := gs.store.SaveProgress(k.prefix()+kana, t.progress[kana]); err == nil {
				delete(t.progressDirty, kana)
			}
		}
	}
```

and change its doc comment's first line to `// saveProgress writes changed kana progress and both kinds of word progress.`.

- [ ] **Step 6: `fyne/wordgame.go`.** Replace `recordWordCorrect`, `applyWordMistake`, `recordWordMiss` and `mergeWordStats` with:

```go
// recordWordCorrect records a hit on a word in kind k: that kind's session
// stats, memory and level, and the answer streak. Kana stats are not
// touched. Must be called under lock.
func (gs *GameState) recordWordCorrect(kana string, k wordKind) {
	t := gs.track(k)
	stat := t.sessionStats[kana]
	stat.Kana = kana
	stat.CorrectCount++
	t.sessionStats[kana] = stat
	*t.sessionDirty = true

	prev, known := t.memory[kana]
	t.memory[kana] = memory.Correct(prev, known, gs.now())
	t.memoryDirty[kana] = true
	gs.raiseWord(kana, k)

	gs.raiseTempo()
}

// applyWordMistake records a mistake on a word in kind k: session miss,
// memory miss (if known), level down and flag, all of that kind. Its kana
// and its other kind are not touched. Must be called under lock.
func (gs *GameState) applyWordMistake(kana string, k wordKind) {
	t := gs.track(k)
	stat := t.sessionStats[kana]
	stat.Kana = kana
	stat.MissCount++
	t.sessionStats[kana] = stat
	*t.sessionDirty = true

	if prev, known := t.memory[kana]; known {
		t.memory[kana] = memory.Miss(prev, gs.now())
		t.memoryDirty[kana] = true
	}
	gs.lowerWord(kana, k)
}

// recordWordMiss records a word that fell through, in kind k: a mistake
// plus the tempo. Must be called under lock.
func (gs *GameState) recordWordMiss(kana string, k wordKind) {
	gs.applyWordMistake(kana, k)
	gs.lowerTempo()
}

// mergeWordStats writes (baseline + session) word stats of both kinds,
// independently of the kana stats and of each other. Must be called under
// lock.
func (gs *GameState) mergeWordStats() {
	for _, k := range wordKinds {
		gs.mergeWordStatsOf(k)
	}
}

// mergeWordStatsOf writes (baseline + session) stats of kind k. A failed
// baseline read keeps the session dirty so the next merge retries. Must be
// called under lock.
func (gs *GameState) mergeWordStatsOf(k wordKind) {
	t := gs.track(k)
	if !*t.sessionDirty {
		return
	}

	baseline := map[string]store.WordStats{}
	if gs.store != nil {
		stats, err := gs.loadWordStats(k)
		if err != nil {
			return
		}
		baseline = stats
	}

	for kana, session := range t.sessionStats {
		if session.CorrectCount == 0 && session.MissCount == 0 {
			continue
		}
		base := baseline[kana]
		base.Kana = kana
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		if gs.store != nil {
			if err := gs.saveWordStats(k, kana, base.CorrectCount, base.MissCount); err != nil {
				continue
			}
		}
		t.overallStats[kana] = base
		delete(t.sessionStats, kana)
	}

	*t.sessionDirty = false
}
```

In `introducedCounts` change `if gs.wordLearned(kana) {` to `if gs.wordLearned(kana, readingKind) {` (Task 8 makes it follow the mode). Replace `sessionWordCorrect`'s body:

```go
// sessionWordCorrect is the number of word hits this session, reading and
// meaning alike. Must be called under lock.
func (gs *GameState) sessionWordCorrect() int {
	n := 0
	for _, k := range wordKinds {
		for _, st := range gs.track(k).sessionStats {
			n += st.CorrectCount
		}
	}
	return n
}
```

In `fyne/packages.go`, `newestPackageDone`: `if gs.wordLearned(kana) {` → `if gs.wordLearned(kana, readingKind) {` (Task 8 changes it).

- [ ] **Step 7: Run** `go build ./... && go vet ./... && gofmt -l . && go test ./fyne/ -run 'TestWordKind|TestMeaningTrack|TestExistingWordProgress|TestRecordWord|TestWordHit|TestMerge|TestProgressSaved' -timeout 300s` → PASS.

- [ ] **Step 8: Run the whole suite once** `go test ./... -timeout 600s` → PASS (no behaviour changed yet).

- [ ] **Step 9: Commit.**

```bash
git add fyne/wordkind.go fyne/wordkind_test.go fyne/game.go fyne/learning.go fyne/wordgame.go fyne/packages.go fyne/game_test.go fyne/learning_test.go fyne/wordgame_test.go
git commit -m "feat(fyne): track reading and meaning progress per word apart" -m "Every word keeps stats, memory and a learning level per kind. The reading uses the existing word: items and word_stats, the meaning uses meaning: items and meaning_stats. No answer counts for the meaning yet."
```

---

### Task 4: Meaning tiles with a marker

**Files:**
- Modify: `fyne/tile.go`
- Test: `fyne/tile_test.go`

- [ ] **Step 1: Write the failing tests.** Append to `fyne/tile_test.go`:

```go
func TestMeaningTileHasAMarkerBesideTheWord(t *testing.T) {
	test.NewApp()
	e := vocab.Entry{Kana: "き", Units: []string{"き"}}
	reading := newWordTile(e, 400, fontChoice{})
	tile := newMeaningTile(e, 400, fontChoice{})
	if reading.ask != readingKind || reading.marker != nil || len(reading.Objects()) != 3 {
		t.Fatalf("a reading tile asks for the reading and has no marker: ask %v, marker %v", reading.ask, reading.marker)
	}
	if tile.ask != meaningKind || tile.marker == nil || tile.word.Kana != "き" {
		t.Fatalf("meaning tile: ask %v, marker %v", tile.ask, tile.marker)
	}
	if objs := tile.Objects(); len(objs) != 4 || objs[3] != tile.marker {
		t.Fatalf("objects = %d, want shadow, face, text and the marker on top", len(objs))
	}
	if tile.marker.Text != meaningMarkerText || tile.marker.Color != wordTileBorderColor {
		t.Fatalf("marker %q in %v", tile.marker.Text, tile.marker.Color)
	}
	if tile.Width() <= reading.Width() {
		t.Fatalf("meaning tile %.1f should be wider than the reading tile %.1f", tile.Width(), reading.Width())
	}
	tile.Move(fyne.NewPos(40, 20))
	mp, ms := tile.marker.Position(), tile.marker.Size()
	textLeft := tile.text.Position().X
	textRight := textLeft + tile.text.Size().Width
	if mp.X < textRight {
		t.Fatalf("marker at %.1f covers the word ending at %.1f", mp.X, textRight)
	}
	if mp.X+ms.Width > 40+tile.Width() || mp.Y < 20 || mp.Y+ms.Height > 20+tileH {
		t.Fatalf("marker %v %v outside the face at (40, 20) %.1f×%.0f", mp, ms, tile.Width(), tileH)
	}
	if left, right := textLeft-40, 40+tile.Width()-textRight; left-right > 1 || right-left > 1 {
		t.Fatalf("word not centred (left %.1f, right %.1f)", left, right)
	}
}

func TestMeaningTileShrinksToFitField(t *testing.T) {
	test.NewApp()
	tile := newMeaningTile(vocab.Entry{Kana: "コンピューター"}, 150, fontChoice{})
	if tile.Width() > 150 {
		t.Fatalf("width %.1f exceeds the 150 px field", tile.Width())
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestMeaningTile' -timeout 300s` → build failure: `undefined: newMeaningTile`, `tile.ask undefined`, `undefined: meaningMarkerText`.

- [ ] **Step 3: Implement in `fyne/tile.go`.** Add to the first `const (` block:

```go
	meaningMarkerText          = "?" // the sign on a word tile that asks for the meaning
	meaningMarkerSize  float32 = 13
	meaningMarkerInset float32 = 3 // from the face's top and right edges
```

Add two fields to `KanaTile` after `text`:

```go
	marker      *canvas.Text // a meaning tile's marker; nil otherwise
	ask         wordKind     // what a word tile asks for; readingKind for kana tiles
```

Replace `newWordTile` with:

```go
// newWordTile builds a word tile that asks for the word's reading, in the
// kana font kf: a sage face with a border, as wide as the word's shown form
// plus padding, so even a one-unit word (か) does not look like a kana tile.
// A word wider than maxW (the field width) gets a smaller font, down to
// minWordTextSize, so the tile fits.
func newWordTile(e vocab.Entry, maxW float32, kf fontChoice) *KanaTile {
	return buildWordTile(e, maxW, kf, nil)
}

// newMeaningTile builds a word tile that asks for the word's meaning: a word
// tile with a small "?" in its top-right corner, so a one-kana word (き) is
// not taken for a kana tile. The tile is wider by the marker, so the marker
// sits beside the centred word and never covers it; it is drawn in the
// border colour on the tile's own face, so it reads whatever the window
// background.
func newMeaningTile(e vocab.Entry, maxW float32, kf fontChoice) *KanaTile {
	m := canvas.NewText(meaningMarkerText, wordTileBorderColor)
	m.TextSize = meaningMarkerSize
	m.TextStyle = fyne.TextStyle{Bold: true}
	m.Resize(m.MinSize())
	t := buildWordTile(e, maxW, kf, m)
	t.ask = meaningKind
	return t
}

// buildWordTile builds a word tile (see newWordTile) with an optional
// marker, for which the tile is wider by the marker's width.
func buildWordTile(e vocab.Entry, maxW float32, kf fontChoice, marker *canvas.Text) *KanaTile {
	var extra float32
	if marker != nil {
		extra = marker.Size().Width
	}
	text := canvas.NewText(e.Shown(), kanaTextColor)
	text.TextSize = wordTextSize
	setKanaFont(text, kf)
	size := text.MinSize()
	avail := maxW - 2*wordTilePad - extra
	for size.Width > avail && text.TextSize > minWordTextSize {
		text.TextSize--
		size = text.MinSize()
	}
	text.Resize(size)
	w := size.Width + 2*wordTilePad + extra

	shadow := canvas.NewRectangle(tileShadowColor)
	shadow.Resize(fyne.NewSize(w, tileH))

	face := canvas.NewRectangle(wordTileFaceColor)
	face.StrokeColor = wordTileBorderColor
	face.StrokeWidth = 2
	face.Resize(fyne.NewSize(w, tileH))

	word := e
	t := &KanaTile{word: &word, width: w, shadow: shadow, face: face, text: text, marker: marker}
	t.Move(fyne.NewPos(0, 0))
	return t
}
```

In `Move`, append at the end:

```go
	if t.marker != nil {
		ms := t.marker.Size()
		t.marker.Move(fyne.NewPos(pos.X+t.width-ms.Width-meaningMarkerInset, pos.Y+meaningMarkerInset))
	}
```

Replace `Objects`:

```go
// Objects returns the canvas objects for the renderer, shadow first so the
// face renders on top, and a meaning tile's marker last.
func (t *KanaTile) Objects() []fyne.CanvasObject {
	if t.marker != nil {
		return []fyne.CanvasObject{t.shadow, t.face, t.text, t.marker}
	}
	return []fyne.CanvasObject{t.shadow, t.face, t.text}
}
```

Update the `KanaTile` doc comment: `// KanaTile is a falling card rendered as three canvas objects (four for a meaning tile). It carries either a kana or, when word is set, a word; a word tile's kana is unset.`

- [ ] **Step 4: Run** `go build ./... && go vet ./... && gofmt -l . && go test ./fyne/ -run 'Tile' -timeout 300s` → PASS (old tile tests included).

- [ ] **Step 5: Commit.**

```bash
git add fyne/tile.go fyne/tile_test.go
git commit -m "feat(fyne): add meaning tiles with a marker" -m "A word tile that asks for the meaning carries a small question mark beside the word, so a one-kana word is not taken for a kana tile. The tile is wider by the marker, which never covers the kana."
```

---

### Task 5: Answers count for the reading or the meaning; the hint

**Files:**
- Modify: `fyne/wordkind.go` (`answeredKind`, `typedReading`, `readingTyped`), `fyne/feedback.go` (`feedbackHint`, `feedbackHintColor`, `meaningHintText`, `addFeedback`, `showMeaningHint`), `fyne/game.go` (`tick`, `checkAnswer`, `wrongInput`, `answers`)
- Create: `fyne/meaningmode_test.go`
- Test: `fyne/wordgame_test.go` (`TestWordAnsweredByMeaning`)

- [ ] **Step 1: Write the failing tests.** Create `fyne/meaningmode_test.go`:

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"kana/store"
	"kana/vocab"
)

// kasa is a synthetic word with a German and an English meaning.
var kasa = vocab.Word{Kana: "かさ", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"de": {"Regenschirm"}, "en": {"umbrella"}}}

// meaningTileAt is a word tile asking for the meaning at (10, y) for a
// reading in gs.words.
func meaningTileAt(gs *GameState, kana string, y float32) *KanaTile {
	e, ok := gs.words.Lookup(kana)
	if !ok {
		panic("no word " + kana)
	}
	tile := newMeaningTile(e, gs.canvasW, fontChoice{})
	tile.fallSeconds = 20
	tile.Move(fyne.NewPos(10, y))
	return tile
}

func TestReadingTileCountsTheAnswerItGot(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	gs.levelOfferDeclined = true // かさ is the whole source: no level offer may pause the game
	gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 0)}
	gs.checkAnswer("umbrella")
	if len(gs.tiles) != 0 || gs.meaningProgress["かさ"].Level != 1 || gs.meaningSessionStats["かさ"].CorrectCount != 1 {
		t.Fatalf("a meaning on a reading tile is a meaning hit: tiles %d, meaning %+v", len(gs.tiles), gs.meaningProgress["かさ"])
	}
	if _, ok := gs.wordProgress["かさ"]; ok || len(gs.wordSessionStats) != 0 {
		t.Fatal("a meaning hit must leave the reading level alone")
	}
	if got := gs.feedback[0].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("feedback %q: answered by meaning keeps its look", got)
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 0)}
	gs.checkAnswer("kasa")
	if gs.wordProgress["かさ"].Level != 1 || gs.meaningProgress["かさ"].Level != 1 {
		t.Fatalf("the reading raises the reading level only: reading %+v, meaning %+v", gs.wordProgress["かさ"], gs.meaningProgress["かさ"])
	}
}

func TestMeaningTileTakesOnlyTheMeaning(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	tile := meaningTileAt(gs, "かさ", 100)
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("kasa")
	gs.checkAnswer("kasa")
	if len(gs.tiles) != 1 || gs.tiles[0] != tile {
		t.Fatal("the reading must not answer a meaning tile")
	}
	if len(gs.feedback) != 1 || gs.feedback[0].kind != feedbackHint || gs.feedback[0].text.Text != meaningHintText || gs.feedback[0].text.Color != feedbackHintColor {
		t.Fatalf("want one hint label, got %d labels", len(gs.feedback))
	}
	if len(gs.wordProgress) != 0 || len(gs.meaningProgress) != 0 || len(gs.wordSessionStats) != 0 || len(gs.meaningSessionStats) != 0 || gs.score != 0 || gs.answerStreak != 0 {
		t.Fatal("the hint is no hit and no mistake")
	}
	gs.checkAnswer("Umbrella")
	if len(gs.tiles) != 0 || gs.meaningProgress["かさ"].Level != 1 || len(gs.wordProgress) != 0 {
		t.Fatalf("a meaning answers it and raises the meaning level: tiles %d, meaning %+v", len(gs.tiles), gs.meaningProgress["かさ"])
	}
	if got := gs.feedback[len(gs.feedback)-1].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("hit feedback %q", got)
	}
}

func TestMeaningFirstWhenTheInputIsBoth(t *testing.T) {
	sushi := vocab.Word{Kana: "すし", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"en": {"sushi"}}}
	gs := newTestState()
	withWords(gs, sushi)
	gs.levelOfferDeclined = true // すし is the whole source: no level offer may pause the game
	gs.tiles = []*KanaTile{meaningTileAt(gs, "すし", 0)}
	gs.checkAnswer("sushi")
	if len(gs.tiles) != 0 || gs.meaningProgress["すし"].Level != 1 || len(gs.wordProgress) != 0 {
		t.Fatal("meaning mode: an input that is reading and meaning is a meaning hit")
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "すし", 0)}
	gs.checkAnswer("sushi")
	if gs.wordProgress["すし"].Level != 1 || gs.meaningProgress["すし"].Level != 1 {
		t.Fatal("reading mode: an input that is both stays a reading hit")
	}
}

func TestMeaningTileMissLowersOnlyTheMeaningLevel(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	setWordLevel(gs, "かさ", 3)
	gs.meaningProgress["かさ"] = store.Progress{Level: 4}
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 599)}
	gs.tick()
	if len(gs.tiles) != 0 || gs.missed != 1 {
		t.Fatalf("tiles %d, missed %d", len(gs.tiles), gs.missed)
	}
	if gs.meaningProgress["かさ"] != (store.Progress{Level: 2, Flagged: true}) || gs.meaningSessionStats["かさ"].MissCount != 1 {
		t.Fatalf("meaning %+v, stats %+v", gs.meaningProgress["かさ"], gs.meaningSessionStats["かさ"])
	}
	if gs.wordProgress["かさ"] != (store.Progress{Level: 3}) || len(gs.wordSessionStats) != 0 {
		t.Fatalf("reading must stay: %+v", gs.wordProgress["かさ"])
	}
	if got := gs.feedback[0].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("miss feedback %q", got)
	}
}

func TestWrongInputOnAMeaningTileLowersTheMeaningLevel(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 100)}
	gs.checkAnswer("parasol")
	if len(gs.tiles) != 1 || gs.meaningProgress["かさ"] != (store.Progress{Flagged: true}) || len(gs.wordProgress) != 0 {
		t.Fatalf("meaning %+v, reading %v", gs.meaningProgress["かさ"], gs.wordProgress)
	}
}

func TestLateReadingForAFallenMeaningTileIsNoMistake(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	kana := tileAt("あ", "a", 100)
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 599), kana}
	gs.tick() // かさ falls through
	gs.checkAnswer("kasa")
	if len(gs.tiles) != 1 || gs.kanaProgress["あ"].Flagged {
		t.Fatal("the reading of a meaning tile that just fell is a late Enter, not a mistake on あ")
	}
	if gs.feedback[len(gs.feedback)-1].kind == feedbackHint {
		t.Fatal("no hint for a tile that is gone")
	}
}
```

In `fyne/wordgame_test.go`, `TestWordAnsweredByMeaning`, replace

```go
		if len(gs.tiles) != 0 || gs.wordSessionStats["かさ"].CorrectCount != 1 {
```

with

```go
		hits := gs.wordSessionStats["かさ"].CorrectCount + gs.meaningSessionStats["かさ"].CorrectCount
		if len(gs.tiles) != 0 || hits != 1 {
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestReadingTileCounts|TestMeaningTile|TestMeaningFirst|TestWrongInputOnAMeaning|TestLateReading' -timeout 300s` → build failure: `undefined: feedbackHint`, `undefined: meaningHintText`, `undefined: feedbackHintColor`.

- [ ] **Step 3: Feedback in `fyne/feedback.go`.** Change the kinds to:

```go
const (
	feedbackHit  feedbackKind = iota
	feedbackMiss
	feedbackHint // "type the meaning": no hit, no miss
)
```

After `var feedbackHitColor = …` add:

```go
// feedbackHintColor is the Warm Paper accent, so a hint looks neither like a
// hit nor like a miss.
var feedbackHintColor = color.RGBA{R: 0x8b, G: 0x5e, B: 0x3c, A: 0xff}

// meaningHintText is shown at a meaning tile whose reading was typed.
const meaningHintText = "type the meaning"
```

In `addFeedback`, replace

```go
	var c color.Color = feedbackHitColor
	if kind == feedbackMiss {
		c = cellMissedColor
	}
```

with

```go
	var c color.Color = feedbackHitColor
	switch kind {
	case feedbackMiss:
		c = cellMissedColor
	case feedbackHint:
		c = feedbackHintColor
	}
```

and change its doc comment's second sentence to `A hit or hint label sits at pos (the tile's last position), a miss label at the bottom of the field in the error colour; all are kept inside the field.` Append to `fyne/feedback.go`:

```go
// showMeaningHint shows meaningHintText at the tile. An earlier hint still
// on the field goes, so repeated Enters do not stack labels. Must be called
// under lock.
func (gs *GameState) showMeaningHint(tile *KanaTile) {
	kept := gs.feedback[:0]
	for _, f := range gs.feedback {
		if f.kind != feedbackHint {
			kept = append(kept, f)
		}
	}
	gs.feedback = kept
	gs.addFeedback(feedbackHint, meaningHintText, tile.pos)
	gs.buildSnapshot()
}
```

- [ ] **Step 4: Helpers in `fyne/wordkind.go`.** Append:

```go
// answeredKind is what a hit on a word tile counts for: everything on a
// tile asking for the meaning; on a tile asking for the reading, the
// reading when input is one of its readings, else (answered by meaning) the
// meaning.
func answeredKind(tile *KanaTile, input string) wordKind {
	if tile.ask == meaningKind || !tile.word.ReadingMatches(input) {
		return meaningKind
	}
	return readingKind
}

// typedReading reports whether input is the reading of a word tile that
// asks for its meaning.
func typedReading(tile *KanaTile, input string) bool {
	return tile.word != nil && tile.ask == meaningKind && tile.word.ReadingMatches(input)
}

// readingTyped returns the lowest falling tile that asks for its meaning
// and whose reading input is, or nil. Must be called under lock.
func (gs *GameState) readingTyped(input string) *KanaTile {
	var best *KanaTile
	for _, t := range gs.tiles {
		if typedReading(t, input) && (best == nil || t.pos.Y > best.pos.Y) {
			best = t
		}
	}
	return best
}
```

- [ ] **Step 5: `fyne/game.go`.** In `tick`, `gs.recordWordMiss(tile.word.Kana, readingKind)` → `gs.recordWordMiss(tile.word.Kana, tile.ask)`.

Replace `answers`:

```go
// answers reports whether input answers the tile: a kana by its romaji; a
// word by its meaning (see vocab.Meanings.MeaningMatches) or, if it asks
// for its reading, by one of its readings (see vocab.Entry.ReadingMatches).
// Must be called under lock.
func (gs *GameState) answers(tile *KanaTile, input string) bool {
	if tile.word != nil {
		if tile.word.Meanings.MeaningMatches(gs.meaningLang, input) {
			return true
		}
		return tile.ask == readingKind && tile.word.ReadingMatches(input)
	}
	return gs.charSet.Matches(tile.kana.Char, input)
}
```

In `wrongInput`, replace the late-Enter condition and the word mistake:

```go
	if gs.lastFallen != nil && gs.now().Sub(gs.lastFallenAt) < lateAnswerWindow &&
		(gs.answers(gs.lastFallen, input) || typedReading(gs.lastFallen, input)) {
		return
	}
	tile := gs.tiles[0]
	if tile.word != nil {
		gs.applyWordMistake(tile.word.Kana, tile.ask)
	} else {
```

(and add to its doc comment: `The mistake counts for the tile's question.`).

Replace `checkAnswer` (doc comment and body) with:

```go
// checkAnswer removes the lowest tile that input answers (see answers): a
// word hit counts for the reading or the meaning (see answeredKind). If no
// tile is answered but input is the reading of a tile asking for its
// meaning, the hint "type the meaning" is shown there, which is no mistake.
// Any other input that answers no tile may count as a mistake (see
// wrongInput). Acquires the lock itself; releases before triggering
// canvas.Refresh() so the UI updates immediately.
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
	refresh := matched
	if matched {
		tile := gs.tiles[best]
		gs.tiles = append(gs.tiles[:best], gs.tiles[best+1:]...)
		announced := false
		if tile.word != nil {
			gs.score += 10 * len(tile.word.Units) * gs.pointsFactor()
			kind := answeredKind(tile, input)
			gs.recordWordCorrect(tile.word.Kana, kind)
			gs.addFeedback(feedbackHit, hitFeedbackText(*tile.word, gs.meaningLang, kind == meaningKind), tile.pos)
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
		gs.maybeOfferPackage()
		gs.buildSnapshot()
	} else if hinted := gs.readingTyped(input); hinted != nil {
		gs.showMeaningHint(hinted)
		refresh = true
	} else {
		gs.wrongInput(input)
	}
	canvas := gs.canvas
	gs.mu.Unlock()

	if refresh && canvas != nil {
		canvas.Refresh()
	}
}
```

- [ ] **Step 6: Run** `go build ./... && go vet ./... && gofmt -l . && go test ./fyne/ -run 'TestReadingTileCounts|TestMeaningTile|TestMeaningFirst|TestWrongInput|TestLateReading|TestWordAnsweredByMeaning|TestLowerTileWins|TestWordMiss|TestMeaningLang' -timeout 300s` → PASS.

- [ ] **Step 7: Run the whole package** `go test ./fyne/ -timeout 600s` → PASS. If another existing test counted a meaning answer on a word tile as a reading hit (look for tests that call `checkAnswer` with a meaning and then assert `wordSessionStats` or `wordProgress`; only `TestWordAnsweredByMeaning` is known, handled in Step 1), change that test to expect the meaning track — that is the spec's change — not the code.

- [ ] **Step 8: Commit.**

```bash
git add fyne/wordkind.go fyne/feedback.go fyne/game.go fyne/meaningmode_test.go fyne/wordgame_test.go
git commit -m "feat(fyne): count answers for the reading or the meaning" -m "A meaning tile takes only a meaning; typing its reading shows the hint 'type the meaning' and costs nothing. A meaning typed on a reading tile now raises the meaning level, and a missed tile lowers the level of its own question."
```

---

### Task 6: The mode — new word tiles ask for the meaning

**Files:**
- Modify: `fyne/game.go` (`GameState.wordsAsk`, `NewGameState`, `spawnKana`), `fyne/packages.go` (`wordSettings`, `applyWordSettings`)
- Test: `fyne/meaningmode_test.go`

- [ ] **Step 1: Write the failing tests.** Append to `fyne/meaningmode_test.go` (add `"fyne.io/fyne/v2/test"` to its imports):

```go
func TestWordsAskIsStored(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	if gs.wordsAsk != readingKind {
		t.Fatalf("default wordsAsk %v, want reading", gs.wordsAsk)
	}
	gs.applyWordSettings(wordSettings{source: gs.wordSource, level: gs.wordLevel, packageSize: gs.packageSize, ask: meaningKind})
	if a, _ := st.WordsAsk(); a != store.WordsAskMeaning || gs.wordsAsk != meaningKind {
		t.Fatalf("stored %q, state %v", a, gs.wordsAsk)
	}
	if again := NewGameState(st); again.wordsAsk != meaningKind {
		t.Fatalf("loaded %v, want meaning", again.wordsAsk)
	}
}

func TestMeaningModeSpawnsMeaningTiles(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	withWords(gs, bw("あい", 5, "love"))
	gs.wordsAsk = meaningKind
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].word == nil || gs.tiles[0].ask != meaningKind || gs.tiles[0].marker == nil {
		t.Fatal("in meaning mode a word spawns as a meaning tile")
	}

	gs = newTestState() // no words: kana fall
	selectRows(gs, "vowels")
	gs.wordsAsk = meaningKind
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].word != nil || gs.tiles[0].ask != readingKind || gs.tiles[0].marker != nil {
		t.Fatal("kana tiles still ask for the romaji")
	}
}

func TestChangingWordsAskLeavesFallingTiles(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	learnKana(gs, "あ", "い", "う", "え", "お")
	withWords(gs, bw("あい", 5, "love"), vocab.Word{Kana: "あお", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{}})
	reading := wordTileAt(gs, "あお", 100) // no meaning at all
	meaning := meaningTileAt(gs, "あい", 50)
	gs.tiles = []*KanaTile{reading, meaning}
	// A package size change as well, so dropInactiveTiles runs.
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 3, ask: meaningKind})
	if len(gs.tiles) != 2 || gs.tiles[0] != reading || gs.tiles[1] != meaning || reading.ask != readingKind {
		t.Fatal("switching to meaning mode keeps falling tiles and their questions")
	}
	gs.checkAnswer("ao")
	if len(gs.tiles) != 1 || gs.wordProgress["あお"].Level != 1 {
		t.Fatal("the falling reading tile still takes its reading")
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5, ask: readingKind})
	if len(gs.tiles) != 1 || gs.tiles[0] != meaning || meaning.ask != meaningKind {
		t.Fatal("switching back keeps the meaning tile")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestWordsAsk|TestMeaningModeSpawns|TestChangingWordsAsk' -timeout 300s` → build failure: `unknown field ask in struct literal of type wordSettings`, `gs.wordsAsk undefined`.

- [ ] **Step 3: Implement.** In `fyne/game.go`, `GameState`, after `wordsOnly bool …` add:

```go
	wordsAsk             wordKind            // what new word tiles ask for: the reading or the meaning
```

In `NewGameState`, after the `st.WordsOnly()` block add:

```go
		if a, err := st.WordsAsk(); err == nil {
			gs.wordsAsk = kindFromSetting(a)
		}
```

In `spawnKana`, replace `tile = newWordTile(entry, gs.canvasW, gs.kanaFont)` with:

```go
		if gs.wordsAsk == meaningKind {
			tile = newMeaningTile(entry, gs.canvasW, gs.kanaFont)
		} else {
			tile = newWordTile(entry, gs.canvasW, gs.kanaFont)
		}
```

and add to its doc comment: `A word tile asks for what gs.wordsAsk says.`

In `fyne/packages.go`, `wordSettings`, add after `meaningLang`:

```go
	ask         wordKind // what word tiles ask for
```

In `applyWordSettings`, after the `meaningLang` block add:

```go
	if ws.ask != gs.wordsAsk {
		// Only new tiles follow; falling tiles keep their question, and
		// the playable words do not depend on it, so nothing is dropped.
		gs.wordsAsk = ws.ask
		if gs.store != nil {
			_ = gs.store.SaveWordsAsk(ws.ask.setting())
		}
	}
```

and add to its doc comment: `A new "Words ask for" applies to tiles spawned afterwards.`

- [ ] **Step 4: Run** `go build ./... && go vet ./... && gofmt -l . && go test ./fyne/ -run 'TestWordsAsk|TestMeaningModeSpawns|TestChangingWordsAsk|TestSettings|TestMeaningLang' -timeout 300s` → PASS.

- [ ] **Step 5: Commit.**

```bash
git add fyne/game.go fyne/packages.go fyne/meaningmode_test.go
git commit -m "feat(fyne): ask word tiles for their meaning in meaning mode" -m "The words-ask setting is loaded at start and saved with the Words settings. New word tiles follow it; falling tiles keep their question."
```

---

### Task 7: Eligible words and selection by the meaning level

**Files:**
- Modify: `fyne/wordkind.go` (`askable`, `askableWords`), `fyne/wordgame.go` (`pickWord`, `wordsStatus`), `fyne/focus.go` (`pickReading`), `fyne/words.go` (`WordsStatus.NoMeanings`, `wordsOnlyReason`)
- Test: `fyne/meaningmode_test.go`, `fyne/words_test.go`

- [ ] **Step 1: Write the failing tests.** Append to `fyne/meaningmode_test.go` (add `"math/rand"`, `"time"` and `"kana/memory"` to its imports):

```go
func TestMeaningModeAsksOnlyWordsWithAMeaning(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.rnd = rand.New(rand.NewSource(11)).Float64
	withWords(gs,
		bw("あい", 5, "love"),
		vocab.Word{Kana: "あお", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{}},
		vocab.Word{Kana: "あう", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"en": {"(to meet)"}}},
		vocab.Word{Kana: "いえ", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"de": {"Haus"}}},
	)
	if got := readingsOf(gs.askableWords()); !equalIDs(got, []string{"あい", "あう", "あお", "いえ"}) {
		t.Fatalf("reading mode asks every playable word, got %v", got)
	}
	gs.wordsAsk = meaningKind
	if got := readingsOf(gs.askableWords()); !equalIDs(got, []string{"あい", "いえ"}) {
		t.Fatalf("meaning mode: %v, want あい and いえ (a meaning only in brackets or none does not count)", got)
	}
	for i := 0; i < 300; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if k := gs.tiles[0].word.Kana; k != "あい" && k != "いえ" {
			t.Fatalf("spawned %s in meaning mode", k)
		}
	}
	if got := playableKana(gs); len(got) != 4 {
		t.Fatalf("playable %v: words without a meaning stay playable", got)
	}
}

func TestMeaningModeWithoutMeaningsFallsBackToKana(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	withWords(gs, vocab.Word{Kana: "あお", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{}})
	if got := formatWordsLine(gs.wordsStatus()); got != "Words: JLPT N5 finished" {
		t.Fatalf("reading mode line %q", got)
	}
	gs.wordsAsk = meaningKind
	gs.spawnKana()
	if gs.tiles[0].word != nil {
		t.Fatal("no word with a meaning: a kana falls")
	}
	if got := formatWordsLine(gs.wordsStatus()); got != "Words only: no word has a meaning to ask for" {
		t.Fatalf("meaning mode line %q", got)
	}
}

func TestMeaningModePicksByMeaningLevel(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.focusWeak = true
	gs.wordsAsk = meaningKind
	gs.rnd = rand.New(rand.NewSource(5)).Float64
	words := vowelWords(10)
	withWords(gs, words...)
	meaningWeak, readingWeak := words[3].Kana, words[6].Kana
	gs.meaningProgress[meaningWeak] = store.Progress{Level: 2, Flagged: true}
	gs.wordProgress[readingWeak] = store.Progress{Level: 2, Flagged: true}
	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		gs.tiles = nil
		gs.spawnKana()
		counts[gs.tiles[0].word.Kana]++
	}
	if n := counts[meaningWeak]; n < 480 || n > 640 {
		t.Fatalf("slipped meaning in %d/1000 spawns, want about 550", n)
	}
	if n := counts[readingWeak]; n > 150 {
		t.Fatalf("a slipped reading is no weak item in meaning mode, got %d/1000", n)
	}
}

func TestMeaningModePicksByMeaningMemory(t *testing.T) {
	gs := newTestState()
	now := time.Unix(1_700_000_000, 0)
	gs.now = func() time.Time { return now }
	gs.rnd = rand.New(rand.NewSource(9)).Float64
	fresh := memory.State{Stability: 100, LastSeen: now}
	missed := memory.Miss(fresh, now) // half forgotten
	gs.wordMemory["あい"], gs.wordMemory["あお"] = missed, fresh
	gs.meaningMemory["あい"], gs.meaningMemory["あお"] = fresh, missed
	count := func() int {
		n := 0
		for i := 0; i < 1000; i++ {
			if gs.pickReading([]string{"あい", "あお"}) == "あお" {
				n++
			}
		}
		return n
	}
	if n := count(); n > 200 {
		t.Fatalf("reading mode picked あお %d/1000; its reading is fresh", n)
	}
	gs.wordsAsk = meaningKind
	if n := count(); n < 800 {
		t.Fatalf("meaning mode picked あお %d/1000; its meaning was just missed", n)
	}
}
```

In `fyne/words_test.go`, `TestFormatWordsLine`, add after the `{"mixed, empty lessons", …}` case:

```go
		{"words only, no meanings", WordsStatus{WordsOnly: true, KanaFallback: true, NoMeanings: true, Introduced: 2, Source: "minna"}, "Words only: no word has a meaning to ask for"},
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestMeaningModeAsks|TestMeaningModeWithout|TestMeaningModePicks|TestFormatWordsLine' -timeout 300s` → build failure: `gs.askableWords undefined`, `unknown field NoMeanings`.

- [ ] **Step 3: Implement.** Append to `fyne/wordkind.go` (add `"kana/vocab"` to its imports):

```go
// askable reports whether new tiles may ask about e: always in reading
// mode; in meaning mode only with a meaning to type (see
// vocab.Meanings.Answerable). Must be called under lock.
func (gs *GameState) askable(e vocab.Entry) bool {
	return gs.wordsAsk != meaningKind || e.Meanings.Answerable(gs.meaningLang)
}

// askableWords returns the playable words new tiles may ask about (see
// askable), sorted by reading. In reading mode it is playableWords' cached
// slice; do not modify it. Must be called under lock.
func (gs *GameState) askableWords() []vocab.Entry {
	playable := gs.playableWords()
	if gs.wordsAsk != meaningKind {
		return playable
	}
	var out []vocab.Entry
	for _, e := range playable {
		if gs.askable(e) {
			out = append(out, e)
		}
	}
	return out
}
```

In `fyne/wordgame.go`, `pickWord`: `playable := gs.playableWords()` → `playable := gs.askableWords()`, and in its doc comment "With no playable word" → "With no askable word (see askableWords)".

In `wordsStatus`, replace

```go
	ws.KanaFallback = gs.wordsOnly && len(gs.playableWords()) == 0 && !canCome
```

with

```go
	askable := gs.askableWords()
	ws.KanaFallback = gs.wordsOnly && len(askable) == 0 && !canCome
	ws.NoMeanings = len(askable) == 0 && len(gs.playableWords()) > 0
```

In `fyne/focus.go`, replace `pickReading`:

```go
// pickReading picks the next word reading from candidates, like pickKana
// over the progress and memory of what new tiles ask for (gs.wordsAsk).
// Must be called under lock.
func (gs *GameState) pickReading(candidates []string) string {
	t := gs.track(gs.wordsAsk)
	if c, ok := gs.pickFocus(candidates, t.progress); ok {
		return c
	}
	return memory.Pick(candidates, t.memory, gs.now(), gs.rnd)
}
```

In `fyne/words.go`, `WordsStatus`, add after `Empty`:

```go
	NoMeanings   bool // meaning mode: words can fall, but none has a meaning to ask for
```

and in `wordsOnlyReason`, before the final `return`:

```go
	case ws.NoMeanings:
		return "no word has a meaning to ask for"
```

- [ ] **Step 4: Run** `go build ./... && go vet ./... && gofmt -l . && go test ./fyne/ -run 'TestMeaningMode|TestFormatWordsLine|TestFocus|TestWordShare|TestWordsOnly' -timeout 300s` → PASS.

- [ ] **Step 5: Commit.**

```bash
git add fyne/wordkind.go fyne/wordgame.go fyne/focus.go fyne/words.go fyne/meaningmode_test.go fyne/words_test.go
git commit -m "feat(fyne): pick meaning tiles by meaning level and memory" -m "In meaning mode only words with a meaning to type fall, and the pick and the focus on weak items use the meaning progress. Words-only mode says when no word has a meaning to ask for."
```

---

### Task 8: Package rhythm, status line and summary by the asked levels

**Files:**
- Modify: `fyne/packages.go` (`newestPackageDone`), `fyne/wordgame.go` (`introducedCounts`, `wordsSummary`, `wordsStatus`), `fyne/words.go` (`WordsStatus.Meaning`, `formatWordsLine`)
- Test: `fyne/meaningmode_test.go`, `fyne/words_test.go`

- [ ] **Step 1: Write the failing tests.** Append to `fyne/meaningmode_test.go`:

```go
func TestNewestPackageDoneByTheAskedLevels(t *testing.T) {
	gs := packageReady(40)
	pkg := introduceFirst(gs, 5)
	for _, kana := range pkg {
		setWordLevel(gs, kana, learnedLevel)
	}
	gs.wordsAsk = meaningKind
	for _, kana := range pkg[:3] {
		gs.meaningProgress[kana] = store.Progress{Level: learnedLevel}
	}
	if gs.newestPackageDone() {
		t.Fatal("meaning mode: 3 of 5 meanings learned is not done, whatever the readings")
	}
	gs.meaningProgress[pkg[3]] = store.Progress{Level: learnedLevel}
	if !gs.newestPackageDone() {
		t.Fatal("meaning mode: 4 of 5 meanings learned is done")
	}
	gs.wordsAsk = readingKind
	for _, kana := range pkg {
		delete(gs.wordProgress, kana)
	}
	if gs.newestPackageDone() {
		t.Fatal("reading mode goes by the reading levels")
	}
}

func TestWordsLineCountsTheAskedLevels(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	learnKana(gs, "あ", "い", "う", "え", "お")
	withWords(gs,
		bw("あい", 5, "love"),
		vocab.Word{Kana: "あお", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{}},
		bw("いえ", 5, "house"),
		bw("うえ", 5, "above"),
	)
	delete(gs.introduced, "うえ") // a package can still come, so the line shows counts
	gs.invalidatePlayable()
	setWordLevel(gs, "あい", learnedLevel)
	setWordLevel(gs, "あお", learnedLevel)
	gs.meaningProgress["いえ"] = store.Progress{Level: learnedLevel}

	if got := formatWordsLine(gs.wordsStatus()); got != "Words: 3 introduced · 2 learned · JLPT N5" {
		t.Fatalf("reading mode line %q", got)
	}
	if got := gs.wordsSummary(); got != "Introduced: 3 · learned: 2 · left in source: 1" {
		t.Fatalf("reading mode summary %q", got)
	}
	gs.wordsAsk = meaningKind
	if got := formatWordsLine(gs.wordsStatus()); got != "Meanings: 2 introduced · 1 learned · JLPT N5" {
		t.Fatalf("meaning mode line %q (あお has no meaning)", got)
	}
	if got := gs.wordsSummary(); got != "Introduced: 2 · meanings learned: 1 · left in source: 1" {
		t.Fatalf("meaning mode summary %q", got)
	}
}
```

In `fyne/words_test.go`, `TestFormatWordsLine`, add after the case added in Task 7:

```go
		{"meaning counts", WordsStatus{Meaning: true, Introduced: 25, Learned: 3, Source: "JLPT N5", Level: 5}, "Meanings: 25 introduced · 3 learned · JLPT N5"},
		{"meaning, waiting for kana", WordsStatus{Meaning: true, Introduced: 18, Learned: 5, Waiting: 15, Source: "minna"}, "Meanings: 18 introduced · 5 learned · 15 waiting for kana · minna"},
		{"meaning, needs kana", WordsStatus{Meaning: true, Next: wordsNeedKana, Introduced: 5, Source: "JLPT N5", Level: 5}, "Words: learn more kana for new words"},
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestNewestPackageDoneByTheAsked|TestWordsLineCountsTheAsked|TestFormatWordsLine' -timeout 300s` → build failure: `unknown field Meaning in struct literal of type WordsStatus`.

- [ ] **Step 3: Implement.** In `fyne/packages.go`, replace `newestPackageDone`:

```go
// newestPackageDone reports whether the newest package is learned well
// enough for the next one: at least ceil(0.8 × p) of its p readings that
// can fall now and that new tiles may ask about (see askable) are learned
// in what tiles ask for (gs.wordsAsk). With none of them (a script switched
// off, a list gone, no meanings) it counts as done. Must be called under
// lock.
func (gs *GameState) newestPackageDone() bool {
	playable, learnedN := 0, 0
	for _, kana := range gs.newestPackage {
		e, ok := gs.words.Lookup(kana)
		if !ok || !gs.unitsFit(e.Units) || !gs.askable(e) {
			continue
		}
		playable++
		if gs.wordLearned(kana, gs.wordsAsk) {
			learnedN++
		}
	}
	return playable == 0 || learnedN*5 >= playable*4
}
```

In `fyne/wordgame.go`, replace `introducedCounts`:

```go
// introducedCounts counts the introduced readings in the word index that
// new tiles may ask about (see askable: in meaning mode those with a
// meaning), those of them that are learned in what tiles ask for, and, in
// mixed mode, the practised ones that wait for a kana to be learned before
// they can fall; words-only mode does not wait for kana. Must be called
// under lock.
func (gs *GameState) introducedCounts() (introduced, learnedN, waiting int) {
	src, found := gs.source()
	for kana := range gs.introduced {
		e, ok := gs.words.Lookup(kana)
		if !ok || !gs.askable(e) {
			continue
		}
		introduced++
		if gs.wordLearned(kana, gs.wordsAsk) {
			learnedN++
		}
		if !gs.wordsOnly && gs.practised(e, src, found) && !gs.unitsKnown(e.Units) {
			waiting++
		}
	}
	return introduced, learnedN, waiting
}
```

In `wordsSummary`, replace `s := fmt.Sprintf("Introduced: %d · learned: %d", introduced, learnedN)` with:

```go
	learnedLabel := "learned"
	if gs.wordsAsk == meaningKind {
		learnedLabel = "meanings learned"
	}
	s := fmt.Sprintf("Introduced: %d · %s: %d", introduced, learnedLabel, learnedN)
```

and change its doc comment's example to `"Introduced: 25 · learned: 18 · waiting for kana: 4 · left in source: 101" ("meanings learned" in meaning mode)`.

In `wordsStatus`, add `Meaning: gs.wordsAsk == meaningKind,` to the `WordsStatus{…}` literal.

In `fyne/words.go`, `WordsStatus`, add after `WordsOnly`:

```go
	Meaning      bool // meaning mode: the counts are of meaning levels
```

In `formatWordsLine`, before the `switch` add

```go
	label := "Words"
	if ws.Meaning {
		label = "Meanings"
	}
```

and replace the last two cases with:

```go
	case ws.Waiting > 0:
		s = fmt.Sprintf("%s: %d introduced · %d learned · %d waiting for kana · %s", label, ws.Introduced, ws.Learned, ws.Waiting, ws.Source)
	default:
		s = fmt.Sprintf("%s: %d introduced · %d learned · %s", label, ws.Introduced, ws.Learned, ws.Source)
```

Add to its doc comment: `In meaning mode the counts line starts with "Meanings:".`

- [ ] **Step 4: Run** `go build ./... && go vet ./... && gofmt -l . && go test ./fyne/ -run 'TestNewestPackageDone|TestWordsLineCounts|TestFormatWordsLine|TestNextPackage|TestPackageRule|TestSettingsFormCarriesValues|TestWordsOnly' -timeout 300s` → PASS.

- [ ] **Step 5: Commit.**

```bash
git add fyne/packages.go fyne/wordgame.go fyne/words.go fyne/meaningmode_test.go fyne/words_test.go
git commit -m "feat(fyne): judge packages and the words line by the asked levels" -m "In meaning mode the newest package is done when 80 % of its meanings are learned, and the status line counts meanings: 'Meanings: N introduced · N learned'. Words without a meaning are left out of both."
```

---

### Task 9: "Words ask for" in the settings dialog

**Files:**
- Modify: `fyne/settings.go` (`settingsState`, `gs.settingsState`, `settingsForm`, labels, `newWordsTab`, `askedKind`, `words`)
- Test: `fyne/settings_test.go`

- [ ] **Step 1: Write the failing test.** Append to `fyne/settings_test.go`:

```go
func TestWordsAskInTheWordsTab(t *testing.T) {
	test.NewApp()
	gs := settingsGame()
	f := newSettingsForm(gs.settingsState(), nil)
	if !equalIDs(f.wordsAsk.Options, []string{"Reading", "Meaning"}) || f.wordsAsk.Selected != "Reading" {
		t.Fatalf("options %v, selected %q", f.wordsAsk.Options, f.wordsAsk.Selected)
	}
	if !labelContains(f.wordsTab.Content, "Words ask for") {
		t.Fatal("the Words tab needs the label 'Words ask for'")
	}
	if f.words().ask != readingKind {
		t.Fatal("Reading gives readingKind")
	}
	f.wordsAsk.SetSelected("Meaning")
	f.apply(gs)
	if gs.wordsAsk != meaningKind {
		t.Fatalf("applied %v, want meaning", gs.wordsAsk)
	}
	if f := newSettingsForm(gs.settingsState(), nil); f.wordsAsk.Selected != "Meaning" {
		t.Fatalf("reopened: %q", f.wordsAsk.Selected)
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run TestWordsAskInTheWordsTab -timeout 300s` → build failure: `f.wordsAsk undefined`.

- [ ] **Step 3: Implement in `fyne/settings.go`.** In `settingsState`, after `wordsOnly bool` add `wordsAsk      wordKind`. In `gs.settingsState()`'s literal, after `wordsOnly: gs.wordsOnly,` add `wordsAsk:      gs.wordsAsk,`.

After the "Tiles" choices const block add:

```go
// The Words tab's "Words ask for" choices.
const (
	askReadingLabel = "Reading"
	askMeaningLabel = "Meaning"
)
```

In `settingsForm`, after `tiles *widget.RadioGroup …` add:

```go
	wordsAsk      *widget.RadioGroup  // askReadingLabel or askMeaningLabel
```

In `newWordsTab`, after the `f.tiles` setup block add:

```go
	f.wordsAsk = widget.NewRadioGroup([]string{askReadingLabel, askMeaningLabel}, nil)
	f.wordsAsk.Horizontal = true
	f.wordsAsk.Required = true
	f.wordsAsk.SetSelected(askReadingLabel)
	if st.wordsAsk == meaningKind {
		f.wordsAsk.SetSelected(askMeaningLabel)
	}
```

and in its returned `container.NewVBox(`, after `f.tiles,` add:

```go
		widget.NewLabelWithStyle("Words ask for", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.wordsAsk,
```

Change its doc comment's first line to `// newWordsTab holds the tiles (kana and words, or words only), what word tiles ask for (reading or meaning), where new`.

After `wordSource()` add:

```go
// askedKind is what the Words tab says word tiles ask for.
func (f *settingsForm) askedKind() wordKind {
	if f.wordsAsk.Selected == askMeaningLabel {
		return meaningKind
	}
	return readingKind
}
```

In `words()`, add `ask: f.askedKind(),` to the returned `wordSettings` literal.

- [ ] **Step 4: Run** `go build ./... && go vet ./... && gofmt -l . && go test ./fyne/ -run 'TestSettings|TestWordsAsk|TestLessonBox|TestWordsTab|TestNoLessonBox' -timeout 300s` → PASS (`TestSettingsFormCarriesValues` still matches: `ask` is `readingKind`, the zero value).

- [ ] **Step 5: Commit.**

```bash
git add fyne/settings.go fyne/settings_test.go
git commit -m "feat(fyne): choose what word tiles ask for in the settings" -m "The Words tab gets 'Words ask for: Reading / Meaning' under the tiles choice, saved with the other word settings. Reading stays the default."
```

---

### Task 10: README — the meaning mode for players

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Core Gameplay.** After the "**Words only**" bullet (ending "…and the status panel says why."), insert:

```markdown
- **Meaning mode**: Settings → Words → "Words ask for: Meaning" makes word tiles ask for the
  word's meaning instead of its reading. They carry a small "?" in the corner, so a one-kana
  word such as き is not taken for a kana tile; kana tiles still ask for the romaji. Type a
  meaning in the meaning language or in English, as for any word; typing the reading instead
  shows "type the meaning" at the tile and costs nothing. Every word has a reading level and a
  meaning level of its own: in meaning mode only meanings count and a missed tile lowers its
  meaning level; in reading mode the reading raises the reading level, and a meaning is still
  accepted and raises the meaning level. In meaning mode the choice of tiles, "Focus on weak
  items", the status line ("Meanings: 12 introduced · 4 learned · …") and when the next word
  package comes go by the meaning levels, and words without a meaning do not fall. Together
  with "Words only" it is a pure meaning drill. A change applies to new tiles; tiles already
  falling keep their question. Progress from before counts as reading progress; all meaning
  levels start at 0, and "Already known" in a package intro counts for the reading only.
```

- [ ] **Step 2: Customization.** In the "**Word packages**" bullet, change `(3, 5 or 10) and the meaning language` to `(3, 5 or 10), what word tiles ask for ("Words ask for": Reading or Meaning) and the meaning language`.

- [ ] **Step 3: Architecture.** In the `fyne/` list, after the `tile.go` line, change it and add a line:

```markdown
- `tile.go`: `KanaTile` — shadow + face + text canvas objects, for a kana or a word; a meaning tile adds its "?" marker
- `wordkind.go`: reading and meaning as two kinds of word progress (`wordKind`, `track`), which answer counts for which, and the words a meaning tile may ask about
```

In the `settings.go` line, change `Words (tiles, source, …` to `Words (tiles, words ask for, source, …`. In the `feedback.go` line, change it to `` - `feedback.go`: the labels shown after a word hit or miss, and the "type the meaning" hint ``. In the `store/store.go` line, change `per-word statistics,` to `per-word statistics (reading and meaning apart),` and add `` `words_ask`, `` after `` `focus_weak`, ``.

- [ ] **Step 4: Data Persistence.** Replace the bullet

```markdown
- Per-word statistics (`word_stats`: correct and miss count per reading) and per-word
  spaced-repetition memory (`word:` items in the `memory` table)
```

with

```markdown
- Per-word statistics (`word_stats`: correct and miss count per reading) and per-word
  spaced-repetition memory (`word:` items in the `memory` table), both for typing the reading;
  the same for typing the meaning in `meaning_stats` and as `meaning:` items in the `memory`
  table. Learning levels and slipped flags are `kana:`, `word:` and `meaning:` items in the
  `progress` table
```

and after the bullet with `meaning_lang`, add:

```markdown
- What word tiles ask for (`words_ask`: `reading`, the default, or `meaning`)
```

- [ ] **Step 5: Check** that the README renders sensibly (`grep -n 'Meaning mode\|words_ask\|meaning_stats' README.md` shows the new lines) and contains no issue identifiers.

- [ ] **Step 6: Final verification.** Run `gofmt -l .` (empty), `go vet ./...` (clean), `go build -o /tmp/kana-desktop-check ./fyne/ && rm /tmp/kana-desktop-check`, and `go test ./... -timeout 600s` → PASS.

- [ ] **Step 7: Commit.**

```bash
git add README.md
git commit -m "docs: describe the meaning mode for players" -m "The README explains 'Words ask for: Meaning', the reading and meaning levels, the hint, the marker on meaning tiles and how meaning progress is stored."
```
