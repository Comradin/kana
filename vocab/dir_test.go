package vocab

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadJSON(t *testing.T) {
	in := `[
	 {"kana": "せんせい", "meaning": {"de": "Lehrer; Lehrerin", "en": ["teacher", "master"]}, "lesson": 1, "jlpt": "N5"},
	 {"kana": "ねこ", "meaning": "Katze"},
	 {"kana": "猫", "meaning": "cat"}
	]`
	l := readJSON("x", strings.NewReader(in))
	if len(l.Words) != 2 || l.Skipped != 1 {
		t.Fatalf("words=%d skipped=%d problems=%v", len(l.Words), l.Skipped, l.Problems)
	}
	w := l.Words[0]
	if !reflect.DeepEqual(w.Meanings["de"], []string{"Lehrer", "Lehrerin"}) || !reflect.DeepEqual(w.Meanings["en"], []string{"teacher", "master"}) {
		t.Fatalf("meanings = %v", w.Meanings)
	}
	if w.Lesson != "1" || w.JLPT != 5 {
		t.Fatalf("lesson=%q jlpt=%d", w.Lesson, w.JLPT)
	}
	if got := l.Words[1].Meanings[""]; !reflect.DeepEqual(got, []string{"Katze"}) {
		t.Fatalf("string meaning = %v", got)
	}
	bad := readJSON("x", strings.NewReader(`{"kana":`))
	if len(bad.Words) != 0 || len(bad.Problems) != 1 {
		t.Fatalf("broken JSON: %+v", bad)
	}
	keys := readJSON("x", strings.NewReader(`[{"kana":"ねこ","meaning":{"de":"y","DE":"x"}}]`))
	if got := keys.Words[0].Meanings["de"]; !reflect.DeepEqual(got, []string{"x", "y"}) {
		t.Fatalf("object keys in sorted order: %v", got)
	}
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("minna.csv", "kana,meaning\nせんせい,Lehrer\nせんせい,Lehrerin\n")
	write("a.csv", "kana\nねこ\n")
	write("a.json", `[{"kana":"いぬ"}]`)
	write("jmdict.tsv", "kana\nとり\n")
	write("notes.txt", "ignored")
	lists, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	byName := map[string]List{}
	for _, l := range lists {
		names = append(names, l.Name)
		byName[l.Name] = l
	}
	want := []string{"a (csv)", "a (json)", "jmdict (tsv)", "minna"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	m := byName["minna"]
	if len(m.Words) != 2 {
		t.Fatalf("every row is kept within a list: %+v", m.Words)
	}
	if m.Path != filepath.Join(dir, "minna.csv") {
		t.Fatalf("Path = %q, want the list's file", m.Path)
	}
	if a := byName["a (json)"]; a.Path != filepath.Join(dir, "a.json") {
		t.Fatalf("a (json) Path = %q: each list knows its own file", a.Path)
	}
	ix := NewIndex(nil, lists)
	e, ok := ix.Lookup("せんせい")
	if !ok || !reflect.DeepEqual(e.Meanings[""], []string{"Lehrer", "Lehrerin"}) {
		t.Fatalf("index merges repeated readings: %+v", e)
	}
	if byName["a (json)"].Words[0].Source != "a (json)" {
		t.Fatal("Source is the list name")
	}
	lists, err = LoadDir(filepath.Join(dir, "missing"))
	if err != nil || len(lists) != 0 {
		t.Fatalf("missing dir: %v, %v", lists, err)
	}
}

func TestLoadDirNames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.CSV", "a.csv", "Jmdict.csv", "B.json", "b.tsv", "solo.Csv"} {
		content := "kana\nねこ\n"
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			content = `[{"kana":"ねこ"}]`
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lists, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, l := range lists {
		names = append(names, l.Name)
		if len(l.Words) != 1 || l.Words[0].Source != l.Name {
			t.Fatalf("%s: words %+v", l.Name, l.Words)
		}
	}
	want := []string{"B (json)", "Jmdict (csv)", "a (csv)", "a (csv) 2", "b (tsv)", "solo"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}

func TestLoadDirSkipsDotFiles(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		".csv":         "kana\nねこ\n",                  // would be a list with an empty name
		"._x.csv":      "\x00\x05\x16\x07AppleDouble", // macOS resource fork
		".hidden.json": `[{"kana":"いぬ"}]`,
		"x.csv":        "kana\nとり\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lists, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 || lists[0].Name != "x" || len(lists[0].Problems) != 0 {
		t.Fatalf("lists = %+v, want only x", lists)
	}
}

func TestReadJSONBOMAndEmptyKana(t *testing.T) {
	l := readJSON("x", strings.NewReader("\ufeff[{\"kana\":\"ねこ\"},{\"kana\":\"  \"},{\"meaning\":\"m\"}]"))
	if len(l.Words) != 1 || l.Words[0].Kana != "ねこ" {
		t.Fatalf("BOM: words = %+v, problems %v", l.Words, l.Problems)
	}
	want := []string{"entry 2: empty kana", "entry 3: empty kana"}
	if l.Skipped != 2 || !reflect.DeepEqual(l.Problems, want) {
		t.Fatalf("skipped=%d problems=%q, want %q", l.Skipped, l.Problems, want)
	}
}
