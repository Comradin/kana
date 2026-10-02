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
	if !panel.groupSections[sectionKey{kanacore.ScriptHiragana, kanacore.GroupDakuon}].Visible() {
		t.Error("expected Dakuon group section visible")
	}
	if panel.groupSections[sectionKey{kanacore.ScriptHiragana, kanacore.GroupYoonDakuon}].Visible() {
		t.Error("expected Yōon + Dakuten group section hidden")
	}
	if _, ok := panel.groupSections[sectionKey{kanacore.ScriptHiragana, kanacore.GroupBasic}]; ok {
		t.Error("basic group should have no section")
	}
}

// TestStatsPanelBasicGridWidthUnaffectedByGroupSelection guards against the
// regression where a single shared grid stretched every column to the
// widest cell across all groups, including long group labels like
// "Yōon + Dakuten". Each group now has its own grid, so selecting every
// non-basic group (and thus showing all of their labels, including the
// longest one) must not widen the basic grid.
//
// The baseline selects every basic row (not just "vowels") so both panels
// show the exact same set of basic-grid cells; only the non-basic groups'
// visibility differs between the two. Comparing against "vowels" alone
// would be unreliable even on the fixed code, since different basic row
// labels (e.g. "m" vs "w" vs "–") have slightly different glyph widths in
// the real font — noise that has nothing to do with the regression being
// guarded against here.
func TestStatsPanelBasicGridWidthUnaffectedByGroupSelection(t *testing.T) {
	test.NewApp()

	basicOnly := make(map[string]bool)
	for _, row := range kanacore.BasicRows() {
		basicOnly[row.ID] = true
	}

	narrowPanel := newStatsPanel()
	narrowPanel.Update(StatsSnapshot{
		SessionStats: map[string]store.KanaStats{},
		SelectedRows: basicOnly,
	})
	narrowWidth := narrowPanel.basicGrids[kanacore.ScriptHiragana].MinSize().Width

	widePanel := newStatsPanel()
	allSelected := make(map[string]bool)
	for _, row := range kanacore.AllKanaRows {
		allSelected[row.ID] = true
	}
	widePanel.Update(StatsSnapshot{
		SessionStats: map[string]store.KanaStats{},
		SelectedRows: allSelected,
	})
	wideWidth := widePanel.basicGrids[kanacore.ScriptHiragana].MinSize().Width

	if narrowWidth != wideWidth {
		t.Errorf("expected basic grid width unaffected by non-basic groups' selection, got %v (basic rows only) vs %v (all rows)", narrowWidth, wideWidth)
	}
}

func TestRowShortLabelForSH(t *testing.T) {
	if got := rowShortLabel("sy"); got != "sh" {
		t.Fatalf("rowShortLabel(sy) = %q, want sh", got)
	}
}

func TestStatsPanelShowsKatakanaOnlyWhenActive(t *testing.T) {
	test.NewApp()
	panel := newStatsPanel()
	snap := StatsSnapshot{
		SessionStats:  map[string]store.KanaStats{},
		SelectedRows:  map[string]bool{"vowels": true, "kata:vowels": true},
		ActiveScripts: map[kanacore.Script]bool{kanacore.ScriptHiragana: true},
	}
	panel.Update(snap)
	if panel.scriptSections[kanacore.ScriptKatakana].Visible() {
		t.Error("katakana section should be hidden while katakana is inactive")
	}
	if panel.rowLabels["kata:vowels"].Visible() {
		t.Error("ACTIVE ROWS should hide rows of inactive scripts")
	}
	snap.ActiveScripts[kanacore.ScriptKatakana] = true
	panel.Update(snap)
	if !panel.scriptSections[kanacore.ScriptKatakana].Visible() || !panel.rowLabels["kata:vowels"].Visible() {
		t.Error("katakana section and row should show once katakana is active")
	}
}

func TestRowShortLabelStripsKatakanaPrefix(t *testing.T) {
	if got := rowShortLabel("kata:sy"); got != "sh" {
		t.Fatalf("rowShortLabel(kata:sy) = %q, want sh", got)
	}
	if got := rowShortLabel("kata:vowels"); got != "–" {
		t.Fatalf("rowShortLabel(kata:vowels) = %q, want –", got)
	}
}
