package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/vocab"
)

func testGroupRows() []kanacore.KanaRow {
	return []kanacore.KanaRow{
		{ID: "r1", Label: "Row 1"},
		{ID: "r2", Label: "Row 2"},
		{ID: "r3", Label: "Row 3"},
	}
}

func TestAllChecked(t *testing.T) {
	tests := []struct {
		name    string
		checked []bool
		want    bool
	}{
		{"empty", nil, false},
		{"all true", []bool{true, true, true}, true},
		{"mixed", []bool{true, false, true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.NewApp()
			checks := make([]*widget.Check, len(tt.checked))
			for i, on := range tt.checked {
				checks[i] = widget.NewCheck("", nil)
				checks[i].SetChecked(on)
			}
			if got := allChecked(checks); got != tt.want {
				t.Errorf("allChecked(%v) = %v, want %v", tt.checked, got, tt.want)
			}
		})
	}
}

func TestNewGroupChecksInitialStateAllSelected(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": true, "r2": true, "r3": true}

	all, checks, _ := newGroupChecks(rows, selected)

	if !all.Checked {
		t.Fatalf("expected 'all' checked when every row is selected")
	}
	for _, row := range rows {
		if !checks[row.ID].Checked {
			t.Errorf("expected row %s to stay checked from initial selection", row.ID)
		}
	}
}

func TestNewGroupChecksInitialStateNotAllSelected(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": true, "r2": false, "r3": true}

	all, checks, _ := newGroupChecks(rows, selected)

	if all.Checked {
		t.Fatalf("expected 'all' unchecked when not every row is selected")
	}
	if !checks["r1"].Checked || checks["r2"].Checked || !checks["r3"].Checked {
		t.Fatalf("expected initial row state preserved exactly, got r1=%v r2=%v r3=%v",
			checks["r1"].Checked, checks["r2"].Checked, checks["r3"].Checked)
	}
}

func TestNewGroupChecksToggleAllChecksEveryRow(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": false, "r2": false, "r3": false}

	all, checks, _ := newGroupChecks(rows, selected)

	all.SetChecked(true)
	for _, row := range rows {
		if !checks[row.ID].Checked {
			t.Errorf("expected row %s checked after toggling 'all' on", row.ID)
		}
	}

	all.SetChecked(false)
	for _, row := range rows {
		if checks[row.ID].Checked {
			t.Errorf("expected row %s unchecked after toggling 'all' off", row.ID)
		}
	}
}

func TestNewGroupChecksUncheckingOneRowUnchecksAll(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": true, "r2": true, "r3": true}

	all, checks, _ := newGroupChecks(rows, selected)
	if !all.Checked {
		t.Fatalf("precondition failed: expected 'all' checked initially")
	}

	checks["r2"].SetChecked(false)

	if all.Checked {
		t.Errorf("expected 'all' unchecked after unchecking row r2")
	}
}

func TestNewGroupChecksCheckingLastRowChecksAll(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": true, "r2": false, "r3": true}

	all, checks, _ := newGroupChecks(rows, selected)
	if all.Checked {
		t.Fatalf("precondition failed: expected 'all' unchecked initially")
	}

	checks["r2"].SetChecked(true)

	if !all.Checked {
		t.Errorf("expected 'all' checked after checking the last unchecked row")
	}
}

func TestCheckedRowIDsFollowsRowOrder(t *testing.T) {
	test.NewApp()
	checks := map[string]*widget.Check{
		"ky":     widget.NewCheck("", nil),
		"vowels": widget.NewCheck("", nil),
		"g":      widget.NewCheck("", nil),
		"k":      widget.NewCheck("", nil),
	}
	checks["ky"].SetChecked(true)
	checks["vowels"].SetChecked(true)
	checks["g"].SetChecked(true)

	got := checkedRowIDs(checks)
	if !equalIDs(got, []string{"vowels", "g", "ky"}) {
		t.Fatalf("checkedRowIDs = %v, want [vowels g ky]", got)
	}
}

func TestCheckedScriptsFollowDisplayOrder(t *testing.T) {
	test.NewApp()
	checks := map[kanacore.Script]*widget.Check{
		kanacore.ScriptKatakana: widget.NewCheck("", nil),
		kanacore.ScriptHiragana: widget.NewCheck("", nil),
	}
	checks[kanacore.ScriptKatakana].SetChecked(true)
	checks[kanacore.ScriptHiragana].SetChecked(true)
	got := checkedScripts(checks)
	if len(got) != 2 || got[0] != kanacore.ScriptHiragana || got[1] != kanacore.ScriptKatakana {
		t.Fatalf("checkedScripts = %v", got)
	}
}

// settingsGame is a game with a private list as the word source and every
// setting away from its default.
func settingsGame() *GameState {
	gs := newTestState()
	gs.words = vocab.NewIndex(
		[]vocab.Word{bw("あい", 5, "love"), bw("あお", 5, "blue"), bw("いえ", 4, "house")},
		[]vocab.List{{Name: "minna-no-nihongo", Words: []vocab.Word{
			{Kana: "あい", Meanings: vocab.Meanings{}}, {Kana: "うえ", Meanings: vocab.Meanings{}},
		}}},
	)
	gs.listNames = []string{"minna-no-nihongo"}
	gs.wordSource = listSourcePrefix + "minna-no-nihongo"
	gs.introduced = map[string]bool{"あい": true}
	setWordLevel(gs, "あい", learnedLevel)
	gs.wordLevel = 4
	gs.packageSize = 10
	gs.scoreLimit = 500
	gs.level = LevelByName("fast")
	gs.pointsSetting = 2
	gs.autoProgress = true
	return gs
}

func tabTexts(f *settingsForm) []string {
	var out []string
	for _, item := range f.tabs.Items {
		out = append(out, item.Text)
	}
	return out
}

func TestSettingsFormCarriesValues(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(settingsGame().settingsState())
	if got := tabTexts(f); !equalIDs(got, []string{"Kana", "Words", "Game", "Sources"}) {
		t.Fatalf("tabs = %v", got)
	}
	if !equalIDs(f.sourceSelect.Options, []string{"minna no nihongo", "JLPT"}) || f.sourceSelect.Selected != "minna no nihongo" {
		t.Fatalf("source options %v, selected %q", f.sourceSelect.Options, f.sourceSelect.Selected)
	}
	if f.wordLevel.Selected != "N4" || !f.wordLevel.Disabled() {
		t.Fatalf("level %q, disabled %v; want N4, disabled for a list", f.wordLevel.Selected, f.wordLevel.Disabled())
	}
	if f.packageSize.Selected != "10" {
		t.Fatalf("package size = %q", f.packageSize.Selected)
	}
	if f.summary.Text != "Introduced: 1 · learned: 1 · left in source: 1" {
		t.Fatalf("summary = %q", f.summary.Text)
	}
	if f.limitEntry.Text != "500" || f.difficulty.Selected != "Fast" || f.points.Selected != pointsLabel(2) || !f.autoCheck.Checked {
		t.Fatalf("game tab: limit %q, difficulty %q, points %q, auto %v", f.limitEntry.Text, f.difficulty.Selected, f.points.Selected, f.autoCheck.Checked)
	}
	f.sourceSelect.SetSelected("JLPT")
	if f.wordLevel.Disabled() {
		t.Fatal("the level must be enabled for JLPT")
	}
	if got := f.words(); got != (wordSettings{source: jlptSource, level: 4, packageSize: 10}) {
		t.Fatalf("words() = %+v", got)
	}
}

func TestSettingsFormMissingList(t *testing.T) {
	test.NewApp()
	gs := settingsGame()
	gs.listNames = nil
	f := newSettingsForm(gs.settingsState())
	if !equalIDs(f.sourceSelect.Options, []string{"minna no nihongo", "JLPT"}) || f.wordSource() != "list:minna-no-nihongo" {
		t.Fatalf("a missing list stays selected: options %v, source %q", f.sourceSelect.Options, f.wordSource())
	}
	if f.summary.Text != "List minna no nihongo not found" {
		t.Fatalf("summary = %q", f.summary.Text)
	}
}

func TestSettingsValidationSwitchesToTheTab(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(settingsGame().settingsState())
	if err := f.validate(); err != nil {
		t.Fatalf("valid form: %v", err)
	}
	f.limitEntry.SetText("lots")
	f.tabs.Select(f.kanaTab)
	if err := f.validate(); err == nil {
		t.Fatal("an invalid score limit must fail")
	}
	if f.tabs.Selected() != f.gameTab {
		t.Fatalf("selected tab = %q, want Game", f.tabs.Selected().Text)
	}
}

func TestSettingsApplySavesEveryTab(t *testing.T) {
	test.NewApp()
	gs := settingsGame()
	gs.newestPackage = []string{"あい"}
	f := newSettingsForm(gs.settingsState())
	f.sourceSelect.SetSelected("JLPT")
	f.wordLevel.SetSelected("N3")
	f.packageSize.SetSelected("3")
	f.limitEntry.SetText("200")
	f.difficulty.SetSelected("Relaxed")
	f.apply(gs)
	if gs.wordSource != jlptSource || gs.wordLevel != 3 || gs.packageSize != 3 || gs.newestPackage != nil {
		t.Fatalf("words: source %q, level %d, size %d, newest %v", gs.wordSource, gs.wordLevel, gs.packageSize, gs.newestPackage)
	}
	if gs.scoreLimit != 200 || gs.level.Level != LevelRelaxed {
		t.Fatalf("game: limit %d, level %v", gs.scoreLimit, gs.level.Level)
	}
}

func TestSourcesTabCreditsEverySource(t *testing.T) {
	test.NewApp()
	var texts []string
	for _, o := range test.LaidOutObjects(newSourcesTab()) {
		switch w := o.(type) {
		case *widget.Label:
			texts = append(texts, w.Text)
		case *widget.Hyperlink:
			texts = append(texts, w.Text+" <"+w.URL.String()+">")
		}
	}
	all := strings.Join(texts, "\n")
	for _, want := range []string{
		"Version dev ·", "Releases <" + releasesURL + ">",
		"JMdict", "EDRDG", "CC BY-SA 4.0", "Licence <https://www.edrdg.org/edrdg/licence.html>",
		"JMdict <https://www.edrdg.org/jmdict/j_jmdict.html>",
		"JLPT levels", "Jonathan Waller, CC BY", "Unofficial",
		"JMdict mapping", "stephenmk/yomitan-jlpt-vocab",
		"Noto Sans JP", "M PLUS Rounded 1c", "SIL Open Font License 1.1",
		"© 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'",
		"OFL <" + oflURL + ">",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("Sources tab lacks %q:\n%s", want, all)
		}
	}
}

func TestSettingsDialogSaveAndInvalidValue(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	gs := settingsGame()
	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, statsPanel, gameCanvas, w)

	showSettingsDialog(gs, statsPanel, gameCanvas, ib, w)
	settings := w.Canvas().Overlays().Top()
	if settings == nil || !gs.settingsOpen {
		t.Fatal("settings dialog not open")
	}
	var tabs *container.AppTabs
	for _, o := range test.LaidOutObjects(settings) {
		if at, ok := o.(*container.AppTabs); ok && len(at.Items) == 4 {
			tabs = at
		}
	}
	if tabs == nil {
		t.Fatal("no settings tabs")
	}
	tabs.SelectIndex(2)
	var limit *widget.Entry
	for _, o := range test.LaidOutObjects(settings) {
		if e, ok := o.(*widget.Entry); ok {
			limit = e
		}
	}
	limit.SetText("-1")
	tabs.SelectIndex(0)
	test.Tap(buttonIn(settings, "Save"))
	if !gs.settingsOpen || tabs.SelectedIndex() != 2 || len(w.Canvas().Overlays().List()) != 2 {
		t.Fatalf("invalid value: open %v, tab %d, overlays %d; want the dialog open on Game with an error",
			gs.settingsOpen, tabs.SelectedIndex(), len(w.Canvas().Overlays().List()))
	}
	w.Canvas().Overlays().Remove(w.Canvas().Overlays().Top()) // dismiss the error
	limit.SetText("300")
	test.Tap(buttonIn(settings, "Save"))
	if gs.settingsOpen || gs.scoreLimit != 300 || w.Canvas().Overlays().Top() != nil {
		t.Fatalf("after Save: open %v, limit %d", gs.settingsOpen, gs.scoreLimit)
	}
}
