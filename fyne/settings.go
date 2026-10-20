package main

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
)

// settingsState is what the settings dialog starts from, copied under the
// lock.
type settingsState struct {
	selected map[string]bool
	active   map[kanacore.Script]bool
	auto     bool
	limit    int
	level    Level
	points   int

	wordSource  string   // the stored word source
	lists       []string // the private lists found
	wordLevel   int
	packageSize int
	summary     string // the Words tab's summary line
	wordsDir    string // the word list folder for display; "" if unknown
	kanaFont    string // the kana font's id
}

// settingsState copies the current settings for the dialog.
func (gs *GameState) settingsState() settingsState {
	home, _ := os.UserHomeDir()
	gs.mu.Lock()
	defer gs.mu.Unlock()
	var dir string
	if gs.wordsDir != "" {
		dir = displayDir(gs.wordsDir, home)
	}
	return settingsState{
		selected:    maps.Clone(gs.selectedRows),
		active:      maps.Clone(gs.activeScripts),
		auto:        gs.autoProgress,
		limit:       gs.scoreLimit,
		level:       gs.level.Level,
		points:      gs.pointsSetting,
		wordSource:  gs.wordSource,
		lists:       append([]string(nil), gs.listNames...),
		wordLevel:   gs.wordLevel,
		packageSize: gs.packageSize,
		summary:     gs.wordsSummary(),
		wordsDir:    dir,
		kanaFont:    gs.kanaFont.id,
	}
}

// settingsForm is the settings dialog's content: the tabs Kana, Words, Game
// and Sources, whose widgets carry the settings.
type settingsForm struct {
	tabs                                   *container.AppTabs
	kanaTab, wordsTab, gameTab, sourcesTab *container.TabItem

	scriptChecks map[kanacore.Script]*widget.Check
	rowChecks    map[string]*widget.Check
	autoCheck    *widget.Check

	sourceSelect *widget.Select
	sourceValues []string // the word source of each sourceSelect option
	storedSource string   // the stored word source, kept as an option while missing
	wordLevel    *widget.RadioGroup
	packageSize  *widget.RadioGroup
	summary      *widget.Label
	folder       *widget.Label // where the word lists are kept

	difficulty *widget.RadioGroup
	points     *widget.Select
	limitEntry *widget.Entry
	kanaFont   *widget.RadioGroup
	limit      int // the score limit the dialog started with

	onReset  func(kanacore.Script) // a reset button was clicked; may be nil
	onImport func()                // the import button was clicked; may be nil
}

// newSettingsForm builds the tabs from st; onReset is called with the script
// whose reset button is clicked.
func newSettingsForm(st settingsState, onReset func(kanacore.Script)) *settingsForm {
	f := &settingsForm{limit: st.limit, onReset: onReset}
	f.kanaTab = container.NewTabItem("Kana", f.newKanaTab(st))
	f.wordsTab = container.NewTabItem("Words", f.newWordsTab(st))
	f.gameTab = container.NewTabItem("Game", f.newGameTab(st))
	f.sourcesTab = container.NewTabItem("Sources", newSourcesTab())
	f.tabs = container.NewAppTabs(f.kanaTab, f.wordsTab, f.gameTab, f.sourcesTab)
	return f
}

// newKanaTab holds the scripts, one row tab per script and auto-progression.
func (f *settingsForm) newKanaTab(st settingsState) fyne.CanvasObject {
	f.scriptChecks = make(map[kanacore.Script]*widget.Check)
	scriptRow := container.NewHBox(widget.NewLabelWithStyle("Scripts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for _, info := range kanacore.Scripts() {
		c := widget.NewCheck(info.Label, nil)
		c.SetChecked(st.active[info.Script])
		f.scriptChecks[info.Script] = c
		scriptRow.Add(c)
	}

	f.rowChecks = make(map[string]*widget.Check)
	rowTabs := container.NewAppTabs()
	for _, info := range kanacore.Scripts() {
		sections := container.NewVBox()
		for _, g := range kanacore.GroupsFor(info.Script) {
			all, checks, grid := newGroupChecks(kanacore.RowsInGroup(info.Script, g.Group), st.selected)
			for id, c := range checks {
				f.rowChecks[id] = c
			}
			sections.Add(container.NewHBox(
				widget.NewLabelWithStyle(g.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				all,
			))
			sections.Add(grid)
		}
		scroll := container.NewVScroll(sections)
		scroll.SetMinSize(fyne.NewSize(460, 220))
		rowTabs.Append(container.NewTabItem(info.Label, scroll))
	}

	f.autoCheck = widget.NewCheck("Enable auto-progression", nil)
	f.autoCheck.SetChecked(st.auto)

	resets := container.NewHBox()
	for _, info := range kanacore.Scripts() {
		s := info.Script
		resets.Add(widget.NewButton("Reset "+strings.ToLower(info.Label)+" progress…", func() {
			if f.onReset != nil {
				f.onReset(s)
			}
		}))
	}
	return container.NewVBox(scriptRow, rowTabs, widget.NewSeparator(), f.autoCheck, resets)
}

// scriptLabel is the display name of a script ("Hiragana").
func scriptLabel(s kanacore.Script) string {
	for _, info := range kanacore.Scripts() {
		if info.Script == s {
			return info.Label
		}
	}
	return string(s)
}

// sourceOptions are the source select's labels and word sources: the lists
// by name, a missing stored list (it stays selected until the learner picks
// another), then JLPT.
func sourceOptions(lists []string, stored string) (labels, values []string) {
	for _, name := range lists {
		labels = append(labels, prettyListName(name))
		values = append(values, listSourcePrefix+name)
	}
	if name, ok := strings.CutPrefix(stored, listSourcePrefix); ok && !slices.Contains(lists, name) {
		labels = append(labels, prettyListName(name))
		values = append(values, stored)
	}
	return append(labels, "JLPT"), append(values, jlptSource)
}

// setLists replaces the source select's lists and selects the list named
// selected.
func (f *settingsForm) setLists(lists []string, selected string) {
	labels, values := sourceOptions(lists, f.storedSource)
	f.sourceValues = values
	f.sourceSelect.SetOptions(labels)
	if i := slices.Index(values, listSourcePrefix+selected); i >= 0 {
		f.sourceSelect.SetSelectedIndex(i)
	}
}

// newWordsTab holds where new words come from, the import button, the JLPT
// level (for JLPT only), the package size, a summary of the current source
// and the word list folder.
func (f *settingsForm) newWordsTab(st settingsState) fyne.CanvasObject {
	f.storedSource = st.wordSource
	labels, values := sourceOptions(st.lists, st.wordSource)
	f.sourceValues = values

	f.wordLevel = widget.NewRadioGroup([]string{"N5", "N4", "N3", "N2", "N1"}, nil)
	f.wordLevel.Horizontal = true
	f.wordLevel.Required = true
	f.wordLevel.SetSelected(fmt.Sprintf("N%d", st.wordLevel))

	f.sourceSelect = widget.NewSelect(labels, func(string) {
		if f.wordSource() == jlptSource {
			f.wordLevel.Enable()
		} else {
			f.wordLevel.Disable()
		}
	})
	if i := slices.Index(f.sourceValues, st.wordSource); i >= 0 {
		f.sourceSelect.SetSelectedIndex(i)
	} else {
		f.sourceSelect.SetSelectedIndex(len(labels) - 1)
	}

	sizes := make([]string, len(packageSizes))
	for i, n := range packageSizes {
		sizes[i] = strconv.Itoa(n)
	}
	f.packageSize = widget.NewRadioGroup(sizes, nil)
	f.packageSize.Horizontal = true
	f.packageSize.Required = true
	f.packageSize.SetSelected(strconv.Itoa(st.packageSize))

	importButton := widget.NewButton("Import word list…", func() {
		if f.onImport != nil {
			f.onImport()
		}
	})
	f.folder = widget.NewLabel("")
	if st.wordsDir != "" {
		f.folder.SetText("Word lists are kept in " + st.wordsDir)
	}

	f.summary = widget.NewLabel(st.summary)
	return container.NewVBox(
		widget.NewLabelWithStyle("New words from", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.sourceSelect,
		importButton,
		widget.NewLabel("JLPT level"),
		f.wordLevel,
		widget.NewLabel("Words per package"),
		f.packageSize,
		widget.NewSeparator(),
		f.summary,
		f.folder,
	)
}

// newGameTab holds the difficulty, points per hit, the score limit and the
// kana font, each font previewed in itself.
func (f *settingsForm) newGameTab(st settingsState) fyne.CanvasObject {
	f.limitEntry = widget.NewEntry()
	f.limitEntry.SetText(strconv.Itoa(st.limit))
	f.limitEntry.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n < 0 {
			return fmt.Errorf("enter a whole number ≥ 0")
		}
		return nil
	}

	levelLabels := make([]string, 0, len(Levels()))
	for _, l := range Levels() {
		levelLabels = append(levelLabels, l.Label)
	}
	f.difficulty = widget.NewRadioGroup(levelLabels, nil)
	f.difficulty.Horizontal = true
	f.difficulty.Required = true
	f.difficulty.SetSelected(LevelByName(string(st.level)).Label)

	pointsLabels := make([]string, 0, len(pointsChoices))
	for _, p := range pointsChoices {
		pointsLabels = append(pointsLabels, pointsLabel(p))
	}
	f.points = widget.NewSelect(pointsLabels, nil)
	f.points.SetSelected(pointsLabel(st.points))

	fontLabels := make([]string, len(fontChoices))
	previews := container.NewGridWithColumns(2)
	for i, c := range fontChoices {
		fontLabels[i] = c.label
		preview := canvas.NewText(fontPreview, kanaTextColor)
		preview.TextSize = 32 // the tile size
		setKanaFont(preview, c)
		previews.Add(widget.NewLabel(c.label))
		previews.Add(preview)
	}
	f.kanaFont = widget.NewRadioGroup(fontLabels, nil)
	f.kanaFont.Horizontal = true
	f.kanaFont.Required = true
	f.kanaFont.SetSelected(fontByID(st.kanaFont).label)

	return container.NewVBox(
		widget.NewLabelWithStyle("Difficulty", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.difficulty,
		container.NewHBox(widget.NewLabel("Points per hit"), f.points),
		widget.NewSeparator(),
		widget.NewLabel("Score limit (0 = endless)"),
		f.limitEntry,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Kana font", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.kanaFont,
		previews,
	)
}

// kanaFontID is the id of the kana font the Game tab has selected.
func (f *settingsForm) kanaFontID() string {
	for _, c := range fontChoices {
		if c.label == f.kanaFont.Selected {
			return c.id
		}
	}
	return fontChoices[0].id
}

// wordSource is the word source the Words tab has selected.
func (f *settingsForm) wordSource() string {
	if i := f.sourceSelect.SelectedIndex(); i >= 0 {
		return f.sourceValues[i]
	}
	return jlptSource
}

// words returns the Words tab's values.
func (f *settingsForm) words() wordSettings {
	level, _ := strconv.Atoi(strings.TrimPrefix(f.wordLevel.Selected, "N"))
	size, _ := strconv.Atoi(f.packageSize.Selected)
	return wordSettings{source: f.wordSource(), level: level, packageSize: size}
}

// validate checks every tab; on an error it switches to the tab that has it.
func (f *settingsForm) validate() error {
	if err := f.limitEntry.Validate(); err != nil {
		f.tabs.Select(f.gameTab)
		return err
	}
	return nil
}

// apply saves every tab: the Words tab first, then the rest, which ends by
// checking whether a word package is due.
func (f *settingsForm) apply(gs *GameState) {
	limit := f.limit
	if n, err := strconv.Atoi(strings.TrimSpace(f.limitEntry.Text)); err == nil && n >= 0 {
		limit = n
	}
	gs.applyWordSettings(f.words())
	gs.SetKanaFont(f.kanaFontID())
	gs.applySettings(checkedRowIDs(f.rowChecks), checkedScripts(f.scriptChecks), f.autoCheck.Checked, limit,
		levelFromLabel(f.difficulty.Selected), pointsFromLabel(f.points.Selected))
}

// showSettingsDialog opens the settings, pausing the game while they are
// open. A script's reset asks for confirmation, then closes the settings and
// restarts the round.
func showSettingsDialog(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, win fyne.Window) {
	var d *dialog.CustomDialog
	onReset := func(s kanacore.Script) {
		name := strings.ToLower(scriptLabel(s))
		c := dialog.NewConfirm("Reset "+name+" progress",
			"Delete the statistics and learning state of all "+name+"? This cannot be undone.",
			func(ok bool) {
				if !ok {
					return
				}
				if err := gs.ResetScript(s); err != nil {
					dialog.ShowError(err, win)
					return
				}
				// closed() is skipped: Reset clears settingsOpen and
				// restartRound shows pending dialogs; unsaved changes in
				// the other tabs are dropped, as with Cancel.
				d.Hide()
				restartRound(gs, statsPanel, gameCanvas, inputBar, win)
			}, win)
		c.SetConfirmText("Reset")
		c.SetDismissText("Cancel")
		c.Show()
	}
	form := newSettingsForm(gs.settingsState(), onReset)
	form.onImport = func() { showImportDialog(gs, form, win) }
	gs.SetSettingsOpen(true)
	gameCanvas.Refresh()

	d = dialog.NewCustomWithoutButtons("Settings", form.tabs, win)
	// closed releases the settings pause on every path; showPendingDialogs
	// then resumes the game or shows whatever is pending.
	closed := func(save bool) {
		d.Hide()
		gs.SetSettingsOpen(false)
		if save {
			form.apply(gs)
			statsPanel.SetKanaFont(gs.KanaFont())
			gs.mu.Lock()
			snap := gs.snapshot()
			gs.mu.Unlock()
			statsPanel.Update(snap)
			inputBar.Update(snap)
		}
		gameCanvas.Refresh()
		showPendingDialogs(gs, statsPanel, inputBar, win)
	}
	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { closed(false) })
	save := widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
		// An invalid value keeps the dialog open on its tab.
		if err := form.validate(); err != nil {
			dialog.ShowError(err, win)
			return
		}
		closed(true)
	})
	save.Importance = widget.HighImportance
	d.SetButtons([]fyne.CanvasObject{cancel, save})
	d.Show()
}

// newGroupChecks builds the per-row checks and the "all" check for one kana
// group, keeping them in sync: checking/unchecking "all" sets every row, and
// checking/unchecking a row updates "all" to match. A per-group syncing flag
// guards against the two handlers recursing into each other.
func newGroupChecks(rows []kanacore.KanaRow, selected map[string]bool) (all *widget.Check, checks map[string]*widget.Check, grid *fyne.Container) {
	checks = make(map[string]*widget.Check, len(rows))
	checkList := make([]*widget.Check, 0, len(rows))
	grid = container.NewGridWithColumns(2)

	for _, row := range rows {
		c := widget.NewCheck(row.Label, nil)
		c.SetChecked(selected[row.ID])
		checks[row.ID] = c
		checkList = append(checkList, c)
		grid.Add(c)
	}

	all = widget.NewCheck("all", nil)
	all.SetChecked(allChecked(checkList))

	syncing := false

	all.OnChanged = func(on bool) {
		if syncing {
			return
		}
		syncing = true
		for _, c := range checkList {
			c.SetChecked(on)
		}
		syncing = false
	}

	for _, c := range checkList {
		c.OnChanged = func(bool) {
			if syncing {
				return
			}
			syncing = true
			all.SetChecked(allChecked(checkList))
			syncing = false
		}
	}

	return all, checks, grid
}

// checkedRowIDs returns the IDs of checked rows in progression order.
func checkedRowIDs(checks map[string]*widget.Check) []string {
	ids := make([]string, 0, len(checks))
	for _, row := range kanacore.AllKanaRows {
		if c, ok := checks[row.ID]; ok && c.Checked {
			ids = append(ids, row.ID)
		}
	}
	return ids
}

// checkedScripts returns the checked scripts in display order.
func checkedScripts(checks map[kanacore.Script]*widget.Check) []kanacore.Script {
	var scripts []kanacore.Script
	for _, info := range kanacore.Scripts() {
		if c, ok := checks[info.Script]; ok && c.Checked {
			scripts = append(scripts, info.Script)
		}
	}
	return scripts
}

func allChecked(checks []*widget.Check) bool {
	for _, c := range checks {
		if !c.Checked {
			return false
		}
	}
	return len(checks) > 0
}

// levelFromLabel maps a difficulty label back to its level; unknown → Normal.
func levelFromLabel(label string) Level {
	for _, l := range Levels() {
		if l.Label == label {
			return l.Level
		}
	}
	return LevelNormal
}

// pointsFromLabel maps a "Points per hit" label back to its setting.
func pointsFromLabel(label string) int {
	for _, p := range pointsChoices {
		if pointsLabel(p) == label {
			return p
		}
	}
	return pointsAuto
}
