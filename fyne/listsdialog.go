package main

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// skippedRowsHint ends the skipped rows' details.
const skippedRowsHint = "Correct these lines in your source file and import it again; the rest of the list plays meanwhile."

// skippedRowBullet starts each skipped row's message in the details and the
// import preview.
const skippedRowBullet = "• "

// A run of more skipped-row messages than skippedRowsScrollAfter scrolls in
// an area skippedRowsScrollH high, so a dialog with 100 of them stays on
// screen.
const (
	skippedRowsScrollAfter         = 8
	skippedRowsScrollH     float32 = 200
)

// listCounts is a list's line in the word list dialog: "4 words · 2
// lessons · 1 row skipped", without the parts that are 0.
func listCounts(li listInfo) string {
	parts := []string{plural(li.words, "word")}
	if li.lessons > 0 {
		parts = append(parts, plural(li.lessons, "lesson"))
	}
	if li.skipped > 0 {
		parts = append(parts, plural(li.skipped, "row")+" skipped")
	}
	return strings.Join(parts, " · ")
}

// skippedRowsLines are the skipped rows' details: a line per message, how
// many rows have no message, then how to fix the file.
func skippedRowsLines(li listInfo) []string {
	lines := make([]string, 0, len(li.problems)+2)
	lines = appendSkippedRows(lines, li.problems, li.skipped)
	return append(lines, skippedRowsHint)
}

// appendSkippedRows appends a line per message and, when skipped rows are
// more than the messages kept, a line saying how many more there are.
func appendSkippedRows(lines, problems []string, skipped int) []string {
	for _, p := range problems {
		lines = append(lines, skippedRowBullet+p)
	}
	if more := skipped - len(problems); more > 0 {
		lines = append(lines, "… and "+plural(more, "more row")+" skipped")
	}
	return lines
}

// linesBody shows lines as wrapped labels. A run of more than
// skippedRowsScrollAfter skipped-row messages (lines starting with
// skippedRowBullet) sits in a scroll area of its own; the lines around it
// stay in view.
func linesBody(lines []string) *fyne.Container {
	body := container.NewVBox()
	for i := 0; i < len(lines); {
		end := i
		for end < len(lines) && strings.HasPrefix(lines[end], skippedRowBullet) {
			end++
		}
		if end-i > skippedRowsScrollAfter {
			rows := container.NewVBox()
			for _, line := range lines[i:end] {
				rows.Add(wrappedLabel(line))
			}
			scroll := container.NewVScroll(rows)
			scroll.SetMinSize(fyne.NewSize(0, skippedRowsScrollH))
			body.Add(scroll)
			i = end
			continue
		}
		end = max(end, i+1)
		for _, line := range lines[i:end] {
			body.Add(wrappedLabel(line))
		}
		i = end
	}
	return body
}

// wrappedLabel is a label that wraps at word boundaries.
func wrappedLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

// listsDialog is the word list dialog over the settings dialog's form f.
type listsDialog struct {
	gs   *GameState
	f    *settingsForm
	win  fyne.Window
	rows *fyne.Container
}

// showListsDialog shows the word lists, each with its counts and skipped
// rows, to rename or delete; another can be imported and the folder opened.
// Changes act at once and show in f. Must run on the Fyne thread.
func showListsDialog(gs *GameState, f *settingsForm, win fyne.Window) {
	ld := &listsDialog{gs: gs, f: f, win: win, rows: container.NewVBox()}
	ld.fill()
	scroll := container.NewVScroll(ld.rows)
	scroll.SetMinSize(fyne.NewSize(560, 280))
	importButton := widget.NewButton("Import…", func() {
		if f.onImport != nil {
			f.onImport()
		}
	})
	dir := gs.WordsDir()
	folderButton := widget.NewButton("Open folder", func() { openWordsFolder(dir, win) })
	if dir == "" {
		importButton.Disable()
		folderButton.Disable()
	}
	content := container.NewBorder(nil, container.NewHBox(importButton, folderButton), nil, nil, scroll)
	d := dialog.NewCustom("Word lists", "Close", content, win)
	f.onListsChanged = ld.fill
	d.SetOnClosed(func() { f.onListsChanged = nil })
	d.Show()
}

// fill shows a row per list, or that there are none. Must run on the Fyne
// thread.
func (ld *listsDialog) fill() {
	ld.rows.RemoveAll()
	infos := ld.gs.ListInfos()
	if len(infos) == 0 {
		ld.rows.Add(widget.NewLabel("No word lists yet"))
		return
	}
	for _, li := range infos {
		ld.rows.Add(ld.row(li))
	}
}

// row is a list's row: its name, counts and skipped rows, and its buttons.
func (ld *listsDialog) row(li listInfo) fyne.CanvasObject {
	buttons := container.NewHBox()
	if li.skipped > 0 {
		buttons.Add(widget.NewButton("Details", func() { showSkippedRows(li, ld.win) }))
	}
	buttons.Add(widget.NewButton("Rename", func() { ld.rename(li) }))
	buttons.Add(widget.NewButton("Delete", func() { ld.confirmDelete(li) }))
	text := container.NewVBox(
		widget.NewLabelWithStyle(prettyListName(li.name), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(listCounts(li)),
	)
	return container.NewBorder(nil, nil, nil, buttons, text)
}

// showSkippedRows shows a list's skipped rows, scrolling when there are
// many, and how to fix the file. Must run on the Fyne thread.
func showSkippedRows(li listInfo, win fyne.Window) {
	d := dialog.NewCustom(prettyListName(li.name)+": "+plural(li.skipped, "row")+" skipped", "OK",
		linesBody(skippedRowsLines(li)), win)
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}

// confirmDelete asks before deleting a list, then deletes it and refreshes
// the settings (and so this dialog). Must run on the Fyne thread.
func (ld *listsDialog) confirmDelete(li listInfo) {
	c := dialog.NewConfirm("Delete word list",
		"Delete "+prettyListName(li.name)+"? Its file is removed; your progress on its words is kept.",
		func(ok bool) {
			if !ok {
				return
			}
			if err := ld.gs.DeleteList(li.name); err != nil {
				ld.f.listsChanged(ld.gs.settingsState(), ld.f.wordSource())
				dialog.ShowError(err, ld.win)
				return
			}
			ld.f.listDeleted(ld.gs.settingsState(), li.name)
		}, ld.win)
	c.SetConfirmText("Delete")
	c.SetDismissText("Cancel")
	c.Show()
}

// rename asks for a list's new file name, checked while typing, then renames
// it and refreshes the settings (and so this dialog). Must run on the Fyne
// thread.
func (ld *listsDialog) rename(li listInfo) {
	entry := widget.NewEntry()
	entry.SetText(li.fileBase())
	entry.Validator = func(s string) error { return ld.gs.CheckListName(li.name, s) }
	item := widget.NewFormItem("Name", entry)
	item.HintText = "The file keeps its extension (" + filepath.Ext(li.path) + ")."
	d := dialog.NewForm("Rename "+prettyListName(li.name), "Rename", "Cancel", []*widget.FormItem{item}, func(ok bool) {
		if !ok {
			return
		}
		to, err := ld.gs.RenameList(li.name, entry.Text)
		if err != nil {
			ld.f.listsChanged(ld.gs.settingsState(), ld.f.wordSource())
			dialog.ShowError(err, ld.win)
			return
		}
		ld.f.listRenamed(ld.gs.settingsState(), li.name, to)
	}, ld.win)
	d.Resize(fyne.NewSize(420, 0))
	d.Show()
}

// openURL opens a URL with the system's handler. Tests replace it.
var openURL = func(u *url.URL) error { return fyne.CurrentApp().OpenURL(u) }

// folderURL is the file URL of a folder: file:///home/… or, on Windows,
// file:///C:/….
func folderURL(dir string) *url.URL {
	p := filepath.ToSlash(dir)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return &url.URL{Scheme: "file", Path: p}
}

// openWordsFolder opens the word list folder in the system file manager,
// creating it first if needed; if that fails it shows the path to copy.
// Must run on the Fyne thread.
func openWordsFolder(dir string, win fyne.Window) {
	if err := os.MkdirAll(dir, wordsDirPerm); err == nil {
		if err := openURL(folderURL(dir)); err == nil {
			return
		}
	}
	showFolderPath(dir, win)
}

// showFolderPath shows the word list folder's path, selectable for copying.
func showFolderPath(dir string, win fyne.Window) {
	path := widget.NewLabel(dir)
	path.Selectable = true
	d := dialog.NewCustom("Word list folder", "OK", container.NewVBox(
		widget.NewLabel("The folder could not be opened. Open it in your file manager:"),
		path,
	), win)
	d.Show()
}
