package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"kana/vocab"
)

// writeList writes a synthetic word list file into dir, creating dir, and
// returns its path.
func writeList(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestListInfos(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	path := writeList(t, dir, "mine.csv", "kana;meaning;lesson\nあお;blue;1\nいえ;house;2\nabc;x;3\n")
	writeList(t, dir, "plain.csv", "kana;meaning\nうえ;up\n")
	gs := NewGameState(openTestStore(t))
	infos := gs.ListInfos()
	if len(infos) != 2 || infos[0].name != "mine" || infos[1].name != "plain" {
		t.Fatalf("infos = %+v, want mine and plain by name", infos)
	}
	m := infos[0]
	if m.path != path || m.fileBase() != "mine" || m.words != 2 || m.lessons != 2 || m.skipped != 1 ||
		!slices.Equal(m.problems, []string{"line 4: abc is not kana"}) {
		t.Fatalf("mine = %+v", m)
	}
	if p := infos[1]; p.lessons != 0 || p.skipped != 0 || p.words != 1 {
		t.Fatalf("plain = %+v", p)
	}
	if gs.WordsDir() != dir {
		t.Fatalf("WordsDir = %q, want %q", gs.WordsDir(), dir)
	}
}

func TestReloadListsPicksUpHandCopiedList(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	if len(gs.listNames) != 0 {
		t.Fatalf("lists = %v, want none yet", gs.listNames)
	}
	writeList(t, dir, "copied.csv", "kana;meaning\nあお;blue\n")
	gs.ReloadLists()
	if !equalIDs(gs.listNames, []string{"copied"}) || len(gs.ListInfos()) != 1 {
		t.Fatalf("lists = %v after reload", gs.listNames)
	}
	if _, ok := gs.words.Lookup("あお"); !ok {
		t.Fatal("the copied list's word is not in the index")
	}
}

func TestReloadWithoutFolderReadsNothing(t *testing.T) {
	gs := newTestState() // no word list folder: must never read the real one
	gs.listNames = []string{"hand-made"}
	gs.ReloadLists()
	if !equalIDs(gs.listNames, []string{"hand-made"}) {
		t.Fatalf("lists = %v; a game without a folder has nothing to reload", gs.listNames)
	}
}

func TestWordsStatusSkippedRowsOfChosenSourceOnly(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	writeList(t, dir, "mine.csv", "kana;meaning\nあお;blue\nabc;x\n")
	writeList(t, dir, "other.csv", "kana;meaning\nいえ;house\n")
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordSource(listSourcePrefix + "mine")
	gs := NewGameState(st)
	gs.mu.Lock()
	mine := gs.wordsStatus()
	gs.wordSource = listSourcePrefix + "other"
	other := gs.wordsStatus()
	gs.wordSource = jlptSource
	jlpt := gs.wordsStatus()
	gs.mu.Unlock()
	if mine.Skipped != 1 || !strings.HasSuffix(formatWordsLine(mine), " · 1 row skipped") {
		t.Fatalf("mine: skipped %d, line %q", mine.Skipped, formatWordsLine(mine))
	}
	if other.Skipped != 0 || jlpt.Skipped != 0 {
		t.Fatalf("other %d, JLPT %d; only the chosen source's skipped rows count", other.Skipped, jlpt.Skipped)
	}
}

// listGame is a game on a fresh store with the given lists in its isolated
// word list folder, the vowels selected (no intro) and the given source.
func listGame(t *testing.T, source string, files map[string]string) (*GameState, string) {
	t.Helper()
	test.NewApp()
	dir := isolateWordsDir(t)
	for name, content := range files {
		writeList(t, dir, name, content)
	}
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordSource(source)
	return NewGameState(st), dir
}

const lessonList = "kana;meaning;lesson\nあお;blue;1\nいえ;house;2\n"

func TestDeleteListKeepsProgressAndFallsBackToJLPT(t *testing.T) {
	gs, dir := listGame(t, listSourcePrefix+"mine", map[string]string{"mine.csv": lessonList})
	_ = gs.store.SaveLessonFilter("mine", []string{"2"})
	gs.ReloadLists() // pick up the stored filter
	gs.mu.Lock()
	setWordLevel(gs, "あお", 3)
	gs.introduced["あお"] = true
	gs.newestPackage = []string{"あお"}
	gs.mu.Unlock()

	if err := gs.DeleteList("mine"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); !os.IsNotExist(err) {
		t.Fatal("the file must be gone")
	}
	if len(gs.listNames) != 0 || len(gs.ListInfos()) != 0 {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if stored, _ := gs.store.WordSource(); gs.wordSource != jlptSource || stored != jlptSource {
		t.Fatalf("source %q, stored %q; want JLPT", gs.wordSource, stored)
	}
	if l, _ := gs.store.LessonFilter("mine"); l != nil || gs.lessonFilters["mine"] != nil {
		t.Fatalf("the lesson filter must be removed: stored %v, game %v", l, gs.lessonFilters["mine"])
	}
	if gs.wordProgress["あお"].Level != 3 || !gs.introduced["あお"] {
		t.Fatal("progress on the list's words must be kept")
	}
	if gs.newestPackage != nil {
		t.Fatalf("newest package %v; a source change clears it", gs.newestPackage)
	}
}

func TestDeleteListThatIsNotTheSource(t *testing.T) {
	gs, _ := listGame(t, listSourcePrefix+"a", map[string]string{"a.csv": lessonList, "b.csv": lessonList})
	gs.newestPackage = []string{"あお"}
	if err := gs.DeleteList("b"); err != nil {
		t.Fatal(err)
	}
	if gs.wordSource != listSourcePrefix+"a" || !equalIDs(gs.listNames, []string{"a"}) || len(gs.newestPackage) != 1 {
		t.Fatalf("source %q, lists %v, newest %v", gs.wordSource, gs.listNames, gs.newestPackage)
	}
}

func TestRenameListCarriesFilterSourceAndNewestPackage(t *testing.T) {
	gs, dir := listGame(t, listSourcePrefix+"mine", map[string]string{"mine.csv": lessonList})
	_ = gs.store.SaveLessonFilter("mine", []string{"2"})
	gs.ReloadLists()
	gs.newestPackage = []string{"いえ"}

	to, err := gs.RenameList("mine", " ours ")
	if err != nil || to != "ours" {
		t.Fatalf("to %q, err %v", to, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ours.csv")); err != nil {
		t.Fatal("the file must be ours.csv")
	}
	if !equalIDs(gs.listNames, []string{"ours"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if stored, _ := gs.store.WordSource(); gs.wordSource != listSourcePrefix+"ours" || stored != gs.wordSource {
		t.Fatalf("source %q, stored %q; want list:ours", gs.wordSource, stored)
	}
	if l, _ := gs.store.LessonFilter("ours"); !equalIDs(l, []string{"2"}) || !equalIDs(gs.lessonFilters["ours"], []string{"2"}) {
		t.Fatalf("filter stored %v, game %v; want [2] under the new name", l, gs.lessonFilters["ours"])
	}
	if l, _ := gs.store.LessonFilter("mine"); l != nil {
		t.Fatalf("old filter %v left behind", l)
	}
	if !equalIDs(gs.newestPackage, []string{"いえ"}) {
		t.Fatalf("newest package %v; a rename keeps it", gs.newestPackage)
	}
	gs.mu.Lock()
	src, found := gs.source()
	gs.mu.Unlock()
	if !found || !equalIDs(src.Lessons, []string{"2"}) {
		t.Fatalf("source %+v, found %v", src, found)
	}
}

func TestRenameListRejected(t *testing.T) {
	gs, _ := listGame(t, jlptSource, map[string]string{"mine.csv": lessonList, "other.csv": lessonList})
	if _, err := gs.RenameList("mine", "OTHER"); !errors.Is(err, vocab.ErrNameTaken) {
		t.Fatalf("err = %v, want ErrNameTaken", err)
	}
	if err := gs.CheckListName("mine", "jmdict"); !errors.Is(err, vocab.ErrNameReserved) {
		t.Fatalf("CheckListName = %v, want ErrNameReserved", err)
	}
	if err := gs.CheckListName("mine", "Mine"); err != nil {
		t.Fatalf("a change of case only: %v", err)
	}
	if !equalIDs(gs.listNames, []string{"mine", "other"}) {
		t.Fatalf("lists = %v, want unchanged", gs.listNames)
	}
	if _, err := gs.RenameList("gone", "x"); err == nil {
		t.Fatal("renaming an unknown list must fail")
	}
}
