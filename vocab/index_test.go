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
	if _, ok := ea.ListPos["b"]; ok {
		t.Fatal("list b's position leaked into a's entry")
	}
	if e, _ := base.Lookup("いぬ"); len(e.ListPos) != 0 {
		t.Fatalf("base list positions changed: %v", e.ListPos)
	}
}

func TestEntryListPosIsTheFirstRow(t *testing.T) {
	lists := []List{
		{Name: "minna", Words: []Word{
			word("せんせい", 0, "minna", "1", "de", "Lehrer"),
			word("ねこ", 0, "minna", "1", "de", "Katze"),
			word("せんせい", 0, "minna", "2", "de", "Meister"),
		}},
		{Name: "extra", Words: []Word{
			word("ねこ", 0, "extra", "", "en", "cat"),
		}},
	}
	bundled := []Word{word("ねこ", 5, BundledSource, "", "en", "cat")}
	for name, ix := range map[string]*Index{
		"NewIndex":  NewIndex(bundled, lists),
		"WithLists": NewIndex(bundled, nil).WithLists(lists),
	} {
		s, _ := ix.Lookup("せんせい")
		n, _ := ix.Lookup("ねこ")
		if !reflect.DeepEqual(s.ListPos, map[string]int{"minna": 0}) {
			t.Errorf("%s: せんせい ListPos = %v, want minna 0 (its first row)", name, s.ListPos)
		}
		if !reflect.DeepEqual(n.ListPos, map[string]int{"minna": 1, "extra": 0}) {
			t.Errorf("%s: ねこ ListPos = %v", name, n.ListPos)
		}
	}
}

func TestEntryRankFromBundledWordsOnly(t *testing.T) {
	a := word("はし", 5, BundledSource, "", "en", "bridge")
	a.Rank = 49
	b := word("はし", 4, BundledSource, "", "en", "chopsticks")
	b.Rank = 5
	c := word("ねこ", 5, BundledSource, "", "en", "cat") // no rank
	d := word("ねこ", 5, BundledSource, "", "en", "kitty")
	d.Rank = 60
	own := word("はし", 0, "mine", "", "en", "edge")
	own.Rank = 1 // a private word's rank is ignored
	ix := NewIndex([]Word{a, b, c, d}, []List{{Name: "mine", Words: []Word{own, word("いぬ", 0, "mine", "", "en", "dog")}}})
	if e, _ := ix.Lookup("はし"); e.Rank != 5 {
		t.Errorf("はし rank = %d, want the best bundled rank 5", e.Rank)
	}
	if e, _ := ix.Lookup("ねこ"); e.Rank != 60 {
		t.Errorf("ねこ rank = %d, want 60 (a known rank beats an unknown one)", e.Rank)
	}
	if e, _ := ix.Lookup("いぬ"); e.Rank != 0 {
		t.Errorf("private-only いぬ rank = %d, want 0", e.Rank)
	}
}

func TestIndexLessons(t *testing.T) {
	ix := NewIndex(nil, []List{
		lessonList([2]string{"いぬ", "10"}, [2]string{"ねこ", "2"}, [2]string{"あめ", ""}, [2]string{"き", "1"}),
		{Name: "named", Words: []Word{
			word("いぬ", 0, "named", "Tiere", "de", "Hund"),
			word("みず", 0, "named", "Wetter", "de", "Wasser"),
		}},
	})
	if lessons, none := ix.Lessons("minna"); !reflect.DeepEqual(lessons, []string{"1", "2", "10"}) || !none {
		t.Fatalf("minna: %v, no lesson %v; want [1 2 10] in number order and words without a lesson", lessons, none)
	}
	if lessons, none := ix.Lessons("named"); !reflect.DeepEqual(lessons, []string{"Tiere", "Wetter"}) || none {
		t.Fatalf("named: %v, no lesson %v", lessons, none)
	}
	if lessons, none := ix.Lessons("gone"); lessons != nil || none {
		t.Fatalf("unknown list: %v, %v", lessons, none)
	}
}

func TestIndexLanguages(t *testing.T) {
	if got := testIndex().Languages(); !reflect.DeepEqual(got, []string{"de", "en"}) {
		t.Fatalf("Languages = %v, want [de en]", got)
	}
	ix := NewIndex([]Word{word("ねこ", 5, BundledSource, "", "", "cat")}, nil)
	if got := ix.Languages(); len(got) != 0 {
		t.Fatalf("meanings without a language count as none, got %v", got)
	}
}

func TestIndexShownForm(t *testing.T) {
	bundled := []Word{word("さん", 5, BundledSource, "", "en", "Mr./Ms.")}
	if e, _ := NewIndex(bundled, nil).Lookup("さん"); e.Shown() != "さん" {
		t.Fatalf("a bundled word alone is shown as %q, want its reading", e.Shown())
	}
	san := Word{Kana: "さん", Display: "～さん", Meanings: Meanings{}}
	plain := Word{Kana: "さん", Meanings: Meanings{}}
	cases := []struct {
		name  string
		lists []List
		want  string
	}{
		{"a list over the bundled word", []List{{Name: "a", Words: []Word{san}}}, "～さん"},
		{"the first list wins", []List{{Name: "a", Words: []Word{san}}, {Name: "b", Words: []Word{plain}}}, "～さん"},
		{"a plain first list wins too", []List{{Name: "a", Words: []Word{plain}}, {Name: "b", Words: []Word{san}}}, "さん"},
		{"the first row of a list wins", []List{{Name: "a", Words: []Word{plain, san}}}, "さん"},
	}
	for _, c := range cases {
		for how, ix := range map[string]*Index{
			"NewIndex":  NewIndex(bundled, c.lists),
			"WithLists": NewIndex(bundled, nil).WithLists(c.lists),
		} {
			e, ok := ix.Lookup("さん")
			if !ok || !e.Bundled || e.Shown() != c.want {
				t.Errorf("%s, %s: shown %q, bundled %v; want %q, one entry", c.name, how, e.Shown(), e.Bundled, c.want)
			}
		}
	}
}

func TestIndexMergesAlts(t *testing.T) {
	a := List{Name: "a", Words: []Word{{Kana: "アメリカ", Alts: []string{"あめりか"}, Meanings: Meanings{}}}}
	b := List{Name: "b", Words: []Word{{Kana: "アメリカ", Alts: []string{"めりけん", "あめりか"}, Meanings: Meanings{}}}}
	want := []string{"あめりか", "めりけん"}
	if e, _ := NewIndex(nil, []List{a, b}).Lookup("アメリカ"); !reflect.DeepEqual(e.Alts, want) {
		t.Fatalf("NewIndex alts = %v, want %v", e.Alts, want)
	}
	base := NewIndex(nil, []List{a})
	if e, _ := base.WithLists([]List{b}).Lookup("アメリカ"); !reflect.DeepEqual(e.Alts, want) {
		t.Fatalf("WithLists alts = %v, want %v", e.Alts, want)
	}
	if e, _ := base.Lookup("アメリカ"); !reflect.DeepEqual(e.Alts, []string{"あめりか"}) {
		t.Fatalf("base alts = %v; WithLists must not change the base index", e.Alts)
	}

	// Three alts appended one by one leave the base entry's Alts spare
	// capacity: two indexes from the same base must not share it.
	three := List{Name: "a", Words: []Word{{Kana: "アメリカ", Alts: []string{"あめりか", "めりけん", "べいこく"}, Meanings: Meanings{}}}}
	base = NewIndex(nil, []List{three})
	x := base.WithLists([]List{{Name: "x", Words: []Word{{Kana: "アメリカ", Alts: []string{"あめりかがっしゅうこく"}, Meanings: Meanings{}}}}})
	base.WithLists([]List{{Name: "y", Words: []Word{{Kana: "アメリカ", Alts: []string{"あめりけん"}, Meanings: Meanings{}}}}})
	if e, _ := x.Lookup("アメリカ"); !reflect.DeepEqual(e.Alts, []string{"あめりか", "めりけん", "べいこく", "あめりかがっしゅうこく"}) {
		t.Fatalf("alts = %v; another WithLists of the same base changed them", e.Alts)
	}
	if e, _ := base.Lookup("アメリカ"); !reflect.DeepEqual(e.Alts, three.Words[0].Alts) {
		t.Fatalf("base alts = %v; WithLists must not change the base index", e.Alts)
	}
}

func TestEntryReadingMatches(t *testing.T) {
	e := Entry{Kana: "どうぞよろしくおねがいします", Alts: []string{"よろしく"}}
	for in, want := range map[string]bool{
		"douzoyoroshikuonegaishimasu": true,
		"yoroshiku":                   true,
		"douzoyoroshiku":              false, // no mix of the full and the short form
		"douzo yoroshiku":             false,
		"":                            false,
	} {
		if got := e.ReadingMatches(in); got != want {
			t.Errorf("ReadingMatches(%q) = %v, want %v", in, got, want)
		}
	}
	v := Entry{Kana: "ローソク", Alts: []string{"ろうそく"}}
	if !v.ReadingMatches("roosoku") || !v.ReadingMatches("rousoku") {
		t.Fatal("every variant's reading must match")
	}
}

func TestIndexMergesPhrase(t *testing.T) {
	phrase := Word{Kana: "さん", Display: "～さん", Phrase: true, Meanings: Meanings{}}
	plain := Word{Kana: "さん", Meanings: Meanings{}}
	for name, lists := range map[string][]List{
		"phrase first": {{Name: "a", Words: []Word{phrase}}, {Name: "b", Words: []Word{plain}}},
		"phrase later": {{Name: "a", Words: []Word{plain}}, {Name: "b", Words: []Word{phrase}}},
	} {
		if e, _ := NewIndex(nil, lists).Lookup("さん"); !e.Phrase {
			t.Errorf("%s: an entry is a phrase if any of its rows is", name)
		}
		base := NewIndex(nil, lists[:1])
		if e, _ := base.WithLists(lists[1:]).Lookup("さん"); !e.Phrase {
			t.Errorf("%s, WithLists: an entry is a phrase if any of its rows is", name)
		}
	}
	if e, _ := NewIndex(nil, []List{{Name: "a", Words: []Word{plain}}}).Lookup("さん"); e.Phrase {
		t.Error("a plain row is no phrase")
	}
}

func TestEntryReadingMatchesParticles(t *testing.T) {
	phrase := Entry{Kana: "こちらはさんです", Phrase: true}
	for _, in := range []string{"kochirawasandesu", "kochirahasandesu"} {
		if !phrase.ReadingMatches(in) {
			t.Errorf("phrase: ReadingMatches(%q) = false, want true", in)
		}
	}
	short := Entry{Kana: "にほんへいきます", Alts: []string{"にほんへ"}, Phrase: true}
	if !short.ReadingMatches("nihone") {
		t.Error("the particle reading applies to other readings too")
	}
	if (Entry{Kana: "はな"}).ReadingMatches("wana") {
		t.Error("outside a phrase は is ha")
	}
}
