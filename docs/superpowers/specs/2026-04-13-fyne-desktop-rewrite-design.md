# Design: Fyne Desktop Rewrite

**Date:** 2026-04-13  
**Status:** Approved  

## Overview

Rewrite the terminal-based kana typing game as a Fyne desktop application with proper graphical kana tiles, while preserving all existing game mechanics, progression logic, and SQLite persistence.

## Visual Design

### Kana Tiles
Each falling tile uses the **Paper/Stamp** style: a parchment-coloured rounded card with a slight 3D shadow offset in darker brown. Only the kana character is shown — no romaji hint, as showing it would contradict the learning objective. Tile colours are read from the active theme rather than hardcoded.

### Layout
**Left/right split** using `container.NewBorder`:
- **Centre** (left portion): `GameCanvas` — the falling tile play field
- **Right** (fixed ~280px): `StatsPanel` — hiragana progress grid, active rows, missed characters
- **Bottom**: `InputBar` — score label, romaji text input, missed counter, gear icon

### Theme
Default theme: **Warm Paper**
- Background: `#f0e6d3`
- Tile face gradient: `#e8d5b7` → `#f5e6c8`, shadow `#b8956a`
- Kana text: `#2c1a0e`
- Stats panel bg: `#e8dbc8`
- Input bar bg: `#e0d4bc`
- Accent: `#8b5e3c`
- Miss colour: `#c44`

The `KanaTheme` struct implements `fyne.Theme` and holds named colour palettes, enabling future themes (Dark Ink, Slate Blue) to be added without changing rendering code.

## Architecture

### Package Structure

```
kana/
├── main.go              ← terminal entry point (unchanged)
├── game.go              ← terminal model/update (unchanged)
├── kana.go              ← shared character data (unchanged)
├── kana_rows.go         ← shared row definitions (unchanged)
├── store/store.go       ← shared SQLite persistence (unchanged)
├── fyne/
│   ├── main.go          ← desktop entry point
│   ├── app.go           ← fyne.App setup, window creation, theme wiring
│   ├── game.go          ← GameState struct + game loop goroutines
│   ├── canvas.go        ← GameCanvas widget (BaseWidget + custom renderer)
│   ├── tile.go          ← KanaTile: parchment card as canvas objects
│   ├── stats.go         ← StatsPanel widget (right column)
│   ├── input.go         ← InputBar widget (bottom bar)
│   ├── settings.go      ← Settings dialog (gear icon → dialog.Custom)
│   └── theme.go         ← KanaTheme implementing fyne.Theme
```

The terminal build and desktop build share `store/`, `kana.go`, and `kana_rows.go` directly. They are built separately: `go build .` for the terminal app, `go build ./fyne` for the desktop app.

## Game Loop

The Bubble Tea tick/spawn model is replaced by two goroutines:

**Tick goroutine** (100ms interval):
1. Lock `GameState.mu`
2. Move all tiles down by their speed
3. Remove tiles that pass the bottom edge, record misses, check game-over
4. Unlock
5. Call `canvas.Refresh()` on the `GameCanvas`

**Spawn goroutine** (4s interval):
1. Lock `GameState.mu`
2. Pick a random character from available set
3. Append a new `KanaTile` at random X, Y=0
4. Unlock

Both goroutines stop when `GameState.over` is set. All UI mutations called from goroutines are wrapped in `fyne.Do` where required for thread safety.

### GameState

```go
type GameState struct {
    mu            sync.Mutex
    tiles         []*KanaTile
    score         int
    scoreLimit    int
    missed        int
    over          bool
    overReason    string
    missedKanas   []Kana
    sessionStats  map[string]store.KanaStats
    overallStats  map[string]store.KanaStats
    currentStreak map[string]int
    sessionDirty  bool
    selectedRows  map[string]bool
    autoProgress  bool
    store         *store.Store
}
```

All existing game logic (`checkAnswer`, `spawnKana`, `recordCorrect`, `recordMiss`, `checkAutoProgression`, `isRowMastered`, `mergeSessionStats`) is ported from `game.go` as methods on `GameState`.

## UI Components

### KanaTile (`tile.go`)
Three canvas objects per tile:
- `canvas.Rectangle` — shadow offset (+3px, +3px), fill `#b8956a`, rounded
- `canvas.Rectangle` — face, gradient fill, rounded corners
- `canvas.Text` — kana character, large, centred, colour `#2c1a0e`

Position is updated directly on the objects each tick. No widget overhead.

### GameCanvas (`canvas.go`)
Implements `fyne.CanvasObject` via `widget.BaseWidget`. Custom renderer's `Objects()` returns the background rectangle followed by all live tile objects. On resize, game area dimensions update and tile X positions are clamped to stay in bounds.

### StatsPanel (`stats.go`)
Vertical `container.VBox` containing:
- "HIRAGANA PROGRESS" header
- Grid of character labels (session correct counts, styled by the theme)
- "ACTIVE ROWS" section listing selected row labels
- "MISSED CHARACTERS" section

Refreshed by calling `statsPanel.Refresh()` after each correct answer or miss. Reads a snapshot of stats passed from `GameState` — does not hold a reference to the mutex-protected state.

### InputBar (`input.go`)
Horizontal `container.HBox`:
- Score `widget.Label` (left)
- Romaji input `widget.Entry` (centred, stretches)
- Missed counter `widget.Label` (right)
- Gear `widget.Button` (opens settings dialog)

`Entry.OnSubmitted` calls `gameState.checkAnswer(text)`, clears the entry, and refreshes the stats panel.

### Settings Dialog (`settings.go`)
`dialog.Custom` opened by the gear button, containing:
- `widget.CheckGroup` — kana row selection
- `widget.Check` — auto-progression toggle
- `widget.Entry` (numeric, validated) — score limit

On confirm: applies changes to live `GameState` and persists via existing `store` methods. The game continues running while the dialog is open.

## Game Over

When `GameState.over` becomes true, both goroutines exit. A `dialog.Custom` is shown over the game window with: title (GAME OVER / SESSION COMPLETE), final score, missed count, reason, and list of missed characters. Two buttons: **Play Again** (resets state, restarts goroutines) and **Quit** (saves session stats, closes window).

## Persistence

`store/store.go` is used without modification. The desktop app opens the same `kana.db` file. On launch, settings and overall stats are loaded into `GameState`. On correct/miss, stats are updated in memory immediately. On game end (or quit), `mergeSessionStats` flushes to SQLite — same behaviour as the terminal version.

## Theming Architecture

```go
type KanaTheme struct {
    name   string
    colors map[themeColorKey]color.Color
}
```

`KanaTheme` implements `fyne.Theme`. All canvas objects and widgets resolve colours via `theme.Color(key, variant)`. Adding a new theme is defining a new `colors` map. A theme selector can be added to the settings dialog later without changing any rendering code.

## Out of Scope

- Katakana support (not in current terminal version)
- Sound effects
- Animated tile effects (particle bursts etc.) — possible later via Option C custom renderer
- Cloud sync of stats
