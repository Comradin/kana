# Status Panel, Window Size and Fall Time Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the right-hand status panel with a compact session line, per-script learning-path lines and a side-by-side hiragana/katakana mastery grid. Start the window at 1000 × 760. Let tiles fall in a fixed time instead of at a fixed pixel speed.

**Architecture:**
- GameState computes everything the panel shows: total correct per character and a `PathStatus` per active script. It also notifies the UI after a miss.
- The panel only maps the snapshot to widgets. It uses a small fixed-column layout and a fixed-size kana cell widget, so a wide yōon cell can never stretch the panel.
- Tile movement is derived from the canvas height and a per-tile fall time.

**Tech Stack:** Go 1.25, Fyne v2.7.3 (CGO), `testing` with `fyne.io/fyne/v2/test`.

**Spec:** `docs/superpowers/specs/2026-10-03-status-panel-design.md`
**Mockup:** https://claude.ai/artifact/1YqLRLs7wg7vCUxazhQsbJ ("A + C kombiniert")

**Branch:** `feat/katakana`.

**Conventions:**
- Commit messages are `type(scope): summary` plus a short paragraph. Never add `Co-Authored-By` or any AI/session trailer. Do not push.
- `fyne/` is `package main`, so tests can reach unexported fields.
- The first `go test ./fyne/` builds Fyne via CGO and can take minutes.
- Run `gofmt -w` on changed files before each commit.
- Do not stage the untracked screenshot PNG in the repo root.
- Existing test helpers in `fyne/game_test.go`: `newTestState()`, `tileAt(char, romaji, y)`, `masterRows(gs, ids...)`, `selectRows(gs, ids...)`, `equalIDs`.

---

## File Map

| File | Responsibility | Tasks |
|---|---|---|
| `kanacore/kana.go` | `Kana` loses `Speed` | 1 |
| `fyne/tile.go` | `KanaTile.fallSeconds` | 1 |
| `fyne/game.go` | fall step, spawn fall time, tick interval, stats hook | 1, 2 |
| `fyne/progress.go` (new) | `StepState`, `PathStatus`, `pathStatus`, `totalCorrect` | 3 |
| `fyne/statswidgets.go` (new) | `fixedColumns` layout, `kanaCell` widget | 4 |
| `fyne/stats.go` | the new `StatsPanel` | 5 |
| `fyne/app.go` | window size, stats hook wiring | 1, 2 |
| tests: `fyne/game_test.go`, `fyne/progress_test.go`, `fyne/statswidgets_test.go`, `fyne/stats_test.go`, `fyne/intro_test.go` | | 1–5 |
| `README.md` | panel description | 6 |

---

### Task 1: Fall time and window size

**Files:**
- Modify: `kanacore/kana.go`, `fyne/tile.go`, `fyne/game.go`, `fyne/app.go`
- Test: `fyne/game_test.go`

- [ ] **Step 1: Write the failing tests**

In `fyne/game_test.go`, change `tileAt` so it no longer sets `Speed` and sets a fall time instead:

```go
func tileAt(char, romaji string, y float32) *KanaTile {
	tile := newKanaTile(kanacore.Kana{Char: char, Romaji: romaji})
	tile.fallSeconds = 20
	tile.Move(fyne.NewPos(10, y))
	return tile
}
```

Search `fyne/*_test.go` for any other use of `Speed:` (`grep -n "Speed" fyne/*_test.go`). Remove `Speed` from those literals. Where a test needs movement, set `tile.fallSeconds = 20`.

Append:

```go
func TestFallStep(t *testing.T) {
	cases := []struct{ h, secs, want float32 }{
		{600, 20, 3},
		{300, 20, 1.5},
		{600, 0, 600 / (minFallSeconds * ticksPerSecond)},
	}
	for _, c := range cases {
		if got := fallStep(c.h, c.secs); got != c.want {
			t.Errorf("fallStep(%v, %v) = %v, want %v", c.h, c.secs, got, c.want)
		}
	}
}

func TestTickMovesByFallTime(t *testing.T) {
	gs := newTestState()
	gs.canvasH = 600
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	gs.tick()
	if y := gs.tiles[0].pos.Y; y != 3 {
		t.Fatalf("y after one tick = %v, want 3", y)
	}
	gs.canvasH = 300
	gs.tick()
	if y := gs.tiles[0].pos.Y; y != 4.5 {
		t.Fatalf("y after a tick at half height = %v, want 4.5", y)
	}
}

func TestSpawnSetsFallTimeInRange(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	for i := 0; i < 50; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if tile.fallSeconds < minFallSeconds || tile.fallSeconds > maxFallSeconds {
			t.Fatalf("fallSeconds = %v, want [%v, %v]", tile.fallSeconds, minFallSeconds, maxFallSeconds)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'FallStep|FallTime'`
Expected: FAIL to compile (`tile.fallSeconds undefined`, `undefined: fallStep`).

- [ ] **Step 3: Implement**

`kanacore/kana.go`: remove the `Speed float32` field from `Kana`.

`fyne/tile.go`: add a field to `KanaTile`:

```go
	fallSeconds float32 // time to cross the canvas; see fallStep
```

`fyne/game.go`:
- Add near `tickLoop`:

```go
const (
	ticksPerSecond         = 10
	minFallSeconds float32 = 16
	maxFallSeconds float32 = 24
)

// fallStep is how far a tile moves per tick so that it crosses a canvas of
// height h in fallSeconds, whatever the window size. A missing fall time
// falls back to the minimum.
func fallStep(h, fallSeconds float32) float32 {
	if fallSeconds <= 0 {
		fallSeconds = minFallSeconds
	}
	return h / (fallSeconds * ticksPerSecond)
}
```

- In `tickLoop`, replace `time.NewTicker(100 * time.Millisecond)` with `time.NewTicker(time.Second / ticksPerSecond)`.
- In `tick`, replace `tile.pos.Y+tile.kana.Speed` with `tile.pos.Y+fallStep(gs.canvasH, tile.fallSeconds)`.
- In `spawnKana`, delete `speed := 3.75 + rand.Float32()*2.5` and the `Speed: speed,` line. After `tile := newKanaTile(kana)`, add:

```go
	tile.fallSeconds = minFallSeconds + rand.Float32()*(maxFallSeconds-minFallSeconds)
```

`fyne/app.go`: replace `w.Resize(fyne.NewSize(900, 620))` with `w.Resize(fyne.NewSize(1000, 760))`.

- [ ] **Step 4: Run all tests**

Run: `go vet ./... && go test -race ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kanacore/kana.go fyne/tile.go fyne/game.go fyne/app.go fyne/*_test.go
git commit -F - <<'EOF'
feat(fyne): fall in a fixed time and start with a larger window

Each tile now takes a random 16–24 s to cross the canvas, derived from
the current height on every tick, so the pace no longer depends on the
window size. The window starts at 1000 × 760.
EOF
```

---

### Task 2: Refresh the UI after a miss

**Files:**
- Modify: `fyne/game.go`, `fyne/app.go`
- Test: `fyne/game_test.go`

- [ ] **Step 1: Write the failing test**

Append to `fyne/game_test.go`:

```go
func TestMissCallsStatsHookOnce(t *testing.T) {
	gs := newTestState() // calls test.NewApp(), so fyne.Do runs the hook
	calls := 0
	gs.onStatsChanged = func() { calls++ }
	gs.canvasH = 600
	gs.tiles = []*KanaTile{tileAt("あ", "a", 599), tileAt("い", "i", 0)}

	gs.tick() // あ falls out, い moves
	if calls != 1 {
		t.Fatalf("hook calls after a miss = %d, want 1", calls)
	}
	gs.tick() // nothing falls out
	if calls != 1 {
		t.Fatalf("hook calls after a tick without a miss = %d, want 1", calls)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./fyne/ -run TestMissCallsStatsHookOnce`
Expected: FAIL to compile (`gs.onStatsChanged undefined`).

- [ ] **Step 3: Implement**

`fyne/game.go`:
- Add to `GameState`, next to `canvas`:

```go
	// onStatsChanged refreshes the stats panel and input bar. buildWindow
	// sets it before Start; tick calls it via fyne.Do after a miss.
	onStatsChanged func()
```

- In `tick`, declare `missedNow := false` before the loop. Set `missedNow = true` in the branch that records a miss. Replace the tail of the function:

```go
	gs.buildSnapshot()
	canvas := gs.canvas
	hook := gs.onStatsChanged
	gs.mu.Unlock()

	if canvas != nil {
		fyne.Do(func() { canvas.Refresh() })
	}
	if missedNow && hook != nil {
		fyne.Do(hook)
	}
```

`fyne/app.go`: in `buildWindow`, immediately before `gs.Start(gameCanvas)`, add:

```go
	gs.onStatsChanged = func() {
		gs.mu.Lock()
		snap := gs.snapshot()
		gs.mu.Unlock()
		statsPanel.Update(snap)
		inputBar.Update(snap)
	}
```

- [ ] **Step 4: Run all tests**

Run: `go vet ./... && go test -race ./fyne/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/app.go fyne/game_test.go
git commit -F - <<'EOF'
fix(fyne): refresh the stats and input bar when a tile is missed

A tile falling off the canvas changed the miss count without updating
the UI until the next answer. tick now calls a stats hook through
fyne.Do after a miss.
EOF
```

---

### Task 3: Path status and total correct in the snapshot

**Files:**
- Create: `fyne/progress.go`, `fyne/progress_test.go`
- Modify: `fyne/game.go` (`snapshot`), `fyne/stats.go` (`StatsSnapshot` fields only)

- [ ] **Step 1: Write the failing tests**

Create `fyne/progress_test.go`:

```go
package main

import (
	"testing"

	"kana/kanacore"
	"kana/store"
)

func TestPathStatusStatesAndNext(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels", "k", "s")
	masterRows(gs, "vowels")
	ps := gs.pathStatus(kanacore.ScriptHiragana)
	if ps.Steps[0] != StepMastered || ps.Steps[1] != StepLearning || ps.Steps[2] != StepOpen {
		t.Fatalf("steps = %v", ps.Steps[:3])
	}
	if ps.Unlocked != 2 || !equalIDs(ps.Next, []string{"t", "n"}) {
		t.Fatalf("unlocked=%d next=%v, want 2 [t n]", ps.Unlocked, ps.Next)
	}
}

func TestPathStatusCountsStepsSelectedOutOfOrder(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels", "k", "s", "g", "z")
	ps := gs.pathStatus(kanacore.ScriptHiragana)
	if ps.Unlocked != 3 || !equalIDs(ps.Next, []string{"t", "n"}) {
		t.Fatalf("unlocked=%d next=%v, want 3 [t n]", ps.Unlocked, ps.Next)
	}
}

func TestPathStatusAllUnlocked(t *testing.T) {
	gs := newTestState()
	var ids []string
	for _, row := range kanacore.RowsFor(kanacore.ScriptHiragana) {
		ids = append(ids, row.ID)
	}
	selectRows(gs, ids...)
	ps := gs.pathStatus(kanacore.ScriptHiragana)
	if ps.Unlocked != 15 || ps.Next != nil {
		t.Fatalf("unlocked=%d next=%v, want 15 nil", ps.Unlocked, ps.Next)
	}
}

func TestSnapshotPathsOnlyForActiveScripts(t *testing.T) {
	gs := newTestState()
	snap := gs.snapshot()
	if _, ok := snap.Paths[kanacore.ScriptHiragana]; !ok {
		t.Fatal("missing hiragana path")
	}
	if _, ok := snap.Paths[kanacore.ScriptKatakana]; ok {
		t.Fatal("inactive katakana must have no path")
	}
}

func TestSnapshotTotalCorrectAddsStoredAndSession(t *testing.T) {
	gs := newTestState()
	gs.overallStats["か"] = store.KanaStats{Char: "か", CorrectCount: 2}
	gs.sessionStats["か"] = store.KanaStats{Char: "か", CorrectCount: 1}
	if got := gs.snapshot().TotalCorrect["か"]; got != 3 {
		t.Fatalf("TotalCorrect[か] = %d, want 3", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'PathStatus|TotalCorrect|SnapshotPaths'`
Expected: FAIL to compile (`gs.pathStatus undefined`, `undefined: StepMastered`).

- [ ] **Step 3: Implement**

Create `fyne/progress.go`:

```go
package main

import "kana/kanacore"

// StepState is how far one learning-path step has come.
type StepState int

const (
	StepOpen     StepState = iota // not every row of the step is selected
	StepLearning                  // all rows selected, not all mastered
	StepMastered                  // all rows selected and mastered
)

// PathStatus summarises one script's learning path for the status panel.
type PathStatus struct {
	Steps    []StepState
	Unlocked int      // steps whose rows are all selected, in any order
	Next     []string // rows the next unlock would add; nil when complete
}

// pathStatus computes the learning-path status of a script. Must be called
// under lock.
func (gs *GameState) pathStatus(s kanacore.Script) PathStatus {
	steps := kanacore.ProgressionStepsFor(s)
	ps := PathStatus{Steps: make([]StepState, len(steps))}
	for i, step := range steps {
		selected, mastered := true, true
		for _, id := range step {
			if !gs.selectedRows[id] {
				selected = false
				break
			}
			if row, ok := kanacore.RowByID(id); !ok || !gs.isRowMastered(row) {
				mastered = false
			}
		}
		switch {
		case !selected:
			ps.Steps[i] = StepOpen
		case mastered:
			ps.Steps[i] = StepMastered
			ps.Unlocked++
		default:
			ps.Steps[i] = StepLearning
			ps.Unlocked++
		}
	}
	ps.Next = gs.nextStepRows(s)
	return ps
}

// totalCorrect returns stored plus session correct answers per character.
// Must be called under lock.
func (gs *GameState) totalCorrect() map[string]int {
	total := make(map[string]int, len(gs.overallStats))
	for char, st := range gs.overallStats {
		total[char] += st.CorrectCount
	}
	for char, st := range gs.sessionStats {
		total[char] += st.CorrectCount
	}
	return total
}
```

`fyne/stats.go`: add to `StatsSnapshot`:

```go
	TotalCorrect map[string]int
	Paths        map[kanacore.Script]PathStatus
```

`fyne/game.go` `snapshot()`: before `return`, add

```go
	paths := make(map[kanacore.Script]PathStatus)
	for _, info := range kanacore.Scripts() {
		if gs.activeScripts[info.Script] {
			paths[info.Script] = gs.pathStatus(info.Script)
		}
	}
```

and set `TotalCorrect: gs.totalCorrect(), Paths: paths,` in the returned struct.

- [ ] **Step 4: Run all tests**

Run: `go vet ./... && go test -race ./fyne/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add fyne/progress.go fyne/progress_test.go fyne/game.go fyne/stats.go
git commit -F - <<'EOF'
feat(fyne): compute learning-path status for the stats panel

The snapshot now carries total correct answers per character and, for
each active script, the state of every learning-path step, how many
steps are unlocked and which rows come next.
EOF
```

---

### Task 4: Fixed-column layout and kana cell widget

**Files:**
- Create: `fyne/statswidgets.go`, `fyne/statswidgets_test.go`

- [ ] **Step 1: Write the failing tests**

Create `fyne/statswidgets_test.go`:

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

func TestGridColumnsAre288Wide(t *testing.T) {
	if w := gridColumns().width(); w != 288 {
		t.Fatalf("grid width = %v, want 288", w)
	}
}

func TestFixedColumnsSkipsHiddenRows(t *testing.T) {
	test.NewApp()
	l := &fixedColumns{widths: []float32{10, 10}, rowHeight: 18, gap: 2}
	objs := []fyne.CanvasObject{
		canvas.NewRectangle(nil), canvas.NewRectangle(nil),
		canvas.NewRectangle(nil), canvas.NewRectangle(nil),
	}
	c := container.New(l, objs...)
	if h := c.MinSize().Height; h != 38 {
		t.Fatalf("two rows height = %v, want 38", h)
	}
	objs[2].Hide()
	objs[3].Hide()
	if h := c.MinSize().Height; h != 18 {
		t.Fatalf("one visible row height = %v, want 18", h)
	}
}

func TestKanaCellStates(t *testing.T) {
	test.NewApp()
	c := newKanaCell("か")
	c.set(cellLearning, false)
	if c.bg.FillColor != cellLearningColor {
		t.Fatalf("learning fill = %v", c.bg.FillColor)
	}
	c.set(cellNew, true)
	if c.bg.StrokeColor != cellMissedColor || c.bg.StrokeWidth != 2 {
		t.Fatalf("missed stroke = %v / %v", c.bg.StrokeColor, c.bg.StrokeWidth)
	}
	if newKanaCell("きゃ").text.TextSize != 10 {
		t.Fatal("two-glyph cells use 10 px text")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'GridColumns|FixedColumns|KanaCell'`
Expected: FAIL to compile.

- [ ] **Step 3: Implement** — create `fyne/statswidgets.go`:

```go
package main

import (
	"image/color"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	cellW     float32 = 24
	cellH     float32 = 18
	rowLabelW float32 = 20
	halfGapW  float32 = 6
	gridGap   float32 = 2
)

var (
	cellOutlineColor  = color.RGBA{R: 0xcd, G: 0xb9, B: 0x9c, A: 0xff}
	cellNewTextColor  = color.RGBA{R: 0x9b, G: 0x8a, B: 0x76, A: 0xff}
	cellLearningColor = color.RGBA{R: 0xe9, G: 0xc4, B: 0x6a, A: 0xff}
	cellMasteredColor = color.RGBA{R: 0x7f, G: 0xa3, B: 0x6b, A: 0xff}
	cellMissedColor   = color.RGBA{R: 0xcc, G: 0x44, B: 0x44, A: 0xff}
)

// fixedColumns lays objects out row by row in columns of fixed widths, so a
// wide cell can never widen its neighbours (container.NewGridWithColumns
// sizes every column to the widest cell). Rows whose objects are all hidden
// take no space.
type fixedColumns struct {
	widths    []float32
	rowHeight float32
	gap       float32
}

func (l *fixedColumns) width() float32 {
	w := l.gap * float32(len(l.widths)-1)
	for _, cw := range l.widths {
		w += cw
	}
	return w
}

func (l *fixedColumns) visibleRows(objs []fyne.CanvasObject) [][]fyne.CanvasObject {
	n := len(l.widths)
	var rows [][]fyne.CanvasObject
	for i := 0; i < len(objs); i += n {
		end := i + n
		if end > len(objs) {
			end = len(objs)
		}
		row := objs[i:end]
		for _, o := range row {
			if o.Visible() {
				rows = append(rows, row)
				break
			}
		}
	}
	return rows
}

func (l *fixedColumns) MinSize(objs []fyne.CanvasObject) fyne.Size {
	rows := len(l.visibleRows(objs))
	if rows == 0 {
		return fyne.NewSize(l.width(), 0)
	}
	return fyne.NewSize(l.width(), float32(rows)*l.rowHeight+float32(rows-1)*l.gap)
}

func (l *fixedColumns) Layout(objs []fyne.CanvasObject, _ fyne.Size) {
	var y float32
	for _, row := range l.visibleRows(objs) {
		var x float32
		for j, o := range row {
			o.Resize(fyne.NewSize(l.widths[j], l.rowHeight))
			o.Move(fyne.NewPos(x, y))
			x += l.widths[j] + l.gap
		}
		y += l.rowHeight + l.gap
	}
}

// gridColumns is the kana grid: row label, five hiragana cells, a spacer and
// five katakana cells – 288 px wide.
func gridColumns() *fixedColumns {
	w := []float32{rowLabelW}
	for i := 0; i < 5; i++ {
		w = append(w, cellW)
	}
	w = append(w, halfGapW)
	for i := 0; i < 5; i++ {
		w = append(w, cellW)
	}
	return &fixedColumns{widths: w, rowHeight: cellH, gap: gridGap}
}

type cellState int

const (
	cellNew      cellState = iota // never answered correctly
	cellLearning                  // 1–2 correct
	cellMastered                  // 3 or more correct
)

// stateFor maps a total correct count to a cell state.
func stateFor(correct int) cellState {
	switch {
	case correct >= 3:
		return cellMastered
	case correct > 0:
		return cellLearning
	default:
		return cellNew
	}
}

// kanaCell is one character in the stats grid.
type kanaCell struct {
	widget.BaseWidget
	bg     *canvas.Rectangle
	text   *canvas.Text
	state  cellState
	missed bool
}

func newKanaCell(char string) *kanaCell {
	c := &kanaCell{bg: canvas.NewRectangle(color.Transparent), text: canvas.NewText(char, kanaTextColor)}
	c.bg.CornerRadius = 3
	c.text.TextSize = 12
	if utf8.RuneCountInString(char) > 1 {
		c.text.TextSize = 10
	}
	c.ExtendBaseWidget(c)
	c.set(cellNew, false)
	return c
}

func (c *kanaCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(c.bg, container.NewCenter(c.text)))
}

func (c *kanaCell) MinSize() fyne.Size {
	return fyne.NewSize(cellW, cellH)
}

// set applies a learning state and the missed-this-session border.
func (c *kanaCell) set(state cellState, missed bool) {
	c.state, c.missed = state, missed
	switch state {
	case cellMastered:
		c.bg.FillColor, c.text.Color = cellMasteredColor, color.White
	case cellLearning:
		c.bg.FillColor, c.text.Color = cellLearningColor, kanaTextColor
	default:
		c.bg.FillColor, c.text.Color = color.Transparent, cellNewTextColor
	}
	c.bg.StrokeColor, c.bg.StrokeWidth = cellOutlineColor, 1
	if state != cellNew {
		c.bg.StrokeWidth = 0
	}
	if missed {
		c.bg.StrokeColor, c.bg.StrokeWidth = cellMissedColor, 2
	}
	c.bg.Refresh()
	c.text.Refresh()
}
```

- [ ] **Step 4: Run the tests**

Run: `go vet ./fyne/ && go test -race ./fyne/ -run 'GridColumns|FixedColumns|KanaCell'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add fyne/statswidgets.go fyne/statswidgets_test.go
git commit -m "feat(fyne): add a fixed-column layout and kana cell for the stats grid"
```

---

### Task 5: The new status panel

**Files:**
- Modify: `fyne/stats.go` (rewrite the panel, keep `StatsSnapshot`, `vowelColIndex`, `rowShortLabel`)
- Rewrite: `fyne/stats_test.go` (keep the two `rowShortLabel` tests)
- Modify: `fyne/intro_test.go` (the one assertion on the old `scriptSections`)

- [ ] **Step 1: Write the failing tests**

Replace everything in `fyne/stats_test.go` except `TestRowShortLabelForSH` and `TestRowShortLabelStripsKatakanaPrefix` with:

```go
func panelSnap(selected []string, scripts []kanacore.Script) StatsSnapshot {
	snap := StatsSnapshot{
		SessionStats:  map[string]store.KanaStats{},
		SelectedRows:  map[string]bool{},
		ActiveScripts: map[kanacore.Script]bool{},
		TotalCorrect:  map[string]int{},
		Paths:         map[kanacore.Script]PathStatus{},
	}
	for _, id := range selected {
		snap.SelectedRows[id] = true
	}
	for _, s := range scripts {
		snap.ActiveScripts[s] = true
		snap.Paths[s] = PathStatus{Steps: make([]StepState, 15)}
	}
	return snap
}

var both = []kanacore.Script{kanacore.ScriptHiragana, kanacore.ScriptKatakana}

func TestPanelRowHalves(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.Update(panelSnap([]string{"k"}, both))
	if !p.cells["か"].Visible() || p.cells["カ"].Visible() {
		t.Fatal("k selected only in hiragana: hiragana half visible, katakana half blank")
	}
	if !p.rowLabels["k"].Visible() || p.rowLabels["t"].Visible() {
		t.Fatal("row k shown, row t (selected nowhere) hidden")
	}
}

func TestPanelCellStatesAndMissed(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"k", "n"}, both)
	snap.TotalCorrect["き"] = 2
	snap.TotalCorrect["く"] = 3
	snap.MissedKanas = []kanacore.Kana{{Char: "ぬ", Romaji: "nu"}}
	p.Update(snap)
	if p.cells["か"].state != cellNew || p.cells["き"].state != cellLearning || p.cells["く"].state != cellMastered {
		t.Fatal("cell states must follow TotalCorrect")
	}
	if !p.cells["ぬ"].missed || p.cells["な"].missed {
		t.Fatal("only ぬ carries the missed border")
	}
}

func TestPanelPathLineFollowsActiveScripts(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.Update(panelSnap([]string{"vowels"}, []kanacore.Script{kanacore.ScriptHiragana}))
	if !p.pathLines[kanacore.ScriptHiragana].Visible() || p.pathLines[kanacore.ScriptKatakana].Visible() {
		t.Fatal("only the hiragana path line is shown")
	}
}

func TestPanelAccuracy(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
	p.Update(snap)
	if p.accuracyText.Text != "–" {
		t.Fatalf("accuracy with no answers = %q", p.accuracyText.Text)
	}
	snap.SessionStats["あ"] = store.KanaStats{Char: "あ", CorrectCount: 34}
	snap.Missed = 4
	p.Update(snap)
	if p.accuracyText.Text != "89 %" {
		t.Fatalf("accuracy 34/4 = %q", p.accuracyText.Text)
	}
	snap.SessionStats["あ"] = store.KanaStats{Char: "あ", CorrectCount: 2}
	snap.Missed = 1
	p.Update(snap)
	if p.accuracyText.Text != "67 %" {
		t.Fatalf("accuracy 2/1 = %q, want rounding", p.accuracyText.Text)
	}
}

func TestPanelHidesEmptyGroupsAndKeepsWidth(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.Update(panelSnap([]string{"vowels", "ky"}, both))
	if p.groupGrids[kanacore.GroupDakuon].Visible() {
		t.Fatal("dakuon grid hidden with no dakuon rows selected")
	}
	if !p.groupGrids[kanacore.GroupYoon].Visible() {
		t.Fatal("yōon grid visible")
	}
	if w := p.groupGrids[kanacore.GroupYoon].MinSize().Width; w != 288 {
		t.Fatalf("yōon grid width = %v, want 288", w)
	}
}
```

In `fyne/intro_test.go`, change `statsPanel.scriptSections[kanacore.ScriptKatakana].Visible()` to `statsPanel.pathLines[kanacore.ScriptKatakana].Visible()`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'Panel|Intro'`
Expected: FAIL to compile (`p.cells undefined`, `p.pathLines undefined`, …).

- [ ] **Step 3: Implement** — replace everything in `fyne/stats.go` after `rowShortLabel` (delete `groupLabel`, the old `StatsPanel`, `sectionKey`, `newStatsPanel`, `CreateRenderer`, `Update`). Update the imports as needed: `fmt`, `math`, `strings`, `image/color`, fyne, canvas, container, widget, kanacore, store.

```go
// StatsPanel shows the session, each active script's learning path and a
// side-by-side hiragana/katakana grid of every character's learning state.
type StatsPanel struct {
	widget.BaseWidget

	correctText, missedText, accuracyText *canvas.Text

	pathLines map[kanacore.Script]*fyne.Container
	pathSegs  map[kanacore.Script][]*canvas.Rectangle
	pathCount map[kanacore.Script]*canvas.Text
	pathNext  map[kanacore.Script]*canvas.Text

	cells      map[string]*kanaCell            // by character
	rowCells   map[string][]*kanaCell          // by row ID (hiragana or kata:), in column order
	rowLabels  map[string]*canvas.Text         // by hiragana row ID
	rowFillers map[string][]fyne.CanvasObject  // spacer and empty slots of a base row
	groupGrids map[kanacore.Group]*fyne.Container

	content *fyne.Container
}

var (
	segOpenColor  = color.RGBA{R: 0xd8, G: 0xc6, B: 0xaa, A: 0xff}
	panelSubColor = color.RGBA{R: 0x6b, G: 0x54, B: 0x40, A: 0xff}
)

func smallText(s string, size float32) *canvas.Text {
	t := canvas.NewText(s, kanaTextColor)
	t.TextSize = size
	return t
}

func newStatsPanel() *StatsPanel {
	p := &StatsPanel{
		pathLines:  map[kanacore.Script]*fyne.Container{},
		pathSegs:   map[kanacore.Script][]*canvas.Rectangle{},
		pathCount:  map[kanacore.Script]*canvas.Text{},
		pathNext:   map[kanacore.Script]*canvas.Text{},
		cells:      map[string]*kanaCell{},
		rowCells:   map[string][]*kanaCell{},
		rowLabels:  map[string]*canvas.Text{},
		rowFillers: map[string][]fyne.CanvasObject{},
		groupGrids: map[kanacore.Group]*fyne.Container{},
	}

	// Session line: correct · missed · accuracy.
	tile := func(caption string) (*canvas.Text, fyne.CanvasObject) {
		num := smallText("0", 15)
		num.TextStyle = fyne.TextStyle{Bold: true}
		num.Alignment = fyne.TextAlignCenter
		sub := smallText(caption, 11)
		sub.Color = panelSubColor
		sub.Alignment = fyne.TextAlignCenter
		bg := canvas.NewRectangle(tileFaceColor)
		bg.CornerRadius = 4
		return num, container.NewStack(bg, container.NewVBox(num, sub))
	}
	var c1, c2, c3 fyne.CanvasObject
	p.correctText, c1 = tile("correct")
	p.missedText, c2 = tile("missed")
	p.accuracyText, c3 = tile("accuracy")
	session := container.NewGridWithColumns(3, c1, c2, c3)

	// One path line per script: name, 15 segments, count; "Next" below.
	top := container.NewVBox(session)
	for _, info := range kanacore.Scripts() {
		widths := []float32{62}
		objs := []fyne.CanvasObject{smallText(info.Label, 12)}
		for i := 0; i < 15; i++ {
			seg := canvas.NewRectangle(segOpenColor)
			seg.CornerRadius = 2
			p.pathSegs[info.Script] = append(p.pathSegs[info.Script], seg)
			widths = append(widths, 10)
			objs = append(objs, seg)
		}
		count := smallText("0/15", 11)
		count.Color = panelSubColor
		p.pathCount[info.Script] = count
		widths = append(widths, 30)
		objs = append(objs, count)
		line := container.New(&fixedColumns{widths: widths, rowHeight: 14, gap: 2}, objs...)
		next := smallText("", 11)
		next.Color = panelSubColor
		p.pathNext[info.Script] = next
		box := container.NewVBox(line, next)
		box.Hide()
		p.pathLines[info.Script] = box
		top.Add(box)
	}

	// Grid header and one fixed-column grid per group.
	head := func(s string) *canvas.Text {
		t := smallText(s, 11)
		t.Alignment = fyne.TextAlignCenter
		t.Color = panelSubColor
		return t
	}
	half := 5*cellW + 4*gridGap
	header := container.New(&fixedColumns{widths: []float32{rowLabelW, half, halfGapW, half}, rowHeight: 14, gap: gridGap},
		smallText("", 11), head("ひらがな"), smallText("", 11), head("カタカナ"))
	grids := container.NewVBox(header)
	for _, g := range kanacore.Groups() {
		var objs []fyne.CanvasObject
		for _, row := range kanacore.RowsInGroup(kanacore.ScriptHiragana, g.Group) {
			lbl := smallText(rowShortLabel(row.ID), 11)
			lbl.Alignment = fyne.TextAlignTrailing
			lbl.Color = panelSubColor
			p.rowLabels[row.ID] = lbl
			objs = append(objs, lbl)
			objs = append(objs, p.half(row)...)
			spacer := canvas.NewRectangle(color.Transparent)
			p.rowFillers[row.ID] = append(p.rowFillers[row.ID], spacer)
			objs = append(objs, spacer)
			kata, _ := kanacore.RowByID("kata:" + row.ID)
			objs = append(objs, p.half(kata)...)
		}
		grid := container.New(gridColumns(), objs...)
		p.groupGrids[g.Group] = grid
		grids.Add(grid)
	}

	legend := container.NewHBox()
	for _, item := range []struct {
		label string
		st    cellState
		miss  bool
	}{{"new", cellNew, false}, {"learning", cellLearning, false}, {"mastered", cellMastered, false}, {"missed", cellNew, true}} {
		sw := newKanaCell("")
		sw.set(item.st, item.miss)
		t := smallText(item.label, 11)
		t.Color = panelSubColor
		legend.Add(container.NewGridWrap(fyne.NewSize(12, 12), sw))
		legend.Add(t)
	}

	p.content = container.NewBorder(top, legend, nil, nil, container.NewVScroll(grids))
	p.ExtendBaseWidget(p)
	return p
}

// half builds the five cell slots of one row in vowel-column order; slots
// without a kana (yōon i/e) are transparent fillers.
func (p *StatsPanel) half(row kanacore.KanaRow) []fyne.CanvasObject {
	slots := make([]fyne.CanvasObject, 5)
	for _, e := range row.Entries {
		if col := vowelColIndex(e.Romaji); col >= 0 && col < 5 {
			cell := newKanaCell(e.Char)
			p.cells[e.Char] = cell
			p.rowCells[row.ID] = append(p.rowCells[row.ID], cell)
			slots[col] = cell
		}
	}
	base := kanacore.BaseRowID(row.ID)
	for i, s := range slots {
		if s == nil {
			filler := canvas.NewRectangle(color.Transparent)
			p.rowFillers[base] = append(p.rowFillers[base], filler)
			slots[i] = filler
		}
	}
	return slots
}

func (p *StatsPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.content)
}

func (p *StatsPanel) MinSize() fyne.Size {
	return fyne.NewSize(300, p.content.MinSize().Height)
}

// Update maps a snapshot onto the panel.
func (p *StatsPanel) Update(snap StatsSnapshot) {
	correct := 0
	for _, st := range snap.SessionStats {
		correct += st.CorrectCount
	}
	p.correctText.Text = fmt.Sprint(correct)
	p.missedText.Text = fmt.Sprint(snap.Missed)
	if answered := correct + snap.Missed; answered > 0 {
		p.accuracyText.Text = fmt.Sprintf("%d %%", int(math.Round(100*float64(correct)/float64(answered))))
	} else {
		p.accuracyText.Text = "–"
	}

	for _, info := range kanacore.Scripts() {
		box := p.pathLines[info.Script]
		ps, ok := snap.Paths[info.Script]
		if !ok {
			box.Hide()
			continue
		}
		box.Show()
		for i, seg := range p.pathSegs[info.Script] {
			seg.FillColor = segOpenColor
			if i < len(ps.Steps) {
				switch ps.Steps[i] {
				case StepMastered:
					seg.FillColor = cellMasteredColor
				case StepLearning:
					seg.FillColor = cellLearningColor
				}
			}
			seg.Refresh()
		}
		p.pathCount[info.Script].Text = fmt.Sprintf("%d/%d", ps.Unlocked, len(ps.Steps))
		if len(ps.Next) == 0 {
			p.pathNext[info.Script].Text = "All rows unlocked"
		} else {
			labels := make([]string, 0, len(ps.Next))
			for _, id := range ps.Next {
				if row, ok := kanacore.RowByID(id); ok {
					labels = append(labels, row.Label)
				}
			}
			p.pathNext[info.Script].Text = "Next: " + strings.Join(labels, " · ")
		}
	}

	missed := map[string]bool{}
	for _, k := range snap.MissedKanas {
		missed[k.Char] = true
	}
	for char, cell := range p.cells {
		cell.set(stateFor(snap.TotalCorrect[char]), missed[char])
	}

	for _, g := range kanacore.Groups() {
		groupVisible := false
		for _, row := range kanacore.RowsInGroup(kanacore.ScriptHiragana, g.Group) {
			hOn := snap.ActiveScripts[kanacore.ScriptHiragana] && snap.SelectedRows[row.ID]
			kOn := snap.ActiveScripts[kanacore.ScriptKatakana] && snap.SelectedRows["kata:"+row.ID]
			show := hOn || kOn
			groupVisible = groupVisible || show
			setVisible(p.rowLabels[row.ID], show)
			for _, f := range p.rowFillers[row.ID] {
				setVisible(f, show)
			}
			for _, c := range p.rowCells[row.ID] {
				setVisible(c, hOn)
			}
			for _, c := range p.rowCells["kata:"+row.ID] {
				setVisible(c, kOn)
			}
		}
		setVisible(p.groupGrids[g.Group], groupVisible)
		p.groupGrids[g.Group].Refresh()
	}

	p.content.Refresh()
}

func setVisible(o fyne.CanvasObject, on bool) {
	if on {
		o.Show()
	} else {
		o.Hide()
	}
}
```

Note on hidden cells: when only one half of a row is visible, the hidden cells keep their slot because the row still has visible objects (the label and fillers). `fixedColumns` positions by index, so the halves stay aligned.

- [ ] **Step 4: Run all tests**

Run: `go vet ./... && go test -race ./...`
Expected: PASS. If `TestPanelHidesEmptyGroupsAndKeepsWidth` reports a width other than 288, check that every grid object sits in exactly 12 columns per row: 1 label, 5 slots, 1 spacer, 5 slots.

- [ ] **Step 5: Build and commit**

Run: `go build -o /tmp/claude-1000/-home-marcus-git-Comradin-kana/fb97d1f8-609d-4b58-abed-f955011bd160/scratchpad/kana-desktop-panel ./fyne/`

```bash
git add fyne/stats.go fyne/stats_test.go fyne/intro_test.go
git commit -F - <<'EOF'
feat(fyne): show learning path and a kana grid in the status panel

The status panel shows correct, missed and accuracy for the session,
one learning-path line per active script with the next rows to unlock,
and one grid with hiragana and katakana side by side, each character
coloured by its learning state and outlined when missed. The active rows
and missed lists are gone.
EOF
```

---

### Task 6: Docs and verification

- [ ] **Step 1:** In `README.md`, update the description of the stats panel and the `stats.go` architecture bullet to match: session line, learning path per script, kana grid with new/learning/mastered/missed, no active-rows or missed lists. Also mention `progress.go` and `statswidgets.go`, the 1000 × 760 window and the 16–24 s fall time.
- [ ] **Step 2:** Run `go vet ./... && go test -race -count=1 ./...` and `gofmt -l .`. Expected: all pass, `gofmt` prints nothing.
- [ ] **Step 3:** Commit `docs: describe the new status panel and fall time`.
- [ ] **Step 4:** Report. The manual check is for the user, on a display:
  - the panel fits without the whole panel scrolling, and only the grid scrolls when many rows are active;
  - the colours and the red border are readable;
  - tiles take about 16–24 s to fall, at any window height.
