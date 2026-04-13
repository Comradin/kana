# Fyne Desktop Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the kana typing game as a Fyne v2 desktop application with paper/stamp kana tiles, left/right split layout, in-game settings, and full SQLite persistence — sharing all game data and store logic with the existing terminal app.

**Architecture:** Extract shared types into a new `kanacore/` package so both the terminal and desktop entry points can import them. The Fyne app lives in `fyne/` as a separate binary within the same Go module. Game state is mutex-guarded; tile positions are pushed to an `atomic.Value` snapshot after each tick so the Fyne renderer never needs to acquire the mutex.

**Tech Stack:** Go 1.25, `fyne.io/fyne/v2` (GUI), `modernc.org/sqlite` (persistence, already present), `fyne.io/fyne/v2/test` (headless widget testing)

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `kanacore/kana.go` | Create | `Kana`, `CharacterSet`, `Hiragana()` — moved from root |
| `kanacore/kana_rows.go` | Create | `KanaRow`, `AllKanaRows`, `CharToRow`, `DefaultRowIDs()` — moved from root |
| `kanacore/kana_test.go` | Create | Tests for character set and row helpers |
| `main.go` | Modify | Import `kana/kanacore` instead of inlining types |
| `game.go` | Modify | Import `kana/kanacore`; fix stats double-counting |
| `ui.go` | Modify | Import `kana/kanacore` |
| `settings_form.go` | Modify | Import `kana/kanacore` |
| `fyne/main.go` | Create | Desktop entry point: open store, build window, run app |
| `fyne/app.go` | Create | `buildWindow()`: layout assembly, event watcher goroutine |
| `fyne/theme.go` | Create | `KanaTheme` implementing `fyne.Theme`; Warm Paper palette |
| `fyne/tile.go` | Create | `KanaTile`: three canvas objects per falling tile |
| `fyne/game.go` | Create | `GameState` struct, all game logic methods, goroutines |
| `fyne/canvas.go` | Create | `GameCanvas` widget: `BaseWidget` + custom renderer |
| `fyne/stats.go` | Create | `StatsPanel` widget + `StatsSnapshot` type |
| `fyne/input.go` | Create | `InputBar` widget: score, entry, missed, gear button |
| `fyne/settings.go` | Create | Settings `dialog.Custom`: rows, auto-progress, score limit |
| `fyne/game_test.go` | Create | Unit tests for `GameState` logic (no Fyne dependency) |
| `fyne/tile_test.go` | Create | Tests for `KanaTile` canvas object positions |
| `fyne/stats_test.go` | Create | Tests for `StatsPanel.Update` and snapshot building |

---

## Task 1: Extract `kanacore/` package

**Files:**
- Create: `kanacore/kana.go`
- Create: `kanacore/kana_rows.go`
- Create: `kanacore/kana_test.go`

- [ ] **Step 1.1: Create `kanacore/kana.go`**

```go
package kanacore

// Kana represents a falling character in the game.
type Kana struct {
	Char   string
	Romaji string
	X      float32
	Y      float32
	Speed  float32
}

// CharacterSet represents a collection of kana characters with their romaji.
type CharacterSet struct {
	Name string
	Data map[string]string
}

// Hiragana returns the basic hiragana character set.
func Hiragana() CharacterSet {
	return CharacterSet{
		Name: "Hiragana",
		Data: map[string]string{
			"あ": "a", "い": "i", "う": "u", "え": "e", "お": "o",
			"か": "ka", "き": "ki", "く": "ku", "け": "ke", "こ": "ko",
			"さ": "sa", "し": "shi", "す": "su", "せ": "se", "そ": "so",
			"た": "ta", "ち": "chi", "つ": "tsu", "て": "te", "と": "to",
			"な": "na", "に": "ni", "ぬ": "nu", "ね": "ne", "の": "no",
			"は": "ha", "ひ": "hi", "ふ": "fu", "へ": "he", "ほ": "ho",
			"ま": "ma", "み": "mi", "む": "mu", "め": "me", "も": "mo",
			"や": "ya", "ゆ": "yu", "よ": "yo",
			"ら": "ra", "り": "ri", "る": "ru", "れ": "re", "ろ": "ro",
			"わ": "wa", "を": "wo", "ん": "n",
		},
	}
}

// GetCharacters returns a slice of all characters in the set.
func (cs CharacterSet) GetCharacters() []string {
	chars := make([]string, 0, len(cs.Data))
	for char := range cs.Data {
		chars = append(chars, char)
	}
	return chars
}

// GetRomaji returns the romaji for a given character.
func (cs CharacterSet) GetRomaji(char string) (string, bool) {
	romaji, exists := cs.Data[char]
	return romaji, exists
}
```

Note: `Kana.X` and `Y` are now `float32` (Fyne device-independent pixels) instead of `int`/`float64`. The root terminal `game.go` uses these fields — update the root package in Task 2.

- [ ] **Step 1.2: Create `kanacore/kana_rows.go`**

```go
package kanacore

// KanaRow groups related kana characters by their consonant row.
type KanaRow struct {
	ID         string
	Label      string
	Characters []string
}

// AllKanaRows lists the basic hiragana rows used for practice.
var AllKanaRows = []KanaRow{
	{ID: "vowels", Label: "Vowels (あ)", Characters: []string{"あ", "い", "う", "え", "お"}},
	{ID: "k", Label: "K-row (か)", Characters: []string{"か", "き", "く", "け", "こ"}},
	{ID: "s", Label: "S-row (さ)", Characters: []string{"さ", "し", "す", "せ", "そ"}},
	{ID: "t", Label: "T-row (た)", Characters: []string{"た", "ち", "つ", "て", "と"}},
	{ID: "n", Label: "N-row (な)", Characters: []string{"な", "に", "ぬ", "ね", "の"}},
	{ID: "h", Label: "H-row (は)", Characters: []string{"は", "ひ", "ふ", "へ", "ほ"}},
	{ID: "m", Label: "M-row (ま)", Characters: []string{"ま", "み", "む", "め", "も"}},
	{ID: "y", Label: "Y-row (や)", Characters: []string{"や", "ゆ", "よ"}},
	{ID: "r", Label: "R-row (ら)", Characters: []string{"ら", "り", "る", "れ", "ろ"}},
	{ID: "w", Label: "W-row (わ)", Characters: []string{"わ", "を"}},
	{ID: "n-only", Label: "N (ん)", Characters: []string{"ん"}},
}

// CharToRow maps each kana character to its row ID.
var CharToRow map[string]string

func init() {
	CharToRow = make(map[string]string)
	for _, row := range AllKanaRows {
		for _, char := range row.Characters {
			CharToRow[char] = row.ID
		}
	}
}

// DefaultRowIDs returns IDs for all rows (used for initial/reset selection).
func DefaultRowIDs() []string {
	ids := make([]string, 0, len(AllKanaRows))
	for _, row := range AllKanaRows {
		ids = append(ids, row.ID)
	}
	return ids
}
```

- [ ] **Step 1.3: Write failing tests in `kanacore/kana_test.go`**

```go
package kanacore_test

import (
	"testing"
	"kana/kanacore"
)

func TestHiraganaGetRomaji(t *testing.T) {
	cs := kanacore.Hiragana()
	romaji, ok := cs.GetRomaji("か")
	if !ok || romaji != "ka" {
		t.Fatalf("expected ka, got %q (ok=%v)", romaji, ok)
	}
}

func TestHiraganaGetCharactersCount(t *testing.T) {
	cs := kanacore.Hiragana()
	if len(cs.GetCharacters()) != 46 {
		t.Fatalf("expected 46 characters, got %d", len(cs.GetCharacters()))
	}
}

func TestAllKanaRowsCount(t *testing.T) {
	if len(kanacore.AllKanaRows) != 11 {
		t.Fatalf("expected 11 rows, got %d", len(kanacore.AllKanaRows))
	}
}

func TestCharToRowMapping(t *testing.T) {
	if kanacore.CharToRow["か"] != "k" {
		t.Fatalf("expected k, got %q", kanacore.CharToRow["か"])
	}
	if kanacore.CharToRow["あ"] != "vowels" {
		t.Fatalf("expected vowels, got %q", kanacore.CharToRow["あ"])
	}
}

func TestDefaultRowIDsCount(t *testing.T) {
	ids := kanacore.DefaultRowIDs()
	if len(ids) != 11 {
		t.Fatalf("expected 11 IDs, got %d", len(ids))
	}
}
```

- [ ] **Step 1.4: Run tests — expect failure (package doesn't exist yet)**

```bash
cd /home/marcus/git/Comradin/kana && go test ./kanacore/...
```

Expected: compile error (package not found or no Go files)

- [ ] **Step 1.5: Run tests — expect pass**

```bash
cd /home/marcus/git/Comradin/kana && go test ./kanacore/... -v
```

Expected: all 5 tests PASS

- [ ] **Step 1.6: Update root package to import `kanacore`**

In `kana.go` (root): delete the file — its types now live in `kanacore/`.

In `kana_rows.go` (root): delete the file — its types now live in `kanacore/`.

In `game.go` (root): add `"kana/kanacore"` import; replace all references:
- `Kana` → `kanacore.Kana`
- `CharacterSet` → `kanacore.CharacterSet`
- `Hiragana()` → `kanacore.Hiragana()`
- `KanaRow` → `kanacore.KanaRow`  
- `AllKanaRows` → `kanacore.AllKanaRows`
- `charToRow` → `kanacore.CharToRow`
- `defaultRowIDs()` → `kanacore.DefaultRowIDs()`

Also fix the stats double-counting in `game.go` (root): in `recordCorrect`, **remove** the lines that update `m.OverallStats` directly. In `mergeSessionStats`, change the base loading to read from store:
```go
func (m *Model) mergeSessionStats() {
	if !m.SessionDirty {
		return
	}
	// Load persisted baseline to avoid double-counting
	var baseline map[string]store.KanaStats
	if m.Store != nil {
		if stats, err := m.Store.KanaStatistics(); err == nil {
			baseline = stats
		}
	}
	for char, session := range m.SessionStats {
		if session.CorrectCount == 0 && session.MissCount == 0 {
			continue
		}
		base := baseline[char]
		base.Char = char
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		base.Streak = m.CurrentStreak[char]
		if m.Store != nil {
			_ = m.Store.SaveKanaStats(char, base.CorrectCount, base.MissCount, base.Streak)
		}
		m.OverallStats[char] = base
	}
	m.SessionDirty = false
}
```

In `ui.go` (root): add `"kana/kanacore"` import; replace `Kana` → `kanacore.Kana`.

In `settings_form.go` (root): add `"kana/kanacore"` import; replace `AllKanaRows` → `kanacore.AllKanaRows`, `KanaRow` → `kanacore.KanaRow`.

- [ ] **Step 1.7: Verify terminal build still compiles and passes tests**

```bash
cd /home/marcus/git/Comradin/kana && go build . && go test ./kanacore/... -v
```

Expected: builds with no errors, all kanacore tests pass

- [ ] **Step 1.8: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add kanacore/ kana.go kana_rows.go game.go ui.go settings_form.go
git commit -m "refactor: extract kanacore package; fix stats double-counting"
```

---

## Task 2: Add Fyne dependency and scaffold

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `fyne/main.go` (stub)

- [ ] **Step 2.1: Add Fyne v2 dependency**

```bash
cd /home/marcus/git/Comradin/kana && go get fyne.io/fyne/v2@latest
```

Expected: `go.mod` and `go.sum` updated

- [ ] **Step 2.2: Create stub `fyne/main.go` that compiles**

```go
package main

import (
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.New()
	w := a.NewWindow("Kana")
	w.ShowAndRun()
}
```

- [ ] **Step 2.3: Verify the stub builds**

```bash
cd /home/marcus/git/Comradin/kana && go build ./fyne/
```

Expected: compiles, binary produced

- [ ] **Step 2.4: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add go.mod go.sum fyne/
git commit -m "feat: add Fyne dependency and stub desktop entry point"
```

---

## Task 3: Theme

**Files:**
- Create: `fyne/theme.go`

- [ ] **Step 3.1: Create `fyne/theme.go`**

```go
package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type themeColorKey int

const (
	colorBackground themeColorKey = iota
	colorTileFace
	colorTileShadow
	colorKanaText
	colorStatsBg
	colorInputBg
	colorAccent
	colorMiss
)

// KanaTheme implements fyne.Theme with a named colour palette.
type KanaTheme struct {
	name   string
	colors map[themeColorKey]color.Color
}

// WarmPaperTheme returns the default warm parchment theme.
func WarmPaperTheme() *KanaTheme {
	return &KanaTheme{
		name: "Warm Paper",
		colors: map[themeColorKey]color.Color{
			colorBackground: hexColor("#f0e6d3"),
			colorTileFace:   hexColor("#eedfc0"),
			colorTileShadow: hexColor("#b8956a"),
			colorKanaText:   hexColor("#2c1a0e"),
			colorStatsBg:    hexColor("#e8dbc8"),
			colorInputBg:    hexColor("#e0d4bc"),
			colorAccent:     hexColor("#8b5e3c"),
			colorMiss:       hexColor("#cc4444"),
		},
	}
}

func (t *KanaTheme) kanaColor(key themeColorKey) color.Color {
	if c, ok := t.colors[key]; ok {
		return c
	}
	return color.Black
}

// fyne.Theme implementation — delegate most tokens to the default theme,
// override only background and button colours.

func (t *KanaTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return t.kanaColor(colorBackground)
	case theme.ColorNameButton:
		return t.kanaColor(colorTileFace)
	case theme.ColorNamePrimary:
		return t.kanaColor(colorAccent)
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *KanaTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *KanaTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *KanaTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// hexColor parses a "#rrggbb" string into color.RGBA.
func hexColor(hex string) color.RGBA {
	var r, g, b uint8
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
```

Add `"fmt"` to the import block.

- [ ] **Step 3.2: Verify it compiles**

```bash
cd /home/marcus/git/Comradin/kana && go build ./fyne/
```

Expected: no errors

- [ ] **Step 3.3: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add fyne/theme.go
git commit -m "feat(fyne): add KanaTheme with Warm Paper palette"
```

---

## Task 4: KanaTile

**Files:**
- Create: `fyne/tile.go`
- Create: `fyne/tile_test.go`

- [ ] **Step 4.1: Write failing test in `fyne/tile_test.go`**

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"kana/kanacore"
)

func TestKanaTileMoveUpdatesAllObjects(t *testing.T) {
	test.NewApp()
	k := kanacore.Kana{Char: "か", Romaji: "ka"}
	tile := newKanaTile(k)
	pos := fyne.NewPos(100, 50)
	tile.Move(pos)

	// Face should be at the given position
	if tile.face.Position() != pos {
		t.Errorf("face position: got %v, want %v", tile.face.Position(), pos)
	}
	// Shadow should be offset by (3,3)
	wantShadow := fyne.NewPos(103, 53)
	if tile.shadow.Position() != wantShadow {
		t.Errorf("shadow position: got %v, want %v", tile.shadow.Position(), wantShadow)
	}
}

func TestKanaTileObjectsReturnsThree(t *testing.T) {
	test.NewApp()
	k := kanacore.Kana{Char: "あ", Romaji: "a"}
	tile := newKanaTile(k)
	if len(tile.Objects()) != 3 {
		t.Errorf("expected 3 canvas objects, got %d", len(tile.Objects()))
	}
}
```

- [ ] **Step 4.2: Run test — expect failure**

```bash
cd /home/marcus/git/Comradin/kana && go test ./fyne/ -run TestKanaTile -v
```

Expected: compile error (`newKanaTile` undefined)

- [ ] **Step 4.3: Create `fyne/tile.go`**

```go
package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"kana/kanacore"
)

const (
	tileW float32 = 52
	tileH float32 = 60
)

// KanaTile is a falling kana card rendered as three canvas objects.
type KanaTile struct {
	kana   kanacore.Kana
	pos    fyne.Position
	shadow *canvas.Rectangle
	face   *canvas.Rectangle
	text   *canvas.Text
}

func newKanaTile(k kanacore.Kana) *KanaTile {
	shadow := canvas.NewRectangle(color.RGBA{R: 0xb8, G: 0x95, B: 0x6a, A: 0xff})
	shadow.Resize(fyne.NewSize(tileW, tileH))

	face := canvas.NewRectangle(color.RGBA{R: 0xee, G: 0xdf, B: 0xc0, A: 0xff})
	face.Resize(fyne.NewSize(tileW, tileH))

	text := canvas.NewText(k.Char, color.RGBA{R: 0x2c, G: 0x1a, B: 0x0e, A: 0xff})
	text.TextSize = 32
	text.Alignment = fyne.TextAlignCenter

	t := &KanaTile{kana: k, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// Move updates the positions of all three canvas objects atomically.
func (t *KanaTile) Move(pos fyne.Position) {
	t.pos = pos
	t.shadow.Move(fyne.NewPos(pos.X+3, pos.Y+3))
	t.face.Move(pos)
	// Centre text within face
	textX := pos.X + (tileW-float32(len(t.kana.Char))*18)/2
	t.text.Move(fyne.NewPos(textX, pos.Y+10))
}

// Objects returns the canvas objects for the renderer, shadow first.
func (t *KanaTile) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{t.shadow, t.face, t.text}
}
```

- [ ] **Step 4.4: Run tests — expect pass**

```bash
cd /home/marcus/git/Comradin/kana && go test ./fyne/ -run TestKanaTile -v
```

Expected: PASS

- [ ] **Step 4.5: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add fyne/tile.go fyne/tile_test.go
git commit -m "feat(fyne): add KanaTile canvas component"
```

---

## Task 5: GameState and game logic

**Files:**
- Create: `fyne/game.go`
- Create: `fyne/game_test.go`

- [ ] **Step 5.1: Write failing tests in `fyne/game_test.go`**

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
)

func newTestState() *GameState {
	test.NewApp()
	gs := &GameState{
		sessionStats:  make(map[string]store.KanaStats),
		overallStats:  make(map[string]store.KanaStats),
		currentStreak: make(map[string]int),
		selectedRows:  make(map[string]bool),
		eventCh:       make(chan gameEvent, 4),
		stopCh:        make(chan struct{}),
		canvasW:       400,
		canvasH:       600,
	}
	gs.charSet = kanacore.Hiragana()
	for _, id := range kanacore.DefaultRowIDs() {
		gs.selectedRows[id] = true
	}
	return gs
}

func TestCheckAnswerRemovesTile(t *testing.T) {
	gs := newTestState()
	tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("ka")
	if len(gs.tiles) != 0 {
		t.Errorf("expected tile removed, got %d tiles", len(gs.tiles))
	}
	if gs.score != 10 {
		t.Errorf("expected score 10, got %d", gs.score)
	}
}

func TestCheckAnswerNoMatchLeavestTile(t *testing.T) {
	gs := newTestState()
	tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("ki")
	if len(gs.tiles) != 1 {
		t.Errorf("expected tile to remain, got %d tiles", len(gs.tiles))
	}
}

func TestRecordCorrectUpdatesSessionOnly(t *testing.T) {
	gs := newTestState()
	gs.recordCorrect("か")
	if gs.sessionStats["か"].CorrectCount != 1 {
		t.Errorf("session correct count: got %d, want 1", gs.sessionStats["か"].CorrectCount)
	}
	// overallStats must NOT be touched by recordCorrect
	if gs.overallStats["か"].CorrectCount != 0 {
		t.Errorf("overallStats should not be updated by recordCorrect, got %d",
			gs.overallStats["か"].CorrectCount)
	}
}

func TestRecordMissResetsStreak(t *testing.T) {
	gs := newTestState()
	gs.currentStreak["か"] = 5
	gs.recordMiss("か")
	if gs.currentStreak["か"] != 0 {
		t.Errorf("expected streak 0, got %d", gs.currentStreak["か"])
	}
	if gs.sessionStats["か"].MissCount != 1 {
		t.Errorf("expected miss count 1, got %d", gs.sessionStats["か"].MissCount)
	}
}

func TestScoreLimitEndsGame(t *testing.T) {
	gs := newTestState()
	gs.scoreLimit = 10
	tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("ka")
	if !gs.over {
		t.Error("expected game over when score limit reached")
	}
	if gs.overReason != "score" {
		t.Errorf("expected reason 'score', got %q", gs.overReason)
	}
}

func TestMissLimitEndsGame(t *testing.T) {
	gs := newTestState()
	for i := 0; i < 9; i++ {
		gs.recordMiss("か")
		gs.missed++
	}
	gs.recordMiss("あ")
	gs.missed++
	gs.checkMissedLimit()
	if !gs.over {
		t.Error("expected game over at 10 misses")
	}
}

func TestIsRowMastered(t *testing.T) {
	gs := newTestState()
	row := kanacore.AllKanaRows[0] // vowels
	// Give 3 correct answers to 4 out of 5 characters (80%)
	for _, char := range row.Characters[:4] {
		gs.overallStats[char] = store.KanaStats{Char: char, CorrectCount: 3}
	}
	if !gs.isRowMastered(row) {
		t.Error("expected row to be mastered at 80% threshold")
	}
}
```

Add `"kana/store"` to imports.

- [ ] **Step 5.2: Run tests — expect failure**

```bash
cd /home/marcus/git/Comradin/kana && go test ./fyne/ -run "TestCheckAnswer|TestRecord|TestScore|TestMiss|TestIsRow" -v
```

Expected: compile error (`GameState` undefined)

- [ ] **Step 5.3: Create `fyne/game.go`**

```go
package main

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"kana/kanacore"
	"kana/store"
)

type gameEventType int

const (
	gameOverEvent gameEventType = iota
)

type gameEvent struct {
	kind gameEventType
}

// GameState holds all mutable game state, protected by mu.
type GameState struct {
	mu sync.Mutex

	tiles         []*KanaTile
	charSet       kanacore.CharacterSet
	canvasW       float32
	canvasH       float32
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
	newlyUnlocked []string
	unlockMessage string
	unlockAt      time.Time
	store         *store.Store

	stopCh         chan struct{}
	eventCh        chan gameEvent
	objectSnapshot atomic.Value // stores []fyne.CanvasObject

	// back-reference set by app.go after construction
	canvas     *GameCanvas
	statsPanel *StatsPanel
}

// NewGameState creates a GameState loaded from the store.
func NewGameState(st *store.Store) *GameState {
	gs := &GameState{
		charSet:       kanacore.Hiragana(),
		sessionStats:  make(map[string]store.KanaStats),
		overallStats:  make(map[string]store.KanaStats),
		currentStreak: make(map[string]int),
		selectedRows:  make(map[string]bool),
		scoreLimit:    store.DefaultScoreLimit,
		store:         st,
		stopCh:        make(chan struct{}),
		eventCh:       make(chan gameEvent, 4),
	}

	for _, id := range kanacore.DefaultRowIDs() {
		gs.selectedRows[id] = true
	}

	if st != nil {
		if rows, err := st.SelectedRows(); err == nil && len(rows) > 0 {
			gs.applySelectedRows(rows)
		}
		if auto, err := st.AutoProgress(); err == nil {
			gs.autoProgress = auto
		}
		if limit, err := st.ScoreLimit(); err == nil && limit >= 0 {
			gs.scoreLimit = limit
		}
		if stats, err := st.KanaStatistics(); err == nil {
			for k, v := range stats {
				gs.overallStats[k] = v
				gs.currentStreak[k] = v.Streak
			}
		}
	}

	return gs
}

// Start launches the tick and spawn goroutines.
func (gs *GameState) Start(canvas *GameCanvas) {
	gs.canvas = canvas
	go gs.tickLoop()
	go gs.spawnLoop()
}

// Reset stops existing goroutines and reinitialises state for a new game.
// It does NOT restart goroutines — call Start() after Reset() to do that.
func (gs *GameState) Reset() {
	// Stop old tick/spawn goroutines
	close(gs.stopCh)
	// Close eventCh so any watchEvents goroutine ranging over it exits cleanly.
	// Drain first to avoid a panic from sending to a closed channel.
	close(gs.eventCh)
	for range gs.eventCh {
	}

	gs.mu.Lock()
	gs.tiles = nil
	gs.score = 0
	gs.missed = 0
	gs.over = false
	gs.overReason = ""
	gs.missedKanas = nil
	gs.sessionStats = make(map[string]store.KanaStats)
	gs.currentStreak = make(map[string]int)
	gs.newlyUnlocked = nil
	gs.unlockMessage = ""
	gs.sessionDirty = false
	gs.stopCh = make(chan struct{})
	gs.eventCh = make(chan gameEvent, 4) // new channel; old watchEvents goroutine has exited

	// Reload overall stats from store to get the true persisted baseline
	if gs.store != nil {
		if stats, err := gs.store.KanaStatistics(); err == nil {
			gs.overallStats = stats
			for k, v := range stats {
				gs.currentStreak[k] = v.Streak
			}
		}
	}
	gs.mu.Unlock()
	// Goroutines are NOT started here. Caller must call Start() after Reset().
}

// Snapshot builds a StatsSnapshot under the caller's lock.
// Caller must hold gs.mu.
func (gs *GameState) snapshot() StatsSnapshot {
	sessionCopy := make(map[string]store.KanaStats, len(gs.sessionStats))
	for k, v := range gs.sessionStats {
		sessionCopy[k] = v
	}
	rowsCopy := make(map[string]bool, len(gs.selectedRows))
	for k, v := range gs.selectedRows {
		rowsCopy[k] = v
	}
	return StatsSnapshot{
		SessionStats:  sessionCopy,
		SelectedRows:  rowsCopy,
		MissedKanas:   append([]kanacore.Kana{}, gs.missedKanas...),
		Score:         gs.score,
		ScoreLimit:    gs.scoreLimit,
		Missed:        gs.missed,
		UnlockMessage: gs.unlockMessage,
		UnlockAt:      gs.unlockAt,
	}
}

func (gs *GameState) tickLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-gs.stopCh:
			return
		case <-ticker.C:
			gs.tick()
		}
	}
}

func (gs *GameState) spawnLoop() {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()
	// Spawn one immediately so screen isn't empty at start
	gs.spawnKana()
	for {
		select {
		case <-gs.stopCh:
			return
		case <-ticker.C:
			gs.spawnKana()
		}
	}
}

func (gs *GameState) tick() {
	gs.mu.Lock()
	if gs.over {
		gs.mu.Unlock()
		return
	}

	for i := len(gs.tiles) - 1; i >= 0; i-- {
		t := gs.tiles[i]
		newPos := fyne.NewPos(t.pos.X, t.pos.Y+t.kana.Speed)
		t.Move(newPos)
		if newPos.Y > gs.canvasH {
			gs.recordMiss(t.kana.Char)
			gs.missedKanas = append(gs.missedKanas, t.kana)
			gs.missed++
			gs.tiles = append(gs.tiles[:i], gs.tiles[i+1:]...)
			gs.checkMissedLimit()
		}
	}

	// Build object snapshot while holding the lock
	objs := gs.buildSnapshot()
	gs.mu.Unlock()

	gs.objectSnapshot.Store(objs)

	if gs.canvas != nil {
		gs.canvas.Refresh()
	}
}

func (gs *GameState) checkMissedLimit() {
	if gs.missed >= 10 && !gs.over {
		gs.over = true
		gs.overReason = "misses"
		gs.mergeSessionStats()
		select {
		case gs.eventCh <- gameEvent{kind: gameOverEvent}:
		default:
		}
	}
}

func (gs *GameState) buildSnapshot() []fyne.CanvasObject {
	objs := make([]fyne.CanvasObject, 0, len(gs.tiles)*3+1)
	for _, t := range gs.tiles {
		objs = append(objs, t.Objects()...)
	}
	return objs
}

func (gs *GameState) spawnKana() {
	gs.mu.Lock()
	if gs.over {
		gs.mu.Unlock()
		return
	}
	chars := gs.availableCharacters()
	if len(chars) == 0 {
		gs.mu.Unlock()
		return
	}
	char := chars[rand.Intn(len(chars))]
	romaji, _ := gs.charSet.GetRomaji(char)

	maxX := gs.canvasW - tileW
	if maxX < 0 {
		maxX = 0
	}
	kana := kanacore.Kana{
		Char:   char,
		Romaji: romaji,
		X:      rand.Float32() * maxX,
		Y:      0,
		Speed:  3.75 + rand.Float32()*2.5,
	}
	tile := newKanaTile(kana)
	tile.Move(fyne.NewPos(kana.X, 0))
	gs.tiles = append(gs.tiles, tile)
	objs := gs.buildSnapshot()
	gs.mu.Unlock()

	gs.objectSnapshot.Store(objs)
	if gs.canvas != nil {
		gs.canvas.Refresh()
	}
}

func (gs *GameState) checkAnswer(input string) {
	for i, t := range gs.tiles {
		if t.kana.Romaji == input {
			gs.tiles = append(gs.tiles[:i], gs.tiles[i+1:]...)
			gs.score += 10
			gs.recordCorrect(t.kana.Char)
			if gs.scoreLimit > 0 && gs.score >= gs.scoreLimit {
				gs.over = true
				gs.overReason = "score"
				gs.mergeSessionStats()
				select {
				case gs.eventCh <- gameEvent{kind: gameOverEvent}:
				default:
				}
			}
			return
		}
	}
}

func (gs *GameState) recordCorrect(char string) {
	streak := gs.currentStreak[char] + 1
	gs.currentStreak[char] = streak

	stat := gs.sessionStats[char]
	stat.Char = char
	stat.CorrectCount++
	stat.Streak = streak
	gs.sessionStats[char] = stat
	gs.sessionDirty = true

	if unlocked := gs.checkAutoProgression(); len(unlocked) > 0 {
		gs.newlyUnlocked = append(gs.newlyUnlocked, unlocked...)
		gs.showUnlockMessage(unlocked)
	}
}

func (gs *GameState) recordMiss(char string) {
	gs.currentStreak[char] = 0
	stat := gs.sessionStats[char]
	stat.Char = char
	stat.MissCount++
	stat.Streak = 0
	gs.sessionStats[char] = stat
	gs.sessionDirty = true
}

func (gs *GameState) mergeSessionStats() {
	if !gs.sessionDirty {
		return
	}
	var baseline map[string]store.KanaStats
	if gs.store != nil {
		if stats, err := gs.store.KanaStatistics(); err == nil {
			baseline = stats
		}
	}
	if baseline == nil {
		baseline = make(map[string]store.KanaStats)
	}
	for char, session := range gs.sessionStats {
		if session.CorrectCount == 0 && session.MissCount == 0 {
			continue
		}
		base := baseline[char]
		base.Char = char
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		base.Streak = gs.currentStreak[char]
		if gs.store != nil {
			_ = gs.store.SaveKanaStats(char, base.CorrectCount, base.MissCount, base.Streak)
		}
		gs.overallStats[char] = base
	}
	gs.sessionDirty = false
}

func (gs *GameState) availableCharacters() []string {
	chars := gs.charSet.GetCharacters()
	if len(gs.selectedRows) == 0 {
		return chars
	}
	filtered := make([]string, 0, len(chars))
	for _, char := range chars {
		rowID, ok := kanacore.CharToRow[char]
		if !ok || gs.selectedRows[rowID] {
			filtered = append(filtered, char)
		}
	}
	if len(filtered) == 0 {
		return chars
	}
	return filtered
}

func (gs *GameState) applySelectedRows(rows []string) {
	for k := range gs.selectedRows {
		delete(gs.selectedRows, k)
	}
	for _, id := range rows {
		gs.selectedRows[id] = true
	}
}

func (gs *GameState) checkAutoProgression() []string {
	if !gs.autoProgress {
		return nil
	}
	var nextRow *kanacore.KanaRow
	for i := range kanacore.AllKanaRows {
		if !gs.selectedRows[kanacore.AllKanaRows[i].ID] {
			nextRow = &kanacore.AllKanaRows[i]
			break
		}
	}
	if nextRow == nil {
		return nil
	}
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] && !gs.isRowMastered(row) {
			return nil
		}
	}
	gs.selectedRows[nextRow.ID] = true
	if gs.store != nil {
		ids := make([]string, 0, len(gs.selectedRows))
		for id := range gs.selectedRows {
			ids = append(ids, id)
		}
		_ = gs.store.SaveSelectedRows(ids)
	}
	return []string{nextRow.ID}
}

func (gs *GameState) isRowMastered(row kanacore.KanaRow) bool {
	if len(row.Characters) == 0 {
		return true
	}
	mastered := 0
	for _, char := range row.Characters {
		if gs.overallStats[char].CorrectCount+gs.sessionStats[char].CorrectCount >= 3 {
			mastered++
		}
	}
	threshold := int(float64(len(row.Characters)) * 0.8)
	if threshold == 0 {
		threshold = 1
	}
	return mastered >= threshold
}

func (gs *GameState) showUnlockMessage(rowIDs []string) {
	labels := make([]string, 0, len(rowIDs))
	for _, id := range rowIDs {
		for _, row := range kanacore.AllKanaRows {
			if row.ID == id {
				labels = append(labels, row.Label)
				break
			}
		}
	}
	if len(labels) == 0 {
		return
	}
	msg := "New row unlocked: " + labels[0]
	for i := 1; i < len(labels); i++ {
		msg += ", " + labels[i]
	}
	gs.unlockMessage = msg
	gs.unlockAt = time.Now()
}
```

- [ ] **Step 5.4: Run tests — expect pass**

```bash
cd /home/marcus/git/Comradin/kana && go test ./fyne/ -run "TestCheckAnswer|TestRecord|TestScore|TestMiss|TestIsRow" -v
```

Expected: all 6 tests PASS

- [ ] **Step 5.5: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add fyne/game.go fyne/game_test.go
git commit -m "feat(fyne): add GameState with full game logic and tests"
```

---

## Task 6: GameCanvas widget

**Files:**
- Create: `fyne/canvas.go`

- [ ] **Step 6.1: Create `fyne/canvas.go`**

```go
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// GameCanvas is the falling-tile play field.
type GameCanvas struct {
	widget.BaseWidget
	state *GameState
}

func newGameCanvas(state *GameState) *GameCanvas {
	gc := &GameCanvas{state: state}
	gc.ExtendBaseWidget(gc)
	return gc
}

func (gc *GameCanvas) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.BackgroundColor())
	return &gameCanvasRenderer{canvas: gc, bg: bg}
}

// Resize stores dimensions so the game loop can use them for bounds checking.
func (gc *GameCanvas) Resize(size fyne.Size) {
	gc.BaseWidget.Resize(size)
	gc.state.mu.Lock()
	gc.state.canvasW = size.Width
	gc.state.canvasH = size.Height
	gc.state.mu.Unlock()
}

type gameCanvasRenderer struct {
	canvas *GameCanvas
	bg     *canvas.Rectangle
}

func (r *gameCanvasRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))
}

func (r *gameCanvasRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 300)
}

func (r *gameCanvasRenderer) Refresh() {
	r.bg.FillColor = theme.BackgroundColor()
	canvas.Refresh(r.bg)
}

func (r *gameCanvasRenderer) Destroy() {}

func (r *gameCanvasRenderer) Objects() []fyne.CanvasObject {
	snap, _ := r.canvas.state.objectSnapshot.Load().([]fyne.CanvasObject)
	if snap == nil {
		return []fyne.CanvasObject{r.bg}
	}
	// Prepend background
	all := make([]fyne.CanvasObject, 0, len(snap)+1)
	all = append(all, r.bg)
	all = append(all, snap...)
	return all
}
```

- [ ] **Step 6.2: Verify it compiles**

```bash
cd /home/marcus/git/Comradin/kana && go build ./fyne/
```

Expected: no errors

- [ ] **Step 6.3: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add fyne/canvas.go
git commit -m "feat(fyne): add GameCanvas widget with atomic object snapshot"
```

---

## Task 7: StatsPanel

**Files:**
- Create: `fyne/stats.go`
- Create: `fyne/stats_test.go`

- [ ] **Step 7.1: Write failing test in `fyne/stats_test.go`**

```go
package main

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/store"
)

func TestStatsPanelUpdateDoesNotPanic(t *testing.T) {
	test.NewApp()
	panel := newStatsPanel()
	snap := StatsSnapshot{
		SessionStats: map[string]store.KanaStats{
			"か": {Char: "か", CorrectCount: 3},
		},
		SelectedRows: map[string]bool{"k": true},
		MissedKanas:  []kanacore.Kana{{Char: "ぬ", Romaji: "nu"}},
		Score:        30,
		ScoreLimit:   100,
		Missed:       1,
	}
	panel.Update(snap) // must not panic
}

func TestStatsPanelUnlockMessageClears(t *testing.T) {
	test.NewApp()
	panel := newStatsPanel()
	snap := StatsSnapshot{
		SessionStats:  make(map[string]store.KanaStats),
		SelectedRows:  make(map[string]bool),
		UnlockMessage: "New row unlocked: K-row (か)",
		UnlockAt:      time.Now().Add(-6 * time.Second), // expired
	}
	panel.Update(snap)
	// After 5s the unlock label should be hidden (empty text)
	if panel.unlockLabel.Text != "" {
		t.Errorf("expected unlock label cleared, got %q", panel.unlockLabel.Text)
	}
}
```

- [ ] **Step 7.2: Run test — expect failure**

```bash
cd /home/marcus/git/Comradin/kana && go test ./fyne/ -run TestStatsPanel -v
```

Expected: compile error

- [ ] **Step 7.3: Create `fyne/stats.go`**

```go
package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/store"
)

// StatsSnapshot is a lock-free copy of the game state fields needed by the panel.
type StatsSnapshot struct {
	SessionStats  map[string]store.KanaStats
	SelectedRows  map[string]bool
	MissedKanas   []kanacore.Kana
	Score         int
	ScoreLimit    int
	Missed        int
	UnlockMessage string
	UnlockAt      time.Time
}

// StatsPanel shows hiragana progress, active rows, and missed characters.
type StatsPanel struct {
	widget.BaseWidget

	charLabels   map[string]*widget.Label
	rowBox       *fyne.Container
	missBox      *fyne.Container
	unlockLabel  *widget.Label
	container    *fyne.Container
}

func newStatsPanel() *StatsPanel {
	p := &StatsPanel{
		charLabels:  make(map[string]*widget.Label),
		unlockLabel: widget.NewLabel(""),
	}

	// Build progress grid
	gridItems := make([]fyne.CanvasObject, 0)
	for _, row := range kanacore.AllKanaRows {
		for _, char := range row.Characters {
			lbl := widget.NewLabel("-")
			p.charLabels[char] = lbl
			gridItems = append(gridItems, lbl)
		}
	}
	grid := container.NewGridWithColumns(5, gridItems...)

	p.rowBox = container.NewVBox()
	p.missBox = container.NewVBox()

	p.container = container.NewVScroll(container.NewVBox(
		widget.NewLabel("PROGRESS"),
		grid,
		widget.NewSeparator(),
		widget.NewLabel("ACTIVE ROWS"),
		p.rowBox,
		widget.NewSeparator(),
		widget.NewLabel("MISSED"),
		p.missBox,
		p.unlockLabel,
	))

	p.ExtendBaseWidget(p)
	return p
}

func (p *StatsPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}

// Update refreshes all labels from the snapshot. Safe to call on the main thread.
func (p *StatsPanel) Update(snap StatsSnapshot) {
	// Progress counts
	for char, lbl := range p.charLabels {
		count := snap.SessionStats[char].CorrectCount
		if count > 0 {
			lbl.SetText(fmt.Sprintf("%d", count))
		} else {
			lbl.SetText("-")
		}
	}

	// Active rows
	p.rowBox.RemoveAll()
	for _, row := range kanacore.AllKanaRows {
		if snap.SelectedRows[row.ID] {
			p.rowBox.Add(widget.NewLabel("• " + row.Label))
		}
	}

	// Missed characters
	p.missBox.RemoveAll()
	seen := make(map[string]bool)
	for _, k := range snap.MissedKanas {
		if !seen[k.Char] {
			p.missBox.Add(widget.NewLabel(k.Char + " (" + k.Romaji + ")"))
			seen[k.Char] = true
		}
	}
	if len(snap.MissedKanas) == 0 {
		p.missBox.Add(widget.NewLabel("None yet!"))
	}

	// Unlock notification
	if snap.UnlockMessage != "" && time.Since(snap.UnlockAt) < 5*time.Second {
		p.unlockLabel.SetText(snap.UnlockMessage)
	} else {
		p.unlockLabel.SetText("")
	}

	p.container.Refresh()
}
```

- [ ] **Step 7.4: Run tests — expect pass**

```bash
cd /home/marcus/git/Comradin/kana && go test ./fyne/ -run TestStatsPanel -v
```

Expected: PASS

- [ ] **Step 7.5: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add fyne/stats.go fyne/stats_test.go
git commit -m "feat(fyne): add StatsPanel widget"
```

---

## Task 8: InputBar

**Files:**
- Create: `fyne/input.go`

- [ ] **Step 8.1: Create `fyne/input.go`**

```go
package main

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// InputBar holds the score label, romaji entry, missed count, and settings gear.
type InputBar struct {
	scoreLabel  *widget.Label
	missedLabel *widget.Label
	entry       *widget.Entry
	Container   *fyne.Container
}

func newInputBar(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, win fyne.Window) *InputBar {
	ib := &InputBar{
		scoreLabel:  widget.NewLabel("Score: 0"),
		missedLabel: widget.NewLabel("Missed: 0/10"),
		entry:       widget.NewEntry(),
	}
	ib.entry.SetPlaceHolder("type romaji…")

	ib.entry.OnSubmitted = func(text string) {
		gs.mu.Lock()
		gs.checkAnswer(text)
		snap := gs.snapshot()
		gs.mu.Unlock()

		ib.entry.SetText("")
		ib.scoreLabel.SetText(ib.formatScore(snap.Score, snap.ScoreLimit))
		ib.missedLabel.SetText(fmt.Sprintf("Missed: %d/10", snap.Missed))
		statsPanel.Update(snap)
		gameCanvas.Refresh()
	}

	gearBtn := widget.NewButton("⚙", func() {
		showSettingsDialog(gs, statsPanel, gameCanvas, win)
	})

	ib.Container = container.NewBorder(nil, nil, ib.scoreLabel, container.NewHBox(ib.missedLabel, gearBtn), ib.entry)
	return ib
}

func (ib *InputBar) formatScore(score, limit int) string {
	if limit > 0 {
		return fmt.Sprintf("Score: %d/%d", score, limit)
	}
	return fmt.Sprintf("Score: %d", score)
}

func (ib *InputBar) Update(snap StatsSnapshot) {
	ib.scoreLabel.SetText(ib.formatScore(snap.Score, snap.ScoreLimit))
	ib.missedLabel.SetText(fmt.Sprintf("Missed: %d/10", snap.Missed))
}
```

Add `"fyne.io/fyne/v2"` to imports.

- [ ] **Step 8.2: Verify it compiles**

```bash
cd /home/marcus/git/Comradin/kana && go build ./fyne/
```

Expected: no errors (settings stub will be resolved in Task 9)

- [ ] **Step 8.3: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add fyne/input.go
git commit -m "feat(fyne): add InputBar widget"
```

---

## Task 9: Settings dialog

**Files:**
- Create: `fyne/settings.go`

- [ ] **Step 9.1: Create `fyne/settings.go`**

```go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
)

func showSettingsDialog(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, win fyne.Window) {
	gs.mu.Lock()
	currentRows := make([]string, 0, len(gs.selectedRows))
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] {
			currentRows = append(currentRows, row.ID)
		}
	}
	currentAuto := gs.autoProgress
	currentLimit := gs.scoreLimit
	gs.mu.Unlock()

	// Build options
	options := make([]string, len(kanacore.AllKanaRows))
	for i, row := range kanacore.AllKanaRows {
		options[i] = row.Label
	}

	selectedLabels := make([]string, 0)
	for _, id := range currentRows {
		for _, row := range kanacore.AllKanaRows {
			if row.ID == id {
				selectedLabels = append(selectedLabels, row.Label)
			}
		}
	}

	rowCheck := widget.NewCheckGroup(options, nil)
	rowCheck.SetSelected(selectedLabels)

	autoCheck := widget.NewCheck("Enable auto-progression", nil)
	autoCheck.SetChecked(currentAuto)

	limitEntry := widget.NewEntry()
	limitEntry.SetText(strconv.Itoa(currentLimit))
	limitEntry.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return fmt.Errorf("enter a whole number ≥ 0")
		}
		return nil
	}

	form := container.NewVBox(
		widget.NewLabel("Kana Rows"),
		rowCheck,
		widget.NewSeparator(),
		autoCheck,
		widget.NewSeparator(),
		widget.NewLabel("Score limit (0 = endless)"),
		limitEntry,
	)

	dialog.ShowCustomConfirm("Settings", "Save", "Cancel", form, func(save bool) {
		if !save {
			return
		}

		// Map selected labels back to IDs
		labelToID := make(map[string]string)
		for _, row := range kanacore.AllKanaRows {
			labelToID[row.Label] = row.ID
		}
		newRows := make([]string, 0)
		for _, lbl := range rowCheck.Selected {
			if id, ok := labelToID[lbl]; ok {
				newRows = append(newRows, id)
			}
		}
		if len(newRows) == 0 {
			newRows = kanacore.DefaultRowIDs()
		}

		newAuto := autoCheck.Checked
		newLimit := currentLimit
		if n, err := strconv.Atoi(strings.TrimSpace(limitEntry.Text)); err == nil && n >= 0 {
			newLimit = n
		}

		gs.mu.Lock()
		gs.applySelectedRows(newRows)
		gs.autoProgress = newAuto
		gs.scoreLimit = newLimit

		// Remove in-flight tiles for deselected rows
		filtered := gs.tiles[:0]
		for _, t := range gs.tiles {
			if rowID, ok := kanacore.CharToRow[t.kana.Char]; ok && !gs.selectedRows[rowID] {
				continue
			}
			filtered = append(filtered, t)
		}
		gs.tiles = filtered

		snap := gs.snapshot()
		objs := gs.buildSnapshot()
		gs.mu.Unlock()

		gs.objectSnapshot.Store(objs)

		if gs.store != nil {
			_ = gs.store.SaveSelectedRows(newRows)
			_ = gs.store.SaveAutoProgress(newAuto)
			_ = gs.store.SaveScoreLimit(newLimit)
		}

		statsPanel.Update(snap)
		gameCanvas.Refresh()
	}, win)
}
```

- [ ] **Step 9.2: Verify it compiles**

```bash
cd /home/marcus/git/Comradin/kana && go build ./fyne/
```

Expected: no errors

- [ ] **Step 9.3: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add fyne/settings.go
git commit -m "feat(fyne): add settings dialog"
```

---

## Task 10: App wiring and game-over dialog

**Files:**
- Create: `fyne/app.go`
- Modify: `fyne/main.go`

- [ ] **Step 10.1: Create `fyne/app.go`**

```go
package main

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/store"
)

func buildWindow(a fyne.App, st *store.Store) fyne.Window {
	kanaTheme := WarmPaperTheme()
	a.Settings().SetTheme(kanaTheme)

	w := a.NewWindow("Kana")
	w.Resize(fyne.NewSize(900, 620))

	gs := NewGameState(st)

	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	gs.statsPanel = statsPanel
	gs.canvas = gameCanvas

	inputBar := newInputBar(gs, statsPanel, gameCanvas, w)

	statsContainer := container.NewPadded(statsPanel)
	statsContainer.Resize(fyne.NewSize(280, 0))

	layout := container.NewBorder(
		nil,
		inputBar.Container,
		nil,
		statsContainer,
		gameCanvas,
	)

	w.SetContent(layout)

	// Initial stats render
	gs.mu.Lock()
	snap := gs.snapshot()
	gs.mu.Unlock()
	statsPanel.Update(snap)

	// Start game loop
	gs.Start(gameCanvas)

	// Watch for game events
	go watchEvents(gs, statsPanel, gameCanvas, inputBar, w)

	w.SetOnClosed(func() {
		close(gs.stopCh)
		gs.mu.Lock()
		gs.mergeSessionStats()
		gs.mu.Unlock()
	})

	return w
}

func watchEvents(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, w fyne.Window) {
	for event := range gs.eventCh {
		switch event.kind {
		case gameOverEvent:
			gs.mu.Lock()
			snap := gs.snapshot()
			reason := gs.overReason
			gs.mu.Unlock()

			go showGameOverDialog(gs, snap, reason, statsPanel, gameCanvas, inputBar, w)
		}
	}
}

func showGameOverDialog(gs *GameState, snap StatsSnapshot, reason string, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, w fyne.Window) {
	title := "GAME OVER"
	if reason == "score" {
		title = "SESSION COMPLETE"
	}

	scoreText := fmt.Sprintf("Score: %d", snap.Score)
	if snap.ScoreLimit > 0 {
		scoreText = fmt.Sprintf("Score: %d/%d", snap.Score, snap.ScoreLimit)
	}

	var reasonText string
	switch reason {
	case "score":
		reasonText = "You reached your target score!"
	case "misses":
		reasonText = "10 kana slipped through."
	default:
		reasonText = "Session ended."
	}

	// Build missed characters list
	unique := make(map[string]kanacore.Kana)
	for _, k := range snap.MissedKanas {
		if _, exists := unique[k.Char]; !exists {
			unique[k.Char] = k
		}
	}
	keys := make([]string, 0, len(unique))
	for k := range unique {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	missedParts := make([]string, 0, len(keys))
	for _, char := range keys {
		k := unique[char]
		missedParts = append(missedParts, fmt.Sprintf("%s (%s)", k.Char, k.Romaji))
	}

	missedText := "None!"
	if len(missedParts) > 0 {
		missedText = strings.Join(missedParts, ", ")
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel(scoreText),
		widget.NewLabel(fmt.Sprintf("Missed: %d/10", snap.Missed)),
		widget.NewLabel(reasonText),
		widget.NewSeparator(),
		widget.NewLabel("Characters missed:"),
		widget.NewLabel(missedText),
	)

	dialog.ShowCustomConfirm("", "Play Again", "Quit", content, func(playAgain bool) {
		if !playAgain {
			gs.mu.Lock()
			gs.mergeSessionStats()
			gs.mu.Unlock()
			w.Close()
			return
		}
		gs.Reset()
		gs.Start(gameCanvas)
		go watchEvents(gs, statsPanel, gameCanvas, inputBar, w)

		gs.mu.Lock()
		snap := gs.snapshot()
		gs.mu.Unlock()
		statsPanel.Update(snap)
		inputBar.Update(snap)
		gameCanvas.Refresh()
	}, w)
}
```

- [ ] **Step 10.2: Update `fyne/main.go` to use `buildWindow`**

```go
package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"
	"kana/store"
)

func main() {
	st, err := store.Open("kana.db")
	if err != nil {
		fmt.Printf("Error opening store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	a := app.New()
	w := buildWindow(a, st)
	w.ShowAndRun()
}
```

- [ ] **Step 10.3: Build and verify**

```bash
cd /home/marcus/git/Comradin/kana && go build ./fyne/ && go build .
```

Expected: both binaries compile cleanly

- [ ] **Step 10.4: Run all tests**

```bash
cd /home/marcus/git/Comradin/kana && go test ./... -v 2>&1 | tail -30
```

Expected: all tests pass, no failures

- [ ] **Step 10.5: Manual smoke test**

Run the desktop app:
```bash
cd /home/marcus/git/Comradin/kana && ./fyne/kana
```

Verify:
- [ ] Window opens, warm paper background visible
- [ ] Kana tiles appear and fall
- [ ] Typing correct romaji + Enter removes a tile and increments score
- [ ] Typing wrong romaji + Enter does nothing
- [ ] Stats panel on the right updates on correct answers
- [ ] Gear icon opens settings dialog with row checkboxes, auto-progress, score limit
- [ ] Settings changes apply immediately (deselected tiles removed)
- [ ] Game over dialog appears at 10 misses or score limit
- [ ] "Play Again" resets the game
- [ ] "Quit" closes the window

- [ ] **Step 10.6: Commit**

```bash
cd /home/marcus/git/Comradin/kana
git add fyne/app.go fyne/main.go
git commit -m "feat(fyne): wire app, game-over dialog, event watcher"
```

---

## Done

Both apps now build from the same module:
- Terminal: `go build .` → `./kana`
- Desktop: `go build ./fyne` → `./fyne/kana`

Katakana support is the next milestone — `kanacore/` is already structured to accept a second `CharacterSet` without changes to `GameState` or the UI.
