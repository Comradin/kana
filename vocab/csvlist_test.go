package vocab

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestReadCSVSeparators(t *testing.T) {
	inputs := map[string]string{
		"comma":     "kana,meaning_de,meaning_en,lesson\nせんせい,Lehrer; Lehrerin,teacher,1\n",
		"semicolon": "kana;meaning_de;meaning_en;lesson\nせんせい;\"Lehrer; Lehrerin\";teacher;1\n",
		"tab":       "kana\tmeaning_de\tmeaning_en\tlesson\nせんせい\tLehrer; Lehrerin\tteacher\t1\n",
		"bom":       "\ufeffkana,meaning_de,meaning_en,lesson\nせんせい,Lehrer; Lehrerin,teacher,1\n",
		"case":      "Kana,Meaning_DE,MEANING_EN,Lesson\nせんせい,Lehrer; Lehrerin,teacher,1\n",
	}
	for name, in := range inputs {
		l := readCSV("minna", strings.NewReader(in), maxProblems)
		if len(l.Words) != 1 || len(l.Problems) != 0 {
			t.Fatalf("%s: words=%d problems=%v", name, len(l.Words), l.Problems)
		}
		w := l.Words[0]
		if w.Kana != "せんせい" || w.Lesson != "1" || w.Source != "minna" {
			t.Fatalf("%s: word = %+v", name, w)
		}
		if !reflect.DeepEqual(w.Meanings["de"], []string{"Lehrer", "Lehrerin"}) || !reflect.DeepEqual(w.Meanings["en"], []string{"teacher"}) {
			t.Fatalf("%s: meanings = %v", name, w.Meanings)
		}
	}
}

func TestReadCSVGenericMeaningAndLevel(t *testing.T) {
	in := "kana,meaning,jlpt\nねこ,Katze,N5\nいぬ,Hund,4\n"
	l := readCSV("x", strings.NewReader(in), maxProblems)
	if len(l.Words) != 2 {
		t.Fatalf("words = %d, problems %v", len(l.Words), l.Problems)
	}
	if got := l.Words[0].Meanings[""]; !reflect.DeepEqual(got, []string{"Katze"}) {
		t.Fatalf("generic meaning = %v", got)
	}
	if l.Words[0].JLPT != 5 || l.Words[1].JLPT != 4 {
		t.Fatalf("levels = %d, %d", l.Words[0].JLPT, l.Words[1].JLPT)
	}
}

func TestReadCSVProblems(t *testing.T) {
	l := readCSV("x", strings.NewReader("word,meaning\nねこ,Katze\n"), maxProblems)
	if len(l.Words) != 0 || len(l.Problems) != 1 || !strings.Contains(l.Problems[0], "kana") {
		t.Fatalf("missing kana column: %+v", l)
	}
	l = readCSV("x", strings.NewReader("kana,meaning\n先生,Lehrer\nねこ,Katze\n,leer\n"), maxProblems)
	if len(l.Words) != 1 || l.Skipped != 2 {
		t.Fatalf("words=%d skipped=%d", len(l.Words), l.Skipped)
	}
	if !strings.Contains(l.Problems[0], "line 2") || !strings.Contains(l.Problems[0], "先生") {
		t.Fatalf("problem = %q", l.Problems[0])
	}
	if l.Problems[1] != "line 4: empty kana" {
		t.Fatalf("empty kana problem = %q", l.Problems[1])
	}
	var b strings.Builder
	b.WriteString("kana\n")
	for i := 0; i < 150; i++ {
		b.WriteString("x\n")
	}
	l = readCSV("x", strings.NewReader(b.String()), maxProblems)
	if len(l.Problems) != maxProblems || l.Skipped != 150 {
		t.Fatalf("problems=%d skipped=%d", len(l.Problems), l.Skipped)
	}
	l = readCSV("x", strings.NewReader(b.String()), allProblems)
	if len(l.Problems) != 150 || l.Skipped != 150 {
		t.Fatalf("all problems: problems=%d skipped=%d", len(l.Problems), l.Skipped)
	}
	l = readCSV("x", strings.NewReader(""), maxProblems)
	if len(l.Words) != 0 || len(l.Problems) != 1 {
		t.Fatalf("empty file: %+v", l)
	}
}

// failingReader returns its data, then a non-EOF error forever.
type failingReader struct{ data io.Reader }

func (f failingReader) Read(p []byte) (int, error) {
	n, err := f.data.Read(p)
	if err == io.EOF {
		return n, errors.New("disk on fire")
	}
	return n, err
}

func TestReadCSVReadError(t *testing.T) {
	l := readCSV("x", failingReader{strings.NewReader("kana\nねこ\n")}, maxProblems)
	if len(l.Words) != 1 {
		t.Fatalf("words = %d, problems %v", len(l.Words), l.Problems)
	}
	if len(l.Problems) != 1 || !strings.Contains(l.Problems[0], "disk on fire") {
		t.Fatalf("problems = %v", l.Problems)
	}
	if l.Skipped != 0 {
		t.Fatalf("a read error is not a skipped row: skipped=%d", l.Skipped)
	}
}

func TestReadCSVColumnOrder(t *testing.T) {
	// The language-less meaning and meaning_de land in header order; with a
	// map iteration they would not be deterministic.
	in := "meaning_de,kana,meaning,meaning_\nKatze,ねこ,cat,ignored\n"
	for i := 0; i < 20; i++ {
		l := readCSV("x", strings.NewReader(in), maxProblems)
		if len(l.Words) != 1 {
			t.Fatalf("words = %d, problems %v", len(l.Words), l.Problems)
		}
		m := l.Words[0].Meanings
		want := Meanings{"de": {"Katze"}, "": {"cat"}}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("meanings = %v, want %v (meaning_ ignored)", m, want)
		}
	}
}

func TestReadCSVDuplicateHeader(t *testing.T) {
	in := "kana,meaning,lesson,Kana,meaning,lesson\nねこ,Katze,1,いぬ,Hund,2\n"
	l := readCSV("x", strings.NewReader(in), maxProblems)
	if len(l.Words) != 1 {
		t.Fatalf("words = %d, problems %v", len(l.Words), l.Problems)
	}
	w := l.Words[0]
	if w.Kana != "ねこ" || w.Lesson != "1" || !reflect.DeepEqual(w.Meanings[""], []string{"Katze"}) {
		t.Fatalf("first column of a duplicate name wins: %+v", w)
	}
}

func TestReadCSVHeaderAliases(t *testing.T) {
	in := "Japanische Schrift,Japanisch (Romaji),Deutsch,Lektion\nせんせい,sensei,Lehrer; Lehrerin,1\n"
	l := readCSV("minna", strings.NewReader(in), maxProblems)
	if len(l.Words) != 1 || len(l.Problems) != 0 {
		t.Fatalf("words=%d problems=%v", len(l.Words), l.Problems)
	}
	w := l.Words[0]
	want := Meanings{"de": {"Lehrer", "Lehrerin"}}
	if w.Kana != "せんせい" || w.Lesson != "1" || !reflect.DeepEqual(w.Meanings, want) {
		t.Fatalf("word = %+v, want せんせい, lesson 1, meanings %v (romaji ignored)", w, want)
	}

	aliases := map[string]string{
		"japanisch,english":        "kana,meaning_en",
		" Reading ,GERMAN,lektion": "kana,meaning_de,lesson",
		"japanisch,englisch":       "kana,meaning_en",
	}
	for alias, canon := range aliases {
		a := readCSV("x", strings.NewReader(alias+"\nねこ,cat,2\n"), maxProblems)
		c := readCSV("x", strings.NewReader(canon+"\nねこ,cat,2\n"), maxProblems)
		if len(a.Words) != 1 || !reflect.DeepEqual(a.Words, c.Words) {
			t.Errorf("%q: %+v, want %+v", alias, a.Words, c.Words)
		}
	}
}

func TestReadCSVCanonicalColumnWinsOverAlias(t *testing.T) {
	in := "deutsch,reading,kana,meaning_de,lektion,lesson\nfalsch,いぬ,ねこ,Katze,9,1\n"
	l := readCSV("x", strings.NewReader(in), maxProblems)
	if len(l.Words) != 1 {
		t.Fatalf("words = %d, problems %v", len(l.Words), l.Problems)
	}
	w := l.Words[0]
	if w.Kana != "ねこ" || w.Lesson != "1" || !reflect.DeepEqual(w.Meanings, Meanings{"de": {"Katze"}}) {
		t.Fatalf("canonical columns must win: %+v", w)
	}
}

func TestReadCSVNotation(t *testing.T) {
	in := "kana;meaning;lesson\n" +
		"～さん;Mr./Ms.;1\n" +
		"（どうぞ）よろしく（おねがいします）;nice to meet you;1\n" +
		"あめりか // アメリカ;America;2\n" +
		"（どうぞ;x;2\n" +
		"～;x;2\n" +
		"あめりか // 米国;x;2\n"
	l := readCSV("course", strings.NewReader(in), maxProblems)
	want := []Word{
		{Kana: "さん", Display: "～さん", Meanings: Meanings{"": {"Mr./Ms."}}, Source: "course", Lesson: "1"},
		{Kana: "どうぞよろしくおねがいします", Display: "（どうぞ）よろしく（おねがいします）", Alts: []string{"よろしく"},
			Meanings: Meanings{"": {"nice to meet you"}}, Source: "course", Lesson: "1"},
		{Kana: "アメリカ", Alts: []string{"あめりか"}, Meanings: Meanings{"": {"America"}}, Source: "course", Lesson: "2"},
	}
	if !reflect.DeepEqual(l.Words, want) {
		t.Fatalf("words:\n%+v\nwant\n%+v", l.Words, want)
	}
	wantProblems := []string{
		"line 5: （どうぞ has unpaired brackets",
		"line 6: empty kana",
		"line 7: あめりか // 米国 is not kana",
	}
	if l.Skipped != 3 || !reflect.DeepEqual(l.Problems, wantProblems) {
		t.Fatalf("skipped %d, problems %q; want %q", l.Skipped, l.Problems, wantProblems)
	}
}
