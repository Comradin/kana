package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/vocab"
)

// listsDialogGame opens the word list dialog over a settings form for a game
// with the given lists; source as in listGame.
func listsDialogGame(t *testing.T, source string, files map[string]string) (*GameState, *settingsForm, fyne.Window, string) {
	t.Helper()
	gs, dir := listGame(t, source, files)
	t.Cleanup(func() { test.NewApp() })
	w := test.NewWindow(nil)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(1000, 760))
	f := newSettingsForm(gs.settingsState(), nil)
	showListsDialog(gs, f, w)
	return gs, f, w, dir
}

// entryIn finds the first entry in a dialog's objects.
func entryIn(o fyne.CanvasObject) *widget.Entry {
	for _, obj := range test.LaidOutObjects(o) {
		if e, ok := obj.(*widget.Entry); ok {
			return e
		}
	}
	return nil
}

func TestListCountsAndSkippedLines(t *testing.T) {
	li := listInfo{name: "minna", words: 4, lessons: 2, skipped: 1, problems: []string{"line 6: abc is not kana"}}
	if got := listCounts(li); got != "4 words · 2 lessons · 1 row skipped" {
		t.Fatalf("listCounts = %q", got)
	}
	if got := listCounts(listInfo{words: 1}); got != "1 word" {
		t.Fatalf("listCounts without lessons = %q", got)
	}
	if got := skippedRowsLines(li); !equalIDs(got, []string{"• line 6: abc is not kana", skippedRowsHint}) {
		t.Fatalf("skippedRowsLines = %q", got)
	}
	cut := listInfo{skipped: 103, problems: []string{"line 2: x is not kana", "line 3: y is not kana"}}
	want := []string{"• line 2: x is not kana", "• line 3: y is not kana", "… and 101 more rows skipped", skippedRowsHint}
	if got := skippedRowsLines(cut); !equalIDs(got, want) {
		t.Fatalf("skippedRowsLines with messages cut off = %q, want %q", got, want)
	}
}

func TestListsDialogRows(t *testing.T) {
	_, _, w, _ := listsDialogGame(t, jlptSource, map[string]string{
		"minna-no-test.csv": "kana;meaning;lesson\nあお;blue;1\nいえ;house;2\nabc;x;3\n",
		"plain.csv":         "kana;meaning\nうえ;up\n",
	})
	top := w.Canvas().Overlays().Top()
	for _, s := range []string{"minna no test", "2 words · 2 lessons · 1 row skipped", "plain", "1 word"} {
		if !labelContains(top, s) {
			t.Errorf("no %q in the dialog", s)
		}
	}
	details := 0
	for _, o := range test.LaidOutObjects(top) {
		if b, ok := o.(*widget.Button); ok && b.Text == "Details" {
			details++
		}
	}
	if details != 1 {
		t.Fatalf("%d Details buttons, want one for the list with skipped rows", details)
	}
	for _, b := range []string{"Rename", "Delete", "Import…", "Open folder", "Close"} {
		if buttonIn(top, b) == nil {
			t.Errorf("no %q button", b)
		}
	}
	test.Tap(buttonIn(top, "Details"))
	if d := w.Canvas().Overlays().Top(); !labelContains(d, "line 4: abc is not kana") || !labelContains(d, "source file") {
		t.Fatal("Details must show the messages and how to fix the file")
	}
}

func TestListsDialogEmpty(t *testing.T) {
	_, _, w, _ := listsDialogGame(t, jlptSource, nil)
	if !labelContains(w.Canvas().Overlays().Top(), "No word lists yet") {
		t.Fatal("want No word lists yet")
	}
}

func TestListsDialogDelete(t *testing.T) {
	gs, f, w, dir := listsDialogGame(t, listSourcePrefix+"mine", map[string]string{"mine.csv": lessonList})
	lists := w.Canvas().Overlays().Top()
	test.Tap(buttonIn(lists, "Delete"))
	confirm := w.Canvas().Overlays().Top()
	if !labelContains(confirm, "Delete mine? Its file is removed; your progress on its words is kept.") {
		t.Fatal("no confirmation text")
	}
	test.Tap(buttonIn(confirm, "Cancel"))
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); err != nil {
		t.Fatal("Cancel must keep the file")
	}
	test.Tap(buttonIn(lists, "Delete"))
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Delete"))
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); !os.IsNotExist(err) {
		t.Fatal("the file must be gone")
	}
	if gs.wordSource != jlptSource || f.wordSource() != jlptSource || !equalIDs(f.sourceSelect.Options, []string{"JLPT"}) {
		t.Fatalf("game source %q, form source %q, options %v", gs.wordSource, f.wordSource(), f.sourceSelect.Options)
	}
	if w.Canvas().Overlays().Top() != lists || !labelContains(lists, "No word lists yet") {
		t.Fatal("the dialog must stay open and show no lists")
	}
}

func TestListsDialogRename(t *testing.T) {
	gs, f, w, dir := listsDialogGame(t, listSourcePrefix+"mine", map[string]string{
		"mine.csv": lessonList, "other.csv": lessonList,
	})
	_ = gs.store.SaveLessonFilter("mine", []string{"2"})
	gs.ReloadLists()
	f.listsChanged(gs.settingsState(), f.wordSource())
	f.lessonChecks["1"].SetChecked(false) // an unsaved edit: only lesson 2
	lists := w.Canvas().Overlays().Top()

	var rename *widget.Button // the first row's: mine
	for _, o := range test.LaidOutObjects(lists) {
		if b, ok := o.(*widget.Button); ok && b.Text == "Rename" && rename == nil {
			rename = b
		}
	}
	test.Tap(rename)
	form := w.Canvas().Overlays().Top()
	entry := entryIn(form)
	if entry == nil || entry.Text != "mine" {
		t.Fatal("the entry must hold the file name")
	}
	entry.SetText("jmdict")
	if err := entry.Validate(); !errors.Is(err, vocab.ErrNameReserved) {
		t.Fatalf("Validate = %v, want ErrNameReserved", err)
	}
	if b := buttonIn(form, "Rename"); b == nil || !b.Disabled() {
		t.Fatal("Rename must be disabled while the name is invalid")
	}
	entry.SetText("ours")
	test.Tap(buttonIn(form, "Rename"))
	if _, err := os.Stat(filepath.Join(dir, "ours.csv")); err != nil {
		t.Fatal("the file must be ours.csv")
	}
	if f.wordSource() != listSourcePrefix+"ours" || f.storedSource != listSourcePrefix+"ours" {
		t.Fatalf("form source %q, stored %q; want list:ours", f.wordSource(), f.storedSource)
	}
	if !equalIDs(checkedLessons(f), []string{"2"}) || !equalIDs(f.words().lessons, []string{"2"}) {
		t.Fatalf("checked %v; the lesson checks must follow the list", checkedLessons(f))
	}
	if !labelContains(lists, "ours") {
		t.Fatal("the dialog must show the new name")
	}
}

func TestFolderURL(t *testing.T) {
	for in, want := range map[string]string{
		"/home/x/my words": "file:///home/x/my%20words",
		"C:/Users/x":       "file:///C:/Users/x",
	} {
		if got := folderURL(in).String(); got != want {
			t.Fatalf("folderURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOpenFolderShowsPathWhenItFails(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	orig := openURL
	t.Cleanup(func() { openURL = orig })
	dir := filepath.Join(t.TempDir(), "words")

	var opened *url.URL
	openURL = func(u *url.URL) error { opened = u; return nil }
	openWordsFolder(dir, w)
	if opened == nil || opened.Path != filepath.ToSlash(dir) || w.Canvas().Overlays().Top() != nil {
		t.Fatalf("opened %v; want the folder opened without a dialog", opened)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatal("a missing folder is created first")
	}

	openURL = func(*url.URL) error { return errors.New("no file manager") }
	openWordsFolder(dir, w)
	if top := w.Canvas().Overlays().Top(); top == nil || !labelContains(top, dir) {
		t.Fatal("a failure must show the path")
	}
}

// scrollIn finds the first scroll area in a dialog's objects.
func scrollIn(o fyne.CanvasObject) *container.Scroll {
	for _, obj := range test.LaidOutObjects(o) {
		if s, ok := obj.(*container.Scroll); ok {
			return s
		}
	}
	return nil
}

// skippedLines returns n skipped-row lines as the dialogs show them.
func skippedLines(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("%sline %d: x is not kana", skippedRowBullet, i+2))
	}
	return out
}

func TestLinesBodyScrollsLongRunsOfSkippedRows(t *testing.T) {
	test.NewApp()
	lines := append(append([]string{"Word list: a", "9 rows skipped:"}, skippedLines(9)...), "First words:")
	body := linesBody(lines)
	if len(body.Objects) != 4 {
		t.Fatalf("%d objects, want two labels, a scroll area, a label", len(body.Objects))
	}
	scroll, ok := body.Objects[2].(*container.Scroll)
	if !ok || len(scroll.Content.(*fyne.Container).Objects) != 9 {
		t.Fatalf("object 2 is %T; want a scroll area with the 9 messages", body.Objects[2])
	}
	short := linesBody(append([]string{"8 rows skipped:"}, skippedLines(8)...))
	for _, o := range short.Objects {
		if _, ok := o.(*widget.Label); !ok {
			t.Fatalf("8 messages need no scroll area, got a %T", o)
		}
	}
}

func TestDetailsScrollWithManySkippedRows(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	li := listInfo{name: "many", words: 1, skipped: 150}
	for i := 0; i < 100; i++ {
		li.problems = append(li.problems, fmt.Sprintf("line %d: x is not kana", i+2))
	}
	showSkippedRows(li, w)
	top := w.Canvas().Overlays().Top()
	if scrollIn(top) == nil {
		t.Fatal("100 messages must scroll")
	}
	if !labelContains(top, skippedRowsHint) {
		t.Fatal("the hint stays below the scroll area")
	}
}
