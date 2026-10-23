package main

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/store"
)

// learnKana puts each character at the learned level.
func learnKana(gs *GameState, chars ...string) {
	for _, c := range chars {
		gs.kanaProgress[c] = store.Progress{Level: learnedLevel}
	}
	gs.invalidatePlayable()
}

// setKanaLevel puts one character at level n, not flagged.
func setKanaLevel(gs *GameState, char string, n int) {
	gs.kanaProgress[char] = store.Progress{Level: n}
	gs.invalidatePlayable()
}

// setWordLevel puts one reading at level n, not flagged.
func setWordLevel(gs *GameState, kana string, n int) {
	gs.wordProgress[kana] = store.Progress{Level: n}
}

func TestHitAndMistake(t *testing.T) {
	p := store.Progress{}
	for want := 1; want <= 5; want++ {
		if p = hit(p); p.Level != want {
			t.Fatalf("hit: level %d, want %d", p.Level, want)
		}
	}
	if p = hit(p); p.Level != learnedLevel {
		t.Fatalf("level capped at 5, got %d", p.Level)
	}
	if p = mistake(p); p != (store.Progress{Level: 3, Flagged: true}) {
		t.Fatalf("mistake from 5 = %+v, want level 3 flagged", p)
	}
	if p = mistake(mistake(p)); p != (store.Progress{Level: 0, Flagged: true}) {
		t.Fatalf("level floored at 0, got %+v", p)
	}
	p = store.Progress{Level: 4, Flagged: true}
	if p = hit(p); p != (store.Progress{Level: 5}) {
		t.Fatalf("reaching 5 clears the flag, got %+v", p)
	}
	if !learned(5) || learned(4) {
		t.Fatal("learned means level 5")
	}
}

func TestKanaHitAndMissChangeTheLevel(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	setKanaLevel(gs, "か", 4)
	gs.recordCorrect("か")
	if p := gs.kanaProgress["か"]; p.Level != 5 || !gs.kanaProgressDirty["か"] {
		t.Fatalf("after a hit: %+v dirty %v", p, gs.kanaProgressDirty["か"])
	}
	gs.recordMiss("か")
	if p := gs.kanaProgress["か"]; p != (store.Progress{Level: 3, Flagged: true}) {
		t.Fatalf("after a miss: %+v", p)
	}
}

func TestWordHitAndMissChangeTheLevel(t *testing.T) {
	gs := newTestState()
	gs.recordWordCorrect("あい", readingKind)
	gs.recordWordCorrect("あい", readingKind)
	if p := gs.wordProgress["あい"]; p.Level != 2 || !gs.wordProgressDirty["あい"] {
		t.Fatalf("after two hits: %+v", p)
	}
	gs.recordWordMiss("あい", readingKind)
	if p := gs.wordProgress["あい"]; p != (store.Progress{Level: 0, Flagged: true}) {
		t.Fatalf("after a miss: %+v", p)
	}
	if gs.kanaProgress["あ"].Level != 0 {
		t.Fatal("a word miss must not touch its kana")
	}
}

func TestLevelRisesBeforeAutoProgression(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels")
	learnKana(gs, "あ", "い", "う")
	setKanaLevel(gs, "え", 4)
	gs.recordCorrect("え") // the hit that reaches 5 completes 80 % of the row
	if gs.kanaProgress["え"].Level != 5 {
		t.Fatal("level must rise in recordCorrect")
	}
	if !gs.selectedRows["k"] || !equalIDs(gs.pendingIntro, []string{"k", "s"}) {
		t.Fatalf("the hit that learns え must unlock the k/s step: rows %v, pending %v", gs.selectedRows, gs.pendingIntro)
	}
}

func TestRowsStayUnlockedAfterALearnedKanaDrops(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels", "k")
	learnKana(gs, "あ", "い", "う", "え", "お")
	gs.recordMiss("あ")
	if !gs.selectedRows["k"] {
		t.Fatal("unlocked rows stay when a kana drops below 5")
	}
}

func TestWordPausesWhileItsKanaIsBelowLearned(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	learnKana(gs, "あ", "い")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable = %v", got)
	}
	gs.recordMiss("あ")
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("あ dropped to 3: playable = %v, want none", got)
	}
	gs.recordCorrect("あ")
	gs.recordCorrect("あ")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("あ learned again: playable = %v", got)
	}
}

func TestFallingWordKeepsFallingWhenItsKanaDrops(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	learnKana(gs, "あ", "い")
	word := wordTileAt(gs, "あい", 100)
	gs.tiles = []*KanaTile{word}
	gs.recordMiss("あ")
	gs.tick()
	if len(gs.tiles) != 1 || gs.tiles[0] != word {
		t.Fatal("a word already falling keeps falling; only new spawns stop")
	}
}

func TestProgressSavedAndReloaded(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.recordCorrect("か")
	gs.recordMiss("き")
	gs.recordWordCorrect("あい", readingKind)
	gs.mergeSessionStats()
	dirty := len(gs.kanaProgressDirty) + len(gs.wordProgressDirty)
	gs.mu.Unlock()
	if dirty != 0 {
		t.Fatalf("%d progress entries still dirty after the merge", dirty)
	}
	again := NewGameState(st)
	if again.kanaProgress["か"].Level != 1 || again.kanaProgress["き"] != (store.Progress{Flagged: true}) || again.wordProgress["あい"].Level != 1 {
		t.Fatalf("reloaded kana %v words %v", again.kanaProgress, again.wordProgress)
	}
}

func TestProgressNotWrittenWhenLoadFailed(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveProgress("kana:か", store.Progress{Level: 5})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.kanaProgressLoadFailed = true
	gs.kanaProgress = map[string]store.Progress{}
	gs.recordCorrect("か")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if p, _ := st.Progress("kana:"); p["か"].Level != 5 {
		t.Fatalf("stored progress overwritten: %+v", p["か"])
	}
}

func TestFlagShownAfterRestart(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.recordMiss("か")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	again := NewGameState(st)
	again.mu.Lock()
	snap := again.snapshot()
	again.mu.Unlock()
	if !snap.Flagged["か"] {
		t.Fatal("the red frame must survive a restart")
	}
}

func TestWrongInputWithOneKanaTile(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	setKanaLevel(gs, "か", 4)
	tile := tileAt("か", "ka", 100)
	gs.tiles = []*KanaTile{tile}
	gs.tempoStep, gs.answerStreak = 3, 2
	gs.checkAnswer("ki")
	if len(gs.tiles) != 1 || gs.tiles[0] != tile {
		t.Fatal("the tile stays on the field")
	}
	if p := gs.kanaProgress["か"]; p != (store.Progress{Level: 2, Flagged: true}) {
		t.Fatalf("progress = %+v, want level 2 flagged", p)
	}
	if gs.sessionStats["か"].MissCount != 1 || gs.currentStreak["か"] != 0 {
		t.Fatal("a wrong input is a session miss and resets the character's streak")
	}
	if gs.missed != 0 || gs.tempoStep != 3 || gs.answerStreak != 2 || len(gs.missedKanas) != 0 {
		t.Fatal("a wrong input does not count towards the miss limit or the tempo")
	}
}

func TestWrongInputWithOneWordTile(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	learnKana(gs, "あ", "い")
	setWordLevel(gs, "あい", 3)
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 100)}
	gs.checkAnswer("ao")
	if p := gs.wordProgress["あい"]; p != (store.Progress{Level: 1, Flagged: true}) {
		t.Fatalf("word progress = %+v", p)
	}
	if gs.kanaProgress["あ"].Level != 5 || gs.kanaProgress["い"].Level != 5 {
		t.Fatal("a word mistake does not touch its kana")
	}
}

func TestWrongInputIgnored(t *testing.T) {
	test.NewApp()
	cases := map[string]func(gs *GameState) string{
		"two tiles": func(gs *GameState) string {
			gs.tiles = []*KanaTile{tileAt("か", "ka", 100), tileAt("き", "ki", 50)}
			return "ku"
		},
		"blank input": func(gs *GameState) string {
			gs.tiles = []*KanaTile{tileAt("か", "ka", 100)}
			return "  "
		},
		"paused": func(gs *GameState) string {
			gs.tiles = []*KanaTile{tileAt("か", "ka", 100)}
			gs.paused = true
			return "ki"
		},
		"over": func(gs *GameState) string {
			gs.tiles = []*KanaTile{tileAt("か", "ka", 100)}
			gs.over = true
			return "ki"
		},
	}
	for name, setup := range cases {
		gs := newTestState()
		gs.checkAnswer(setup(gs))
		if len(gs.kanaProgress) != 0 || len(gs.sessionStats) != 0 {
			t.Errorf("%s: progress %v, stats %v; want no mistake", name, gs.kanaProgress, gs.sessionStats)
		}
	}
}

func TestLateEnterForAFallenTileIsIgnored(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	gs.now = func() time.Time { return now }
	fallen := tileAt("か", "ka", gs.canvasH-1)
	fallen.fallSeconds = 0.1 // falls through on the next tick
	other := tileAt("き", "ki", 10)
	gs.tiles = []*KanaTile{other, fallen}
	gs.tick()
	if len(gs.tiles) != 1 {
		t.Fatalf("want か fallen through, tiles = %d", len(gs.tiles))
	}
	now = now.Add(time.Second)
	gs.checkAnswer("ka") // meant for か, too late
	if p := gs.kanaProgress["き"]; p.Level != 0 || p.Flagged {
		t.Fatalf("late Enter punished き: %+v", p)
	}
	now = now.Add(time.Second) // 2 s after the fall
	gs.checkAnswer("ka")
	if !gs.kanaProgress["き"].Flagged {
		t.Fatal("after 1.5 s a wrong input counts again")
	}
}
