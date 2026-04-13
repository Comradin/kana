# Design: Fyne Desktop Rewrite

**Date:** 2026-04-13  
**Status:** Approved  

## Overview

Rewrite the terminal-based kana typing game as a Fyne desktop application with proper graphical kana tiles, while preserving all existing game mechanics, progression logic, and SQLite persistence.

## Visual Design

### Kana Tiles
Each falling tile uses the **Paper/Stamp** style: a parchment-coloured rounded card with a slight 3D shadow offset in darker brown. Only the kana character is shown — no romaji hint, as showing it would contradict the learning objective. Tile colours are read from the active theme rather than hardcoded.

**Note on Fyne rendering:** `canvas.Rectangle` does not support rounded corners or gradient fills in Fyne v2. Tiles are rendered as flat-filled rectangles (no gradient). The shadow is a second `canvas.Rectangle` offset by (+3, +3) device-independent pixels. Rounded corners can be approximated by rasterising to an `image.RGBA` wrapped in `canvas.NewImageFromImage` if desired in a future pass; the initial implementation uses flat fills.

### Layout
**Left/right split** using `container.NewBorder`:
- **Centre** (left portion): `GameCanvas` — the falling tile play field
- **Right** (fixed ~280px): `StatsPanel` — hiragana progress grid, active rows, missed characters
- **Bottom**: `InputBar` — score label, romaji text input, missed counter, gear icon

### Theme
Default theme: **Warm Paper**
- Background: `#f0e6d3`
- Tile face: `#eedfc0` (flat fill, no gradient)
- Tile shadow: `#b8956a`
- Kana text: `#2c1a0e`
- Stats panel bg: `#e8dbc8`
- Input bar bg: `#e0d4bc`
- Accent: `#8b5e3c`
- Miss colour: `#c44`

The `KanaTheme` struct implements `fyne.Theme` and holds named colour palettes, enabling future themes (Dark Ink, Slate Blue) to be added without changing rendering code.

## Module & Package Structure

The `fyne/` subdirectory is part of the **same Go module** (`module kana`). Because `kana.go` and `kana_rows.go` are currently declared `package main`, they cannot be imported by the `fyne/` package directly.

**Required refactor:** Move shared types and data into a new `kanacore/` package before writing any Fyne code:

```
kana/
├── go.mod               ← module kana (unchanged)
├── main.go              ← terminal entry point; imports kanacore
├── game.go              ← terminal model/update; imports kanacore
├── settings_form.go     ← terminal settings; imports kanacore
├── ui.go                ← terminal rendering; imports kanacore
├── store/store.go       ← SQLite persistence (unchanged)
├── kanacore/
│   ├── kana.go          ← Kana, CharacterSet, Hiragana() — moved from root
│   └── kana_rows.go     ← KanaRow, AllKanaRows, charToRow — moved from root
└── fyne/
    ├── main.go          ← desktop entry point
    ├── app.go           ← fyne.App setup, window creation, theme wiring
    ├── game.go          ← GameState struct + game loop goroutines
    ├── canvas.go        ← GameCanvas widget (BaseWidget + custom renderer)
    ├── tile.go          ← KanaTile: parchment card as canvas objects
    ├── stats.go         ← StatsPanel widget (right column)
    ├── input.go         ← InputBar widget (bottom bar)
    ├── settings.go      ← Settings dialog (gear icon → dialog.Custom)
    └── theme.go         ← KanaTheme implementing fyne.Theme
```

Both entry points import `kana/kanacore` and `kana/store`. The terminal files (`main.go`, `game.go`, `ui.go`, `settings_form.go`) need minimal updates: change `package main` declarations that reference the moved types to import `kanacore` instead. Game logic in `game.go` stays in the root package; only the data definitions move.

## Game Loop

The Bubble Tea tick/spawn model is replaced by two goroutines plus an event channel:

**Tick goroutine** (100ms interval):
1. Lock `GameState.mu`
2. Move all tiles down by `speed` device-independent pixels per tick
3. Remove tiles whose Y position exceeds the canvas height, record misses, check game-over
4. If `over` was just set, send to `GameState.eventCh`
5. Unlock
6. Call `gameCanvas.Refresh()` — safe to call from any goroutine in Fyne v2

**Spawn goroutine** (4s interval):
1. Lock `GameState.mu`
2. Pick a random character from the available set
3. Append a new `KanaTile` at random X within canvas bounds, Y=0
4. Unlock
5. Call `gameCanvas.Refresh()`

Both goroutines are stopped via a `stopCh chan struct{}` that is closed when the game ends or the app quits.

**Event channel** — `GameState.eventCh chan gameEvent` is a buffered channel (size 4). Events include `gameOverEvent`. A watcher goroutine started by `app.go` reads from this channel and dispatches UI actions by calling `go func() { dialog.ShowCustom(...) }()` — Fyne's dialog calls schedule themselves onto the main thread internally, so this pattern is safe from any goroutine.

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
    missedKanas   []kanacore.Kana
    sessionStats  map[string]store.KanaStats
    overallStats  map[string]store.KanaStats
    currentStreak map[string]int
    sessionDirty  bool
    selectedRows  map[string]bool
    autoProgress  bool
    newlyUnlocked []string        // row IDs unlocked this session
    unlockMessage string          // text for the unlock notification
    unlockAt      time.Time       // when unlockMessage was set
    store          *store.Store
    stopCh         chan struct{}    // closed to stop goroutines
    eventCh        chan gameEvent  // game events dispatched to UI
    objectSnapshot atomic.Value   // []fyne.CanvasObject, written by tick, read by renderer
}
```

All existing game logic (`checkAnswer`, `spawnKana`, `recordCorrect`, `recordMiss`, `checkAutoProgression`, `isRowMastered`, `mergeSessionStats`, `showUnlockMessage`) is ported from the root `game.go` as methods on `GameState`, using `kanacore` types.

**Stats double-counting fix:** The existing terminal `recordCorrect` increments `overallStats` immediately *and* `mergeSessionStats` later adds `sessionStats` on top, double-counting correct answers in the persisted total. The Fyne version fixes this:

- `recordCorrect` updates only `sessionStats` and `currentStreak` in memory; it does **not** touch `overallStats`.
- `mergeSessionStats` loads the persisted baseline by calling `store.KanaStatistics()` — **not** from `GameState.overallStats` — and writes `baseline.CorrectCount + session.CorrectCount` back to the store. This avoids accumulating on an already-incremented in-memory total.
- `GameState.overallStats` is used only for `checkAutoProgression` during a session. It is reloaded from the store at the start of each new game (in `Reset()`) to ensure it reflects the true persisted baseline.

## UI Components

### KanaTile (`tile.go`)

Three canvas objects per tile, all positioned in device-independent pixels:

```go
type KanaTile struct {
    kana   kanacore.Kana
    pos    fyne.Position  // top-left of the face rectangle
    shadow *canvas.Rectangle
    face   *canvas.Rectangle
    text   *canvas.Text
}
```

- `shadow`: `canvas.Rectangle`, fill `#b8956a`, size `(tileW, tileH)`, positioned at `pos.Add(fyne.NewPos(3, 3))`
- `face`: `canvas.Rectangle`, fill `#eedfc0`, size `(tileW, tileH)`, positioned at `pos`
- `text`: `canvas.Text`, the kana character, colour `#2c1a0e`, centred within the face

`tileW = 52`, `tileH = 60` (device-independent pixels). Text size is set to 32.

`KanaTile.Move(pos fyne.Position)` updates all three objects' positions atomically.

`KanaTile.Objects() []fyne.CanvasObject` returns `[shadow, face, text]` for the renderer.

### GameCanvas (`canvas.go`)

Embeds `widget.BaseWidget`. Implements `fyne.Widget` (which satisfies `fyne.CanvasObject`).

```go
type GameCanvas struct {
    widget.BaseWidget
    state *GameState
}
```

`CreateRenderer()` returns a `gameCanvasRenderer` implementing `fyne.WidgetRenderer`:

```go
type gameCanvasRenderer struct {
    canvas *GameCanvas
    bg     *canvas.Rectangle
}

func (r *gameCanvasRenderer) Layout(size fyne.Size) { r.bg.Resize(size) }
func (r *gameCanvasRenderer) MinSize() fyne.Size    { return fyne.NewSize(200, 300) }
func (r *gameCanvasRenderer) Refresh()              { r.bg.FillColor = theme.BackgroundColor(); canvas.Refresh(r.bg) }
func (r *gameCanvasRenderer) Destroy()              {}
func (r *gameCanvasRenderer) Objects() []fyne.CanvasObject {
    // Do NOT acquire mu here — Objects() is called on the Fyne render thread,
    // and OnSubmitted also runs on the main thread holding mu, which would deadlock.
    // Instead, the tick goroutine maintains r.canvas.state.objectSnapshot, a
    // []fyne.CanvasObject slice that is atomically swapped (via atomic.Value)
    // after each tick before calling Refresh(). Objects() reads only the snapshot.
    snap, _ := r.canvas.state.objectSnapshot.Load().([]fyne.CanvasObject)
    if snap == nil {
        return []fyne.CanvasObject{r.bg}
    }
    return snap
}
```

`GameCanvas.Resize(size fyne.Size)` stores the canvas dimensions in `GameState` (under the mutex) so the tick goroutine can use them for boundary checks and spawn X randomisation.

After each tick, the tick goroutine builds the full `[]fyne.CanvasObject` slice (background + all tile objects) under the mutex and stores it via `GameState.objectSnapshot.Store(objs)` (an `atomic.Value`). This snapshot is what `Objects()` reads, avoiding any mutex acquisition on the render thread.

### Tile Coordinate System

All positions are **Fyne device-independent pixels** (float32). The game area dimensions (`canvasW`, `canvasH`) are stored in `GameState` and updated on each `GameCanvas.Resize()` call. 

- Spawn X: `rand.Float32() * (canvasW - tileW)`
- Spawn Y: `0`
- Speed: pixels per tick. At a typical window height of ~600px with ~24 terminal rows, one character cell ≈ 25px. The original speed range (0.15–0.25 cells/tick) therefore translates to approximately **3.75–6.25 px/tick**. Starting value: `3.75 + rand.Float32()*2.5`.
- A tile is considered missed when `pos.Y > canvasH`.

### StatsPanel (`stats.go`)

Vertical `container.VScroll` wrapping a `container.VBox` containing:
- "HIRAGANA PROGRESS" header label
- A `container.NewGridWithColumns(5)` of character count labels
- "ACTIVE ROWS" section
- "MISSED CHARACTERS" section
- Unlock notification label (shown for 5 seconds after `unlockAt`, then cleared on next refresh)

`StatsPanel.Update(snapshot StatsSnapshot)` takes a value-copy snapshot and refreshes all labels. Never holds a reference to `GameState` directly.

```go
type StatsSnapshot struct {
    SessionStats  map[string]store.KanaStats // shallow-copied map (values are structs)
    SelectedRows  map[string]bool            // shallow-copied map
    MissedKanas   []kanacore.Kana            // deep-copied: append([]kanacore.Kana{}, gs.missedKanas...)
    Score         int
    ScoreLimit    int
    Missed        int
    UnlockMessage string
    UnlockAt      time.Time
}
```

The caller builds `StatsSnapshot` while holding `GameState.mu`, then releases the lock before calling `statsPanel.Update(snapshot)`. Slice fields must be deep-copied (not just header-copied) since the underlying arrays may be modified by subsequent ticks.

### InputBar (`input.go`)

`container.NewBorder` with:
- Left: score `widget.Label`
- Right: missed counter `widget.Label`
- Centre: romaji `widget.Entry`
- Far right: gear `widget.Button` (⚙)

`Entry.OnSubmitted`: lock `GameState.mu`, call `checkAnswer(text)`, unlock, call `gameCanvas.Refresh()`, call `statsPanel.Update(snapshot)`, clear entry.

### Settings Dialog (`settings.go`)

`dialog.ShowCustom` opened by the gear button. Content is a `container.VBox` with:
- `widget.CheckGroup` — kana row selection (options from `kanacore.AllKanaRows`)
- `widget.Check` — auto-progression
- `widget.Entry` — score limit (numeric, validated)

On confirm:
1. Lock `GameState.mu`
2. Apply new `selectedRows`, `autoProgress`, `scoreLimit`
3. **Remove any in-flight tiles whose character belongs to a now-deselected row** — iterate `tiles`, filter out tiles where `charToRow[tile.kana.Char]` is no longer in `selectedRows`
4. Unlock
5. Persist via `store` methods
6. Call `gameCanvas.Refresh()` and `statsPanel.Update(snapshot)`

The game continues running while the dialog is open.

## Game Over

When `GameState.over` is set true by the tick goroutine, a `gameOverEvent` is sent to `eventCh`. The watcher goroutine receives it and calls `go func() { dialog.ShowCustom(..., w) }()` — Fyne dialog calls schedule themselves onto the main thread internally, so this is safe from any goroutine. The dialog contains:
- Title: "GAME OVER" or "SESSION COMPLETE"
- Final score (with limit if set)
- Missed count
- Reason text
- List of unique missed characters with their romaji

Two buttons:
- **Play Again**: calls `GameState.Reset()`, which must: (1) close the existing `stopCh` to stop the old goroutines, (2) drain or discard any pending events on `eventCh`, (3) reload `overallStats` from `store.KanaStatistics()`, (4) reinitialise all state fields, (5) create a new `stopCh`, (6) restart the tick and spawn goroutines, then refresh the canvas and stats panel.
- **Quit**: calls `mergeSessionStats()`, then `w.Close()`.

## Persistence

`store/store.go` is used without modification. The desktop app opens `kana.db` in the working directory (same as the terminal version). On launch, `GameState` is populated from the store. Stats are flushed to SQLite in `mergeSessionStats`, called at game end or quit.

## Theming Architecture

```go
type KanaTheme struct {
    name   string
    colors map[themeColorKey]color.Color
}
```

`KanaTheme` implements `fyne.Theme`. All components resolve colours from the active theme. A theme selector can be added to the settings dialog later; switching theme calls `app.Settings().SetTheme(newTheme)` and refreshes all widgets.

## Out of Scope

- Katakana support
- Sound effects
- Animated tile effects (rounded corners, gradients, particles) — possible later via raster images
- Cloud sync of stats
