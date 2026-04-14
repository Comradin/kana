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
	if panel.unlockLabel.Text != "" {
		t.Errorf("expected unlock label cleared, got %q", panel.unlockLabel.Text)
	}
}
