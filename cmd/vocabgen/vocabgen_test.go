package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"reflect"
	"sort"
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
	if want := []string{"アイディア", "アイデア", "せんせい", "ねこ", "はし"}; !reflect.DeepEqual(order, sortedCopy(want)) {
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
		{"{biol.} subfamilie", "", false},
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
