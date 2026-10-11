# Design: Difficulty

**Date:** 2026-10-11
**Status:** Approved (design discussion and spec review)
**Branch:** `feat/difficulty` (from `main` after PR #9)

## Overview

The game is too easy for an experienced learner. Difficulty becomes a chosen **level** that
sets fall time, spawn interval and the miss limit, plus a **per-character factor** (known
characters fall faster) and a **session tempo** that rises with streaks of correct answers and
drops on misses. A configurable **points factor** lets learners shape their own game.

## Decisions

### Levels

| Level | Fall time | New tile every | Misses allowed | Auto points factor |
|---|---|---|---|---|
| Relaxed | 18–26 s | 4.5 s | 10 | ×1 |
| Normal (today) | 16–24 s | 4.0 s | 10 | ×1 |
| Fast | 10–15 s | 2.5 s | 7 | ×2 |
| Pro | 6–9 s | 1.6 s | 5 | ×3 |

- The level is chosen in the settings and stored. If unset or unknown, it is **Normal**, which
  matches today's game exactly, so existing learners see no change until they pick another level.
- Changing the level in the settings takes effect for tiles spawned from then on and for the
  miss limit immediately. If the misses so far already reach the new limit, the game ends on
  the next miss, not instantly.

### Per-character factor

The fall time of a new tile is multiplied by a factor from the character's learning state (same
rule as the status panel: total correct = stored + session):

| State | Correct answers | Fall-time factor |
|---|---|---|
| new | 0 | ×1.2 (more time) |
| learning | 1–2 | ×1.0 |
| mastered | ≥ 3 | ×0.7 (less time) |

### Session tempo

- `tempoStep` starts at 0 each game (new game and Play Again).
- Every **5 correct answers in a row** raise `tempoStep` by 1, up to **`maxTempoStep = 10`**
  (0.95¹⁰ ≈ 0.6, where the floor is reached). A miss resets the streak and lowers `tempoStep`
  by 2, not below 0, so every miss slows the game noticeably even after a long streak.
- Tempo factor = `max(0.95 ^ tempoStep, 0.6)`. It multiplies both the fall time of
  newly spawned tiles and the spawn interval. Tiles already falling keep their fall time.
- The tempo shown to the player is the inverse speed-up `1 / factor`, formatted `%.2f`, so
  step 0 shows "×1.00", then "×1.05", "×1.11", … up to "×1.67".
- A level change in the settings applies from the next spawn timer re-arm; the interval
  already pending keeps its length.

### Points

- Settings field **"Points per hit"** with **Auto (by level)**, ×1, ×2, ×3, ×5, stored.
  Default is Auto.
- A correct answer scores `10 × factor`, where Auto uses the level's factor from the table.
- The score limit stays as it is (default 1000, 0 = endless).

### Display

- The input bar shows the level and tempo next to the score, e.g. `Score: 240/1000 · Fast ·
  Tempo ×1.15`, and the missed counter uses the level's limit, e.g. `Missed: 2/5`.
- The game-over dialog shows `Missed: n/limit` and the reason "n kana slipped through.", both
  with `n = snap.Missed` (the true count; it can exceed the limit after a mid-game level change)
  and the level's limit (both hard-code 10 today). The text comes from
  `missReason(missed int) string`.
- The settings save path and the initial render in `buildWindow` both call
  `inputBar.Update(snap)` as well as `statsPanel.Update(snap)`, so level, tempo and miss limit
  show at once (on start, a stored Fast or Pro must not show "Missed: 0/10").

## Architecture

### `fyne/difficulty.go` (new)

```go
type Level string // "relaxed", "normal", "fast", "pro"

type LevelInfo struct {
	Level          Level
	Label          string  // "Relaxed", "Normal", "Fast", "Pro"
	MinFall        float32 // seconds
	MaxFall        float32
	SpawnEvery     time.Duration
	MissLimit      int
	PointsFactor   int     // used when the points setting is Auto
}

func Levels() []LevelInfo          // display order
func LevelByName(s string) LevelInfo // unknown → Normal

const pointsAuto = 0 // the stored "Auto" value

func charFallFactor(correct int) float32 // 1.2 / 1.0 / 0.7
func tempoFactor(step int) float32       // max(0.95^step, 0.6)
func validPoints(p int) int              // 0,1,2,3,5 kept; anything else → 0 (Auto)
func missReason(missed int) string       // "5 kana slipped through." 
func formatStatus(score, limit int, level string, tempo float32) string
	// "Score: 240/1000 · Fast · Tempo ×1.15"; limit 0 → "Score: 240 · Fast · Tempo ×1.15"
```

These are pure functions with table tests.

### GameState (`fyne/game.go`)

- New fields: `level LevelInfo`, `pointsSetting int` (0 = Auto, else 1/2/3/5),
  `answerStreak int` (named apart from the per-character `currentStreak`), `tempoStep int`.
- **Loading:** `NewGameState` sets `level = LevelByName("")` (Normal) before the `st != nil`
  block, so a nil store also gets Normal, then reads `Difficulty()` and `PointsFactor()`
  (passed through `validPoints`). `newTestState` in `fyne/game_test.go` sets the same default
  level, so the existing tick and miss tests keep a limit of 10.
- `minFallSeconds` / `maxFallSeconds` are removed. `fallStep(h, fallSeconds, fallback float32)`
  gains a fallback parameter (the level's `MinFall`) for its ≤ 0 guard. `TestFallStep` and
  `TestSpawnSetsFallTimeInRange` are updated accordingly.
- **Spawn:** `fallSeconds = rand(level.MinFall, level.MaxFall) × charFallFactor(c) × tempoFactor(tempoStep)`,
  where `c = overallStats[char].CorrectCount + sessionStats[char].CorrectCount`, computed inline
  rather than building the whole `totalCorrect()` map.
- **Spawn loop:** the fixed 4 s ticker becomes a `time.Timer` (`defer timer.Stop()`) re-armed
  after every spawn attempt with `gs.spawnInterval()` = `level.SpawnEvery × tempoFactor(tempoStep)`
  (read under the lock). The first spawn comes one interval after start, as today. It still
  stops on `stopCh`; `Reset` closes the old `stopCh` and `Start` launches a fresh loop, as today.
  While paused or over, `spawnKana` returns early and the timer is simply re-armed.
- `pointsFactor()` (under lock) = `level.PointsFactor` if `pointsSetting == pointsAuto`, else
  `pointsSetting`.
- **Answers:** `checkAnswer` adds `10 × pointsFactor()`; `recordCorrect` increments
  `answerStreak` and, every 5, `tempoStep` (capped at `maxTempoStep`).
- **Misses:** `recordMiss` sets `answerStreak = 0` and `tempoStep = max(0, tempoStep-2)`.
  `checkMissedLimit` uses `level.MissLimit`.
- `Reset` sets `answerStreak` and `tempoStep` to 0.
- `applySettings` gains `level Level` and `points int` parameters, applied (points through
  `validPoints`) and persisted under the same lock. Its existing test call sites are updated to
  pass `"normal"` and `pointsAuto`.
- `StatsSnapshot` gains `LevelLabel string`, `Tempo float32` (the shown speed-up, `1/factor`)
  and `MissLimit int`.

### Store (`store/store.go`)

Two settings keys, no schema change:

| Key | Methods | Default |
|---|---|---|
| `difficulty` | `Difficulty() (string, error)`, `SaveDifficulty(string) error` | `""` (unset; GameState maps it to Normal) |
| `points_factor` | `PointsFactor() (int, error)`, `SavePointsFactor(int) error` | `0` (Auto) |

### Settings dialog (`fyne/settings.go`)

Below the scripts row and tabs, above auto-progression: a "Difficulty" `widget.RadioGroup`
(Relaxed / Normal / Fast / Pro, horizontal) and a "Points per hit" `widget.Select`
(Auto (by level) / ×1 / ×2 / ×3 / ×5). Both feed `applySettings`. The dialog must still fit the
1000 × 760 window; if it does not, reduce the tab scroll height.

### Input bar and game-over dialog

`InputBar.Update` and `OnSubmitted` format the score line through `formatStatus` and use
`snap.MissLimit` for the missed label. The game-over dialog uses `snap.MissLimit` for both
the count and the reason text.

## Testing

- `difficulty.go`:
  - `LevelByName` maps all four names and falls back to Normal.
  - `charFallFactor` returns 1.2 / 1.0 / 0.7 at 0 / 2 / 3 correct.
  - `tempoFactor(0) == 1`, `tempoFactor(1) == 0.95`, `tempoFactor(10) == 0.6` (floor).
  - `validPoints` keeps 0/1/2/3/5 and maps 4, -1 and 7 to 0.
  - `formatStatus(240, 1000, "Fast", 1.15)` == `"Score: 240/1000 · Fast · Tempo ×1.15"` and
    `formatStatus(240, 0, "Normal", 1)` == `"Score: 240 · Normal · Tempo ×1.00"`.
- GameState:
  - Spawned fall time stays within the level's range × char factor × tempo. Use a mastered
    character on Pro with `tempoStep = 0`, so the range is [4.2, 6.3].
  - Five correct answers raise `tempoStep` to 1; four don't. A miss after that lowers it to 0,
    never below. 60 correct answers in a row leave `tempoStep` at 10 (cap), and one miss then
    gives 8.
  - Points: Normal + Auto gives +10, Pro + Auto gives +30, Pro + ×1 gives +10, Relaxed + ×5
    gives +50.
  - The miss limit follows the level: on Pro the game ends at the 5th miss.
  - `Reset` clears streak and tempo.
  - `applySettings` stores the level and the points factor.
  - Unknown stored level → Normal.
  - The spawn interval used after a step is `level.SpawnEvery × tempoFactor`. Test the helper
    that computes it, not the timer.
- Store: both keys round-trip and default when unset.
- Input bar: covered by the `formatStatus` test; the missed label on Pro reads "Missed: 2/5"
  (tested through `InputBar.Update` with a full bar, as `pause_test.go` builds one).
- Game over: `missReason(5) == "5 kana slipped through."`.

## Out of Scope

- Speed-ups based on the learning path step (only per-character state and session tempo).
- High scores per level.
- The vocabulary trainer (next milestone).
