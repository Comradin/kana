package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/vocab"
)

func TestStartupCreatesTheWordsFolder(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	NewGameState(openTestStore(t))
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("words folder: %v", err)
	}
}

func TestImportListReloadsAndKeepsProgress(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	gs := NewGameState(openTestStore(t))
	c, err := vocab.Inspect("mine.csv", []byte("kana;meaning\nあお;blue\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := gs.ListFiles("mine"); len(got) != 0 {
		t.Fatal("ListFiles before the import")
	}
	if name, err := gs.ImportList(c); err != nil || name != "mine" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if !equalIDs(gs.listNames, []string{"mine"}) || !slices.Equal(gs.ListFiles("MINE"), []string{"mine.csv"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if _, ok := gs.words.Lookup("あお"); !ok {
		t.Fatal("the imported word is not in the index")
	}
	gs.mu.Lock()
	setWordLevel(gs, "あお", 3)
	gs.introduced["あお"] = true
	gs.wordSource = listSourcePrefix + "mine"
	gs.mu.Unlock()

	c, _ = vocab.Inspect("Mine.json", []byte(`[{"kana":"いえ","meaning":"house"}]`))
	if name, err := gs.ImportList(c); err != nil || name != "mine" {
		t.Fatalf("name %q, err %v; the list keeps its name", name, err)
	}
	if !equalIDs(gs.listNames, []string{"mine"}) {
		t.Fatalf("after replacing: lists = %v", gs.listNames)
	}
	gs.mu.Lock()
	_, found := gs.source()
	gs.mu.Unlock()
	if !found || !gs.introduced["あお"] {
		t.Fatal("the selected list must still be found and its introduced words kept")
	}
	if _, ok := gs.words.Lookup("いえ"); !ok {
		t.Fatal("the new word is not in the index")
	}
	if gs.wordProgress["あお"].Level != 3 {
		t.Fatal("replacing a list must keep the progress of its old words")
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.json")); err != nil {
		t.Fatal("the new file is stored under the list's name: mine.json")
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); !os.IsNotExist(err) {
		t.Fatal("the old file must be gone")
	}
}

func TestImportPreviewLines(t *testing.T) {
	c, err := vocab.Inspect("minna-no-nihongo.csv", []byte(
		"kana;meaning_de;meaning_en;lesson\nせんせい;Lehrer;teacher;1\nがくせい;Student;student;1\nえき;Bahnhof;station;2\nいえ;Haus;house;2\nabc;x;x;3\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := importPreviewLines(c, "de", []string{"minna.csv"})
	want := []string{
		"Word list: minna no nihongo",
		"Replaces the word list of the same name; your progress is kept.",
		"4 words · 2 lessons · meanings: de, en",
		"1 row skipped:",
		"• line 6: abc is not kana",
		"First words:",
		"せんせい · Lehrer · lesson 1",
		"がくせい · Student · lesson 1",
		"えき · Bahnhof · lesson 2",
	}
	if !equalIDs(got, want) {
		t.Fatalf("lines:\n%q\nwant\n%q", got, want)
	}

	c, _ = vocab.Inspect("a.csv", []byte("kana;meaning\nせんせい;teacher\n"))
	if got := importPreviewLines(c, "en", nil); !equalIDs(got, []string{
		"Word list: a", "1 word", "First words:", "せんせい · teacher",
	}) {
		t.Fatalf("plain list: %q", got)
	}
}

func TestImportPreviewLinesSeveralFiles(t *testing.T) {
	c, _ := vocab.Inspect("genki.csv", []byte("kana;meaning\nせんせい;teacher\n"))
	got := importPreviewLines(c, "en", []string{"Genki.json", "genki.csv"})
	if want := "Replaces Genki.json, genki.csv; your progress is kept."; len(got) < 2 || got[1] != want {
		t.Fatalf("lines: %q, want line 2 %q", got, want)
	}
}

func TestImportChosenShowsPreviewAndImports(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	isolateWordsDir(t)
	gs := NewGameState(openTestStore(t))
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	f := newSettingsForm(gs.settingsState(), nil)

	importChosen(gs, f, "mine.csv", []byte("kana;meaning\nあお;blue\n"), w)
	preview := w.Canvas().Overlays().Top()
	if preview == nil || buttonIn(preview, "Import") == nil {
		t.Fatal("no preview with an Import button")
	}
	test.Tap(buttonIn(preview, "Import"))
	if !equalIDs(gs.listNames, []string{"mine"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if f.wordSource() != "list:mine" || !slices.Contains(f.sourceSelect.Options, "mine") {
		t.Fatalf("source %q, options %v; want the new list preselected", f.wordSource(), f.sourceSelect.Options)
	}

	importChosen(gs, f, "MINE.csv", []byte("kana;meaning\nいえ;house\n"), w)
	if buttonIn(w.Canvas().Overlays().Top(), "Replace") == nil {
		t.Fatal("an existing name must offer Replace")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Replace"))
	if !equalIDs(gs.listNames, []string{"mine"}) || f.wordSource() != "list:mine" {
		t.Fatalf("lists %v, source %q; the replaced list keeps its name", gs.listNames, f.wordSource())
	}
}

func TestImportChosenRejectsWithHint(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	isolateWordsDir(t)
	gs := NewGameState(openTestStore(t))
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	f := newSettingsForm(gs.settingsState(), nil)
	importChosen(gs, f, "list.xlsx", []byte("PK"), w)
	top := w.Canvas().Overlays().Top()
	if top == nil || !labelContains(top, "This is a spreadsheet file.") || !labelContains(top, "CSV UTF-8") {
		t.Fatal("a rejected file must show the reason and the hint")
	}
	if len(gs.listNames) != 0 {
		t.Fatal("nothing may be imported")
	}
}

func TestImportDoesNotChangeTheSource(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	before, _ := st.WordSource()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	f := newSettingsForm(gs.settingsState(), nil)
	importChosen(gs, f, "mine.csv", []byte("kana;meaning\nあお;blue\n"), w)
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if after, _ := st.WordSource(); gs.wordSource != before || after != before {
		t.Fatalf("source %q, stored %q; want %q until the settings are saved", gs.wordSource, after, before)
	}
	if !equalIDs(gs.listNames, []string{"mine"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
}

// labelContains reports whether a label or rich text under o contains s.
func labelContains(o fyne.CanvasObject, s string) bool {
	for _, obj := range test.LaidOutObjects(o) {
		switch v := obj.(type) {
		case *widget.Label:
			if strings.Contains(v.Text, s) {
				return true
			}
		case *widget.RichText:
			if strings.Contains(v.String(), s) {
				return true
			}
		}
	}
	return false
}
