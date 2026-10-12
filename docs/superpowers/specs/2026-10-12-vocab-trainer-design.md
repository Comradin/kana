# Design: Vocabulary Trainer

**Date:** 2026-10-12
**Status:** Approved (design discussion)
**Branch:** `feat/vocab-trainer` (from `main` after PR #11)

## Overview

Kana grows into a vocabulary trainer. Words written in kana fall like tiles; the learner types
the reading in romaji and, after a hit, sees the meaning. Words appear only when the learner
can read them (their kana are known) and only up to the learner's JLPT level. Words come from
a bundled list derived from JMdict with JLPT levels from Jonathan Waller's lists, and from
private word lists the learner keeps in their data directory, never in the repository.

Words need three things the app does not have yet: the small っ/ッ, the long vowel mark ー and
the extended katakana (ティ, ファ, ヴ …). They are added here.

The work ships in four stages, each with its own plan and PR (see Stages).

## Decisions

| Topic | Decision |
|---|---|
| Play forms | **Mixed** (default): words fall among kana tiles. **Words only**: a mode for targeted practice of a level, a private list or one lesson of it. |
| Answer | The romaji reading of the whole word. The meaning is shown after a hit and, with the solution, after a miss. |
| Meaning language | A setting. Bundled words carry all JMdict languages; fallback to English, then to any available language. Private lists carry their own meanings. Input is always Hepburn romaji. |
| Word playable (mixed) | Every kana unit of the word is **secure** (≥ 3 correct, the panel's "mastered"). っ/ッ and ー count as known once their script is active. |
| Word playable (words only) | Every kana unit is **unlocked** (in an active row). The learner chose the practice explicitly. |
| Extended katakana | Both: a new katakana learning-path group with tiles and intros, and part of words. A word with ティ needs ティ itself to be secure. |
| Bundled words | Only words with a JLPT level (about 8,000, N5–N1), shown in their kana reading. No kanji. |
| JLPT level | Setting "Vocabulary up to N5 … N1", default N5. Bundled words above the level are never played. Private lists ignore the level. |
| Level offer | When 90 % of all bundled words of the current level are secure, a dialog offers the next level. Declined → not asked again for that level. |
| Private lists | CSV/TSV and JSON in `~/.local/share/kana/words/` (honours `XDG_DATA_HOME`). |
| Progress key | Per reading: `word:<kana>` in `memory`, and a `word_stats` row. Homophones share progress; their meanings are merged. |

## Stage 1: Kana extensions and reading match (`kanacore`)

### Extended katakana group

A new group `GroupExtended` ("Extended katakana"), katakana only, after Yōon in the katakana
learning path. Like the other groups, the path unlocks two rows per step, each with the intro
dialog.

| Row ID | Entries |
|---|---|
| `kata:ext-t` | ティ ti, ディ di, トゥ tu, ドゥ du |
| `kata:ext-f` | ファ fa, フィ fi, フェ fe, フォ fo |
| `kata:ext-w` | ウィ wi, ウェ we, ウォ wo |
| `kata:ext-sh` | シェ she, ジェ je, チェ che |
| `kata:ext-v` | ヴァ va, ヴィ vi, ヴ vu, ヴェ ve, ヴォ vo |
| `kata:ext-ts` | ツァ tsa, ツィ tsi, ツェ tse, ツォ tso |

These rows are defined directly in katakana; they are not generated from hiragana rows and
`RowsFor(Hiragana)` does not contain them. Input collisions (ティ/ち "ti", トゥ/つ "tu",
ウォ/を "wo") need no new rule: `checkAnswer` already picks the lowest matching tile.

### Word units

`KanaUnits(word string) ([]string, bool)` splits a word into units, longest match first:

1. a two-character combination known to any row (yōon きゃ, extended ティ);
2. a single kana known to any row;
3. っ/ッ (sokuon) and ー (long vowel mark) as units of their own.

It returns `false` if any character cannot be placed (e.g. ゐ, ヶ, a kanji), or if the word
starts with っ/ッ or ー.

### `ReadingMatches(word, input string) bool`

Input is lowercased and trimmed. It matches when the input can be split, left to right, so
that each unit is matched by one of its spellings:

- A normal unit: its canonical romaji or any alternative from its row (`shi`/`si`,
  `chi`/`ti`, `tsu`/`tu`, `ja`/`zya`/`jya` …).
- ん/ン: `n`, `nn` or `n'`. Because one known word is checked, `n` before a vowel is
  not ambiguous.
- っ/ッ: doubles the first letter of the next unit's chosen spelling (`kitte`); before a
  spelling starting with `ch`, `t` is also accepted (`matcha` and `maccha`). At the end of a
  word it matches nothing.
- ー: repeats the last vowel of the previous unit's chosen spelling (`koohii`), or `-`
  (`ko-hi-`).
- Long vowels written with う/お/い are typed kana by kana: とうきょう is `toukyou`;
  `tokyo` does not match. This is deliberate, so learners notice long vowels.

Implementation: recursion or dynamic programming over (unit index, input position); words are
short, so no optimisation beyond memoising positions is needed.

### `Romaji(word string) string`

The canonical spelling, built from each unit's canonical romaji with the っ and ー rules
above (`kitte`, `matcha`, `koohii`, `paatii`). Used as the solution shown after a miss.
っ before `ch` gives `tch`.

## Stage 2: Word data (`vocab`, `cmd/vocabgen`)

### Model

```go
type Word struct {
	Kana     string            // reading, e.g. "せんせい"
	Meanings map[string]string // ISO 639-1 language → "teacher; master"
	JLPT     int               // 5…1, 0 = none
	Source   string            // "jmdict" or a list name, e.g. "minna-no-nihongo"
	Lesson   string            // optional, private lists only
}
```

`vocab` depends on `kanacore` (for `KanaUnits`) and nothing from the app.

### Bundled list

- `vocab/data/words.json.gz`, embedded with `go:embed`, loaded by `vocab.Bundled() ([]Word, error)`.
  Format: a JSON array of `{"k": kana, "j": level, "m": {"en": …, "de": …}}`.
- Generated by `go run ./cmd/vocabgen -jmdict JMdict.gz -jlpt <dir> -out vocab/data/words.json.gz`,
  run by hand, not during build. Inputs are downloaded by the maintainer; the tool does not
  download.
  - JMdict: the full multilingual `JMdict.gz` (EDRDG). Languages kept: en, de, fr, ru, es, hu,
    nl, sv, sl (JMdict codes eng, ger, fre, rus, spa, hun, dut, swe, slv).
  - JLPT levels: Jonathan Waller's lists (tanos.co.uk, CC BY), matched to JMdict entries via
    an existing published mapping (e.g. `stephenmk/yomitan-jlpt-vocab`); its licence is
    checked in stage 2 and recorded in `ATTRIBUTION.md`. If no usable mapping exists, match
    by kanji form + reading.
  - Per entry: the first reading element (preferring one with a priority tag), the level, and
    per language the first sense's glosses joined with "; " (at most three).
  - Entries whose reading fails `KanaUnits` are dropped and counted in the tool's output.
  - Entries with the same reading are merged into one word: the easiest level (the highest
    number, e.g. N5 over N3) wins, meanings joined with "; " per language.
- `vocab/data/ATTRIBUTION.md`: JMdict © Electronic Dictionary Research and Development Group,
  CC BY-SA 4.0; JLPT levels © Jonathan Waller, CC BY; the mapping's source and licence. The
  data file is CC BY-SA 4.0; the code stays MIT. The README links to it.

### Private lists

- Directory: `filepath.Join(dataDir, "words")`, where `dataDir` is the directory of
  `store.DefaultPath()`. Not created automatically (only by "Open folder"). `.gitignore` gains
  `words/` as a guard.
- `vocab.LoadDir(dir string) ([]List, error)`; a missing directory returns no lists and no
  error. Each `List` has `Name` (file name without extension), `Words []Word` and
  `Problems []string` (at most the first 10, plus a count of skipped rows).
- **CSV/TSV** (`*.csv`, `*.tsv`): UTF-8, a BOM is ignored. The header row is required; the
  separator is detected from it: tab, then semicolon, then comma. Columns (case-insensitive):
  `kana` (required), `meaning` (applies to every language not given separately),
  `meaning_<lang>`, `lesson`, `jlpt` (accepts `5` or `N5`). A file without a `kana` column is
  one problem and no words. A row whose `kana` fails `KanaUnits` is skipped with a problem
  "line 17: 先生 is not kana".
- **JSON** (`*.json`): an array of `{"kana": …, "meaning": "…" or {"de": …}, "lesson": …,
  "jlpt": …}`. Same row rules.
- Within one list, duplicate readings are merged like bundled homophones.
- A later Anki loader returns the same `List` values; see Out of Scope.

### Playable filter

```go
// Playable returns the words whose units all satisfy known, and, for bundled
// words, whose level is at or below maxLevel (JLPT >= maxLevel numerically).
func Playable(words []Word, known func(unit string) bool, maxLevel int) []Word
```

Private-list words skip the level check. Bundled words without a level do not exist (the
generator keeps only levelled words).

## Stage 3: Words in the game (mixed mode)

### Store

- New table, created in `migrate`:
  ```sql
  CREATE TABLE IF NOT EXISTS word_stats (
  	kana          TEXT PRIMARY KEY,
  	correct_count INTEGER NOT NULL DEFAULT 0,
  	miss_count    INTEGER NOT NULL DEFAULT 0
  );
  ```
  Separate from `kana_stats`, because the word か (mosquito) must not share the character か's
  stats.
- `WordStatistics() (map[string]WordStats, error)`, `IncrementWordCorrect(kana)`,
  `IncrementWordMiss(kana)`, written in `mergeSessionStats` like kana stats.
- Memory states for words use item `word:<kana>` in the existing `memory` table.

### GameState

- Words are loaded once in `NewGameState` (bundled + private lists) and kept in `gs.words`.
  A load error of the bundled list disables bundled words and is reported in the settings
  dialog; kana play is unaffected.
- A tile carries either a kana or a word (`tile.word *vocab.Word`, nil for kana).
- `known(unit)` for mixed mode: the unit is `っ`/`ッ`/`ー` and its script is active, or the
  unit's stored + session correct count is ≥ 3.
- **Spawn:** with `p = min(0.3, len(playable) / 100)`, a spawn is a word; otherwise a kana as
  today. The word is chosen with `memory.Pick` over the playable readings (falling words
  excluded as for kana), using the `word:` memory states and `gs.rnd`.
- **Fall time:** level range × tempo × `min(1 + 0.2 × (units − 1), 2)`. The per-character
  factor does not apply to words.
- **Answer:** `checkAnswer` tests kana tiles with `Matches` and word tiles with
  `ReadingMatches`; the lowest matching tile wins. A word hit scores `10 × units ×
  pointsFactor()`, records word correct (stats, memory, answer streak and tempo as for kana).
  It does not change any kana's stats.
- **Miss:** a word that falls through counts towards the miss limit, records a word miss and
  `memory.Miss` when the word has a state, and lowers the tempo as a kana miss does.
- **Feedback:** after a hit, a label "はい — yes" stays at the tile's last position for 2 s;
  after a miss, "はい · hai — yes" appears at the bottom in the error colour for 2 s. The game
  keeps running. The meaning comes from `meaning_lang`, then `en`, then any language; a word
  without meanings shows only kana (and romaji for a miss).
- Word tiles are as wide as their text plus padding; the spawn x-position keeps them inside
  the field.
- **Status panel:** a line "Words: 42 playable · 17 secure · up to N5" (in words-only mode
  it names the source instead of the level).

## Stage 4: Words-only mode, settings, level offer

### Words-only mode

- Setting `game_mode`: `mixed` (default) or `words`.
- Setting `word_source`: `jmdict` (default) or a list name; `word_lesson`: empty (all) or a
  lesson value of that list.
- In words mode every spawn is a word from the chosen source (and lesson), playable by the
  **unlocked** rule, chosen with `memory.Pick`. With no playable word, kana spawn instead and
  the status line says "No playable words — unlock more kana or pick another list".
- If the stored `word_source` no longer exists, `jmdict` is used and the settings dialog says
  so.

### Level offer

- Setting `word_level` (5 … 1, default 5) and `word_level_declined` (the highest level number
  declined, 0 = none).
- After a word hit, if `word_level > 1` and at least 90 % of all bundled words of level
  `word_level` are secure (≥ 3 correct), and the next level has not been declined, the offer
  dialog "Add N4 vocabulary?" is queued through `showPendingDialogs`, pausing like the
  katakana offer. Accept lowers `word_level` by one; Decline stores it in
  `word_level_declined`.

### Settings dialog

The dialog gets four outer tabs; it must still fit the 1000 × 760 window:

| Tab | Content |
|---|---|
| Kana | today's scripts row, per-script row tabs and auto-progression |
| Words | mode, level (N5–N1), source and lesson (words-only), meaning language, private lists with word count and problems, "Reload lists", "Open folder" |
| Game | difficulty, points per hit, score limit |
| Sources | attribution for JMdict, Waller and the mapping, with licences and links |

- `meaning_lang` default: the system language if JMdict has it, else `en`. The choice lists
  the nine JMdict languages plus any `meaning_<lang>` found in private lists.
- "Reload lists" re-reads the directory; "Open folder" creates it if missing and opens it via
  `fyne.CurrentApp().OpenURL`.
- New settings round-trip through the store with defaults when unset, like the existing keys.

## Testing

- **kanacore:**
  - `KanaUnits`: きゃく, パーティー, ヴァイオリン, きって; failures for 先生, ゐ, a leading ー.
  - `ReadingMatches` table: きって `kitte`; まっちゃ `matcha`/`maccha`; コーヒー
    `koohii`/`ko-hi-`; でんしゃ `densha`/`densya`/`dennsha`/`den'sha`; パーティー
    `paatii`; とうきょう `toukyou` yes, `tokyo` no; ふじさん `fujisan`/`huzisan`; wrong
    inputs, a prefix and an overlong input do not match.
  - `Romaji` for the same words.
  - The katakana learning path contains the extended group after Yōon; hiragana does not.
- **vocab:**
  - CSV with comma, semicolon and tab; BOM; missing `kana` column; non-kana rows reported;
    `meaning` vs `meaning_de`; `jlpt` as `4` and `N4`.
  - JSON with meaning as string and as object.
  - Homophones merged; `Playable` with level and unit filter; private words ignore the level.
  - `Bundled()` loads, has more than 5,000 words, and every word passes `KanaUnits`.
  - `LoadDir` on a missing directory returns nothing and no error.
- **store:** `word_stats` and every new setting round-trip and default.
- **fyne:**
  - Word share in mixed mode with a seeded `rnd` (0 with no playable word, about 0.3 with
    ≥ 30).
  - Word tile fall time, points, hit and miss (stats, memory, miss limit, tempo), feedback
    label text.
  - A kana tile and a word tile both matching: the lower one wins.
  - Words-only: unlocked rule, source and lesson filter, kana fallback with the hint.
  - Level offer at 90 % and not after a decline.
  - Status line text.

## Stages

1. **Kana extensions:** extended katakana group, `KanaUnits`, `ReadingMatches`, `Romaji`.
   Playable result: the new katakana group.
2. **Word data:** `vocab`, loaders, `cmd/vocabgen`, the bundled list and `ATTRIBUTION.md`.
   No UI.
3. **Words in the game:** mixed mode, `word_stats`, word tiles, feedback, status line.
4. **Words-only mode and settings:** tabs, private lists in the UI, level offer, sources tab.

Each stage gets its own plan and PR.

## Out of Scope

- An Anki loader (`.apkg`: a zip with a SQLite collection, newer exports zstd-compressed) with
  a per-deck field mapping. It would return `vocab.List` values. Anki's plain-text export
  (tab-separated) already works through the TSV loader.
- Other input systems (Polivanov Cyrillic, kana input via IME). They would replace the
  romaji spellings used by `ReadingMatches`.
- A translated UI.
- Kanji and furigana; asking for the meaning instead of the reading.
- Thierry Bézecourt's pre-2010 JLPT lists (no free licence) and Wagotabi's own data
  (proprietary).
