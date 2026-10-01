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

func TestStatsPanelShowsGroupLabelForVisibleRows(t *testing.T) {
	test.NewApp()
	panel := newStatsPanel()
	panel.Update(StatsSnapshot{
		SessionStats: map[string]store.KanaStats{},
		SelectedRows: map[string]bool{"vowels": true, "g": true},
	})
	if !panel.groupCells[kanacore.GroupDakuon][0].Visible() {
		t.Error("expected Dakuon group label visible")
	}
	if panel.groupCells[kanacore.GroupYoon][0].Visible() {
		t.Error("expected Yōon group label hidden")
	}
	if _, ok := panel.groupCells[kanacore.GroupBasic]; ok {
		t.Error("basic group should have no label row")
	}
}

func TestRowShortLabelForSH(t *testing.T) {
	if got := rowShortLabel("sy"); got != "sh" {
		t.Fatalf("rowShortLabel(sy) = %q, want sh", got)
	}
}
