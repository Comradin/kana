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

func TestInputBarKeepsItsWidthAsTheScoreGrows(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	ib := newInputBar(gs, newStatsPanel(), newGameCanvas(gs), test.NewWindow(nil))
	widths := func() (float32, float32) {
		return ib.scoreBox.MinSize().Width, ib.missedBox.MinSize().Width
	}
	ib.Update(gs.snapshot())
	score0, missed0 := widths()
	gs.score, gs.missed = 12340, 4
	ib.Update(gs.snapshot())
	if score, missed := widths(); score != score0 || missed != missed0 {
		t.Fatalf("widths %v/%v → %v/%v, want the status to keep its width", score0, missed0, score, missed)
	}
}
