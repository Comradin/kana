package main

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"
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
	if gs.HasList("mine") {
		t.Fatal("HasList before the import")
	}
	if name, err := gs.ImportList(c); err != nil || name != "mine" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if !equalIDs(gs.listNames, []string{"mine"}) || !gs.HasList("MINE") {
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
