# Design: Spaced Repetition for Spawning

**Date:** 2026-10-12
**Status:** Approved (design discussion and spec review)
**Branch:** `feat/spaced-repetition` (from `main` after PR #10)

## Overview

Today a new tile picks a random character from the selected rows, so a learner who knows most
kana keeps seeing characters they recognise instantly. Spawning becomes **weighted by a
forgetting-curve model**: characters the learner is likely to have forgotten come more often,
secure ones less often but never not at all, and new characters are capped so they cannot
dominate. The model lives in its own package, so the vocabulary trainer can reuse it for words.

The model only decides **which kana fall**. The learning-state colours in the status panel, the
row unlock rule, the difficulty levels and the per-character fall factor keep counting correct
answers, as they do today.

## Decisions

| Topic | Decision |
|---|---|
| Model | Forgetting curve: stability `S` (days) and last review time per item. Retrievability `R = 0.9^(Δt / S)`, so after `Δt = S` recall is 90 %. |
| First correct answer | `S = 10 min` (`minStability`). |
| Further correct answer | `S ← min(S × (1.5 + 2 × (1 − R)), 365 days)`, using `R` at the moment of the answer. A hard recall (low `R`) grows `S` more. |
| Miss (tile fell through) | `S ← max(S × 0.3, minStability)`, and the last review time is set back so that `R = 0.5` right after the miss (`LastSeen = now − S × ln 0.5 / ln 0.9` days). A just-missed item therefore gets weight 0.5, ten times a secure one, and comes back soon. A miss on an item never answered correctly creates no memory state, so the item stays "new". |
| Selection | Two stages. With probability `min(0.4, newShare)` pick uniformly among **new** candidates; otherwise pick among **known** candidates weighted by `max(1 − R, 0.05)`. With no known candidates, always pick new; with no new ones, always known. |
| `newShare` | The number of new candidates divided by all candidates. With only 1 new of 20 candidates, the new one is not forced to 40 %. |
| Duplicates | Characters currently falling are excluded from the candidates while at least one other candidate remains. |
| Scope | Only the spawn choice. Colours, unlocks, difficulty and the char fall factor are unchanged. |

"New" means: no memory state (never answered correctly). "Known" means: has a memory state.

## Architecture

### Package `memory` (new, `memory/memory.go`)

No dependency on Fyne, kanacore or store; times are passed in.

```go
// State is what the model knows about one item.
type State struct {
	Stability float64   // days
	LastSeen  time.Time
}

const (
	MinStability = 10.0 / (24 * 60) // 10 minutes, in days
	MaxStability = 365.0
	NewCap       = 0.4
	WeightFloor  = 0.05
)

func Retrievability(s State, now time.Time) float64      // 0.9^(Δt/S), 1 when Δt ≤ 0
func Correct(s State, known bool, now time.Time) State    // known=false → first-answer rule; s ignored then
func Miss(s State, now time.Time) State                   // only for known items
func Weight(s State, now time.Time) float64               // max(1 − R, WeightFloor)
func Seed(correct int, now time.Time) State               // stability from a correct count, for migration

// Pick chooses one candidate. states holds the known ones; candidates
// missing from states are new. rnd returns a float in [0, 1).
func Pick(candidates []string, states map[string]State, now time.Time, rnd func() float64) string
```

`Correct` and `Seed` set `LastSeen = now`; `Miss` sets it so that `R = 0.5` at `now`. Δt in days is
`now.Sub(LastSeen).Hours() / 24`. `Seed` gives `min(MinStability × 2^min(correct, 10), MaxStability)`. `Pick` returns `""` only for an empty candidate list.
It is deterministic for a given `rnd` sequence, which the tests rely on.

### Store (`store/store.go`)

A new table, created by `migrate` (inside the seeding transaction below), keyed by item ID so
words can share it later:

```sql
CREATE TABLE IF NOT EXISTS memory (
	item      TEXT PRIMARY KEY,     -- "kana:か", later "word:…"
	stability REAL NOT NULL,        -- days
	last_seen INTEGER NOT NULL      -- unix seconds
);
```

- `MemoryStates(prefix string) (map[string]memory.State, error)`: all items with the prefix,
  with the prefix stripped from the keys, e.g. `"か"`.
- `SaveMemoryState(item string, s memory.State) error`: upsert.
- `last_seen` is stored as unix seconds (sub-second precision is dropped).
- **Seeding, once:** after the existing statements (the `PRAGMA journal_mode` cannot run in a
  transaction), `migrate` runs one transaction that
  1. checks `SELECT name FROM sqlite_master WHERE type='table' AND name='memory'`;
  2. if absent, creates the table and inserts one row per `kana_stats` row with
     `correct_count > 0`, using `memory.Seed(correct_count, now)` (so `kana:` + char,
     stability, `last_seen` = now);
  3. commits.
  If anything fails, the transaction rolls back and the next open seeds again. Existing
  progress therefore starts as "known", with a stability that grows with how often the
  character was answered (10 correct gives about 7 days).

`store` imports `memory` for the `State` type; `memory` imports nothing from the app.

### GameState (`fyne/game.go`)

- New field `memory map[string]memory.State` (keyed by character), loaded in `NewGameState` from
  `MemoryStates("kana:")` (when the store is set). It defaults to an empty map. `Reset` does
  **not** reload it: the in-memory map is always current, and reloading could drop states not
  yet written.
- `memoryDirty map[string]bool` marks changed characters.
- `recordCorrect(char)`: `st, known := gs.memory[char]`; `gs.memory[char] = memory.Correct(st, known, gs.now())`;
  mark dirty.
- `recordMiss(char)`: only if a state exists, `gs.memory[char] = memory.Miss(...)`; mark dirty.
- `mergeSessionStats` first writes every dirty memory state with
  `SaveMemoryState("kana:"+char, …)` (with a store), clearing each mark that saved, **before**
  its existing early returns, so a failing stats read cannot drop memory. It is called at game
  end, Play Again and window close, as today.
- `spawnKana`: build the candidates from `availableCharacters()` minus the characters of tiles
  currently on the canvas (kept if that would leave none), then
  `char := memory.Pick(candidates, gs.memory, gs.now(), rand.Float64)`.
- `gs.now` is a `func() time.Time` field (default `time.Now`), so tests can control time.
  `newTestState()` in `fyne/game_test.go` builds `&GameState{…}` directly, so it must also
  initialise `memory`, `memoryDirty` and `now`.

## Testing

- **`memory`:**
  - `Retrievability` is 1 at Δt = 0, 0.9 at Δt = S, about 0.81 at Δt = 2S.
  - `Correct` on a new item gives `minStability`.
  - `Correct` at R = 0.9 grows S by 1.7×; at R ≈ 0.5 it grows S by about 2.5×.
  - `S` is capped at 365 days.
  - `Miss` multiplies S by 0.3, with a floor at `minStability`, and leaves `R = 0.5` (weight 0.5) at the time of the miss.
  - `Weight` is `1 − R` with a floor of 0.05.
  - `Pick`, driven by a seeded `math/rand` source:
    - with 10 new and 10 known (all at R ≈ 1), the new share over 10,000 draws is 0.40 ± 0.02;
    - with 1 new and 19 known, the new share is about 1/20 (not 40 %);
    - among known items, one at R = 0.5 is drawn about 10× as often as one at R ≈ 1 (weight
      0.5 vs 0.05), within tolerance;
    - only new candidates → always new; only known → always known; empty → `""`.
- **Store:**
  - `SaveMemoryState` and `MemoryStates` round-trip and strip the prefix (compare
    `LastSeen.Unix()`, not `time.Time` values).
  - A database with existing `kana_stats` (correct 0 and 3) gets a seeded memory row only for the
    character with correct 3, with stability `memory.MinStability × 8`. Re-opening does not
    re-seed or overwrite a changed row. The tests check stability and row presence, not
    `last_seen`.
  - `memory.Seed(10, now).Stability` ≈ 7.1 days; `Seed` stops growing after 10 correct answers.
- **GameState:**
  - A correct answer creates a memory state for a new character.
  - A miss on a new character creates none; a miss on a known one lowers S.
  - `mergeSessionStats` persists dirty states.
  - `Reset` keeps memory states that are already in the map.
  - `spawnKana` never spawns a character already falling when another candidate exists.
  - With a fixed `now` and memory states set, spawning over many draws prefers the weak
    character. Use the `Pick` statistics, not the GUI.

## Out of Scope

- Changing the status-panel colours or the unlock rule to use retrievability.
- Showing "due" counts or a review queue.
- Tuning the constants from data (FSRS-style optimisation).
- Words (vocabulary trainer, next).
