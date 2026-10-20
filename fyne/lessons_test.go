package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/vocab"
)

// lw is a word of a private list in a lesson ("" = none), with its reading
// as German meaning.
func lw(list, kana, lesson string) vocab.Word {
	return vocab.Word{Kana: kana, Source: list, Lesson: lesson, Meanings: vocab.Meanings{"de": {kana}}}
}

// withList makes the private list the word source, with all its words
// introduced.
func withList(gs *GameState, list string, words ...vocab.Word) {
	gs.words = vocab.NewIndex(nil, []vocab.List{{Name: list, Words: words}})
	gs.listNames = []string{list}
	gs.wordSource = listSourcePrefix + list
	gs.introduced = map[string]bool{}
	for _, w := range words {
		gs.introduced[w.Kana] = true
	}
	gs.invalidatePlayable()
}

func TestLessonFilterLimitsFallingWords(t *testing.T) {
	for _, wordsOnly := range []bool{false, true} {
		gs := newTestState()
		gs.wordsOnly = wordsOnly
		withList(gs, "minna", lw("minna", "あい", "1"), lw("minna", "あお", "2"), lw("minna", "いえ", ""))
		gs.words = vocab.NewIndex([]vocab.Word{bw("うえ", 5, "above")},
			[]vocab.List{{Name: "minna", Words: []vocab.Word{lw("minna", "あい", "1"), lw("minna", "あお", "2"), lw("minna", "いえ", "")}}})
		gs.introduced["うえ"] = true
		learnKana(gs, "あ", "い", "う", "え", "お")

		all := []string{"あい", "あお", "いえ"}
		if !wordsOnly {
			all = []string{"あい", "あお", "いえ", "うえ"} // mixed, all lessons: every introduced word
		}
		if got := playableKana(gs); !equalIDs(got, all) {
			t.Fatalf("words-only %v, all lessons: %v, want %v", wordsOnly, got, all)
		}
		gs.lessonFilters = map[string][]string{"minna": {"2", ""}}
		gs.invalidatePlayable()
		if got := playableKana(gs); !equalIDs(got, []string{"あお", "いえ"}) {
			t.Fatalf("words-only %v, lesson 2 and no lesson: %v, want [あお いえ]", wordsOnly, got)
		}
	}
}

func TestPackagesComeFromTheChosenLessons(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	learnKana(gs, "あ", "い", "う", "え", "お")
	withList(gs, "minna", lw("minna", "あい", "1"), lw("minna", "あう", "2"), lw("minna", "あえ", "3"), lw("minna", "あお", "3"))
	gs.introduced = map[string]bool{}
	gs.lessonFilters = map[string][]string{"minna": {"3", "2"}}
	gs.maybeOfferPackage()
	if !equalIDs(gs.pendingPackage, []string{"あう"}) {
		t.Fatalf("first package = %v, want lesson 2: [あう]", gs.pendingPackage)
	}
	if got := gs.PackageIntro().Title; got != "New words · minna · lesson 2" {
		t.Fatalf("title = %q", got)
	}
	gs.FinishPackage(nil)
	setWordLevel(gs, "あう", learnedLevel)
	drainEvents(gs)
	gs.maybeOfferPackage()
	if !equalIDs(gs.pendingPackage, []string{"あえ", "あお"}) {
		t.Fatalf("second package = %v, want lesson 3", gs.pendingPackage)
	}
}

func TestLessonFilterChangeStartsThePackageRhythmAfresh(t *testing.T) {
	gs := newTestState()
	withList(gs, "minna", lw("minna", "あい", "1"), lw("minna", "あお", "2"))
	gs.newestPackage = []string{"あい"}
	gs.applyWordSettings(wordSettings{source: listSourcePrefix + "minna", level: 5, packageSize: 5})
	if !equalIDs(gs.newestPackage, []string{"あい"}) {
		t.Fatal("an unchanged filter keeps the newest package")
	}
	gs.applyWordSettings(wordSettings{source: listSourcePrefix + "minna", level: 5, packageSize: 5, lessons: []string{"2"}})
	if gs.newestPackage != nil || !slices.Equal(gs.lessonFilters["minna"], []string{"2"}) {
		t.Fatalf("newest %v, filter %v", gs.newestPackage, gs.lessonFilters["minna"])
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5})
	if !slices.Equal(gs.lessonFilters["minna"], []string{"2"}) {
		t.Fatal("choosing JLPT must not touch the list's filter")
	}
}

func TestLessonFilterIsStoredPerList(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"minna.csv": "kana;meaning;lesson\nあい;love;1\nあお;blue;2\n",
		"other.csv": "kana;meaning;lesson\nいえ;house;1\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordSource("list:minna")
	gs := NewGameState(st)
	gs.applyWordSettings(wordSettings{source: "list:minna", level: 5, packageSize: 5, lessons: []string{"2"}})
	if l, _ := st.LessonFilter("minna"); !slices.Equal(l, []string{"2"}) {
		t.Fatalf("stored minna = %v", l)
	}
	if l, _ := st.LessonFilter("other"); l != nil {
		t.Fatalf("stored other = %v, want none", l)
	}
	gs = NewGameState(st)
	gs.mu.Lock()
	src, _ := gs.source()
	gs.wordSource = "list:other"
	other, _ := gs.source()
	gs.mu.Unlock()
	if !slices.Equal(src.Lessons, []string{"2"}) || other.Lessons != nil {
		t.Fatalf("loaded: minna %v, other %v", src.Lessons, other.Lessons)
	}
}

func TestSourceUsesOnlyLessonsThatStillExist(t *testing.T) {
	cases := []struct {
		name   string
		words  []vocab.Word
		stored []string
		want   []string
	}{
		{"lesson column removed", []vocab.Word{lw("minna", "あい", ""), lw("minna", "あお", "")}, []string{"2"}, nil},
		{"one lesson gone", []vocab.Word{lw("minna", "あい", "1"), lw("minna", "あお", "2")}, []string{"9", "2"}, []string{"2"}},
		{"no lesson still exists", []vocab.Word{lw("minna", "あい", "1"), lw("minna", "あお", "")}, []string{""}, []string{""}},
		{"no lesson gone", []vocab.Word{lw("minna", "あい", "1"), lw("minna", "あお", "2")}, []string{""}, nil},
	}
	for _, c := range cases {
		gs := newTestState()
		gs.wordsOnly = true
		withList(gs, "minna", c.words...)
		gs.lessonFilters = map[string][]string{"minna": c.stored}
		gs.mu.Lock()
		src, ok := gs.source()
		gs.mu.Unlock()
		if !ok || !slices.Equal(src.Lessons, c.want) {
			t.Errorf("%s: lessons %v, want %v", c.name, src.Lessons, c.want)
		}
	}

	gs := newTestState()
	gs.wordsOnly = true
	withList(gs, "minna", lw("minna", "あい", ""), lw("minna", "あお", ""))
	gs.lessonFilters = map[string][]string{"minna": {"2"}}
	if got := playableKana(gs); !equalIDs(got, []string{"あい", "あお"}) {
		t.Fatalf("a filter of lessons gone falls back to all lessons: %v", got)
	}
}

func TestStaleOrReorderedFilterKeepsThePackageRhythm(t *testing.T) {
	for _, c := range []struct {
		name          string
		stored, given []string
	}{
		{"stale lesson", []string{"2", "9"}, []string{"2"}},
		{"other order", []string{"2", "1"}, []string{"1", "2"}},
		{"all lessons gone", []string{"9"}, nil},
	} {
		gs := newTestState()
		withList(gs, "minna", lw("minna", "あい", "1"), lw("minna", "あお", "2"), lw("minna", "あう", "3"))
		gs.lessonFilters = map[string][]string{"minna": c.stored}
		gs.newestPackage = []string{"あい"}
		gs.applyWordSettings(wordSettings{source: listSourcePrefix + "minna", level: 5, packageSize: 5, lessons: c.given})
		if !equalIDs(gs.newestPackage, []string{"あい"}) {
			t.Errorf("%s: the same effective filter must keep the newest package", c.name)
		}
		if !slices.Equal(gs.lessonFilters["minna"], c.given) {
			t.Errorf("%s: filter %v, want what the dialog gave: %v", c.name, gs.lessonFilters["minna"], c.given)
		}
	}
}
