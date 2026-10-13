package vocab

import (
	"testing"

	"kana/kanacore"
)

func TestBundled(t *testing.T) {
	words, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if len(words) < 5000 {
		t.Fatalf("bundled words = %d, want > 5000", len(words))
	}
	levels := map[int]int{}
	for _, w := range words {
		if _, ok := kanacore.KanaUnits(w.Kana); !ok {
			t.Fatalf("%s is not typeable", w.Kana)
		}
		if w.Source != BundledSource || w.JLPT < 1 || w.JLPT > 5 {
			t.Fatalf("bad word %+v", w)
		}
		levels[w.JLPT]++
	}
	for l := 1; l <= 5; l++ {
		if levels[l] == 0 {
			t.Fatalf("no words for N%d", l)
		}
	}
	ix := NewIndex(words, nil)
	s, ok := ix.Lookup("せんせい")
	if !ok || s.JLPT != 5 || !contains(s.Meanings["en"], "teacher") {
		t.Fatalf("せんせい = %+v", s)
	}
	if len(s.Meanings["de"]) == 0 {
		t.Fatalf("せんせい has no German meaning: %v", s.Meanings)
	}
}
