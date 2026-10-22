package vocab

import (
	"reflect"
	"testing"
)

func TestSplitMeanings(t *testing.T) {
	cases := map[string][]string{
		"Lehrer; Lehrerin":   {"Lehrer", "Lehrerin"},
		" teacher ":          {"teacher"},
		"a;;a; b ;":          {"a", "b"},
		"":                   nil,
		" ; ":                nil,
		"Brücke;Essstäbchen": {"Brücke", "Essstäbchen"},
	}
	for in, want := range cases {
		if got := SplitMeanings(in); !reflect.DeepEqual(got, want) {
			t.Errorf("SplitMeanings(%q) = %#v, want %#v", in, got, want)
		}
	}
}

func TestMeaningsAddAndFor(t *testing.T) {
	m := Meanings{}
	m.Add("de", "Lehrer", "Lehrerin")
	m.Add("de", "Lehrer", "Meister")
	m.Add("en", "teacher")
	if got := m["de"]; !reflect.DeepEqual(got, []string{"Lehrer", "Lehrerin", "Meister"}) {
		t.Fatalf("de = %v", got)
	}
	if got := m.For("de"); got[0] != "Lehrer" {
		t.Fatalf("For(de) = %v", got)
	}
	if got := m.For("fr"); !reflect.DeepEqual(got, []string{"teacher"}) {
		t.Fatalf("For(fr) falls back to en, got %v", got)
	}
	g := Meanings{}
	g.Add("", "Lehrer")
	g.Add("en", "teacher")
	if got := g.For("de"); !reflect.DeepEqual(got, []string{"Lehrer"}) {
		t.Fatalf("generic meaning wins over en for an unlisted language, got %v", got)
	}
	only := Meanings{}
	only.Add("sv", "lärare")
	if got := only.For("de"); !reflect.DeepEqual(got, []string{"lärare"}) {
		t.Fatalf("any language as last resort, got %v", got)
	}
	if got := (Meanings{}).For("de"); got != nil {
		t.Fatalf("empty meanings = %v", got)
	}
	if got := m.Display("de"); got != "Lehrer; Lehrerin; Meister" {
		t.Fatalf("Display = %q", got)
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]int{"5": 5, "N5": 5, "n4": 4, " 3 ": 3, "": 0, "6": 0, "x": 0, "N0": 0}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestWordShown(t *testing.T) {
	if got := (Word{Kana: "さん"}).Shown(); got != "さん" {
		t.Fatalf("plain word shown as %q", got)
	}
	if got := (Word{Kana: "さん", Display: "～さん"}).Shown(); got != "～さん" {
		t.Fatalf("word with notation shown as %q", got)
	}
}
