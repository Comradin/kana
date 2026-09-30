# Design: Complete Hiragana (Dakuon, Handakuon, Yōon)

**Date:** 2026-09-30
**Status:** Approved (design discussion and spec review)

## Overview

Extend the desktop app (`kana-desktop`) from the 46 basic hiragana to the full practical set: Dakuon, Handakuon, Yōon and Yōon with (han)dakuten. That adds 58 characters in 16 new rows, 104 characters in total. Katakana is the next milestone after this one, so the data model must take a second script without structural changes.

The terminal app (`kana`, Bubble Tea) is **removed** as the first step of this work. Development concentrates on the Fyne app. A different frontend can be built later, once the core logic has settled, and the old code stays available in the git history.

## Decisions

| Topic | Decision |
|---|---|
| Scope | Dakuon (20), Handakuon (5), Yōon (21), Yōon with (han)dakuten (12). ぢゃ/ぢゅ/ぢょ are left out: they are practically unused. |
| Romaji | Hepburn is the canonical answer and is what the UI shows. Common alternatives (Kunrei/Nihon-shiki and IME spellings) are also accepted. |
| Duplicate romaji | じ/ぢ (ji) and ず/づ (zu) share an answer. The input hits the **lowest** matching tile on the canvas. ぢ/づ can also be typed unambiguously as di/du. |
| Progression order | By group: Basic → Dakuon → Handakuon → Yōon → Yōon with dakuten. |
| Learning path | New desktop users start with the vowel row only, with auto-progression on. The rest unlock in steps of two rows, and a step never crosses a group boundary. Learners with prior knowledge add groups or rows in the settings. |
| Terminal app | Removed before the feature work (see *Removing the terminal app*). |
| Order of work | 1. Remove the terminal app. 2. Rebase PR #5. 3. Feature work. Doing the removal first means the Charm lines are already gone from `go.mod`, which leaves fewer conflicts for the rebase. PR #5 (aligned hiragana stats table) is rebased onto `main` **without** its dependency bumps, and merged. This feature then builds on it. The bumps sit in commit `9d95806`, together with a `fyne/stats.go` change, so while replaying that commit `go.mod`/`go.sum` have to be reset to `main`, followed by `go mod tidy`. |
| Default selection | A fresh database starts the learning path: the vowel row only, with auto-progression on. An empty selection in the settings dialog falls back to the 11 basic rows. |

## Character Data

### New rows

Alternatives in parentheses.

**Dakuon**

| Row ID | Label | a | i | u | e | o |
|---|---|---|---|---|---|---|
| `g` | G-row (が) | が ga | ぎ gi | ぐ gu | げ ge | ご go |
| `z` | Z-row (ざ) | ざ za | じ ji (zi) | ず zu | ぜ ze | ぞ zo |
| `d` | D-row (だ) | だ da | ぢ ji (di) | づ zu (du) | で de | ど do |
| `b` | B-row (ば) | ば ba | び bi | ぶ bu | べ be | ぼ bo |

**Handakuon**

| Row ID | Label | a | i | u | e | o |
|---|---|---|---|---|---|---|
| `p` | P-row (ぱ) | ぱ pa | ぴ pi | ぷ pu | ぺ pe | ぽ po |

**Yōon**

| Row ID | Label | ya | yu | yo |
|---|---|---|---|---|
| `ky` | KY (きゃ) | きゃ kya | きゅ kyu | きょ kyo |
| `sy` | SH (しゃ) | しゃ sha (sya) | しゅ shu (syu) | しょ sho (syo) |
| `ch` | CH (ちゃ) | ちゃ cha (tya, cya) | ちゅ chu (tyu, cyu) | ちょ cho (tyo, cyo) |
| `ny` | NY (にゃ) | にゃ nya | にゅ nyu | にょ nyo |
| `hy` | HY (ひゃ) | ひゃ hya | ひゅ hyu | ひょ hyo |
| `my` | MY (みゃ) | みゃ mya | みゅ myu | みょ myo |
| `ry` | RY (りゃ) | りゃ rya | りゅ ryu | りょ ryo |

**Yōon with (han)dakuten**

| Row ID | Label | ya | yu | yo |
|---|---|---|---|---|
| `gy` | GY (ぎゃ) | ぎゃ gya | ぎゅ gyu | ぎょ gyo |
| `j` | J (じゃ) | じゃ ja (zya, jya) | じゅ ju (zyu, jyu) | じょ jo (zyo, jyo) |
| `by` | BY (びゃ) | びゃ bya | びゅ byu | びょ byo |
| `py` | PY (ぴゃ) | ぴゃ pya | ぴゅ pyu | ぴょ pyo |

### Alternatives added to existing characters

し `si`, ち `ti`, つ `tu`, ふ `hu`, ん `nn`. Canonical answers are unchanged.

`o` is **not** added for を: it would collide with お. を stays `wo`.

## Architecture

### `kanacore` data model

Rows become the single source of truth; everything else is derived from them.

```go
type Group string

const (
    GroupBasic      Group = "basic"
    GroupDakuon     Group = "dakuon"
    GroupHandakuon  Group = "handakuon"
    GroupYoon       Group = "yoon"
    GroupYoonDakuon Group = "yoon-dakuon"
)

type Entry struct {
    Char   string
    Romaji string   // canonical Hepburn, shown in the UI
    Alt    []string // additionally accepted inputs
}

type KanaRow struct {
    ID      string
    Label   string
    Group   Group
    Entries []Entry
}

func (r KanaRow) Characters() []string
```

- `AllKanaRows` holds all 27 rows in progression order: 11 basic rows, then groups in the order of the `Group` constants.
- `ProgressionSteps [][]string` lists the row IDs unlocked together, in order (see *Learning path*). A test checks that the steps cover every row in `AllKanaRows` exactly once and in the same order, and that no step crosses a group boundary.
- `BasicRows()` returns the 11 basic rows only.
- `Groups() []GroupInfo` returns the groups in display order, with `GroupInfo{Group Group; Label string}` (labels “Basic”, “Dakuon”, “Handakuon”, “Yōon”, “Yōon + Dakuten”), for the settings dialog and the stats panel.
- `Hiragana()` returns the `CharacterSet` built from `AllKanaRows`, using a builder `NewCharacterSet(name, rows)` that Katakana will reuse later.
- `CharacterSet` keeps `Data map[string]string` (char → canonical romaji) so existing callers keep working. It gains `Matches(char, input string) bool`, which is true if `input` equals the canonical romaji or any alternative. Input is compared lower-cased and trimmed.
- `CharToRow` is derived from `AllKanaRows` as today. `DefaultRowIDs()` returns the IDs of `BasicRows()` (see *Persistence and migration*).

Katakana later becomes a second set of rows with the same shape (plus a script dimension). That is out of scope here, but nothing in this design assumes hiragana.

`KanaRow.Characters` changes from a field to a method. The call sites are `fyne/game.go`, `fyne/stats.go`, `fyne/game_test.go` and `kanacore` itself; all of them are updated mechanically.

### Removing the terminal app

This is a separate commit before the feature work:

- Delete the root `package main` files `main.go`, `game.go`, `ui.go` and `settings_form.go`.
- Remove the dependencies with `go mod tidy`: `bubbletea`, `lipgloss` and `huh`, plus their indirect dependencies.
- Leave `store/` unchanged.
- Update the docs:
  - `README.md` describes the desktop app only, fixes the stale `kana.go` reference and the empty “Terminal App (``)” heading, and builds with `go build -o kana-desktop ./fyne/`.
  - `CLAUDE.md` describes the current structure: Fyne app, `kanacore`, `store`. It is ignored by the global gitignore, so this is a local change only and will not appear in the commit.
- Add `kana-desktop` to `.gitignore`.
- Remove the `Kana.X`/`Kana.Y` fields, which only the terminal app used. Also remove the terminal-bell item in `PLAN.md`.
- Keep `kana.db` compatible: the desktop app already reads and writes the same keys.

Moving the game logic out of `fyne/` into a UI-independent package is **not** part of this work. It is the natural first step if another frontend comes back.

### Desktop answer matching (`fyne/game.go`)

Input is now trimmed and lower-cased before matching (today it is compared raw), so `Ka ` counts as `ka`.

`checkAnswer` looks at every tile whose character `Matches` the input and removes the one with the **largest Y**, the lowest tile on the canvas. Today the first tile in spawn order wins. That is the same tile in most cases, but the new rule is predictable when romaji are duplicated.

The missed list and the game-over dialog keep showing the canonical romaji.

### Tiles (`fyne/tile.go`)

- The tile width depends on the number of runes: 52 px for one glyph, 84 px for two. The height stays at 60 px.
- The text is centred from its measured size (`fyne.MeasureText`) instead of the hard-coded 18 px glyph width.
- `KanaTile` exposes `Width()`. `spawnKana` uses it for `maxX`, so wide tiles do not spawn past the right edge.

### Settings dialog (`fyne/settings.go`)

The single `CheckGroup` over all rows is replaced by one section per group:

- a bold group heading with an “all” check that toggles every row in the group, and
- a `CheckGroup` with that group's rows,

all inside a vertical scroll container with a fixed minimum height, so the dialog does not grow off-screen.

Saving works as today: the selected row IDs are collected across all groups, an empty selection falls back to `DefaultRowIDs()`, and tiles of rows that were deselected are removed.

### Stats panel (`fyne/stats.go`, on top of PR #5)

PR #5 places each character in a vowel column by the last letter of its romaji. That already fits the new rows: Dakuon and Handakuon fill all five columns, Yōon fill the a/u/o columns.

Additions:

- Rows stay visible **only while they are selected**, as PR #5 already does in `Update`. That keeps the table short despite 27 rows.
- Every group except the basic one gets a group label row in the same 6-column grid: the group label in the first cell and five empty cells. It is shown while at least one row of the group is visible. Caveat: `NewGridWithColumns` sizes all columns to the widest cell. If a long label such as “Yōon + Dakuten” widens the table visibly, fall back to one grid per group with a full-width label between them.
- The short label in the first column keeps PR #5's `rowShortLabel`, extended so that `sy` shows as `sh`. All other new IDs are shown as they are.
- The missed list uses the new row data. Its labels are pre-created for all 104 characters, as today.

### Persistence and migration

The schema does not change: stats are keyed by character, and selected rows are stored as ID strings.

Existing users who have all 11 basic rows selected no longer count as “everything unlocked”. With auto-progression on, they continue with the step `g`+`z`. With auto-progression off, nothing changes until they select new rows in the settings.

**Default selection.** `DefaultRowIDs()` returns the IDs of `BasicRows()` from now on. The settings dialog uses it as the fallback for an empty selection. A fresh start does not use it: that follows the learning path below.

## Learning path (desktop)

Beginners should not face 46 characters at once. The desktop app gets a learning path, similar to what the old terminal app did when auto-progression was switched on without stats.

**Fresh start.** If the store has no saved row selection, `NewGameState` selects only the first step (`vowels`) and turns auto-progression on. It persists both, so the next launch counts as an existing user. This also applies to an existing desktop user who never saved a selection. If that user has stats, the catch-up described under *Introducing new rows → Flow* restores their mastered steps right at startup.

**Steps.** `kanacore.ProgressionSteps`:

| Group | Steps |
|---|---|
| Basic | `vowels` → `k s` → `t n` → `h m` → `y r` → `w n-only` |
| Dakuon/Handakuon | `g z` → `d b` → `p` |
| Yōon | `ky sy` → `ch ny` → `hy my` → `ry` |
| Yōon with dakuten | `gy j` → `by py` |

Dakuon and Handakuon are separate groups, but `p` stands alone anyway, so no step crosses that boundary.

**Unlock rule.** The mastery condition stays as it is: every selected row must have at least 80 % of its characters answered correctly three times or more. When it holds, `checkAutoProgression` takes the first step that still has an unselected row and selects **all** unselected rows of that step. If the user already added some rows by hand, only the rest of the step is added. The unlock message names every row it unlocked.

**Prior knowledge.** Nothing extra is needed. The user ticks groups or rows in the settings dialog, and progression continues from the first step that is not complete.

The fresh-start path switches auto-progression on even if the store says it is off. That includes existing desktop users who never saved a selection. If `SelectedRows()` returns an error, the app does **not** treat it as a fresh start: it keeps today's behaviour and does not overwrite anything.

## Introducing new rows (desktop)

Before a row is asked, it is introduced, so the learner has a chance to memorise it.

**When the dialog appears**
- On every automatic unlock.
- On a fresh start, for the vowel row, before the first tile falls.
- **Not** when rows are added by hand in the settings. The learner chose them and probably knows them already.

**Content**
- The title names the rows, for example “New: K-row (か), S-row (さ)”.
- For each row, a line of large tiles in the same paper style as the falling tiles, at about twice the size (text around 64 px), with the romaji below each tile. Two-glyph kana use the wide tile.
- One button: “Let's go”.

The UI stays in English, as today.

**Pause**
- The game pauses while the dialog is open: tiles do not move, nothing spawns, and the input is disabled.
- The pause is set under the lock at the moment of unlocking, inside `recordCorrect`, and not only when the dialog opens. A tick in the gap between the two therefore cannot move tiles.
- Closing the dialog resumes the game. Characters of the new rows can only spawn from then on, because the rows are selected at unlock time and spawning is paused.
- `GameState` gets `Pause()`/`Resume()` and a `paused` flag, which `tick` and `spawnKana` check. The flag also gives `PLAN.md`'s general pause feature something to build on later.

- `checkAnswer` ignores input while paused. Otherwise a quick Enter in the gap before the dialog opens could match a tile already on screen and trigger a second unlock.

**Flow**
- An automatic unlock sends a new event kind, `rowsUnlockedEvent`, which carries the row IDs, to the existing buffered `eventCh`. The send is non-blocking, like `gameOverEvent`.
- `pendingIntro` lives in `GameState`:
  - `recordCorrect` sets it to the unlocked IDs together with `paused`, both under the lock and only if the send succeeds.
  - “Let's go” clears `pendingIntro` and calls `Resume()`.
- The game pauses **only if the send succeeds**. A full channel is not expected (4 slots, and at most one unlock per pause), but if it happens the rows are still unlocked, just without an intro and without a pause.
- `watchEvents` opens the intro dialog via `fyne.Do`. The game-over dialog is currently opened as `go showGameOverDialog(...)`, from outside the UI thread. It moves to `fyne.Do` as well, in line with commit `a370975`.
- **Game over wins.** The answer that unlocks a step can also reach the score limit. In that case `rowsUnlockedEvent` sits in `eventCh` before `gameOverEvent`.
  - The intro handler skips the dialog when the game is already over.
  - Instead, the unlocked IDs stay in `pendingIntro`.
- **`Reset()`** (“Play Again”) clears `paused`. If `pendingIntro` is not empty, the new game starts paused, and the Play Again handler shows the intro before the first tile falls.
- **Quit** with a pending intro leaves the rows selected but not introduced. That is accepted, and `pendingIntro` is not persisted.
- **Fresh start.** `NewGameState` sets `pendingIntro = [vowels]` and starts paused. `buildWindow` shows the intro dialog directly after `w.SetContent(...)`. That happens before `ShowAndRun`, and Fyne displays a dialog on a window that is not yet visible once the window appears. If that turns out not to work, queue the dialog via `fyne.Do` instead. The implementation has to verify which one works.
- **Returning users without a saved selection.** The fresh-start check also runs **catch-up**: starting from `[vowels]`, the app adds the next step **only if that step's own rows are all mastered** according to the overall stats, and stops at the first step that is not. The catch-up rows get no intro, since the learner knows them. The first unmastered step then arrives through normal auto-progression, **with** an intro, as soon as the learner answers a tile from the selected rows correctly. `pendingIntro = [vowels]` is only set when the user has no stats at all. A returning user who has stats but has not mastered the vowels starts on `[vowels]` directly, without an intro and without a pause. This avoids up to 15 dialogs in a row for someone who already knows all hiragana.

**Replaces** the 5-second unlock message in the stats panel: `unlockMessage`, `unlockAt`, `StatsSnapshot.UnlockMessage/UnlockAt`, the label and its test all go. The label text built by `showUnlockMessage` moves into the dialog title.

## Testing

- `kanacore`
  - `AllKanaRows` contains 104 characters and no duplicates.
  - Every row has a known group.
  - The groups appear in progression order.
  - `BasicRows()` returns exactly the 11 existing rows with their 46 characters.
- `Matches`: a table test over canonical and alternative inputs (shi/si, ji/zi/di, zu/du, sha/sya, cha/tya/cya, ja/jya/zya, n/nn), plus negative cases (`o` does not match を, and `di` does not match じ).
- Desktop game
  - With two tiles that share a romaji (じ and ぢ), the lower one is removed.
  - Alternative input removes a tile.
  - Auto-progression unlocks `k`+`s` after `vowels`, `g`+`z` after the last basic step, and `ky`+`sy` after `p`.
  - If `k` was added by hand, the next unlock adds only `s`.
  - The existing `TestCheckAutoProgressionUnlocksNextRow` is **rewritten** for the step rule. No second test is added next to it.
  - An unlock pauses the game and puts a `rowsUnlockedEvent` with the unlocked IDs on `eventCh`.
  - While paused, `tick` moves no tiles and `spawnKana` adds none. After `Resume()` both work again.
  - A fresh start begins paused with `pendingIntro == [vowels]`.
  - A fresh start with stats that master the first three steps selects those steps and sets no `pendingIntro`.
  - While paused, `checkAnswer` removes no tile.
  - When an unlock and the score limit happen in the same answer, the game is over and `pendingIntro` holds the unlocked IDs. After `Reset()` the game is paused with the same `pendingIntro`.
  - After `Reset()` without a pending intro, the game is not paused.
  - The fresh-start path forces auto-progression on, even over a stored “off”.
  - A `SelectedRows()` error does not trigger a fresh start.
  - A fresh store yields `[vowels]` with auto-progression on, and both are persisted.
  - A store with a saved selection keeps it unchanged.
- Tiles: a two-glyph tile is wider than a one-glyph tile, and a spawn keeps the whole tile inside the canvas.
- `ProgressionSteps` covers `AllKanaRows` exactly once and in order, and no step crosses a group boundary.
- `go build ./...` and `go vet ./...` stay clean after the terminal app is removed.

## Out of Scope

- Katakana (next milestone).
- ぢゃ/ぢゅ/ぢょ, ゔ, small っ (sokuon) and long vowels.
- Mastery colours, spawn weighting and other items from `PLAN.md`, including a user-facing pause button (the `Pause()`/`Resume()` mechanism is built, the button is not).
- An intro dialog for rows added by hand, and a way to reopen intros later.
- Moving `kana.db` out of the working directory.
- CI setup.
