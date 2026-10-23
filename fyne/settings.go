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

	wordsOnly     bool
	wordsAsk      wordKind
	lessons       map[string][]string // per list: its lessons in learning order, "" last when it has words without a lesson
	lessonFilters map[string][]string // per list: the stored lesson filter; absent = all lessons
	meaningLang   string              // the stored meaning language; "" = System
	languages     []string            // the meaning languages of the words, by code
	focusWeak     bool
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
	lessons := make(map[string][]string, len(gs.listNames))
	var langs []string
	if gs.words != nil {
		for _, name := range gs.listNames {
			ls, none := gs.words.Lessons(name)
			if none {
				ls = append(ls, "")
			}
			lessons[name] = ls
		}
		langs = gs.words.Languages()
	}
	filters := make(map[string][]string, len(gs.lessonFilters))
	for name, ls := range gs.lessonFilters {
		filters[name] = slices.Clone(ls)
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

		wordsOnly:     gs.wordsOnly,
		wordsAsk:      gs.wordsAsk,
		lessons:       lessons,
		lessonFilters: filters,
		meaningLang:   gs.meaningSetting,
		languages:     langs,
		focusWeak:     gs.focusWeak,
	}
}

// The Words tab's "Tiles" choices.
const (
	tilesMixed     = "Kana and words"
	tilesWordsOnly = "Words only"
)

// The Words tab's "Words ask for" choices.
const (
	askReadingLabel = "Reading"
	askMeaningLabel = "Meaning"
)

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

	tiles         *widget.RadioGroup  // tilesMixed or tilesWordsOnly
	wordsAsk      *widget.RadioGroup  // askReadingLabel or askMeaningLabel
	listLessons   map[string][]string // per list: its lessons, "" last for words without one
	lessonFilters map[string][]string // per list: its lesson filter as edited in this dialog
	lessonBox     *fyne.Container     // "All lessons" and the selected list's lesson checks
	lessonScroll  *container.Scroll
	lessonList    string   // the list lessonBox shows; "" = none
	lessonOrder   []string // its lessons, in order
	lessonChecks  map[string]*widget.Check
	allLessons    *widget.Check
	meaningSelect *widget.Select
	meaningValues []string // the meaning language of each meaningSelect option; "" = System
	focusCheck    *widget.Check

	difficulty *widget.RadioGroup
	points     *widget.Select
	limitEntry *widget.Entry
	kanaFont   *widget.RadioGroup
	limit      int // the score limit the dialog started with

	onReset        func(kanacore.Script) // a reset button was clicked; may be nil
	onImport       func()                // the import button was clicked; may be nil
	onManage       func()                // the word lists button was clicked; may be nil
	onListsChanged func()                // the lists shown changed; set while the word list dialog is open
}

// openSettings is the form of the settings dialog while it is open, so a
// dropped file's import can show its list there; nil otherwise.
// Fyne-thread-only, like dialogShowing.
var openSettings *settingsForm

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

// setLists replaces the source select's lists and selects the word source
// selected; JLPT when it is not among them. Must run on the Fyne thread.
func (f *settingsForm) setLists(lists []string, selected string) {
	labels, values := sourceOptions(lists, f.storedSource)
	f.sourceValues = values
	f.sourceSelect.SetOptions(labels)
	i := slices.Index(values, selected)
	if i < 0 {
		i = len(values) - 1 // JLPT
	}
	f.sourceSelect.SetSelectedIndex(i)
	f.showLessons()
}

// listsChanged shows the word lists of st after an import, delete or rename:
// the source select with selected (a word source) selected, the lessons, the
// meaning languages and the summary. An open word list dialog refreshes
// too. Must run on the Fyne thread.
func (f *settingsForm) listsChanged(st settingsState, selected string) {
	f.storedSource = st.wordSource
	f.listLessons = st.lessons
	f.setLists(st.lists, selected)
	f.setMeaningLangs(st.languages)
	f.summary.SetText(st.summary)
	if f.onListsChanged != nil {
		f.onListsChanged()
	}
}

// listImported shows an imported list, selected. Its lesson checks start
// from its stored filter: checks shown so far belonged to the list it
// replaced. Must run on the Fyne thread.
func (f *settingsForm) listImported(st settingsState, name string) {
	if f.lessonList == name {
		f.lessonList = ""
	}
	if ls, ok := st.lessonFilters[name]; ok {
		f.lessonFilters[name] = ls
	} else {
		delete(f.lessonFilters, name)
	}
	f.listsChanged(st, listSourcePrefix+name)
}

// listDeleted drops a deleted list: its lesson checks go, and JLPT is
// selected if it was selected. Must run on the Fyne thread.
func (f *settingsForm) listDeleted(st settingsState, name string) {
	if f.lessonList == name {
		f.lessonList = ""
	}
	delete(f.lessonFilters, name)
	selected := f.wordSource()
	if selected == listSourcePrefix+name {
		selected = jlptSource
	}
	f.listsChanged(st, selected)
}

// listRenamed follows a renamed list: its lesson checks as edited in this
// dialog and its selection move to the new name. Must run on the Fyne
// thread.
func (f *settingsForm) listRenamed(st settingsState, from, to string) {
	f.keepLessons()
	if f.lessonList == from {
		f.lessonList = ""
	}
	if ls, ok := f.lessonFilters[from]; ok {
		delete(f.lessonFilters, from)
		f.lessonFilters[to] = ls
	}
	selected := f.wordSource()
	if selected == listSourcePrefix+from {
		selected = listSourcePrefix + to
	}
	f.listsChanged(st, selected)
}

// newWordsTab holds the tiles (kana and words, or words only), what word
// tiles ask for (reading or meaning), where new words come from, the import
// button, the lessons of a list, the JLPT level (for JLPT only), the package
// size, the meaning language, a summary of the current source and the word
// list folder.
func (f *settingsForm) newWordsTab(st settingsState) fyne.CanvasObject {
	f.storedSource = st.wordSource
	labels, values := sourceOptions(st.lists, st.wordSource)
	f.sourceValues = values

	f.wordLevel = widget.NewRadioGroup([]string{"N5", "N4", "N3", "N2", "N1"}, nil)
	f.wordLevel.Horizontal = true
	f.wordLevel.Required = true
	f.wordLevel.SetSelected(fmt.Sprintf("N%d", st.wordLevel))

	f.tiles = widget.NewRadioGroup([]string{tilesMixed, tilesWordsOnly}, nil)
	f.tiles.Horizontal = true
	f.tiles.Required = true
	f.tiles.SetSelected(tilesMixed)
	if st.wordsOnly {
		f.tiles.SetSelected(tilesWordsOnly)
	}

	f.wordsAsk = widget.NewRadioGroup([]string{askReadingLabel, askMeaningLabel}, nil)
	f.wordsAsk.Horizontal = true
	f.wordsAsk.Required = true
	f.wordsAsk.SetSelected(askReadingLabel)
	if st.wordsAsk == meaningKind {
		f.wordsAsk.SetSelected(askMeaningLabel)
	}

	// The lesson box comes before the source select, whose OnChanged
	// fills it.
	f.listLessons = st.lessons
	f.lessonFilters = maps.Clone(st.lessonFilters)
	if f.lessonFilters == nil {
		f.lessonFilters = map[string][]string{}
	}
	f.lessonBox = container.NewVBox()
	f.lessonScroll = container.NewVScroll(f.lessonBox)
	f.lessonScroll.SetMinSize(fyne.NewSize(460, 96))

	f.sourceSelect = widget.NewSelect(labels, func(string) {
		if f.wordSource() == jlptSource {
			f.wordLevel.Enable()
		} else {
			f.wordLevel.Disable()
		}
		f.showLessons()
	})
	if i := slices.Index(f.sourceValues, st.wordSource); i >= 0 {
		f.sourceSelect.SetSelectedIndex(i)
	} else {
		f.sourceSelect.SetSelectedIndex(len(labels) - 1)
	}
	f.showLessons()

	mLabels, mValues := meaningLangOptions(st.languages, st.meaningLang)
	f.meaningValues = mValues
	f.meaningSelect = widget.NewSelect(mLabels, nil)
	f.meaningSelect.SetSelectedIndex(slices.Index(mValues, st.meaningLang))

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
	manageButton := widget.NewButton("Manage word lists…", func() {
		if f.onManage != nil {
			f.onManage()
		}
	})
	f.folder = widget.NewLabel("")
	if st.wordsDir != "" {
		f.folder.SetText("Word lists are kept in " + st.wordsDir)
	}

	f.summary = widget.NewLabel(st.summary)
	return container.NewVBox(
		widget.NewLabelWithStyle("Tiles", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.tiles,
		widget.NewLabelWithStyle("Words ask for", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.wordsAsk,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("New words from", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.sourceSelect,
		container.NewHBox(importButton, manageButton),
		f.lessonScroll,
		widget.NewLabel("JLPT level"),
		f.wordLevel,
		widget.NewLabel("Words per package"),
		f.packageSize,
		container.NewHBox(widget.NewLabel("Meanings in"), f.meaningSelect),
		widget.NewSeparator(),
		f.summary,
		f.folder,
	)
}

// newGameTab holds the difficulty, points per hit, the focus on weak items,
// the score limit and the kana font, each font previewed in itself.
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

	f.focusCheck = widget.NewCheck("Focus on weak items", nil)
	f.focusCheck.SetChecked(st.focusWeak)

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
		f.focusCheck,
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

// askedKind is what the Words tab says word tiles ask for.
func (f *settingsForm) askedKind() wordKind {
	if f.wordsAsk.Selected == askMeaningLabel {
		return meaningKind
	}
	return readingKind
}

// words returns the Words tab's values.
func (f *settingsForm) words() wordSettings {
	level, _ := strconv.Atoi(strings.TrimPrefix(f.wordLevel.Selected, "N"))
	size, _ := strconv.Atoi(f.packageSize.Selected)
	return wordSettings{source: f.wordSource(), level: level, packageSize: size,
		wordsOnly: f.tiles.Selected == tilesWordsOnly, lessons: f.selectedLessons(), meaningLang: f.meaningLang(),
		ask: f.askedKind()}
}

// showLessons fills the lesson box for the selected source: "All lessons"
// and a check per lesson of the list ("No lesson" for words without one),
// checked as the list's filter in this dialog says (all when it has none).
// JLPT and a list without lessons hide the box. The checks of the list
// shown so far are kept first. Must run on the Fyne thread.
func (f *settingsForm) showLessons() {
	f.keepLessons()
	f.lessonList, f.lessonOrder, f.lessonChecks, f.allLessons = "", nil, nil, nil
	f.lessonBox.RemoveAll()
	name, isList := strings.CutPrefix(f.wordSource(), listSourcePrefix)
	lessons := f.listLessons[name]
	if !isList || !slices.ContainsFunc(lessons, func(l string) bool { return l != "" }) {
		f.lessonScroll.Hide()
		return
	}
	chosen := f.lessonFilters[name]
	f.lessonList, f.lessonOrder = name, lessons
	f.lessonChecks = make(map[string]*widget.Check, len(lessons))
	checks := make([]*widget.Check, 0, len(lessons))
	grid := container.NewGridWithColumns(4)
	for _, l := range lessons {
		c := widget.NewCheck(lessonLabel(l), nil)
		c.SetChecked(len(chosen) == 0 || slices.Contains(chosen, l))
		f.lessonChecks[l] = c
		checks = append(checks, c)
		grid.Add(c)
	}
	f.allLessons = widget.NewCheck("All lessons", nil)
	f.allLessons.SetChecked(allChecked(checks))
	linkAllCheck(f.allLessons, checks)
	f.lessonBox.Add(f.allLessons)
	f.lessonBox.Add(grid)
	f.lessonScroll.Show()
}

// keepLessons keeps the lesson checks shown as that list's filter in this
// dialog, so switching sources back and forth keeps them.
func (f *settingsForm) keepLessons() {
	if f.lessonList != "" {
		f.lessonFilters[f.lessonList] = f.chosenLessons()
	}
}

// chosenLessons is the filter the lesson checks shown give: the checked
// lessons in list order; nil (all lessons) when all or none are checked.
func (f *settingsForm) chosenLessons() []string {
	var out []string
	for _, l := range f.lessonOrder {
		if f.lessonChecks[l].Checked {
			out = append(out, l)
		}
	}
	if len(out) == len(f.lessonOrder) {
		return nil
	}
	return out
}

// selectedLessons is the lesson filter to save for the selected source:
// the checks for a list that shows them, nil (all lessons) for a list
// without lessons, the list's filter as it is for a missing list, nil for
// JLPT.
func (f *settingsForm) selectedLessons() []string {
	name, ok := strings.CutPrefix(f.wordSource(), listSourcePrefix)
	switch {
	case !ok:
		return nil
	case name == f.lessonList:
		return f.chosenLessons()
	}
	if _, exists := f.listLessons[name]; exists {
		return nil
	}
	return f.lessonFilters[name]
}

// lessonLabel is a lesson's check label: its name, "No lesson" for words
// without one.
func lessonLabel(l string) string {
	if l == "" {
		return "No lesson"
	}
	return l
}

// setMeaningLangs replaces the meaning select's languages and keeps its
// selection. Must run on the Fyne thread.
func (f *settingsForm) setMeaningLangs(langs []string) {
	current := f.meaningLang()
	labels, values := meaningLangOptions(langs, current)
	f.meaningValues = values
	f.meaningSelect.SetOptions(labels)
	f.meaningSelect.SetSelectedIndex(slices.Index(values, current))
}

// meaningLang is the meaning language the Words tab has selected; "" =
// System.
func (f *settingsForm) meaningLang() string {
	if i := f.meaningSelect.SelectedIndex(); i >= 0 {
		return f.meaningValues[i]
	}
	return ""
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
	gs.SetFocusWeak(f.focusCheck.Checked)
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
	// Pause first: the reload may drop tiles, which must not happen in a
	// running round.
	gs.SetSettingsOpen(true)
	gameCanvas.Refresh()
	gs.ReloadLists() // lists copied, edited or removed by hand come in
	form := newSettingsForm(gs.settingsState(), onReset)
	form.onImport = func() { showImportDialog(gs, form, win) }
	form.onManage = func() { showListsDialog(gs, form, win) }

	d = dialog.NewCustomWithoutButtons("Settings", form.tabs, win)
	openSettings = form
	// Every way out (Save, Cancel, a reset) hides the dialog.
	d.SetOnClosed(func() { openSettings = nil })
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
// group, keeping them in sync (linkAllCheck).
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
	linkAllCheck(all, checkList)
	return all, checks, grid
}

// linkAllCheck keeps all and checks in sync: checking or unchecking all
// sets every check, and a check's change sets all to whether every check is
// on. A syncing flag guards against the handlers recursing into each other.
func linkAllCheck(all *widget.Check, checks []*widget.Check) {
	syncing := false
	all.OnChanged = func(on bool) {
		if syncing {
			return
		}
		syncing = true
		for _, c := range checks {
			c.SetChecked(on)
		}
		syncing = false
	}
	for _, c := range checks {
		c.OnChanged = func(bool) {
			if syncing {
				return
			}
			syncing = true
			all.SetChecked(allChecked(checks))
			syncing = false
		}
	}
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
