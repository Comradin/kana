package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestReadingPracticeIsStored(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	if gs.readingPractice {
		t.Fatal("reading practice must be off by default")
	}
	ws := wordSettings{source: gs.wordSource, level: gs.wordLevel, packageSize: gs.packageSize, readingPractice: true}
	gs.applyWordSettings(ws)
	if on, _ := st.ReadingPractice(); !on || !gs.readingPractice {
		t.Fatalf("stored %v, state %v", on, gs.readingPractice)
	}
	if !NewGameState(st).readingPractice {
		t.Fatal("reading practice not loaded")
	}
	ws.readingPractice = false
	gs.applyWordSettings(ws)
	if on, _ := st.ReadingPractice(); on || gs.readingPractice {
		t.Fatalf("off: stored %v, state %v", on, gs.readingPractice)
	}
}
