package main

import (
	"testing"

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
