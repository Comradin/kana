# Design: Vocabulary Trainer Stage 4a — Word Packages, Settings Tabs, Kana Font

**Date:** 2026-10-15
**Status:** Approved (design discussion), revised after spec review — the section "Review resolutions" is authoritative where it is more specific
**Branch:** `feat/word-packages` (from `main` after PR #17)
**Builds on:** `2026-10-12-vocab-trainer-design.md` (stages 1–3, and "Planned for stage 4")

## Overview

Stage 3 made every word with secure kana playable at once, which flooded beginners, and was
throttled to private lists only. Stage 4a introduces words the way kana rows are introduced:
in small **packages**, each shown first in an intro dialog. Only introduced words fall. This
brings the bundled JLPT words back. The settings dialog gets **tabs**, a **Sources** tab takes
over the attribution, and the learner can choose a **kana font** that ships with the app.

Stage 4b (later, own spec): reading practice, words-only mode with list/lesson, list
management (problems, reload, open folder), meaning language setting.

## Decisions

| Topic | Decision |
|---|---|
| Introduced | A word is introduced once it was shown in a package intro. Only introduced words fall (and their kana must still be secure). |
| Package size | Setting: 3, 5 (default) or 10. |
| Next package | When at least `ceil(0.8 × p)` of the newest package's words that are playable now are secure (≥ 3 correct), `p` being that playable count; if none of them is playable, the package counts as complete. The first package comes as soon as the source has a playable word. |
| Learning source | Setting "New words from": one private list (shown by a prettified file name) or "JLPT". Default: the first private list by name if any, else JLPT. |
| JLPT order | Level by level from the setting `word_level` (N5 first); within a level by frequency rank (JMdict priority tags), then fewer kana units, then reading. |
| Private list order | By lesson (numeric if all lessons of the list are integers, else first appearance), then file order; words without lesson last, in file order. |
| Intro dialog | Each word large with romaji and meaning; per word a "Already known" checkbox. Checked words count as secure and the package is refilled from the source. Pauses the game like kana intros. |
| Level offer | When every bundled word with JLPT ≥ `word_level` (i.e. that level and all easier ones) is introduced, offer "N5 finished — continue with N4?". Yes lowers `word_level`; Later: not asked again this session. If words remain but none is playable yet, the status line says to learn more kana instead. |
| Word share | Unchanged: `min(0.1, playable_introduced / 100)`. |
| Existing progress | On first start, every reading with a `word_stats` row or a `word:` memory state counts as introduced. |
| `bundled_words` | Removed; the learning source replaces it. The store key is ignored. |
| Settings dialog | Tabs Kana, Words, Game, Sources. Save/Cancel apply to all tabs. |
| Attribution | Moves from the status panel to the Sources tab. |
| Kana font | Setting: Noto Sans JP (default), M PLUS Rounded 1c, or System. Full font files (Regular + Bold) ship with the app. |

## Review resolutions

1. **80 % rule:** counts only `newestPackage` readings that are playable now (numerator and
   denominator); none playable → complete. Prevents a deadlock when a script is switched off,
   a list file disappears, or words become unplayable.
2. **Package membership:** `pendingPackage` is the single source of truth.
   `ReplacementWord() (vocab.Entry, bool)` takes the next `NextPackage` entry not already in
   `pendingPackage` and appends it under the lock. `FinishPackage(known []string)`: every
   reading in `pendingPackage` (ticked or not) becomes introduced;
   `newestPackage = pendingPackage − known`.
   Unticking a word keeps its replacement (the package just grows by one). No replacement
   available → the box is ticked, nothing is added.
3. **"Already known":** writes `word_stats` with `correct = max(3, current)` directly (store and
   `wordOverallStats`, not session stats, so the session tiles stay honest) and a memory state
   `memory.Seed(3, now)` (dirty). No points, no tempo change.
4. **Re-check points** for `maybeOfferPackage()`: after a word hit; after a kana hit that makes a
   kana secure; at the end of `checkAnswer` next to `maybeOfferKatakana` (after the score/end
   check); at the end of `Reset` (after its `paused` recomputation), `FinishIntro`, `EnableKatakana`, `DeclineKatakana`,
   `FinishPackage`, `AcceptLevelOffer`; after settings Save. At start, `NewGameState` sets
   `pendingPackage` directly (no event), like `startLearningPath` sets `pendingIntro`; the first
   `showPendingDialogs` shows it after any kana intro. If an event send fails (buffer full),
   nothing is set pending and the next re-check point tries again. Whenever
   `maybeOfferPackage` sets a pending package or level offer it also sets `paused = true`
   (as `announceRows` does); `settle()` only ever unpauses.
5. **Load failure:** `introducedLoadFailed` (like `wordMemoryLoadFailed`): introduced words
   cannot be determined, so the playable set is empty (no words fall), no packages are offered
   and nothing is written to `introduced_words`; kana play is unaffected. The status line says
   "Words unavailable: could not load progress".
6. **Persistence:** `FinishPackage` saves the introduced readings and `newest_package`
   immediately (rare event; no batching through the merge). A failed save leaves the reading
   in `introducedDirty`, which `mergeSessionStats` retries. `introduced_words` is created after
   `ensureMemory` with the same `sqlite_master` check and one transaction for create + seed.
7. **Level offer:** condition per the Decisions table (JLPT ≥ `word_level` all introduced).
   `levelOfferDeclined` lives for the app run (survives `Reset`). Exhausted level after Later:
   status "Words: JLPT N5 finished"; after N1: "Words: JLPT finished".
8. **Stage 3 leftovers removed:** `bundled_words` (store getters/setters, `useBundled`,
   `WordsStatus.Bundled`, `Filter.NoBundled`); the stage 3 level gate on playable words.
9. **`applySettings`:** when source, level, package size or scripts change → invalidate the
   playable cache, run `dropInactiveTiles`, and (source change) clear `newestPackage`.
10. **vocab:** `Entry.ListPos` is deep-copied by `clone()`; `Rank` is taken only from bundled
    words; rank 0 (missing `"r"`) sorts as 99; `NextPackage` uses a per-source order computed
    once per index (cached), then filters.
11. **vocabgen:** `xmlEntry` parses `r_ele` (`reb`, `re_pri`) and `k_ele` (`ke_pri`). Rank from the
    `re_pri` of the `reb` equal to the row's reading; if that has no tags, the best tag over all
    `re_pri`/`ke_pri` of the entry. `SOURCES` is rewritten with the inputs actually used.
12. **Counts:** status "introduced" = introduced readings present in the index; Words tab
    "left in source" = not-introduced entries of the source (JLPT: bundled with JLPT ≥
    `word_level`). Intro title for a package spanning lessons: "minna no nihongo · lessons 2–3".
13. **Fonts:**
    - Four package-level `fyne.Resource` vars with distinct names, created once (Fyne caches
      custom fonts by resource and never evicts them).
    - Each font is parsed once with `github.com/go-text/typesetting/font` (`font.ParseTTF`,
      `face.NominalGlyph`) for the coverage check; the module becomes a direct dependency.
    - Set `FontSource` before measuring and size tiles with `text.MinSize()` (not
      `fyne.MeasureText`, which ignores `FontSource`).
    - The coverage check runs on the final text (after `fitText`, so "…" counts). Its purpose
      is a consistent look: without it, missing runes fall back to system fonts rune by rune.
    - A font change after Save applies to new tiles and labels; falling tiles keep theirs. The
      status grid cells get `SetKanaFont(res)`.
14. **Dialog plumbing:** `PendingDialogs` returns a struct (intro rows, katakana offer, package,
    level offer). A validation error in a tab switches to that tab.

## Word packages

### Data

- `vocab/data/words.json.gz` gains a frequency rank per word, field `"r"` (int, lower = more
  common), computed by `cmd/vocabgen` from the JMdict entry of the row's sequence number,
  using the priority tags of the reading element that matches the row's reading
  (`<re_pri>`), falling back to the best tag of any reading/kanji element:
  - `nfXX` (01–48) → rank `XX`;
  - else any of `news1`, `ichi1`, `spec1`, `gai1` → 49;
  - else any of `news2`, `ichi2`, `spec2`, `gai2` → 60;
  - else 99.
  - Merged homophones take the lowest (best) rank. Waller-fallback rows without an entry → 99.
- `vocab.Word` and `vocab.Entry` gain `Rank int`. For private-list words it is 0 (unused).
- The data is regenerated from current inputs; `SOURCES` is rewritten with the inputs
  actually used. `update-words.yml` keeps working
  unchanged.

### Selection (`vocab`)

```go
// Source names where new words come from.
type Source struct {
	List  string // private list name; "" = JLPT
	Level int    // JLPT: lowest level number allowed (5 = N5 only); ignored for lists
}

// NextPackage returns up to n entries from src that are not introduced and
// whose units are all known, in learning order (see Decisions).
func NextPackage(ix *Index, src Source, introduced map[string]bool, n int, known func(string) bool) []Entry
```

- JLPT order: level descending from 5 down to `src.Level` (so N5 before N4), then `Rank`,
  then `len(Units)`, then `Kana`. Only bundled entries.
- List order: per the Decisions table, from the entry's `Lists[src.List]` lessons; an entry
  in several lessons uses its first lesson in list order. To keep file order, `NewIndex` /
  `WithLists` record for each list entry its first row index (`Entry.ListPos
  map[string]int`).
- `known` is the same rule as stage 3 (`unitKnown`).
- Words that are not playable now (unknown kana) are skipped, not blocking: the package takes
  the next playable words. They come later once their kana are secure.

### Game state (`fyne`)

- `introduced map[string]bool`, loaded from the store; saved immediately by `FinishPackage`
  (Resolution 6); `introducedDirty` only holds readings whose immediate save failed, retried
  in `mergeSessionStats`.
- `newestPackage []string` (readings), persisted so the 80 % rule survives restarts.
- **Playable for spawning** = introduced ∩ known (all units known, as in stage 3), across all
  sources and levels. `word_level` and the source only steer `NextPackage`; an introduced word
  keeps falling even if the level or source changes later.
- **Package check** `maybeOfferPackage()` (under lock), called at the re-check points listed in
  Resolution 4:
  1. Not when over, or while a package or level offer is already pending.
  2. Let `p` = readings of `newestPackage` that are playable now. If `p` is non-empty and fewer
     than `ceil(0.8 × len(p))` of them are secure → stop.
  3. `pkg := NextPackage(index, source, introduced, packageSize, unitKnown)`.
  4. If `pkg` is empty: for the JLPT source, if every bundled entry with JLPT ≥ `word_level` is
     introduced, `word_level > 1` and the offer was not declined this run → queue the level
     offer; else stop (the status line explains: "learn more kana", "JLPT N5 finished",
     "JLPT finished", "<list> finished").
  5. Else queue a **package event** with `pkg`, set `pendingPackage` and `paused = true`.
- **Pause invariant** (as for kana intros and the katakana offer): `pendingPackage []string`
  and `pendingLevelOffer bool` keep the game paused (`settle()`), `PendingDialogs` reports
  them, `showPendingDialogs` shows them after kana intros and the katakana offer (order:
  kana intro, katakana offer, word package, level offer), `Reset` clears them, `watchEvents`
  handles the new event kinds and drops them when over. For event-driven offers, set pending
  only when the event send succeeds (the start path sets it directly, Resolution 4).
- **Finishing a package** `FinishPackage(known []string)`: every reading in `pendingPackage`
  becomes introduced (saved immediately); `newestPackage = pendingPackage − known` (saved
  immediately); each reading in `known` is marked secure as in Resolution 3; the playable cache
  is invalidated; `pendingPackage` is cleared; `settle()`; then `maybeOfferPackage()`.
- **Refill in the dialog:** when the learner ticks "Already known", the dialog calls
  `ReplacementWord()`, which appends the next entry to `pendingPackage` and returns it; the
  dialog adds a row for it. Ticked words stay listed (struck through) and are passed in
  `known`. Unticking keeps the replacement. No replacement → the box is ticked, nothing added.
- **Level offer** `AcceptLevelOffer()` lowers `word_level` by one (not below 1), saves it,
  clears pending, invalidates, `settle()`, then `maybeOfferPackage()`.
  `DeclineLevelOffer()` sets `levelOfferDeclined` (for the app run, survives `Reset`), clears
  pending, `settle()`.
- `Reset` keeps `introduced` and `newestPackage` (they are progress, not session state).

### Intro dialog (`fyne/intro.go`)

- Title "New words" (and the source name, e.g. "minna no nihongo · lesson 2" or "JLPT N5").
- One row per word: the kana large (kana font, see below), romaji (`kanacore.Romaji`),
  meaning (`Meanings.Display(lang)`, truncated as in stage 3), and a checkbox "Already known".
- Button "Let's go" closes it and calls `FinishPackage`.
- Must fit the 1000 × 760 window with 10 words: a scroll container if needed.

### Status line

`Words: 25 introduced · 18 secure · minna no nihongo` (or `· JLPT N5`). Hints instead of
counts when nothing can be introduced: "Words: learn more kana for new words" (source has
words, none playable), "Words: <source> finished" (list fully introduced),
"Words: add a list to ~/.local/share/kana/words or choose JLPT" (list source selected but
missing). The stage 3 "Words unavailable: …" for a broken bundled list stays when the source
is JLPT. Fitted to width as in stage 3.

## Settings dialog with tabs (`fyne/settings.go`)

| Tab | Content |
|---|---|
| Kana | Scripts row; per-script row tabs (as today); auto-progression |
| Words | "New words from" (private lists by prettified name, then "JLPT"); JLPT level N5–N1 (enabled for JLPT); package size 3/5/10; a summary line "Introduced: 25 · secure: 18 · left in source: 101" |
| Game | Difficulty; points per hit; score limit; kana font with preview |
| Sources | Version line with releases link (moved from the bottom); JMdict (EDRDG, CC BY-SA 4.0, licence link, jmdict page link); JLPT levels (Jonathan Waller, CC BY, unofficial); JMdict mapping (stephenmk/yomitan-jlpt-vocab, CC BY-SA 4.0); Noto Sans JP and M PLUS Rounded 1c (SIL OFL 1.1) |

- Pretty list name: replace `-` and `_` with spaces (`minna-no-nihongo` → `minna no nihongo`);
  the stored setting is the real list name.
- `word_source` setting: `jlpt` or `list:<name>`. A missing list stays selected: no packages
  are offered, the status line shows the add-a-list hint, and the Words tab says "List <name>
  not found". The automatic default is resolved once (first list by name, else JLPT) and
  saved on first start, so adding a list later does not switch the source silently.
- Changing the source keeps introduced words falling; new packages come from the new source;
  `newestPackage` is cleared so the next package can come at once.
- The dialog still pauses the game; Save applies all tabs and calls `maybeOfferPackage()`.
- The status panel's attribution lines are removed.

## Kana font

- Setting `kana_font`: `noto` (default), `mplus`, `system`.
- Files in `fyne/fonts/`, embedded with `go:embed`:
  - `NotoSansJP-Regular.otf`, `NotoSansJP-Bold.otf` — from the Noto CJK release (pinned tag,
    `SubsetOTF/JP`), OFL 1.1;
  - `MPLUSRounded1c-Regular.ttf`, `MPLUSRounded1c-Bold.ttf` — from google/fonts (pinned
    commit), OFL 1.1;
  - `OFL-NotoSansCJK.txt`, `OFL-MPLUSRounded1c.txt`.
- `scripts/fetch-fonts.sh` downloads the pinned files and checks SHA-256 sums recorded in the
  script; the files are committed (about 16 MB total).
- Where it applies: every `canvas.Text` that shows kana — kana and word tiles, the big kana in
  kana-row and word-package intros, the kana grid cells in the status panel, and feedback
  labels — via `canvas.Text.FontSource` (regular or bold resource matching the text style).
- Coverage check: a text uses the chosen font only if the font has a glyph for every rune of
  the text (checked once per text with the font's cmap via `github.com/go-text/typesetting`,
  already a Fyne dependency); otherwise `FontSource` stays nil (theme font + system fallback).
  The check keeps a text in one consistent font; without it Fyne would fall back to system
  fonts rune by rune for missing glyphs.
- `system`: `FontSource` nil everywhere (today's behaviour).
- Widgets (labels, buttons, entries, dialogs' text) keep the theme font.
- Changing the font applies after Save to new tiles, new labels and the status grid
  (`SetKanaFont`); tiles already falling keep their font.
- The Game tab shows each choice with a preview "あ ア き キ ー" in that font.

## Store

- Table `introduced_words (kana TEXT PRIMARY KEY, introduced_at INTEGER NOT NULL)`; created in
  `migrate`, and on creation seeded in the same transaction from `word_stats` readings and
  `memory` items with prefix `word:`.
- Settings: `package_size` (3/5/10, default 5), `word_source` (default "" = auto, resolved
  on first start to the first list by name or JLPT and saved), `kana_font` (default `noto`), `newest_package` (readings joined with `\n`).
- `IntroducedWords() (map[string]bool, error)`, `SaveIntroduced(kana string, at time.Time)`.
- `bundled_words` getters/setters are removed.

## Testing

- **vocabgen:** rank from `nfXX`, from `news1`/`ichi1`, from `*2`, none → 99, merged homophones
  take the best rank; fixture extended; output still reproducible.
- **vocab:** `NextPackage` — JLPT level then rank then units then kana; skips introduced and
  not-playable words; partial package; empty; list order by numeric lesson, by first
  appearance for non-numeric lessons, no-lesson words last; entry in two lessons uses the
  first; `ListPos` kept through `WithLists`.
- **store:** `introduced_words` seeding from `word_stats` and `word:` memory once; new
  settings round-trip and default.
- **fyne:**
  - only introduced words spawn;
  - first package offered when a source word is playable; next only at ≥ 80 % secure
    (5 → 4, 3 → 3, 10 → 8);
  - `FinishPackage` with known words: `word_stats` correct ≥ 3 and `memory.Seed(3)` state,
    session tiles unchanged; introduced and `newest_package` persisted immediately;
    `newestPackage` excludes known words;
  - refill: `ReplacementWord` appends an entry not yet in `pendingPackage`; none left → false;
  - 80 % rule counts only playable package words; no playable → complete (script switched off);
  - re-check after `Reset` offers a package again and pauses the new game;
  - `introducedLoadFailed`: no words fall, no packages, status hint;
  - pause invariant with a pending package and with a pending level offer; `Reset` clears
    pending but keeps `introduced`/`newestPackage`;
  - level offer when the level is exhausted, not after Later in the same session, accept
    lowers the level and offers a package;
  - source change clears `newestPackage`; missing list → no packages, hint shown, Words tab
    says "List … not found"; auto source resolved and saved on first start;
  - introduced words keep falling after the level or source changes;
  - status line texts;
  - settings tabs carry the values; Sources tab lists all five sources;
  - kana font: resources load; coverage check with a rune neither font has (e.g. Thai ก,
    confirmed against the pinned files) → nil `FontSource`; tiles use the chosen font and are
    sized with it; `system` → nil.
- **Manual:** on macOS, Windows and Linux (release packages), kana render in both fonts;
  a first package appears for a beginner with secure vowels and k-row.

## Out of Scope (stage 4b and later)

- Reading practice, words-only mode with list/lesson filter, list problems/reload/open
  folder, meaning language setting (stage 4b).
- Meaning mode (stage 5).
- Kanji display or furigana.
