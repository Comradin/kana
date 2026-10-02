package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

// TestEntryHasFocusAfterBuild lets a returning learner start typing at once,
// without reaching for the mouse while the first tiles appear.
func TestEntryHasFocusAfterBuild(t *testing.T) {
	a := test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})

	w := buildWindow(a, st)
	defer w.Close()

	if _, ok := w.Canvas().Focused().(*romajiEntry); !ok {
		t.Fatalf("focused = %T, want the romaji entry", w.Canvas().Focused())
	}
}
