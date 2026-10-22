package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/vocab"
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

// TestMissedLines shows "None!" only when neither kana nor words were
// missed, and leaves the characters line out when only words were missed.
func TestMissedLines(t *testing.T) {
	ka := kanacore.Kana{Char: "か", Romaji: "ka"}
	cases := []struct {
		name  string
		kanas []kanacore.Kana
		words []vocab.Entry
		want  []string
	}{
		{"nothing", nil, nil, []string{"Characters missed:", "None!"}},
		{"kana only", []kanacore.Kana{ka, ka}, nil, []string{"Characters missed:", "か (ka)"}},
		{"words only", nil, []vocab.Entry{{Kana: "あい"}}, []string{"Words missed:", "あい (ai)"}},
		{"both", []kanacore.Kana{ka}, []vocab.Entry{{Kana: "あい"}}, []string{"Characters missed:", "か (ka)", "Words missed:", "あい (ai)"}},
		{"shown form", nil, []vocab.Entry{{Kana: "さん", Display: "～さん"}}, []string{"Words missed:", "～さん (san)"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := missedLines(c.kanas, c.words); !equalIDs(got, c.want) {
				t.Fatalf("missedLines = %q, want %q", got, c.want)
			}
		})
	}
}
