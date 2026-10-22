package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"kana/vocab"
)

// importPreviewWords is how many words the import preview shows.
const importPreviewWords = 3

// importFileExts are the extensions the file dialog shows; spreadsheets are
// shown so that their hint can be given.
var importFileExts = []string{".csv", ".tsv", ".txt", ".json", ".xlsx", ".xls", ".ods"}

// importPreviewLines describes a candidate: its name, the word list files it
// replaces, counts, skipped rows and the first words with meanings in lang.
func importPreviewLines(c vocab.Candidate, lang string, files []string) []string {
	lines := []string{"Word list: " + prettyListName(c.Name)}
	switch len(files) {
	case 0:
	case 1:
		lines = append(lines, "Replaces the word list of the same name; your progress is kept.")
	default:
		lines = append(lines, "Replaces "+strings.Join(files, ", ")+"; your progress is kept.")
	}
	lessons := map[string]bool{}
	langs := map[string]bool{}
	for _, w := range c.List.Words {
		if w.Lesson != "" {
			lessons[w.Lesson] = true
		}
		for l := range w.Meanings {
			if l != "" {
				langs[l] = true
			}
		}
	}
	counts := []string{plural(len(c.List.Words), "word")}
	if len(lessons) > 0 {
		counts = append(counts, plural(len(lessons), "lesson"))
	}
	if len(langs) > 0 {
		counts = append(counts, "meanings: "+strings.Join(slices.Sorted(maps.Keys(langs)), ", "))
	}
	lines = append(lines, strings.Join(counts, " · "))
	if c.List.Skipped > 0 {
		lines = append(lines, plural(c.List.Skipped, "row")+" skipped:")
		for _, p := range c.List.Problems {
			lines = append(lines, skippedRowBullet+p)
		}
	}
	lines = append(lines, "First words:")
	for _, w := range c.List.Words[:min(importPreviewWords, len(c.List.Words))] {
		parts := []string{w.Shown()}
		if m := meaningSnippet(w.Meanings, lang); m != "" {
			parts = append(parts, m)
		}
		if w.Lesson != "" {
			parts = append(parts, "lesson "+w.Lesson)
		}
		lines = append(lines, strings.Join(parts, " · "))
	}
	return lines
}

// plural is "1 word" or "4 words".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// showImportDialog lets the learner pick a source file, starting in the
// home directory. Must run on the Fyne thread.
func showImportDialog(gs *GameState, f *settingsForm, win fyne.Window) {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if r == nil {
			return // cancelled
		}
		defer r.Close()
		data, err := io.ReadAll(io.LimitReader(r, vocab.MaxImportSize+1))
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		importChosen(gs, f, r.URI().Name(), data, win, nil)
	}, win)
	d.SetFilter(storage.NewExtensionFileFilter(importFileExts))
	if home, err := os.UserHomeDir(); err == nil {
		if l, err := storage.ListerForURI(storage.NewFileURI(home)); err == nil {
			d.SetLocation(l)
		}
	}
	d.Resize(win.Canvas().Size())
	d.Show()
}

// importChosen checks a chosen source file and shows its preview, or why it
// cannot be imported. Import (Replace for an existing name) stores it and
// reloads the lists; a settings form f (may be nil) preselects it. then (may
// be nil) is called once the preview or error is closed, whatever the
// answer. Must run on the Fyne thread.
func importChosen(gs *GameState, f *settingsForm, fileName string, data []byte, win fyne.Window, then func()) {
	done := func() {
		if then != nil {
			then()
		}
	}
	c, err := vocab.Inspect(fileName, data)
	if err != nil {
		showImportError(err, win, then)
		return
	}
	files := gs.ListFiles(c.Name)
	body := linesBody(importPreviewLines(c, gs.MeaningLang(), files))
	confirm := "Import"
	if len(files) > 0 {
		confirm = "Replace"
	}
	d := dialog.NewCustomConfirm("Import word list", confirm, "Cancel", body, func(ok bool) {
		if !ok {
			done()
			return
		}
		name, err := gs.ImportList(c)
		if err != nil {
			showErrorThen(err, win, then)
			return
		}
		if f != nil {
			f.listImported(gs.settingsState(), name)
		}
		done()
	}, win)
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}

// showImportError shows why a file cannot be imported and what to do; then
// (may be nil) is called once it is closed.
func showImportError(err error, win fyne.Window, then func()) {
	text := err.Error()
	var ie *vocab.ImportError
	if errors.As(err, &ie) && ie.Hint != "" {
		text += "\n\n" + ie.Hint
	}
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom("Cannot import", "OK", label, win)
	if then != nil {
		d.SetOnClosed(then)
	}
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}

// showErrorThen shows err; then (may be nil) is called once it is closed.
func showErrorThen(err error, win fyne.Window, then func()) {
	d := dialog.NewError(err, win)
	if then != nil {
		d.SetOnClosed(then)
	}
	d.Show()
}
