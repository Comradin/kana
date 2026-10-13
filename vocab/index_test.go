package vocab

import (
	"reflect"
	"testing"
)

func word(kana string, level int, source, lesson string, lang, meaning string) Word {
	m := Meanings{}
	m.Add(lang, meaning)
	return Word{Kana: kana, JLPT: level, Source: source, Lesson: lesson, Meanings: m}
}

func testIndex() *Index {
	bundled := []Word{
		word("ねこ", 5, BundledSource, "", "en", "cat"),
		word("はし", 5, BundledSource, "", "en", "bridge"),
		word("きって", 4, BundledSource, "", "en", "stamp"),
		word("かんきょう", 2, BundledSource, "", "en", "environment"),
	}
	lists := []List{
		{Name: "minna", Words: []Word{
			word("はし", 0, "minna", "3", "de", "Essstäbchen"),
			word("せんせい", 0, "minna", "1", "de", "Lehrer"),
			word("かんきょう", 0, "minna", "2", "de", "Umwelt"),
		}},
	}
	return NewIndex(bundled, lists)
}

func kanaOf(es []Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Kana)
	}
	return out
}

func TestNewIndexMerges(t *testing.T) {
	ix := testIndex()
	e, ok := ix.Lookup("はし")
	if !ok || !e.Bundled || e.JLPT != 5 {
		t.Fatalf("はし = %+v", e)
	}
	if !reflect.DeepEqual(e.Meanings["en"], []string{"bridge"}) || !reflect.DeepEqual(e.Meanings["de"], []string{"Essstäbchen"}) {
		t.Fatalf("meanings merged across sources: %v", e.Meanings)
	}
	if !reflect.DeepEqual(e.Lists["minna"], []string{"3"}) {
		t.Fatalf("lists = %v", e.Lists)
	}
	if !reflect.DeepEqual(e.Units, []string{"は", "し"}) {
		t.Fatalf("units = %v", e.Units)
	}
	s, _ := ix.Lookup("せんせい")
	if s.Bundled || s.JLPT != 0 {
		t.Fatalf("private-only entry = %+v", s)
	}
	if len(ix.Entries()) != 5 {
		t.Fatalf("entries = %v", kanaOf(ix.Entries()))
	}
}

func TestPlayable(t *testing.T) {
	ix := testIndex()
	all := func(string) bool { return true }
	cases := []struct {
		name   string
		filter Filter
		known  func(string) bool
		want   []string
	}{
		{"mixed N5", Filter{MaxLevel: 5}, all, []string{"かんきょう", "せんせい", "ねこ", "はし"}},
		{"mixed N4", Filter{MaxLevel: 4}, all, []string{"かんきょう", "きって", "せんせい", "ねこ", "はし"}},
		{"jmdict N5", Filter{MaxLevel: 5, List: BundledSource}, all, []string{"ねこ", "はし"}},
		{"jmdict N2", Filter{MaxLevel: 2, List: BundledSource}, all, []string{"かんきょう", "きって", "ねこ", "はし"}},
		{"minna", Filter{MaxLevel: 5, List: "minna"}, all, []string{"かんきょう", "せんせい", "はし"}},
		{"minna lesson 1", Filter{MaxLevel: 5, List: "minna", Lesson: "1"}, all, []string{"せんせい"}},
		{"unknown list", Filter{MaxLevel: 5, List: "nope"}, all, nil},
		{"known units", Filter{MaxLevel: 5}, func(u string) bool { return u == "ね" || u == "こ" }, []string{"ねこ"}},
	}
	for _, c := range cases {
		got := kanaOf(Playable(ix.Entries(), c.known, c.filter))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
