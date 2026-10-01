package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/store"
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

func TestCheckAnswerIgnoredAfterGameOver(t *testing.T) {
	gs := newTestState()
	tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	gs.tiles = []*KanaTile{tile}
	gs.over = true
	gs.checkAnswer("ka")
	if len(gs.tiles) != 1 {
		t.Fatalf("expected tile to remain after game over, got %d tiles", len(gs.tiles))
	}
	if gs.score != 0 {
		t.Fatalf("expected score unchanged after game over, got %d", gs.score)
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
	// overallStats must NOT be touched by recordCorrect (see spec: stats double-counting fix)
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

func tileAt(char, romaji string, y float32) *KanaTile {
	tile := newKanaTile(kanacore.Kana{Char: char, Romaji: romaji, Speed: 5})
	tile.Move(fyne.NewPos(10, y))
	return tile
}

func TestCheckAnswerAcceptsAlternative(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("し", "shi", 0)}
	gs.checkAnswer("si")
	if len(gs.tiles) != 0 {
		t.Fatalf("expected し removed by 'si', %d tiles left", len(gs.tiles))
	}
}

func TestCheckAnswerNormalisesInput(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("か", "ka", 0)}
	gs.checkAnswer("  KA ")
	if len(gs.tiles) != 0 {
		t.Fatalf("expected か removed by '  KA ', %d tiles left", len(gs.tiles))
	}
}

func TestCheckAnswerRemovesLowestMatchingTile(t *testing.T) {
	gs := newTestState()
	high := tileAt("じ", "ji", 100)
	low := tileAt("ぢ", "ji", 300)
	gs.tiles = []*KanaTile{high, low}
	gs.checkAnswer("ji")
	if len(gs.tiles) != 1 || gs.tiles[0] != high {
		t.Fatalf("expected only the upper じ tile to remain")
	}
}

func TestSpawnKeepsWideTilesInsideCanvas(t *testing.T) {
	gs := newTestState()
	gs.selectedRows = map[string]bool{"ky": true}
	gs.canvasW = 100
	for i := 0; i < 50; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if tile.pos.X < 0 || tile.pos.X+tile.Width() > gs.canvasW {
			t.Fatalf("tile %s at x=%.1f width %.1f exceeds canvas width %.1f",
				tile.kana.Char, tile.pos.X, tile.Width(), gs.canvasW)
		}
	}
}

func TestIsRowMastered(t *testing.T) {
	gs := newTestState()
	row := kanacore.AllKanaRows[0] // vowels
	// Give 3 correct answers to 4 out of 5 characters (80%)
	for _, char := range row.Characters()[:4] {
		gs.overallStats[char] = store.KanaStats{Char: char, CorrectCount: 3}
	}
	if !gs.isRowMastered(row) {
		t.Error("expected row to be mastered at 80% threshold")
	}
}

func masterRows(gs *GameState, ids ...string) {
	for _, id := range ids {
		row, _ := kanacore.RowByID(id)
		for _, char := range row.Characters() {
			gs.overallStats[char] = store.KanaStats{Char: char, CorrectCount: 3}
		}
	}
}

func selectRows(gs *GameState, ids ...string) {
	gs.selectedRows = make(map[string]bool)
	for _, id := range ids {
		gs.selectedRows[id] = true
	}
}

func equalIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestProgressionCrossesIntoDakuon(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	basic := kanacore.DefaultRowIDs()
	selectRows(gs, basic...)
	masterRows(gs, basic...)
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"g", "z"}) {
		t.Fatalf("unlocked %v, want [g z]", got)
	}
}

func TestProgressionAfterHandakuonUnlocksYoon(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	ids := append(kanacore.DefaultRowIDs(), "g", "z", "d", "b", "p")
	selectRows(gs, ids...)
	masterRows(gs, ids...)
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"ky", "sy"}) {
		t.Fatalf("unlocked %v, want [ky sy]", got)
	}
}

func TestProgressionCompletesPartialStep(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels", "k")
	masterRows(gs, "vowels", "k")
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"s"}) {
		t.Fatalf("unlocked %v, want [s]", got)
	}
}

// newUnlockReadyState returns a state where one correct あ unlocks [k s].
func newUnlockReadyState() *GameState {
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels")
	masterRows(gs, "vowels")
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	return gs
}

func TestUnlockPausesAndQueuesEvent(t *testing.T) {
	gs := newUnlockReadyState()
	gs.checkAnswer("a")
	if !gs.paused {
		t.Fatal("expected game paused after unlock")
	}
	if !equalIDs(gs.pendingIntro, []string{"k", "s"}) {
		t.Fatalf("pendingIntro = %v, want [k s]", gs.pendingIntro)
	}
	select {
	case ev := <-gs.eventCh:
		if ev.kind != rowsUnlockedEvent || !equalIDs(ev.rows, []string{"k", "s"}) {
			t.Fatalf("unexpected event %+v", ev)
		}
	default:
		t.Fatal("expected rowsUnlockedEvent on eventCh")
	}
}

func TestUnlockWithFullEventChannelDoesNotPause(t *testing.T) {
	gs := newUnlockReadyState()
	for i := 0; i < cap(gs.eventCh); i++ {
		gs.eventCh <- gameEvent{kind: gameOverEvent}
	}
	gs.checkAnswer("a")
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("expected no pause without a delivered event (paused=%v, pending=%v)", gs.paused, gs.pendingIntro)
	}
	if !gs.selectedRows["k"] || !gs.selectedRows["s"] {
		t.Fatal("rows should still be unlocked")
	}
}

func TestPausedGameDoesNotMoveSpawnOrScore(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	gs.Pause()

	gs.tick()
	if y := gs.tiles[0].pos.Y; y != 0 {
		t.Fatalf("tile moved while paused: y=%.1f", y)
	}
	gs.spawnKana()
	if len(gs.tiles) != 1 {
		t.Fatalf("spawned while paused: %d tiles", len(gs.tiles))
	}
	gs.checkAnswer("a")
	if len(gs.tiles) != 1 || gs.score != 0 {
		t.Fatal("answer accepted while paused")
	}

	gs.Resume()
	gs.tick()
	if y := gs.tiles[0].pos.Y; y == 0 {
		t.Fatal("tile did not move after Resume")
	}
}

func TestFinishIntroClearsPendingAndResumes(t *testing.T) {
	gs := newUnlockReadyState()
	gs.checkAnswer("a")
	gs.FinishIntro()
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("after FinishIntro paused=%v pending=%v", gs.paused, gs.pendingIntro)
	}
}

func TestUnlockAndScoreLimitKeepIntroForPlayAgain(t *testing.T) {
	gs := newUnlockReadyState()
	gs.scoreLimit = 10
	gs.checkAnswer("a")
	if !gs.over {
		t.Fatal("expected game over at score limit")
	}
	if !equalIDs(gs.pendingIntro, []string{"k", "s"}) {
		t.Fatalf("pendingIntro = %v, want [k s]", gs.pendingIntro)
	}
	first := <-gs.eventCh
	second := <-gs.eventCh
	if first.kind != rowsUnlockedEvent || second.kind != gameOverEvent {
		t.Fatalf("event order = %v, %v", first.kind, second.kind)
	}

	gs.Reset()
	if !gs.paused || !equalIDs(gs.PendingIntro(), []string{"k", "s"}) {
		t.Fatalf("after Reset paused=%v pending=%v, want paused with [k s]", gs.paused, gs.pendingIntro)
	}
}

func TestResetWithoutPendingIntroUnpauses(t *testing.T) {
	gs := newTestState()
	gs.Pause()
	gs.Reset()
	if gs.paused {
		t.Fatal("expected Reset to clear pause without a pending intro")
	}
}
