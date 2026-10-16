package vocab

import (
	"reflect"
	"testing"
)

func ranked(kana string, level, rank int) Word {
	w := word(kana, level, BundledSource, "", "en", kana)
	w.Rank = rank
	return w
}

func knowAll(string) bool { return true }

func TestNextPackageJLPTOrder(t *testing.T) {
	ix := NewIndex([]Word{
		ranked("みず", 5, 1),
		ranked("あめ", 5, 49),
		ranked("いぬ", 5, 49),
		ranked("いぬごや", 5, 49), // more units than いぬ
		ranked("き", 5, 0),     // unknown rank sorts as 99, after 60
		ranked("ねこ", 5, 60),
		ranked("あう", 4, 1), // N4 comes after every N5 word
		ranked("かんきょう", 2, 1),
	}, nil)
	got := kanaOf(NextPackage(ix, Source{Level: 4}, nil, 10, knowAll))
	want := []string{"みず", "あめ", "いぬ", "いぬごや", "ねこ", "き", "あう"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if got := kanaOf(NextPackage(ix, Source{Level: 5}, nil, 10, knowAll)); len(got) != 6 {
		t.Fatalf("N5 only = %v, want the six N5 words", got)
	}
	if got := kanaOf(NextPackage(ix, Source{Level: 4}, nil, 3, knowAll)); !reflect.DeepEqual(got, want[:3]) {
		t.Fatalf("n = 3: %v", got)
	}
}

func TestNextPackageSkipsIntroducedAndUnplayable(t *testing.T) {
	ix := NewIndex([]Word{ranked("あい", 5, 1), ranked("かお", 5, 2), ranked("あお", 5, 3), ranked("いえ", 5, 4)}, nil)
	introduced := map[string]bool{"あい": true}
	known := func(u string) bool { return u != "か" } // かお is not playable yet
	got := kanaOf(NextPackage(ix, Source{Level: 5}, introduced, 5, known))
	if !reflect.DeepEqual(got, []string{"あお", "いえ"}) {
		t.Fatalf("package = %v, want [あお いえ]: introduced and unplayable words skipped, a partial package", got)
	}
	if got := NextPackage(ix, Source{Level: 5}, map[string]bool{"あい": true, "かお": true, "あお": true, "いえ": true}, 5, knowAll); got != nil {
		t.Fatalf("everything introduced: %v, want none", kanaOf(got))
	}
	if got := NextPackage(ix, Source{Level: 5}, nil, 0, knowAll); got != nil {
		t.Fatalf("n = 0: %v", kanaOf(got))
	}
	if got := NextPackage(ix, Source{List: "nope"}, nil, 5, knowAll); got != nil {
		t.Fatalf("unknown list: %v", kanaOf(got))
	}
}

func TestNextPackageListOrder(t *testing.T) {
	list := func(rows ...[2]string) List {
		l := List{Name: "minna"}
		for _, r := range rows {
			l.Words = append(l.Words, word(r[0], 0, "minna", r[1], "de", r[0]))
		}
		return l
	}
	cases := []struct {
		name string
		list List
		want []string
	}{
		{"numeric lessons", list(
			[2]string{"いぬ", "10"}, [2]string{"ねこ", "2"}, [2]string{"あめ", ""},
			[2]string{"みず", "2"}, [2]string{"き", "1"},
		), []string{"き", "ねこ", "みず", "いぬ", "あめ"}},
		{"named lessons by first appearance", list(
			[2]string{"いぬ", "Tiere"}, [2]string{"あめ", ""}, [2]string{"みず", "Wetter"},
			[2]string{"ねこ", "Tiere"}, [2]string{"き", "1"},
		), []string{"いぬ", "ねこ", "みず", "き", "あめ"}},
		{"entry in two lessons counts for the first", list(
			[2]string{"いぬ", "3"}, [2]string{"ねこ", "2"}, [2]string{"いぬ", "1"},
		), []string{"いぬ", "ねこ"}},
		{"no lessons: file order", list(
			[2]string{"みず", ""}, [2]string{"あめ", ""}, [2]string{"いぬ", ""},
		), []string{"みず", "あめ", "いぬ"}},
	}
	for _, c := range cases {
		for name, ix := range map[string]*Index{
			"NewIndex":  NewIndex([]Word{ranked("いぬ", 5, 1)}, []List{c.list}),
			"WithLists": NewIndex([]Word{ranked("いぬ", 5, 1)}, nil).WithLists([]List{c.list}),
		} {
			got := allPackages(ix, Source{List: "minna"}, 10, knowAll)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("%s (%s): %v, want %v", c.name, name, got, c.want)
			}
		}
	}
}

func TestLessonIsTheFirstInListOrder(t *testing.T) {
	ix := NewIndex(nil, []List{{Name: "minna", Words: []Word{
		word("いぬ", 0, "minna", "3", "de", "Hund"),
		word("ねこ", 0, "minna", "", "de", "Katze"),
		word("いぬ", 0, "minna", "1", "de", "Hund"),
	}}})
	e, _ := ix.Lookup("いぬ")
	if got := ix.Lesson(e, "minna"); got != "1" {
		t.Fatalf("Lesson(いぬ) = %q, want 1", got)
	}
	n, _ := ix.Lookup("ねこ")
	if got := ix.Lesson(n, "minna"); got != "" {
		t.Fatalf("Lesson(ねこ) = %q, want none", got)
	}
}

func TestNextPackageOrderIsCachedPerSource(t *testing.T) {
	ix := NewIndex([]Word{ranked("あい", 5, 1)}, []List{{Name: "a", Words: []Word{word("いえ", 0, "a", "", "en", "house")}}})
	NextPackage(ix, Source{Level: 5}, nil, 1, knowAll)
	NextPackage(ix, Source{Level: 3}, nil, 1, knowAll)
	NextPackage(ix, Source{List: "a"}, nil, 1, knowAll)
	if len(ix.orders) != 2 || ix.orders[""] == nil || ix.orders["a"] == nil {
		t.Fatalf("orders = %v, want one per source (JLPT for every level, and list a)", ix.orders)
	}
}

func TestSourceHas(t *testing.T) {
	ix := testIndex()
	cases := []struct {
		src  Source
		kana string
		want bool
	}{
		{Source{Level: 5}, "ねこ", true},
		{Source{Level: 5}, "きって", false}, // N4
		{Source{Level: 4}, "きって", true},
		{Source{Level: 5}, "せんせい", false}, // private only
		{Source{List: "minna"}, "せんせい", true},
		{Source{List: "minna"}, "ねこ", false},
	}
	for _, c := range cases {
		e, _ := ix.Lookup(c.kana)
		if got := c.src.Has(e); got != c.want {
			t.Errorf("%+v.Has(%s) = %v, want %v", c.src, c.kana, got, c.want)
		}
	}
}

// allPackages draws packages until the source is exhausted and returns the
// readings in the order they were introduced.
func allPackages(ix *Index, src Source, n int, known func(string) bool) []string {
	introduced := map[string]bool{}
	var out []string
	for {
		pkg := NextPackage(ix, src, introduced, n, known)
		if len(pkg) == 0 {
			return out
		}
		for _, e := range pkg {
			introduced[e.Kana] = true
			out = append(out, e.Kana)
		}
	}
}

func lessonList(rows ...[2]string) List {
	l := List{Name: "minna"}
	for _, r := range rows {
		l.Words = append(l.Words, word(r[0], 0, "minna", r[1], "de", r[0]))
	}
	return l
}

func TestNextPackageStaysWithinOneLesson(t *testing.T) {
	ix := NewIndex(nil, []List{lessonList(
		[2]string{"き", "1"}, [2]string{"め", "1"}, [2]string{"ねこ", "2"},
		[2]string{"みず", "2"}, [2]string{"あめ", ""},
	)})
	src := Source{List: "minna"}
	got := kanaOf(NextPackage(ix, src, nil, 5, knowAll))
	if !reflect.DeepEqual(got, []string{"き", "め"}) {
		t.Fatalf("first package = %v, want only lesson 1", got)
	}
	got = kanaOf(NextPackage(ix, src, map[string]bool{"き": true, "め": true}, 5, knowAll))
	if !reflect.DeepEqual(got, []string{"ねこ", "みず"}) {
		t.Fatalf("second package = %v, want only lesson 2", got)
	}
	got = kanaOf(NextPackage(ix, src, map[string]bool{"き": true, "め": true, "ねこ": true, "みず": true}, 5, knowAll))
	if !reflect.DeepEqual(got, []string{"あめ"}) {
		t.Fatalf("words without lesson form their own package, got %v", got)
	}
}

func TestNextPackageSkipsAheadPastUnreadableWords(t *testing.T) {
	ix := NewIndex(nil, []List{lessonList(
		[2]string{"テレビ", "2"}, [2]string{"ねこ", "2"}, [2]string{"き", "3"},
	)})
	src := Source{List: "minna"}
	noKatakana := func(u string) bool { return u != "テ" && u != "レ" && u != "ビ" }
	introduced := map[string]bool{"ねこ": true}
	got := kanaOf(NextPackage(ix, src, introduced, 5, noKatakana))
	if !reflect.DeepEqual(got, []string{"き"}) {
		t.Fatalf("unreadable テレビ must not block lesson 3, got %v", got)
	}
	introduced["き"] = true
	got = kanaOf(NextPackage(ix, src, introduced, 5, knowAll))
	if !reflect.DeepEqual(got, []string{"テレビ"}) {
		t.Fatalf("held-back lesson 2 word comes as its own package, got %v", got)
	}
}

func TestNextPackageWhereKeepsOnlyMatchingEntries(t *testing.T) {
	ix := NewIndex(nil, []List{lessonList(
		[2]string{"き", "1"}, [2]string{"ねこ", "2"}, [2]string{"みず", "2"},
	)})
	src := Source{List: "minna"}
	inLesson1 := func(e Entry) bool { return ix.Lesson(e, "minna") == "1" }
	if got := NextPackageWhere(ix, src, map[string]bool{"き": true}, 1, knowAll, inLesson1); got != nil {
		t.Fatalf("lesson 1 is used up: %v, want none rather than a lesson 2 word", kanaOf(got))
	}
	if got := kanaOf(NextPackageWhere(ix, src, nil, 5, knowAll, nil)); !reflect.DeepEqual(got, kanaOf(NextPackage(ix, src, nil, 5, knowAll))) {
		t.Fatalf("a nil keep must equal NextPackage, got %v", got)
	}
}
