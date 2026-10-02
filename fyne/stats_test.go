package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/store"
)

func panelSnap(selected []string, scripts []kanacore.Script) StatsSnapshot {
	snap := StatsSnapshot{
		SessionStats:  map[string]store.KanaStats{},
		SelectedRows:  map[string]bool{},
		ActiveScripts: map[kanacore.Script]bool{},
		TotalCorrect:  map[string]int{},
		Paths:         map[kanacore.Script]PathStatus{},
	}
	for _, id := range selected {
		snap.SelectedRows[id] = true
	}
	for _, s := range scripts {
		snap.ActiveScripts[s] = true
		snap.Paths[s] = PathStatus{Steps: make([]StepState, 15)}
	}
	return snap
}

var both = []kanacore.Script{kanacore.ScriptHiragana, kanacore.ScriptKatakana}

func TestPanelRowHalves(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.Update(panelSnap([]string{"k"}, both))
	if !p.cells["か"].Visible() || p.cells["カ"].Visible() {
		t.Fatal("k selected only in hiragana: hiragana half visible, katakana half blank")
	}
	if !p.rowLabels["k"].Visible() || p.rowLabels["t"].Visible() {
		t.Fatal("row k shown, row t (selected nowhere) hidden")
	}
}

func TestPanelCellStatesAndMissed(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"k", "n"}, both)
	snap.TotalCorrect["き"] = 2
	snap.TotalCorrect["く"] = 3
	snap.MissedKanas = []kanacore.Kana{{Char: "ぬ", Romaji: "nu"}}
	p.Update(snap)
	if p.cells["か"].state != cellNew || p.cells["き"].state != cellLearning || p.cells["く"].state != cellMastered {
		t.Fatal("cell states must follow TotalCorrect")
	}
	if !p.cells["ぬ"].missed || p.cells["な"].missed {
		t.Fatal("only ぬ carries the missed border")
	}
}

func TestPanelPathLineFollowsActiveScripts(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.Update(panelSnap([]string{"vowels"}, []kanacore.Script{kanacore.ScriptHiragana}))
	if !p.pathLines[kanacore.ScriptHiragana].Visible() || p.pathLines[kanacore.ScriptKatakana].Visible() {
		t.Fatal("only the hiragana path line is shown")
	}
}

func TestPanelAccuracy(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
	p.Update(snap)
	if p.accuracyText.Text != "–" {
		t.Fatalf("accuracy with no answers = %q", p.accuracyText.Text)
	}
	snap.SessionStats["あ"] = store.KanaStats{Char: "あ", CorrectCount: 34}
	snap.Missed = 4
	p.Update(snap)
	if p.accuracyText.Text != "89 %" {
		t.Fatalf("accuracy 34/4 = %q", p.accuracyText.Text)
	}
	snap.SessionStats["あ"] = store.KanaStats{Char: "あ", CorrectCount: 2}
	snap.Missed = 1
	p.Update(snap)
	if p.accuracyText.Text != "67 %" {
		t.Fatalf("accuracy 2/1 = %q, want rounding", p.accuracyText.Text)
	}
}

func TestPanelHidesEmptyGroupsAndKeepsWidth(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.Update(panelSnap([]string{"vowels", "ky"}, both))
	if p.groupGrids[kanacore.GroupDakuon].Visible() {
		t.Fatal("dakuon grid hidden with no dakuon rows selected")
	}
	if !p.groupGrids[kanacore.GroupYoon].Visible() {
		t.Fatal("yōon grid visible")
	}
	if w := p.groupGrids[kanacore.GroupYoon].MinSize().Width; w != 288 {
		t.Fatalf("yōon grid width = %v, want 288", w)
	}
}

func TestRowShortLabelForSH(t *testing.T) {
	if got := rowShortLabel("sy"); got != "sh" {
		t.Fatalf("rowShortLabel(sy) = %q, want sh", got)
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
