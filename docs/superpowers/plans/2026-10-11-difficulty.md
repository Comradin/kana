# Difficulty Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add four difficulty levels (fall time, spawn interval, miss limit), a per-character fall factor, a session tempo that rises with streaks and drops on misses, and a configurable points factor.

**Architecture:** `fyne/difficulty.go` holds the level table and pure helpers. `GameState` keeps the chosen level, the points setting, the answer streak and the tempo step, and applies them when spawning, scoring and counting misses. The spawn loop becomes a re-armed timer. The settings dialog, input bar and game-over dialog show and change the new values.

**Tech Stack:** Go 1.25, Fyne v2.7.3 (CGO), `testing` with `fyne.io/fyne/v2/test`.

**Spec:** `docs/superpowers/specs/2026-10-11-difficulty-design.md`

**Branch:** `feat/difficulty`.

**Conventions:**
- Commit messages: `type(scope): summary` plus a short paragraph. Never add `Co-Authored-By` or any AI/session trailer. Do not push.
- `fyne/` is `package main`, so tests reach unexported fields.
- The first `go test ./fyne/` builds Fyne via CGO and can take minutes.
- Run `gofmt -w` on changed files before each commit.
- Test helpers in `fyne/game_test.go`: `newTestState()`, `tileAt(char, romaji, y)` (sets `fallSeconds = 20`), `masterRows(gs, ids...)`, `selectRows(gs, ids...)`, `equalIDs`. `fyne/game_lifecycle_test.go` has `openTestStore(t)`. `fyne/pause_test.go` shows how to build a full `InputBar` (`newInputBar(gs, newStatsPanel(), newGameCanvas(gs), test.NewWindow(nil))`).

---

## File Map

| File | Responsibility | Tasks |
|---|---|---|
| `store/store.go`, `store/store_test.go` | `difficulty` and `points_factor` keys | 1 |
| `fyne/difficulty.go` (new), `fyne/difficulty_test.go` (new) | level table and pure helpers | 1 |
| `fyne/game.go`, `fyne/game_test.go`, `fyne/game_lifecycle_test.go` | level in play: spawn, tempo, points, misses, settings, snapshot | 2 |
| `fyne/stats.go` | `StatsSnapshot` fields | 2 |
| `fyne/input.go`, `fyne/app.go`, `fyne/settings.go`, `fyne/difficulty_ui_test.go` (new) | display and settings | 3 |
| `README.md` | docs | 4 |

---

### Task 1: Level table, helpers and store keys

**Files:** create `fyne/difficulty.go`, `fyne/difficulty_test.go`; modify `store/store.go`, `store/store_test.go`.

- [ ] **Step 1: Write the failing tests**

Create `fyne/difficulty_test.go`:

```go
package main

import (
	"testing"
	"time"
)

func TestLevelByName(t *testing.T) {
	cases := map[string]struct {
		label string
		limit int
		every time.Duration
	}{
		"relaxed": {"Relaxed", 10, 4500 * time.Millisecond},
		"normal":  {"Normal", 10, 4 * time.Second},
		"fast":    {"Fast", 7, 2500 * time.Millisecond},
		"pro":     {"Pro", 5, 1600 * time.Millisecond},
		"":        {"Normal", 10, 4 * time.Second},
		"insane":  {"Normal", 10, 4 * time.Second},
	}
	for name, want := range cases {
		got := LevelByName(name)
		if got.Label != want.label || got.MissLimit != want.limit || got.SpawnEvery != want.every {
			t.Errorf("LevelByName(%q) = %+v", name, got)
		}
	}
	if len(Levels()) != 4 || Levels()[0].Level != LevelRelaxed || Levels()[3].Level != LevelPro {
		t.Fatal("Levels() must list relaxed, normal, fast, pro")
	}
}

func TestCharFallFactor(t *testing.T) {
	for correct, want := range map[int]float32{0: 1.2, 2: 1.0, 3: 0.7, 9: 0.7} {
		if got := charFallFactor(correct); got != want {
			t.Errorf("charFallFactor(%d) = %v, want %v", correct, got, want)
		}
	}
}

func TestTempoFactor(t *testing.T) {
	if tempoFactor(0) != 1 || tempoFactor(1) != float32(0.95) || tempoFactor(10) != 0.6 || tempoFactor(20) != 0.6 {
		t.Fatalf("tempoFactor = %v %v %v %v", tempoFactor(0), tempoFactor(1), tempoFactor(10), tempoFactor(20))
	}
}

func TestValidPoints(t *testing.T) {
	for in, want := range map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 5: 5, 4: 0, -1: 0, 7: 0} {
		if got := validPoints(in); got != want {
			t.Errorf("validPoints(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestFormatStatusAndMissReason(t *testing.T) {
	if got := formatStatus(240, 1000, "Fast", 1.15); got != "Score: 240/1000 · Fast · Tempo ×1.15" {
		t.Errorf("formatStatus = %q", got)
	}
	if got := formatStatus(240, 0, "Normal", 1); got != "Score: 240 · Normal · Tempo ×1.00" {
		t.Errorf("formatStatus endless = %q", got)
	}
	if got := missReason(5); got != "5 kana slipped through." {
		t.Errorf("missReason = %q", got)
	}
}
```

Append to `store/store_test.go`:

```go
func TestDifficultyAndPointsRoundTrip(t *testing.T) {
	st := openTemp(t)
	if d, err := st.Difficulty(); err != nil || d != "" {
		t.Fatalf("unset Difficulty = %q, %v", d, err)
	}
	if p, err := st.PointsFactor(); err != nil || p != 0 {
		t.Fatalf("unset PointsFactor = %d, %v", p, err)
	}
	if err := st.SaveDifficulty("pro"); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePointsFactor(3); err != nil {
		t.Fatal(err)
	}
	if d, _ := st.Difficulty(); d != "pro" {
		t.Fatalf("Difficulty = %q", d)
	}
	if p, _ := st.PointsFactor(); p != 3 {
		t.Fatalf("PointsFactor = %d", p)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./store/ ./fyne/ -run 'Level|CharFall|Tempo|ValidPoints|FormatStatus|DifficultyAndPoints'`
Expected: FAIL to compile.

- [ ] **Step 3: Implement**

Create `fyne/difficulty.go`:

```go
package main

import (
	"fmt"
	"math"
	"time"
)

// Level names a difficulty level; the value is what the store keeps.
type Level string

const (
	LevelRelaxed Level = "relaxed"
	LevelNormal  Level = "normal"
	LevelFast    Level = "fast"
	LevelPro     Level = "pro"
)

// LevelInfo is what a level changes about the game.
type LevelInfo struct {
	Level        Level
	Label        string
	MinFall      float32 // seconds a tile takes to fall, before factors
	MaxFall      float32
	SpawnEvery   time.Duration
	MissLimit    int
	PointsFactor int // used when the points setting is Auto
}

var levels = []LevelInfo{
	{LevelRelaxed, "Relaxed", 18, 26, 4500 * time.Millisecond, 10, 1},
	{LevelNormal, "Normal", 16, 24, 4 * time.Second, 10, 1},
	{LevelFast, "Fast", 10, 15, 2500 * time.Millisecond, 7, 2},
	{LevelPro, "Pro", 6, 9, 1600 * time.Millisecond, 5, 3},
}

// Levels returns all levels in display order.
func Levels() []LevelInfo {
	return append([]LevelInfo(nil), levels...)
}

// LevelByName returns the named level; unknown or empty names give Normal,
// which matches the game before levels existed.
func LevelByName(name string) LevelInfo {
	for _, l := range levels {
		if string(l.Level) == name {
			return l
		}
	}
	return levels[1]
}

const (
	pointsAuto     = 0  // stored "Auto": use the level's points factor
	maxTempoStep   = 10 // 0.95^10 ≈ 0.6, where the tempo floor is reached
	streakPerStep  = 5  // correct answers in a row per tempo step
	missTempoSteps = 2  // tempo steps a miss takes back
)

// pointsChoices are the selectable points settings, Auto first.
var pointsChoices = []int{pointsAuto, 1, 2, 3, 5}

// validPoints keeps a selectable points setting and maps anything else to Auto.
func validPoints(p int) int {
	for _, c := range pointsChoices {
		if c == p {
			return p
		}
	}
	return pointsAuto
}

// pointsLabel is the settings label of a points setting.
func pointsLabel(p int) string {
	if p == pointsAuto {
		return "Auto (by level)"
	}
	return fmt.Sprintf("×%d", p)
}

// charFallFactor gives new characters more time and mastered ones less.
func charFallFactor(correct int) float32 {
	switch stateFor(correct) {
	case cellMastered:
		return 0.7
	case cellLearning:
		return 1.0
	default:
		return 1.2
	}
}

// tempoFactor scales fall time and spawn interval for a tempo step, never
// below 0.6.
func tempoFactor(step int) float32 {
	f := float32(math.Pow(0.95, float64(step)))
	if f < 0.6 {
		return 0.6
	}
	return f
}

// formatStatus is the input bar's score line.
func formatStatus(score, limit int, level string, tempo float32) string {
	s := fmt.Sprintf("Score: %d", score)
	if limit > 0 {
		s = fmt.Sprintf("Score: %d/%d", score, limit)
	}
	return fmt.Sprintf("%s · %s · Tempo ×%.2f", s, level, tempo)
}

// missReason is the game-over reason after too many misses.
func missReason(missed int) string {
	return fmt.Sprintf("%d kana slipped through.", missed)
}
```

In `store/store.go`, add keys to the first `const` block:

```go
	difficultyKey   = "difficulty"
	pointsFactorKey = "points_factor"
```

and methods after `SaveKatakanaOffered`:

```go
// Difficulty returns the stored level name, or "" when unset.
func (s *Store) Difficulty() (string, error) {
	value, err := s.getSetting(difficultyKey)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SaveDifficulty persists the level name.
func (s *Store) SaveDifficulty(name string) error {
	return s.setSetting(difficultyKey, name)
}

// PointsFactor returns the stored points setting, or 0 (Auto) when unset or
// unreadable.
func (s *Store) PointsFactor() (int, error) {
	value, err := s.getSetting(pointsFactorKey)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n, convErr := strconv.Atoi(value)
	if convErr != nil {
		return 0, nil
	}
	return n, nil
}

// SavePointsFactor persists the points setting.
func (s *Store) SavePointsFactor(p int) error {
	return s.setSetting(pointsFactorKey, strconv.Itoa(p))
}
```

- [ ] **Step 4: Run the tests**

Run: `go vet ./... && go test ./store/ ./fyne/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add fyne/difficulty.go fyne/difficulty_test.go store/store.go store/store_test.go
git commit -F - <<'EOF'
feat: add difficulty levels and their stored settings

A level table (Relaxed, Normal, Fast, Pro) with fall time, spawn
interval, miss limit and an Auto points factor, helpers for the
per-character fall factor, the session tempo and the status line, and
store keys for the chosen level and points setting.
EOF
```

---

### Task 2: Difficulty in play

**Files:** modify `fyne/game.go`, `fyne/stats.go` (snapshot fields), `fyne/game_test.go`, `fyne/game_lifecycle_test.go`.

- [ ] **Step 1: Write the failing tests**

In `fyne/game_test.go`:
- In `newTestState`'s struct literal add `level: LevelByName(""),`.
- Update `TestFallStep`: call `fallStep(c.h, c.secs, 16)`, and for the zero case expect the literal `3.75` (`600 / (16 * ticksPerSecond)` would be integer division once `minFallSeconds` is gone).
- Replace `TestSpawnSetsFallTimeInRange`:

```go
func TestSpawnSetsFallTimeInRange(t *testing.T) {
	gs := newTestState() // Normal; vowels not answered yet → factor 1.2
	selectRows(gs, "vowels")
	for i := 0; i < 50; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if tile.fallSeconds < 16*1.2-0.01 || tile.fallSeconds > 24*1.2+0.01 {
			t.Fatalf("fallSeconds = %v, want [19.2, 28.8]", tile.fallSeconds)
		}
	}
}

func TestSpawnOnProForMasteredKana(t *testing.T) {
	gs := newTestState()
	gs.level = LevelByName("pro")
	selectRows(gs, "vowels")
	masterRows(gs, "vowels")
	for i := 0; i < 50; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if tile.fallSeconds < 4.2-0.01 || tile.fallSeconds > 6.3+0.01 {
			t.Fatalf("fallSeconds = %v, want [4.2, 6.3]", tile.fallSeconds)
		}
	}
}
```

Append:

```go
func answerOne(gs *GameState) {
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	gs.checkAnswer("a")
}

func TestTempoRisesWithStreakAndDropsOnMiss(t *testing.T) {
	gs := newTestState()
	for i := 0; i < 4; i++ {
		answerOne(gs)
	}
	if gs.tempoStep != 0 {
		t.Fatalf("tempoStep after 4 = %d", gs.tempoStep)
	}
	answerOne(gs)
	if gs.tempoStep != 1 {
		t.Fatalf("tempoStep after 5 = %d", gs.tempoStep)
	}
	gs.recordMiss("あ")
	if gs.tempoStep != 0 || gs.answerStreak != 0 {
		t.Fatalf("after miss tempoStep=%d streak=%d", gs.tempoStep, gs.answerStreak)
	}
	for i := 0; i < 60; i++ {
		answerOne(gs)
	}
	if gs.tempoStep != maxTempoStep {
		t.Fatalf("tempoStep after 60 = %d, want %d", gs.tempoStep, maxTempoStep)
	}
	gs.recordMiss("あ")
	if gs.tempoStep != 8 {
		t.Fatalf("tempoStep after cap and miss = %d, want 8", gs.tempoStep)
	}
}

func TestPointsPerHit(t *testing.T) {
	cases := []struct {
		level  string
		points int
		want   int
	}{
		{"normal", pointsAuto, 10},
		{"pro", pointsAuto, 30},
		{"pro", 1, 10},
		{"relaxed", 5, 50},
	}
	for _, c := range cases {
		gs := newTestState()
		gs.level = LevelByName(c.level)
		gs.pointsSetting = c.points
		answerOne(gs)
		if gs.score != c.want {
			t.Errorf("%s/%d: score = %d, want %d", c.level, c.points, gs.score, c.want)
		}
	}
}

func TestMissLimitFollowsLevel(t *testing.T) {
	gs := newTestState()
	gs.level = LevelByName("pro")
	for i := 0; i < 4; i++ {
		gs.missed++
		gs.checkMissedLimit()
	}
	if gs.over {
		t.Fatal("4 misses must not end a Pro game")
	}
	gs.missed++
	gs.checkMissedLimit()
	if !gs.over {
		t.Fatal("the 5th miss ends a Pro game")
	}
}

func TestResetClearsTempo(t *testing.T) {
	gs := newTestState()
	gs.answerStreak, gs.tempoStep = 7, 3
	gs.Reset()
	if gs.answerStreak != 0 || gs.tempoStep != 0 {
		t.Fatalf("after Reset streak=%d step=%d", gs.answerStreak, gs.tempoStep)
	}
}

func TestSpawnIntervalUsesTempo(t *testing.T) {
	gs := newTestState()
	gs.level = LevelByName("fast")
	gs.tempoStep = 1
	f := float32(0.95) // a variable: a constant expression would not convert to Duration
	if got, want := gs.spawnInterval(), time.Duration(float64(2500*time.Millisecond)*float64(f)); got != want {
		t.Fatalf("spawnInterval = %v, want %v", got, want)
	}
}

func TestSnapshotCarriesLevelTempoAndLimit(t *testing.T) {
	gs := newTestState()
	gs.level = LevelByName("fast")
	gs.tempoStep = 1
	snap := gs.snapshot()
	if snap.LevelLabel != "Fast" || snap.MissLimit != 7 || snap.Tempo != 1/float32(0.95) {
		t.Fatalf("snapshot level=%q limit=%d tempo=%v", snap.LevelLabel, snap.MissLimit, snap.Tempo)
	}
}
```

Add `"time"` to the imports of `fyne/game_test.go`. Update the five existing `gs.applySettings(…)` calls in `fyne/game_test.go` to pass two more arguments, `LevelNormal, pointsAuto`. Append:

```go
func TestApplySettingsStoresLevelAndPoints(t *testing.T) {
	gs := newTestState()
	gs.applySettings([]string{"vowels"}, nil, false, 0, LevelPro, 4)
	if gs.level.Level != LevelPro || gs.pointsSetting != pointsAuto {
		t.Fatalf("level=%v points=%d (4 is invalid → Auto)", gs.level.Level, gs.pointsSetting)
	}
}
```

Append to `fyne/game_lifecycle_test.go`:

```go
func TestStoredDifficultyIsLoaded(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveDifficulty("fast")
	_ = st.SavePointsFactor(5)
	gs := NewGameState(st)
	if gs.level.Level != LevelFast || gs.pointsSetting != 5 {
		t.Fatalf("level=%v points=%d", gs.level.Level, gs.pointsSetting)
	}
	_ = st.SaveDifficulty("nonsense")
	if NewGameState(st).level.Level != LevelNormal {
		t.Fatal("unknown stored level must load as Normal")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/`
Expected: FAIL to compile (`unknown field level`, `gs.tempoStep undefined`, `too many arguments in call to gs.applySettings` …).

- [ ] **Step 3: Implement in `fyne/game.go`**

1. Add to `GameState` (near `autoProgress`):

```go
	level         LevelInfo // chosen difficulty
	pointsSetting int       // pointsAuto or a fixed factor
	answerStreak  int       // correct answers in a row this game
	tempoStep     int       // session tempo, 0..maxTempoStep
```

2. In `NewGameState`'s struct literal add `level: LevelByName(""),`. Inside `if st != nil {`, next to the other settings loads:

```go
		if name, err := st.Difficulty(); err == nil {
			gs.level = LevelByName(name)
		}
		if p, err := st.PointsFactor(); err == nil {
			gs.pointsSetting = validPoints(p)
		}
```

3. Replace the `minFallSeconds`/`maxFallSeconds` constants and `fallStep`:

```go
const ticksPerSecond = 10

// fallStep is how far a tile moves per tick so that it crosses a canvas of
// height h in fallSeconds, whatever the window size. A missing fall time
// uses fallback.
func fallStep(h, fallSeconds, fallback float32) float32 {
	if fallSeconds <= 0 {
		fallSeconds = fallback
	}
	return h / (fallSeconds * ticksPerSecond)
}
```

In `tick`, call `fallStep(gs.canvasH, tile.fallSeconds, gs.level.MinFall)`.

4. Replace `spawnLoop`:

```go
func (gs *GameState) spawnLoop() {
	gs.mu.Lock()
	stop := gs.stopCh
	interval := gs.spawnInterval()
	gs.mu.Unlock()

	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			gs.spawnKana()
			gs.mu.Lock()
			interval = gs.spawnInterval()
			gs.mu.Unlock()
			timer.Reset(interval)
		}
	}
}

// spawnInterval is the level's spawn interval at the current tempo. Must be
// called under lock.
func (gs *GameState) spawnInterval() time.Duration {
	return time.Duration(float64(gs.level.SpawnEvery) * float64(tempoFactor(gs.tempoStep)))
}
```

5. In `spawnKana`, replace the `tile.fallSeconds = …` line with:

```go
	correct := gs.overallStats[char].CorrectCount + gs.sessionStats[char].CorrectCount
	base := gs.level.MinFall + rand.Float32()*(gs.level.MaxFall-gs.level.MinFall)
	tile.fallSeconds = base * charFallFactor(correct) * tempoFactor(gs.tempoStep)
```

6. In `checkAnswer`, replace `gs.score += 10` with `gs.score += 10 * gs.pointsFactor()` and add:

```go
// pointsFactor is the points multiplier in effect. Must be called under lock.
func (gs *GameState) pointsFactor() int {
	if gs.pointsSetting == pointsAuto {
		return gs.level.PointsFactor
	}
	return gs.pointsSetting
}
```

7. In `recordCorrect`, after `gs.sessionDirty = true`:

```go
	gs.answerStreak++
	if gs.answerStreak%streakPerStep == 0 && gs.tempoStep < maxTempoStep {
		gs.tempoStep++
	}
```

8. In `recordMiss`, after `gs.sessionDirty = true`:

```go
	gs.answerStreak = 0
	gs.tempoStep -= missTempoSteps
	if gs.tempoStep < 0 {
		gs.tempoStep = 0
	}
```

9. In `checkMissedLimit`, replace `gs.missed >= 10` with `gs.missed >= gs.level.MissLimit`.

10. In `Reset`, next to `gs.missed = 0`, add `gs.answerStreak = 0` and `gs.tempoStep = 0`.

11. Change `applySettings`'s signature to
`func (gs *GameState) applySettings(rows []string, scripts []kanacore.Script, auto bool, limit int, level Level, points int)`
and, next to `gs.autoProgress = auto`, add:

```go
	gs.level = LevelByName(string(level))
	gs.pointsSetting = validPoints(points)
```

and inside the `if gs.store != nil` block:

```go
		_ = gs.store.SaveDifficulty(string(gs.level.Level))
		_ = gs.store.SavePointsFactor(gs.pointsSetting)
```

12. In `fyne/settings.go`, the existing `gs.applySettings(newRows, checkedScripts(scriptChecks), newAuto, newLimit)` call temporarily passes `, gs.level.Level, gs.pointsSetting` read under the lock at the top of `showSettingsDialog` (`currentLevel := gs.level.Level`, `currentPoints := gs.pointsSetting`). Task 3 replaces this with the real controls.

13. `fyne/stats.go` `StatsSnapshot` (the type) gets the fields; `snapshot()` in `fyne/game.go` fills them. Add to the type

```go
	LevelLabel string
	Tempo      float32 // shown speed-up, 1 / tempoFactor
	MissLimit  int
```

and in `snapshot()` set `LevelLabel: gs.level.Label, Tempo: 1 / tempoFactor(gs.tempoStep), MissLimit: gs.level.MissLimit,`.

- [ ] **Step 4: Run all tests**

Run: `go vet ./... && go test -race ./...`
Expected: PASS. If any other test builds `&GameState{…}` directly (`grep -n "&GameState{" fyne/*_test.go`), give it `level: LevelByName("")` too.

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/stats.go fyne/settings.go fyne/game_test.go fyne/game_lifecycle_test.go
git commit -F - <<'EOF'
feat(fyne): play at the chosen difficulty with a session tempo

Spawned tiles take their fall time from the level, the character's
learning state and the session tempo; the spawn timer follows the same
tempo. Every five correct answers in a row raise the tempo, a miss
takes two steps back. Points use the level's or the chosen factor, and
the miss limit comes from the level.
EOF
```

---

### Task 3: Settings, input bar and game-over dialog

**Files:** modify `fyne/settings.go`, `fyne/input.go`, `fyne/app.go`; create `fyne/difficulty_ui_test.go`.

- [ ] **Step 1: Write the failing tests**

Create `fyne/difficulty_ui_test.go`:

```go
package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestInputBarShowsLevelTempoAndLimit(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.level = LevelByName("pro")
	gs.missed = 2
	ib := newInputBar(gs, newStatsPanel(), newGameCanvas(gs), test.NewWindow(nil))
	ib.Update(gs.snapshot())
	if ib.missedLabel.Text != "Missed: 2/5" {
		t.Fatalf("missed label = %q", ib.missedLabel.Text)
	}
	if !strings.Contains(ib.scoreLabel.Text, "· Pro · Tempo ×1.00") {
		t.Fatalf("score label = %q", ib.scoreLabel.Text)
	}
}

func TestLevelAndPointsSettingsMapping(t *testing.T) {
	if levelFromLabel("Fast") != LevelFast || levelFromLabel("?") != LevelNormal {
		t.Fatal("levelFromLabel maps labels to levels, unknown → Normal")
	}
	if pointsFromLabel("×5") != 5 || pointsFromLabel("Auto (by level)") != pointsAuto {
		t.Fatal("pointsFromLabel maps labels to settings")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'InputBarShowsLevel|LevelAndPointsSettings'`
Expected: FAIL (`undefined: levelFromLabel`; the missed label still reads `/10`).

- [ ] **Step 3: Implement**

`fyne/input.go`:
- Replace the body of `Update`:

```go
func (ib *InputBar) Update(snap StatsSnapshot) {
	ib.scoreLabel.SetText(formatStatus(snap.Score, snap.ScoreLimit, snap.LevelLabel, snap.Tempo))
	ib.missedLabel.SetText(fmt.Sprintf("Missed: %d/%d", snap.Missed, snap.MissLimit))
}
```

- In `OnSubmitted`, replace the two `SetText` lines for the score and missed labels with `ib.Update(snap)`.
- Delete `formatScore` if nothing else uses it (`grep -n formatScore fyne/*.go`).

`fyne/app.go`:
- In `buildWindow`'s initial render, after `statsPanel.Update(snap)`, add `inputBar.Update(snap)`.
- In `showGameOverDialog`, replace `reasonText = "10 kana slipped through."` with `reasonText = missReason(snap.Missed)`. Replace `fmt.Sprintf("Missed: %d/10", snap.Missed)` with `fmt.Sprintf("Missed: %d/%d", snap.Missed, snap.MissLimit)`.

`fyne/settings.go`:
- Add helpers at the end of the file:

```go
// levelFromLabel maps a difficulty label back to its level; unknown → Normal.
func levelFromLabel(label string) Level {
	for _, l := range Levels() {
		if l.Label == label {
			return l.Level
		}
	}
	return LevelNormal
}

// pointsFromLabel maps a "Points per hit" label back to its setting.
func pointsFromLabel(label string) int {
	for _, p := range pointsChoices {
		if pointsLabel(p) == label {
			return p
		}
	}
	return pointsAuto
}
```

- Build the controls after `limitEntry`:

```go
	levelLabels := make([]string, 0, len(Levels()))
	for _, l := range Levels() {
		levelLabels = append(levelLabels, l.Label)
	}
	levelRadio := widget.NewRadioGroup(levelLabels, nil)
	levelRadio.Horizontal = true
	levelRadio.Required = true
	levelRadio.SetSelected(LevelByName(string(currentLevel)).Label)

	pointsLabels := make([]string, 0, len(pointsChoices))
	for _, p := range pointsChoices {
		pointsLabels = append(pointsLabels, pointsLabel(p))
	}
	pointsSelect := widget.NewSelect(pointsLabels, nil)
	pointsSelect.SetSelected(pointsLabel(currentPoints))
```

- In the form `container.NewVBox(…)`, insert after `tabs,`:

```go
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Difficulty", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		levelRadio,
		container.NewHBox(widget.NewLabel("Points per hit"), pointsSelect),
```

- Change the tab scroll's `SetMinSize(fyne.NewSize(460, 280))` to `fyne.NewSize(460, 220)`, so the taller dialog still fits the 1000 × 760 window.
- In the save callback, pass the controls: `gs.applySettings(newRows, checkedScripts(scriptChecks), newAuto, newLimit, levelFromLabel(levelRadio.Selected), pointsFromLabel(pointsSelect.Selected))`. After `statsPanel.Update(snap)`, add `inputBar.Update(snap)`.

- [ ] **Step 4: Run all tests and build**

Run: `go vet ./... && go test -race ./... && go build -o /tmp/claude-1000/-home-marcus-git-Comradin-kana/fb97d1f8-609d-4b58-abed-f955011bd160/scratchpad/kana-desktop-diff ./fyne/`
Expected: PASS and a binary.

- [ ] **Step 5: Commit**

```bash
git add fyne/settings.go fyne/input.go fyne/app.go fyne/difficulty_ui_test.go
git commit -F - <<'EOF'
feat(fyne): choose difficulty and points per hit in the settings

The settings dialog offers the four levels and the points factor. The
input bar shows level and tempo next to the score and the level's miss
limit, also right after start and after saving the settings, and the
game-over dialog uses the same limit.
EOF
```

---

### Task 4: Docs and verification

- [ ] **Step 1:** In `README.md`, describe the four levels (table from the spec), the per-character factor, the session tempo and the points setting. Update the "Game Timing" section (fall time and spawn interval now depend on the level) and the controls/settings description. Mention `fyne/difficulty.go` in the architecture list.
- [ ] **Step 2:** In `PLAN.md`, tick or adjust the difficulty items under "Difficulty & Progression" that this covers: presets, dynamic adjustment, spawn rate, speed by mastery.
- [ ] **Step 3:** Run `go vet ./... && go test -race -count=1 ./...` and `gofmt -l .`. Expected: all pass; `gofmt` prints nothing.
- [ ] **Step 4:** Commit with `docs: describe difficulty levels and the session tempo`.
- [ ] **Step 5: Report.** These checks are manual and for the user:
  - the settings dialog fits the window;
  - Pro feels clearly faster;
  - the tempo shown rises with streaks and drops on misses;
  - the miss limit ends the game at the right count.
