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
		// かんきょう and はし are bundled and in minna, so they stay playable
		// without bundled words, at any level.
		{"mixed without bundled", Filter{MaxLevel: 5, NoBundled: true}, all, []string{"かんきょう", "せんせい", "はし"}},
		{"jmdict without bundled", Filter{MaxLevel: 5, List: BundledSource, NoBundled: true}, all, nil},
		{"minna without bundled", Filter{MaxLevel: 5, List: "minna", NoBundled: true}, all, []string{"かんきょう", "せんせい", "はし"}},
	}
	for _, c := range cases {
		got := kanaOf(Playable(ix.Entries(), c.known, c.filter))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestWithListsMatchesNewIndex(t *testing.T) {
	bundled := []Word{
		word("ねこ", 5, BundledSource, "", "en", "cat"),
		word("はし", 5, BundledSource, "", "en", "bridge"),
		word("はし", 4, BundledSource, "", "en", "chopsticks"),
		word("きって", 4, BundledSource, "", "en", "stamp"),
	}
	lists := []List{
		{Name: "minna", Words: []Word{
			word("はし", 0, "minna", "3", "de", "Essstäbchen"),
			word("はし", 0, "minna", "4", "en", "bridge"),
			word("せんせい", 0, "minna", "1", "de", "Lehrer"),
			word("abc", 0, "minna", "1", "de", "invalid reading"),
		}},
		{Name: "extra", Words: []Word{
			word("ねこ", 0, "extra", "", "en", "kitty"),
			word("あお", 0, "extra", "", "en", "blue"),
		}},
	}
	base := NewIndex(bundled, nil)
	before := NewIndex(bundled, nil)

	merged := base.WithLists(lists)
	want := NewIndex(bundled, lists)
	if !reflect.DeepEqual(merged.Entries(), want.Entries()) {
		t.Fatalf("WithLists entries = %+v\nwant %+v", merged.Entries(), want.Entries())
	}
	for _, e := range want.Entries() {
		got, ok := merged.Lookup(e.Kana)
		if !ok || !reflect.DeepEqual(got, e) {
			t.Fatalf("Lookup(%s) = %+v, %v; want %+v", e.Kana, got, ok, e)
		}
	}
	if !reflect.DeepEqual(base.Entries(), before.Entries()) {
		t.Fatalf("WithLists changed the base index: %+v\nwant %+v", base.Entries(), before.Entries())
	}
	if _, ok := base.Lookup("せんせい"); ok {
		t.Fatal("a private reading leaked into the base index")
	}

	if empty := base.WithLists(nil); !reflect.DeepEqual(empty.Entries(), base.Entries()) {
		t.Fatal("WithLists(nil) must give the base entries")
	}
}

// TestWithListsDoesNotShareMeanings merges two lists into one base whose
// meanings slice has spare capacity: an append in place would let the
// second merge overwrite the first one's meaning.
func TestWithListsDoesNotShareMeanings(t *testing.T) {
	base := NewIndex([]Word{word("いぬ", 5, BundledSource, "", "en", "dog")}, nil)
	// Give the base meanings spare capacity, so an append that does not
	// copy first would write into memory a and b share.
	en := make([]string, 3, 5)
	copy(en, []string{"dog", "hound", "spy"})
	base.entries[base.byKana["いぬ"]].Meanings["en"] = en
	if e, _ := base.Lookup("いぬ"); cap(e.Meanings["en"]) <= len(e.Meanings["en"]) {
		t.Fatal("precondition: the base meanings need spare capacity")
	}
	a := base.WithLists([]List{{Name: "a", Words: []Word{word("いぬ", 0, "a", "1", "en", "doggy")}}})
	b := base.WithLists([]List{{Name: "b", Words: []Word{word("いぬ", 0, "b", "2", "en", "pup")}}})
	ea, _ := a.Lookup("いぬ")
	eb, _ := b.Lookup("いぬ")
	if got := ea.Meanings["en"]; !reflect.DeepEqual(got, []string{"dog", "hound", "spy", "doggy"}) {
		t.Fatalf("a = %v", got)
	}
	if got := eb.Meanings["en"]; !reflect.DeepEqual(got, []string{"dog", "hound", "spy", "pup"}) {
		t.Fatalf("b = %v", got)
	}
	if e, _ := base.Lookup("いぬ"); len(e.Meanings["en"]) != 3 || len(e.Lists) != 0 {
		t.Fatalf("base changed: %+v", e)
	}
	if _, ok := ea.Lists["b"]; ok {
		t.Fatal("list b leaked into a's entry")
	}
}
