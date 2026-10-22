package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
)

func TestImportOpenPausesLikeTheSettings(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("あ", "a", 10)}
	gs.SetSettingsOpen(true)
	gs.SetImportOpen(true)
	if !gs.paused || len(tileObjects(gs)) != 0 {
		t.Fatal("an import preview pauses and hides the tiles")
	}
	gs.SetImportOpen(false)
	if !gs.paused {
		t.Fatal("closing the preview must not release the settings' pause")
	}
	gs.SetSettingsOpen(false)
	if gs.paused {
		t.Fatal("with nothing open the game goes on")
	}
}

// dropGame is a running window setup for drops: a fresh game with the vowels
// selected, a test window and a drop importer; src is a folder for source
// files outside the word list folder.
func dropGame(t *testing.T) (gs *GameState, d *dropImporter, w fyne.Window, src string) {
	t.Helper()
	test.NewApp()
	t.Cleanup(func() { test.NewApp() })
	openSettings, dialogShowing = nil, false
	t.Cleanup(func() { openSettings, dialogShowing = nil, false })
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs = NewGameState(st)
	w = test.NewWindow(nil)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(1000, 760))
	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, statsPanel, gameCanvas, w)
	d = &dropImporter{gs: gs, statsPanel: statsPanel, gameCanvas: gameCanvas, inputBar: ib, win: w}
	return gs, d, w, t.TempDir()
}

func TestDropShowsPreviewAndPauses(t *testing.T) {
	gs, d, w, src := dropGame(t)
	a := writeList(t, src, "a.csv", "kana;meaning\nあお;blue\n")
	d.drop([]fyne.URI{storage.NewFileURI(a)})
	if !gs.importOpen || !gs.paused {
		t.Fatal("the game must pause while the preview shows")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if !equalIDs(gs.listNames, []string{"a"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if gs.importOpen || gs.paused || w.Canvas().Overlays().Top() != nil {
		t.Fatalf("after the last file: importOpen %v, paused %v", gs.importOpen, gs.paused)
	}
}

func TestDropSeveralFilesInTurn(t *testing.T) {
	gs, d, w, src := dropGame(t)
	a := writeList(t, src, "a.csv", "kana;meaning\nあお;blue\n")
	b := writeList(t, src, "b.csv", "kana;meaning\nいえ;house\n")
	d.drop([]fyne.URI{storage.NewFileURI(a), storage.NewFileURI(b)})
	if !labelContains(w.Canvas().Overlays().Top(), "Word list: a") {
		t.Fatal("first the preview of a")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if !labelContains(w.Canvas().Overlays().Top(), "Word list: b") || !gs.importOpen {
		t.Fatal("then the preview of b, still paused")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Cancel"))
	if !equalIDs(gs.listNames, []string{"a"}) || gs.importOpen {
		t.Fatalf("lists %v, importOpen %v", gs.listNames, gs.importOpen)
	}
}

func TestDropFolderShowsTheImportError(t *testing.T) {
	gs, d, w, src := dropGame(t)
	folder := filepath.Join(src, "lists.csv")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	d.drop([]fyne.URI{storage.NewFileURI(folder)})
	top := w.Canvas().Overlays().Top()
	if top == nil || !labelContains(top, "This is a folder.") {
		t.Fatal("a folder must get the import's error")
	}
	test.Tap(buttonIn(top, "OK"))
	if gs.importOpen || len(gs.listNames) != 0 {
		t.Fatalf("importOpen %v, lists %v", gs.importOpen, gs.listNames)
	}
}

func TestDropOnOpenSettingsShowsTheNewList(t *testing.T) {
	gs, d, w, src := dropGame(t)
	showSettingsDialog(gs, d.statsPanel, d.gameCanvas, d.inputBar, w)
	settings := w.Canvas().Overlays().Top()
	a := writeList(t, src, "a.csv", "kana;meaning\nあお;blue\n")
	d.drop([]fyne.URI{storage.NewFileURI(a)})
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if w.Canvas().Overlays().Top() != settings || !gs.settingsOpen || !gs.paused {
		t.Fatal("the settings must be there again, the game still paused")
	}
	if openSettings == nil || !slices.Contains(openSettings.sourceSelect.Options, "a") || openSettings.wordSource() != listSourcePrefix+"a" {
		t.Fatal("the open settings must show and preselect the new list")
	}
}
