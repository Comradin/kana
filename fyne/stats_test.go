package main

import (
	"testing"

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
