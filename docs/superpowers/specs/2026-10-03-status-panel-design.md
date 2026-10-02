# Design: Status Panel, Window Size and Fall Time

**Date:** 2026-10-03
**Status:** Approved (design discussion and spec review)
**Mockups:** https://claude.ai/artifact/1YqLRLs7wg7vCUxazhQsbJ ("A + C kombiniert")
**Builds on:** the katakana, pause and data-path work on `feat/katakana`

## Overview

The status panel on the right is redesigned: a session line, a compact learning-path line per
active script, and one kana grid that shows hiragana and katakana side by side with each
character's learning state. The "ACTIVE ROWS" and "MISSED" lists go. The window starts larger,
and tiles fall at a constant time instead of a constant pixel speed, so the pace no longer
depends on the window height.

## Decisions

| Topic | Decision |
|---|---|
| Panel layout | Top to bottom: session line → one path line per active script → kana grid (scrolls on its own if needed) → legend. The panel as a whole never scrolls. |
| Removed | "ACTIVE ROWS" (the settings show it) and "MISSED" (red cell border instead; romaji stay in the game-over dialog). |
| Learning state per character | Total correct answers = stored + session. **new** = 0, **learning** = 1–2, **mastered** = ≥ 3 (the same threshold the unlock rule uses per character). A character missed in this session gets a red border on top of its state. |
| Grid rows | One line per base row (27, hiragana order). A line is shown when the row is selected in at least one **active** script. Each half (hiragana left, katakana right) shows its five cells only if that script is active and that row is selected in it; otherwise the half stays blank. |
| Window | Starts at **1000 × 760** instead of 900 × 620. |
| Fall time | Each tile gets a random fall time of **16–24 s** (was 9–15 s at the old height). Per tick it moves `canvasHeight / (fallSeconds × 10)`, so the time to fall stays the same at any window height, also after a resize. The spawn interval stays at 4 s. |

## Status panel (`fyne/stats.go`)

### Session line

Three small tiles in one row: **correct** (sum of session correct counts), **missed**
(`StatsSnapshot.Missed`), **accuracy** (correct ÷ (correct + missed), rounded to the nearest
whole percent, shown as "–" with no answers yet). Score stays in the input bar only.

### Path line (one per active script)

- Script name (62 px), then 15 segments of fixed width (10 px, 2 px apart) in the same kind of
  fixed-width layout as the grid (one per `ProgressionStepsFor(script)` step), then
  "`unlocked`/15", where **unlocked = the number of steps whose rows are all selected**, in any
  order. A partly selected step is open and not counted.
- Segment states: **mastered** (green) – every row of the step is selected and mastered;
  **learning** (amber) – every row of the step is selected, not all mastered; **open** (neutral) –
  not every row is selected.
- Below it, one small line: "Next: " plus the labels of the rows the next unlock would add
  (`nextStepRows(script)`, i.e. the first incomplete step, which can differ from `unlocked + 1`
  when rows were selected out of order), e.g. "Next: D-row (だ) · B-row (ば)", or "All rows
  unlocked". Katakana labels show katakana, e.g. "D-row (ダ)".
- A row counts as mastered by the existing `isRowMastered` rule (≥ 80 % of its characters with
  ≥ 3 correct).

### Kana grid

- One grid per group, stacked with a 4 px gap between groups and no group labels. A group is
  shown while at least one of its rows is shown. A header line above the first grid labels the
  halves "ひらがな" and "カタカナ"; there is no a/i/u/e/o header.
- The grids use a small custom layout with **fixed column widths** instead of
  `container.NewGridWithColumns` (which sizes every column to the widest cell and adds theme
  padding): row label 20 px, five cells of 24 px, a 6 px spacer, five cells of 24 px, 2 px
  between columns and rows. That is 288 px wide; the panel's minimum width is 300 px including
  its padding.
- Row labels are `canvas.Text` (11 px), not `widget.Label`, so they add no inner padding.
- Cells sit in the vowel column given by `vowelColIndex` (yōon use a/u/o). A cell is a small
  rounded rectangle with the kana centred:
  - new: transparent fill, 1 px outline `#cdb99c`, text `#9b8a76`;
  - learning: fill `#e9c46a`, text `kanaTextColor`;
  - mastered: fill `#7fa36b`, text white;
  - missed this session: 2 px border `#cc4444`, any fill.
- Cells are a small custom widget with a fixed min size of 24 × 18 px: a `canvas.Rectangle`
  (corner radius 3) with a centred `canvas.Text`, 12 px for one glyph and 10 px for two-glyph
  yōon.
- The grids live in a `container.NewVScroll`, which fills the space between the path lines and
  the legend (`container.NewBorder`).
- Legend: one line, "new · learning · mastered · missed", with small swatches.

### Snapshot (`StatsSnapshot`, built in `fyne/game.go` under the lock)

`StatsSnapshot` keeps `SessionStats`, `SelectedRows`, `ActiveScripts`, `MissedKanas`, `Score`,
`ScoreLimit`, `Missed`, and gains:

- `TotalCorrect map[string]int` – stored + session correct count per character.
- `Paths map[kanacore.Script]PathStatus` for active scripts, with
  `PathStatus{Steps []StepState; Unlocked int; Next []string}` and
  `StepState` one of `StepOpen`, `StepLearning`, `StepMastered`.

GameState computes `Paths` with the same helpers progression uses (`ProgressionStepsFor`,
`isRowMastered`, `nextStepRows`), so the panel holds no learning logic of its own. The panel
only maps these values to widgets.

### Updating

Today the panel only refreshes after an Enter, a dialog or Play Again. A tile that falls off the
canvas changes `missed`, the accuracy and the missed borders, so that must refresh it too:

- `GameState` gains `onStatsChanged func()`, set once by `buildWindow` to a function that takes a
  snapshot and calls `statsPanel.Update` and `inputBar.Update`.
- `tick` remembers whether it recorded a miss; after unlocking, if it did and the hook is set, it
  calls `fyne.Do(gs.onStatsChanged)`, also on the miss that ends the game (harmless; the
  game-over dialog refreshes too). This also fixes the input bar's "Missed" label, which today
  lags in the same way. `buildWindow` sets the hook before `gs.Start`; it is not changed later.
  The test for it runs under `test.NewApp()`, where `fyne.Do` executes the call.

`StatsPanel.Update(snap)` refreshes cell states, row and group visibility, path lines and the
session line. Widgets are created once in `newStatsPanel` for all 27 base rows × 2 scripts and
shown/hidden, as today.

## Window and fall time

- `fyne/app.go`: `w.Resize(fyne.NewSize(1000, 760))`.
- `kanacore.Kana` loses `Speed`. `KanaTile` gains `fallSeconds float32`, set at spawn to a random
  value in [16, 24].
- `tick` moves each tile by `gs.canvasH / (tile.fallSeconds * 10)`. A `fallSeconds` ≤ 0 is
  treated as `minFallSeconds`, so a tile built without one can never divide by zero. The constants
  `minFallSeconds`, `maxFallSeconds` and `ticksPerSecond` (10, matching the 100 ms ticker) live
  next to the tick loop.
- Tests that build tiles directly (`tileAt`) set `fallSeconds`.

## Testing

- **Snapshot:**
  - `TotalCorrect` adds stored and session counts.
  - `Paths` for a state with vowels mastered, k+s selected and not mastered: steps
    `[mastered, learning, open, …]`, `Unlocked == 2`, `Next == [t n]`.
  - `Paths` has no entry for an inactive script; "all unlocked" gives `Next == nil`.
  - Rows selected out of order (vowels, k, s, g, z): `Unlocked == 3`, `Next == [t n]`.
- **Panel:**
  - A row selected only in hiragana shows hiragana cells and a blank katakana half.
  - A row selected in neither active script is hidden.
  - Cell state follows `TotalCorrect` (0 → new, 2 → learning, 3 → mastered).
  - A character in `MissedKanas` carries the missed border.
  - The path line of an inactive script is hidden.
  - The accuracy text is "–" with no answers, "89 %" for 34 correct / 4 missed and "67 %" for
    2 / 1 (rounding, not truncation).
  - A group whose rows are all hidden is hidden.
  - The grid's minimum width is 288 px with yōon rows visible.
  - Missing a tile in `tick` calls the stats hook once.
- **Fall time:**
  - With `canvasH = 600` and `fallSeconds = 20`, one tick moves a tile by 3 px.
  - After changing `canvasH` to 300, the same tile moves 1.5 px per tick.
  - Spawned tiles get a `fallSeconds` in [16, 24].
- The existing stats tests that reference removed parts (ACTIVE ROWS, MISSED, the old snapshot)
  are rewritten or deleted, not left failing.

## Out of Scope

- A difficulty setting for fall time or spawn rate (PLAN.md → Difficulty).
- Speeding up with mastery.
- Remembering the window size between starts.
