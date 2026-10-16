package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func generate(t *testing.T) ([]outWord, Stats, []byte) {
	t.Helper()
	var buf bytes.Buffer
	stats, err := run("testdata/JMdict_sample.xml", "testdata/jlpt", &buf)
	if err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), buf.Bytes()...)
	zr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	var words []outWord
	if err := json.NewDecoder(zr).Decode(&words); err != nil {
		t.Fatal(err)
	}
	return words, stats, raw
}

func TestGenerate(t *testing.T) {
	words, stats, _ := generate(t)
	byK := map[string]outWord{}
	var order []string
	for _, w := range words {
		byK[w.K] = w
		order = append(order, w.K)
	}
	if want := []string{"アイディア", "アイデア", "せんせい", "ねこ", "はし", "ふく"}; !reflect.DeepEqual(order, sortedCopy(want)) {
		t.Fatalf("words = %v", order)
	}
	s := byK["せんせい"]
	if s.J != 5 {
		t.Fatalf("easiest level wins: %d", s.J)
	}
	if !reflect.DeepEqual(s.M["en"], []string{"teacher", "instructor", "master"}) {
		t.Fatalf("en = %v (first sense, at most three)", s.M["en"])
	}
	if !reflect.DeepEqual(s.M["de"], []string{"Lehrer", "Meister"}) || !reflect.DeepEqual(s.M["fr"], []string{"professeur"}) {
		t.Fatalf("other languages = %v", s.M)
	}
	if !reflect.DeepEqual(s.M["ru"], []string{"учитель"}) {
		t.Fatalf("ru = %v (numbering stripped, markup gloss dropped)", s.M["ru"])
	}
	h := byK["はし"]
	if h.J != 5 || !reflect.DeepEqual(h.M["en"], []string{"bridge", "chopsticks"}) || !reflect.DeepEqual(h.M["de"], []string{"Brücke"}) {
		t.Fatalf("homophones merged: %+v", h)
	}
	if byK["アイディア"].J != 4 || !reflect.DeepEqual(byK["アイディア"].M["en"], []string{"idea", "(new) concept"}) {
		t.Fatalf("second reading of a seq, English glosses kept as they are: %+v", byK["アイディア"])
	}
	if !reflect.DeepEqual(byK["ねこ"].M["en"], []string{"cat"}) {
		t.Fatalf("waller fallback: %+v", byK["ねこ"])
	}
	f := byK["ふく"]
	if !reflect.DeepEqual(f.M["ru"], []string{"вытирать, стирать", "чистить", "протирать"}) {
		t.Fatalf("ru = %q (glosses split on \";\", per-sense cap counts items)", f.M["ru"])
	}
	if !reflect.DeepEqual(f.M["en"], []string{"to wipe", "to dry"}) {
		t.Fatalf("en = %q (English glosses are split too)", f.M["en"])
	}
	for _, w := range words {
		for lang, items := range w.M {
			for _, it := range items {
				if strings.Contains(it, ";") {
					t.Fatalf("%s %s: item %q contains \";\"", w.K, lang, it)
				}
			}
		}
	}
	if _, ok := byK["ぬ"]; ok {
		t.Fatal("row without sequence number must not be written")
	}
	if stats.Dropped != 1 || stats.MissingSeq != 1 || stats.NoSeq != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestCleanGloss(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"Lehrer", "Lehrer", true},
		{"  1) учитель ", "учитель", true},
		{"12)работать", "работать", true},
		{"1): {～な} ясный", "ясный", true},
		{": {～な} неясный", "неясный", true},
		{"{～[の]} тот же", "тот же", true},
		{"(ласк.) младенец", "младенец", true},
		{"((сокр. англ.) television) (разг.) телевизор", "телевизор", true},
		{"(почтительное обращение (к врачу)", "", false},
		{"(…に, …と)", "", false},
		{"{～をする} работать", "", false},
		{"{biol.} subfamilie", "subfamilie", true},
		{"calmado (tranquilo", "", false},
		{"bla) bla", "", false},
		{"1. завтрашний день", "завтрашний день", true},
		{"а) конопля", "конопля", true},
		{"een pil van een ~", "", false},
		{"ein ～ machen", "", false},
		{"escena (eng: scene)", "escena", true},
		{"Tanz (eng)", "Tanz", true},
		{"Schule (ger: Schule) Haus", "Schule (ger: Schule) Haus", true},
		{"moški {formalno}", "moški (formalno)", true},
		{"{～と言う} sagen", "", false},
		{"Haus {いえ}", "", false},
		{"Lat.} pons", "", false},
		{"-kant", "", false},
		{"(ср.) あいする", "", false},
		{"sagen (für iu言う)", "", false},
		{"1", "", false},
		{"", "", false},
		{"ok", "ok", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := cleanGloss(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("cleanGloss(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestSplitGloss(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"вытирать, стирать; чистить", []string{"вытирать, стирать", "чистить"}},
		{"y (junto con; al igual que)", []string{"y (junto con, al igual que)"}},
		{"a (b; c); d", []string{"a (b, c)", "d"}},
		{"a; a;; b", []string{"a", "b"}},
		{"a) b; c", []string{"a) b", "c"}},
		{"teacher", []string{"teacher"}},
	}
	for _, tt := range tests {
		if got := splitGloss(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitGloss(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGlossItems(t *testing.T) {
	tests := []struct {
		lang, in string
		want     []string
	}{
		{"ru", "сколько?; (с отриц.) немного", []string{"сколько?", "немного"}},
		{"ru", "сходить с рельсов; (обр.) а) отклоняться от темы; б) не идти", []string{"сходить с рельсов", "отклоняться от темы", "не идти"}},
		{"ru", "{～をする} делать; работать", nil},
		{"ru", "{～する} делать; работать", []string{"делать", "работать"}},
		{"ru", "{～な} ясный; {～の} светлый", []string{"ясный", "светлый"}},
		{"de", "Haus; いえ", []string{"Haus"}},
		{"es", "y (junto con; al igual que)", []string{"y (junto con, al igual que)"}},
		{"en", "to wipe; (of) x", []string{"to wipe", "(of) x"}},
	}
	for _, tt := range tests {
		if got := glossItems(tt.lang, tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("glossItems(%q, %q) = %q, want %q", tt.lang, tt.in, got, tt.want)
		}
	}
}

func TestGenerateIsReproducible(t *testing.T) {
	_, _, a := generate(t)
	_, _, b := generate(t)
	if !bytes.Equal(a, b) {
		t.Fatal("output differs between runs")
	}
}

func TestGenerateMissingInput(t *testing.T) {
	var buf bytes.Buffer
	if _, err := run("testdata/nope.xml", "testdata/jlpt", &buf); err == nil {
		t.Fatal("missing JMdict must fail")
	}
	if _, err := run("testdata/JMdict_sample.xml", os.TempDir()+"/no-such-jlpt", &buf); err == nil {
		t.Fatal("missing JLPT dir must fail")
	}
}

func sortedCopy(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

func TestRankOf(t *testing.T) {
	tests := []struct {
		tags []string
		want int
	}{
		{nil, 99},
		{[]string{"nf01"}, 1},
		{[]string{"nf48"}, 48},
		{[]string{"news1"}, 49},
		{[]string{"ichi1"}, 49},
		{[]string{"spec1"}, 49},
		{[]string{"gai1"}, 49},
		{[]string{"news2"}, 60},
		{[]string{"ichi2"}, 60},
		{[]string{"spec2"}, 60},
		{[]string{"gai2"}, 60},
		{[]string{"news2", "ichi1", "nf17"}, 17},
		{[]string{"news2", "spec1"}, 49},
		{[]string{"nf00", "nf49", "nfx", "other"}, 99},
	}
	for _, tt := range tests {
		if got := rankOf(tt.tags); got != tt.want {
			t.Errorf("rankOf(%v) = %d, want %d", tt.tags, got, tt.want)
		}
	}
}

func TestEntryRank(t *testing.T) {
	e := newJMEntry(xmlEntry{
		Kanji: []xmlKanji{{Pri: []string{"nf02"}}},
		Readings: []xmlReading{
			{Reb: "あ", Pri: []string{"news2"}},
			{Reb: "い"},
		},
	})
	if got := e.rank("あ"); got != 60 {
		t.Errorf("rank(あ) = %d, want 60 from its own tags, not the kanji's nf02", got)
	}
	if got := e.rank("い"); got != 2 {
		t.Errorf("rank(い) = %d, want 2, the best tag of the entry", got)
	}
	if got := e.rank("う"); got != 2 {
		t.Errorf("rank(う) = %d, want 2 for a reading the entry does not list", got)
	}
	if got := newJMEntry(xmlEntry{Readings: []xmlReading{{Reb: "あ"}}}).rank("あ"); got != 99 {
		t.Errorf("rank without any tag = %d, want 99", got)
	}
}

func TestGenerateRanks(t *testing.T) {
	words, _, _ := generate(t)
	want := map[string]int{
		"せんせい":  49, // its reading's ichi1/news1, not the kanji's nf01
		"はし":    5,  // homophones merged: nf05 beats ichi1
		"アイデア":  60, // gai2
		"アイディア": 60, // no tags of its own: the entry's best
		"ふく":    99, // entry without tags
		"ねこ":    99, // Waller fallback, no JMdict entry
	}
	for _, w := range words {
		if w.R != want[w.K] {
			t.Errorf("%s: rank %d, want %d", w.K, w.R, want[w.K])
		}
	}
}
