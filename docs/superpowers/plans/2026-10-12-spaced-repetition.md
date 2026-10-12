# Spaced Repetition Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Choose which kana spawn by a forgetting-curve model: likely-forgotten characters come often, secure ones rarely but never not at all, and new ones at most 40 % of the time.

**Architecture:**
- **`memory`:** a new pure package with state, retrievability, update rules, seeding and a two-stage weighted `Pick`.
- **`store`:** keeps one `memory` row per item and seeds it once from `kana_stats` in a transaction.
- **`GameState`:** loads the states, updates them on answers and misses, saves them first in `mergeSessionStats`, and spawns through `memory.Pick` without duplicating falling characters.

**Tech Stack:** Go 1.25, Fyne v2.7.3 (CGO), `modernc.org/sqlite`, `testing`.

**Spec:** `docs/superpowers/specs/2026-10-12-spaced-repetition-design.md`

**Branch:** `feat/spaced-repetition`.

**Conventions:**
- Commit messages: `type(scope): summary` plus a short paragraph.
- Never add `Co-Authored-By` or any AI/session trailer. Do not push.
- Run `gofmt -w` on changed files.
- The first `go test ./fyne/` builds Fyne via CGO and can take minutes.
- Test helpers:
  - `fyne/game_test.go`: `newTestState()`, `tileAt`, `selectRows`, `masterRows`, `equalIDs`.
  - `fyne/game_lifecycle_test.go`: `openTestStore(t)`.
  - `store/store_test.go`: `openTemp(t)`.

---

### Task 1: The `memory` package

**Files:** create `memory/memory.go`, `memory/memory_test.go`.

- [ ] **Step 1: Write the failing tests** — `memory/memory_test.go`:

```go
package memory

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)

func after(days float64) time.Time {
	return t0.Add(time.Duration(days * 24 * float64(time.Hour)))
}

func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestRetrievability(t *testing.T) {
	s := State{Stability: 2, LastSeen: t0}
	if r := Retrievability(s, t0); r != 1 {
		t.Fatalf("R at 0 = %v", r)
	}
	if r := Retrievability(s, after(2)); !near(r, 0.9, 1e-9) {
		t.Fatalf("R at S = %v", r)
	}
	if r := Retrievability(s, after(4)); !near(r, 0.81, 1e-9) {
		t.Fatalf("R at 2S = %v", r)
	}
}

func TestCorrectAndMiss(t *testing.T) {
	if s := Correct(State{}, false, t0); s.Stability != MinStability || !s.LastSeen.Equal(t0) {
		t.Fatalf("first correct = %+v", s)
	}
	s := State{Stability: 2, LastSeen: t0}
	if got := Correct(s, true, after(2)).Stability; !near(got, 2*1.7, 1e-9) {
		t.Fatalf("correct at R=0.9 → %v, want 3.4", got)
	}
	halfLife := 2 * math.Log(0.5) / math.Log(0.9) // Δt where R = 0.5
	if got := Correct(s, true, after(halfLife)).Stability; !near(got, 2*2.5, 1e-6) {
		t.Fatalf("correct at R=0.5 → %v, want 5", got)
	}
	if got := Correct(State{Stability: 300, LastSeen: t0}, true, after(300)).Stability; got != MaxStability {
		t.Fatalf("cap → %v", got)
	}
	missed := Miss(s, after(1))
	if !near(missed.Stability, 0.6, 1e-9) {
		t.Fatalf("miss → %v, want 0.6", missed.Stability)
	}
	if r := Retrievability(missed, after(1)); !near(r, 0.5, 1e-6) {
		t.Fatalf("R right after a miss = %v, want 0.5", r)
	}
	if got := Miss(State{Stability: MinStability, LastSeen: t0}, t0).Stability; got != MinStability {
		t.Fatalf("miss floor → %v", got)
	}
}

func TestWeightAndSeed(t *testing.T) {
	if w := Weight(State{Stability: 2, LastSeen: t0}, t0); w != WeightFloor {
		t.Fatalf("weight at R=1 = %v", w)
	}
	if w := Weight(State{Stability: 2, LastSeen: t0}, after(4)); !near(w, 0.19, 1e-9) {
		t.Fatalf("weight at R=0.81 = %v", w)
	}
	if s := Seed(3, t0); !near(s.Stability, MinStability*8, 1e-12) {
		t.Fatalf("Seed(3) = %v", s.Stability)
	}
	if s := Seed(10, t0); !near(s.Stability, 7.11, 0.01) {
		t.Fatalf("Seed(10) = %v days", s.Stability)
	}
	if s := Seed(50, t0); s.Stability > MaxStability {
		t.Fatalf("Seed must cap at MaxStability")
	}
}

func share(n int, f func() bool) float64 {
	hits := 0
	for i := 0; i < n; i++ {
		if f() {
			hits++
		}
	}
	return float64(hits) / float64(n)
}

func TestPickNewShare(t *testing.T) {
	rnd := rand.New(rand.NewSource(1)).Float64
	states := map[string]State{}
	var cands []string
	for i := 0; i < 20; i++ {
		c := string(rune('a' + i))
		cands = append(cands, c)
		if i >= 10 {
			states[c] = State{Stability: 10, LastSeen: t0}
		}
	}
	got := share(10000, func() bool { _, known := states[Pick(cands, states, t0, rnd)]; return !known })
	if !near(got, 0.40, 0.02) {
		t.Fatalf("new share with 10/10 = %v, want 0.40", got)
	}

	for i := 1; i < 10; i++ {
		states[cands[i]] = State{Stability: 10, LastSeen: t0}
	}
	got = share(10000, func() bool { _, known := states[Pick(cands, states, t0, rnd)]; return !known })
	if !near(got, 0.05, 0.015) {
		t.Fatalf("new share with 1/19 = %v, want ~0.05", got)
	}
}

func TestPickPrefersWeak(t *testing.T) {
	rnd := rand.New(rand.NewSource(2)).Float64
	halfLife := 2 * math.Log(0.5) / math.Log(0.9)
	states := map[string]State{
		"weak":   {Stability: 2, LastSeen: t0},
		"strong": {Stability: 2, LastSeen: after(halfLife)},
	}
	now := after(halfLife) // weak at R=0.5 (w 0.5), strong at R=1 (w 0.05)
	weak := share(11000, func() bool { return Pick([]string{"weak", "strong"}, states, now, rnd) == "weak" })
	ratio := weak / (1 - weak)
	if ratio < 8.5 || ratio > 11.5 {
		t.Fatalf("weak:strong = %v, want ~10", ratio)
	}
}

func TestPickEdges(t *testing.T) {
	rnd := rand.New(rand.NewSource(3)).Float64
	known := map[string]State{"k": {Stability: 1, LastSeen: t0}}
	if Pick(nil, known, t0, rnd) != "" {
		t.Fatal("empty candidates → empty string")
	}
	for i := 0; i < 100; i++ {
		if Pick([]string{"n1", "n2"}, known, t0, rnd) == "" {
			t.Fatal("only new candidates must still pick one")
		}
		if Pick([]string{"k"}, known, t0, rnd) != "k" {
			t.Fatal("only known candidates → known")
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./memory/`
Expected: FAIL (`undefined: State` …).

- [ ] **Step 3: Implement** — `memory/memory.go`:

```go
// Package memory models how well a learner remembers an item, so practice
// can favour what is about to be forgotten.
package memory

import (
	"math"
	"time"
)

// State is what the model knows about one item.
type State struct {
	Stability float64 // days until recall drops to 90 %
	LastSeen  time.Time
}

const (
	MinStability = 10.0 / (24 * 60) // 10 minutes, in days
	MaxStability = 365.0
	NewCap       = 0.4  // highest share of new items in Pick
	WeightFloor  = 0.05 // lowest pick weight of a known item
)

func elapsedDays(s State, now time.Time) float64 {
	return now.Sub(s.LastSeen).Hours() / 24
}

func clamp(v float64) float64 {
	return math.Min(math.Max(v, MinStability), MaxStability)
}

// Retrievability is the estimated chance of recalling the item now.
func Retrievability(s State, now time.Time) float64 {
	dt := elapsedDays(s, now)
	if dt <= 0 {
		return 1
	}
	return math.Pow(0.9, dt/clamp(s.Stability))
}

// Correct updates a state after a correct answer. An item that was not known
// starts at MinStability; a known one grows more the harder the recall was.
func Correct(s State, known bool, now time.Time) State {
	if !known {
		return State{Stability: MinStability, LastSeen: now}
	}
	r := Retrievability(s, now)
	return State{Stability: clamp(s.Stability * (1.5 + 2*(1-r))), LastSeen: now}
}

// Miss updates a known item's state after it was not recognised. The
// stability shrinks, and the item is treated as half forgotten (R = 0.5 at
// now), so it is picked often again soon after the miss.
func Miss(s State, now time.Time) State {
	st := clamp(s.Stability * 0.3)
	back := st * math.Log(0.5) / math.Log(0.9) // days until R = 0.5
	return State{Stability: st, LastSeen: now.Add(-time.Duration(back * 24 * float64(time.Hour)))}
}

// Weight is how strongly Pick favours a known item: its chance of being
// forgotten, but never below WeightFloor.
func Weight(s State, now time.Time) float64 {
	return math.Max(1-Retrievability(s, now), WeightFloor)
}

// Seed turns a count of past correct answers into a starting state.
func Seed(correct int, now time.Time) State {
	n := math.Min(math.Max(float64(correct), 0), 10)
	return State{Stability: clamp(MinStability * math.Pow(2, n)), LastSeen: now}
}

// Pick chooses one candidate. Candidates missing from states are new: they
// are chosen with probability min(NewCap, their share), uniformly. Otherwise
// a known candidate is chosen, weighted by Weight. rnd returns [0, 1).
func Pick(candidates []string, states map[string]State, now time.Time, rnd func() float64) string {
	if len(candidates) == 0 {
		return ""
	}
	var fresh, known []string
	for _, c := range candidates {
		if _, ok := states[c]; ok {
			known = append(known, c)
		} else {
			fresh = append(fresh, c)
		}
	}
	pNew := math.Min(NewCap, float64(len(fresh))/float64(len(candidates)))
	switch {
	case len(known) == 0:
		pNew = 1
	case len(fresh) == 0:
		pNew = 0
	}
	if rnd() < pNew {
		i := int(rnd() * float64(len(fresh)))
		if i >= len(fresh) {
			i = len(fresh) - 1
		}
		return fresh[i]
	}
	weights := make([]float64, len(known))
	total := 0.0
	for i, c := range known {
		weights[i] = Weight(states[c], now)
		total += weights[i]
	}
	x := rnd() * total
	for i, w := range weights {
		if x < w {
			return known[i]
		}
		x -= w
	}
	return known[len(known)-1]
}
```

- [ ] **Step 4: Run** `go vet ./memory/ && go test ./memory/`. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add memory
git commit -F - <<'EOF'
feat(memory): add a forgetting-curve model with weighted picking

Each item has a stability and a last-seen time; recall decays as
0.9^(days/stability), correct answers grow the stability more the
harder the recall was, misses shrink it. Pick favours items likely to
be forgotten, keeps a floor for secure ones and caps new items at 40 %.
EOF
```

---

### Task 2: Store memory states and seed them once

**Files:** modify `store/store.go`, `store/store_test.go`.

- [ ] **Step 1: Write the failing tests** — append to `store/store_test.go`. Add the imports `"database/sql"`, `"math"`, `"time"` and `"kana/memory"`.

```go
func TestMemoryStatesRoundTrip(t *testing.T) {
	st := openTemp(t)
	seen := time.Date(2026, 10, 12, 8, 30, 0, 0, time.UTC)
	if err := st.SaveMemoryState("kana:か", memory.State{Stability: 2, LastSeen: seen}); err != nil {
		t.Fatal(err)
	}
	_ = st.SaveMemoryState("word:x", memory.State{Stability: 1, LastSeen: seen})
	got, err := st.MemoryStates("kana:")
	if err != nil || len(got) != 1 {
		t.Fatalf("MemoryStates = %v, %v", got, err)
	}
	if s := got["か"]; s.Stability != 2 || s.LastSeen.Unix() != seen.Unix() {
		t.Fatalf("か = %+v", s)
	}
}

func TestMemorySeededOnceFromKanaStats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE kana_stats (char TEXT PRIMARY KEY, correct_count INTEGER NOT NULL DEFAULT 0, miss_count INTEGER NOT NULL DEFAULT 0, streak INTEGER NOT NULL DEFAULT 0)`,
		`INSERT INTO kana_stats VALUES ('あ', 0, 2, 0), ('か', 3, 0, 3)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Close()

	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := st.MemoryStates("kana:")
	if len(got) != 1 || math.Abs(got["か"].Stability-memory.MinStability*8) > 1e-9 {
		t.Fatalf("seeded = %+v", got)
	}
	_ = st.SaveMemoryState("kana:か", memory.State{Stability: 5, LastSeen: time.Now()})
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if got, _ := st.MemoryStates("kana:"); got["か"].Stability != 5 {
		t.Fatalf("re-open must not re-seed: %+v", got["か"])
	}
}
```

- [ ] **Step 2: Run** `go test ./store/`. Expected: FAIL to compile (`st.SaveMemoryState undefined`).

- [ ] **Step 3: Implement** in `store/store.go`:
- Add the import `"kana/memory"`.
- At the end of `migrate`, after its loop, add `return ensureMemory(ctx, db, time.Now())` in place of the final `return nil`.
- Add:

```go
// ensureMemory creates the memory table if it is missing and, in the same
// transaction, seeds it from kana_stats so earlier progress counts as known.
// If anything fails nothing is committed and the next open tries again.
func ensureMemory(ctx context.Context, db *sql.DB, now time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin memory migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var name string
	err = tx.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='memory'`).Scan(&name)
	if err == nil {
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("store: check memory table: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `CREATE TABLE memory (
		item TEXT PRIMARY KEY,
		stability REAL NOT NULL,
		last_seen INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("store: create memory table: %w", err)
	}

	type seed struct {
		char    string
		correct int
	}
	rows, err := tx.QueryContext(ctx, `SELECT char, correct_count FROM kana_stats WHERE correct_count > 0`)
	if err != nil {
		return fmt.Errorf("store: read kana stats for seeding: %w", err)
	}
	var seeds []seed
	for rows.Next() {
		var s seed
		if err := rows.Scan(&s.char, &s.correct); err != nil {
			rows.Close()
			return fmt.Errorf("store: scan kana stats for seeding: %w", err)
		}
		seeds = append(seeds, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: iterate kana stats for seeding: %w", err)
	}

	for _, s := range seeds {
		st := memory.Seed(s.correct, now)
		if _, err := tx.ExecContext(ctx, `INSERT INTO memory (item, stability, last_seen) VALUES (?, ?, ?)`,
			"kana:"+s.char, st.Stability, st.LastSeen.Unix()); err != nil {
			return fmt.Errorf("store: seed memory: %w", err)
		}
	}
	return tx.Commit()
}

// MemoryStates returns the memory states of all items starting with prefix,
// keyed by the rest of the item ID.
func (s *Store) MemoryStates(prefix string) (map[string]memory.State, error) {
	rows, err := s.db.Query(`SELECT item, stability, last_seen FROM memory WHERE substr(item, 1, length(?)) = ?`, prefix, prefix)
	if err != nil {
		return nil, fmt.Errorf("store: query memory: %w", err)
	}
	defer rows.Close()
	states := make(map[string]memory.State)
	for rows.Next() {
		var item string
		var stability float64
		var lastSeen int64
		if err := rows.Scan(&item, &stability, &lastSeen); err != nil {
			return nil, fmt.Errorf("store: scan memory: %w", err)
		}
		states[strings.TrimPrefix(item, prefix)] = memory.State{Stability: stability, LastSeen: time.Unix(lastSeen, 0)}
	}
	return states, rows.Err()
}

// SaveMemoryState stores one item's memory state.
func (s *Store) SaveMemoryState(item string, st memory.State) error {
	_, err := s.db.Exec(`
		INSERT INTO memory (item, stability, last_seen) VALUES (?, ?, ?)
		ON CONFLICT(item) DO UPDATE SET stability = excluded.stability, last_seen = excluded.last_seen
	`, item, st.Stability, st.LastSeen.Unix())
	if err != nil {
		return fmt.Errorf("store: save memory %s: %w", item, err)
	}
	return nil
}
```

`time`, `strings`, `errors` and `context` are already imported by `store.go`. Check and add any that are missing.

- [ ] **Step 4: Run** `go vet ./... && go test ./store/ ./fyne/`. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add store
git commit -F - <<'EOF'
feat(store): keep memory states and seed them from past answers

A memory table holds stability and last-seen time per item. When it is
first created, characters with correct answers get a starting stability
from their count, all in one transaction so a failed seed is retried on
the next start.
EOF
```

---

### Task 3: Spawn by memory

**Files:** modify `fyne/game.go`, `fyne/game_test.go`, `fyne/game_lifecycle_test.go`.

- [ ] **Step 1: Write the failing tests**

In `fyne/game_test.go`:
- Add the imports `"kana/memory"` and `"time"` if they are missing.
- In `newTestState`'s literal, add `memory: map[string]memory.State{}, memoryDirty: map[string]bool{}, now: time.Now,`.
- Append:

```go
func TestCorrectCreatesMemoryState(t *testing.T) {
	gs := newTestState()
	gs.recordCorrect("か")
	if s, ok := gs.memory["か"]; !ok || s.Stability != memory.MinStability || !gs.memoryDirty["か"] {
		t.Fatalf("memory[か] = %+v, %v", s, ok)
	}
}

func TestMissOnlyChangesKnownMemory(t *testing.T) {
	gs := newTestState()
	gs.recordMiss("か")
	if _, ok := gs.memory["か"]; ok {
		t.Fatal("a miss on a new character must not create memory")
	}
	gs.memory["き"] = memory.State{Stability: 1, LastSeen: time.Now()}
	gs.recordMiss("き")
	if s := gs.memory["き"].Stability; s < 0.29 || s > 0.31 {
		t.Fatalf("miss on known → %v, want 0.3", s)
	}
	if w := memory.Weight(gs.memory["き"], gs.now()); w < 0.45 {
		t.Fatalf("a just-missed character must be weighted high, got %v", w)
	}
}

func TestResetKeepsMemory(t *testing.T) {
	gs := newTestState()
	gs.recordCorrect("か")
	gs.Reset()
	if _, ok := gs.memory["か"]; !ok {
		t.Fatal("Reset must keep memory states")
	}
}

func TestSpawnAvoidsFallingDuplicates(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	for i := 0; i < 20; i++ {
		gs.tiles = []*KanaTile{tileAt("あ", "a", 0), tileAt("い", "i", 0), tileAt("う", "u", 0), tileAt("え", "e", 0)}
		gs.spawnKana()
		if got := gs.tiles[len(gs.tiles)-1].kana.Char; got != "お" {
			t.Fatalf("spawned %s while it was falling", got)
		}
	}
}

func TestSpawnPrefersWeakCharacter(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	gs.now = func() time.Time { return now }
	for _, c := range []string{"い", "う", "え", "お"} {
		gs.memory[c] = memory.State{Stability: 5, LastSeen: now} // R = 1 → weight 0.05
	}
	gs.memory["あ"] = memory.State{Stability: 1, LastSeen: now.Add(-7 * 24 * time.Hour)} // R ≈ 0.48
	weak := 0
	for i := 0; i < 500; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if gs.tiles[0].kana.Char == "あ" {
			weak++
		}
	}
	if weak < 300 { // expected about 0.52/(0.52+0.2) ≈ 72 %
		t.Fatalf("あ spawned %d/500 times, want > 300", weak)
	}
}
```

Append to `fyne/game_lifecycle_test.go`:

```go
func TestMergePersistsMemory(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.recordCorrect("か")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	got, _ := st.MemoryStates("kana:")
	if _, ok := got["か"]; !ok || len(gs.memoryDirty) != 0 {
		t.Fatalf("persisted = %v, dirty = %v", got, gs.memoryDirty)
	}
	if _, ok := NewGameState(st).memory["か"]; !ok {
		t.Fatal("memory must load on start")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/`. Expected: FAIL to compile (`unknown field memory`).

- [ ] **Step 3: Implement** in `fyne/game.go`:
1. Add the import `"kana/memory"`. `time` is already imported.
2. Add to `GameState` (near `overallStats`):

```go
	memory      map[string]memory.State // spaced-repetition state per character
	memoryDirty map[string]bool         // characters whose state is not saved yet
	now         func() time.Time        // clock, replaceable in tests
```

3. In `NewGameState`'s literal, add `memory: map[string]memory.State{},`, `memoryDirty: map[string]bool{},` and `now: time.Now,`, one field per line like the rest of the literal. Inside `if st != nil {`, add:

```go
		if ms, err := st.MemoryStates("kana:"); err == nil {
			gs.memory = ms
		}
```

4. In `recordCorrect`, after `gs.sessionDirty = true`:

```go
	prev, known := gs.memory[char]
	gs.memory[char] = memory.Correct(prev, known, gs.now())
	gs.memoryDirty[char] = true
```

5. In `recordMiss`, after `gs.sessionDirty = true`:

```go
	if prev, known := gs.memory[char]; known {
		gs.memory[char] = memory.Miss(prev, gs.now())
		gs.memoryDirty[char] = true
	}
```

6. At the very top of `mergeSessionStats`, before `if !gs.sessionDirty {`, add `gs.saveMemory()`, and add:

```go
// saveMemory writes changed memory states. It runs before the stats merge,
// so a failing stats read cannot lose them. Must be called under lock.
func (gs *GameState) saveMemory() {
	if gs.store == nil {
		return
	}
	for char := range gs.memoryDirty {
		if err := gs.store.SaveMemoryState("kana:"+char, gs.memory[char]); err == nil {
			delete(gs.memoryDirty, char)
		}
	}
}
```

7. In `spawnKana`, replace `char := chars[rand.Intn(len(chars))]` with:

```go
	char := memory.Pick(gs.spawnCandidates(chars), gs.memory, gs.now(), rand.Float64)
```

and add:

```go
// spawnCandidates drops characters that are already falling, unless that
// would leave none. Must be called under lock.
func (gs *GameState) spawnCandidates(chars []string) []string {
	falling := make(map[string]bool, len(gs.tiles))
	for _, t := range gs.tiles {
		falling[t.kana.Char] = true
	}
	out := make([]string, 0, len(chars))
	for _, c := range chars {
		if !falling[c] {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return chars
	}
	return out
}
```

`Reset` stays as it is: it must **not** reload `gs.memory`.

- [ ] **Step 4: Run** `go vet ./... && go test -race ./...`. Expected: PASS. If another test builds `&GameState{…}` directly (`grep -n "&GameState{" fyne/*_test.go`), initialise the three fields there too.

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/game_test.go fyne/game_lifecycle_test.go
git commit -F - <<'EOF'
feat(fyne): spawn kana by spaced-repetition weight

Each answer updates the character's memory state, and new tiles are
chosen with memory.Pick: likely-forgotten characters come more often,
secure ones rarely, new ones at most 40 % of the time, and a character
already falling is not spawned twice. States are saved before the stats
merge and loaded on start.
EOF
```

---

### Task 4: Docs and verification

- [ ] **Step 1:** In `README.md`:
  - Add a short "Spaced repetition" subsection explaining how tiles are chosen: the forgetting curve, recall weighting, the 5 % floor for secure kana, at most 40 % new and no duplicates on screen.
  - Add `memory/` to the architecture list.
  - Mention the `memory` table under persistence.
- [ ] **Step 2:** In `PLAN.md`, tick or annotate "Spawn Distribution" (weighting layer) as done via spaced repetition.
- [ ] **Step 3:** Run `go vet ./... && go test -race -count=1 ./...` and `gofmt -l .`. Expected: all pass, and `gofmt` prints nothing.
- [ ] **Step 4:** Commit `docs: describe spaced-repetition spawning`.
- [ ] **Step 5:** Report. The manual check is for the user: play a few rounds with mostly known kana and confirm that secure characters appear rarely but still appear, and that weak or long-unseen ones come up more often.
