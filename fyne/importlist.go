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
	"fyne.io/fyne/v2/container"
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

// importPreviewLines describes a candidate: its name, whether it replaces a
// word list, counts, skipped rows and the first words with meanings in lang.
func importPreviewLines(c vocab.Candidate, lang string, replaces bool) []string {
	lines := []string{"Word list: " + prettyListName(c.Name)}
	if replaces {
		lines = append(lines, "Replaces the word list of the same name; your progress is kept.")
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
			lines = append(lines, "• "+p)
		}
	}
	lines = append(lines, "First words:")
	for _, w := range c.List.Words[:min(importPreviewWords, len(c.List.Words))] {
		parts := []string{w.Kana}
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
		importChosen(gs, f, r.URI().Name(), data, win)
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
// cannot be imported. Import (Replace for an existing name) stores it,
// reloads the lists and preselects it in f. Must run on the Fyne thread.
func importChosen(gs *GameState, f *settingsForm, fileName string, data []byte, win fyne.Window) {
	c, err := vocab.Inspect(fileName, data)
	if err != nil {
		showImportError(err, win)
		return
	}
	replaces := gs.HasList(c.Name)
	body := container.NewVBox()
	for _, line := range importPreviewLines(c, gs.MeaningLang(), replaces) {
		body.Add(widget.NewLabel(line))
	}
	confirm := "Import"
	if replaces {
		confirm = "Replace"
	}
	dialog.ShowCustomConfirm("Import word list", confirm, "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		name, err := gs.ImportList(c)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		f.setLists(gs.ListNames(), name)
		// The summary may have said the chosen list was missing.
		f.summary.SetText(gs.settingsState().summary)
	}, win)
}

// showImportError shows why a file cannot be imported and what to do.
func showImportError(err error, win fyne.Window) {
	text := err.Error()
	var ie *vocab.ImportError
	if errors.As(err, &ie) && ie.Hint != "" {
		text += "\n\n" + ie.Hint
	}
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom("Cannot import", "OK", label, win)
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}
