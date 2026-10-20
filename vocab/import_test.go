package vocab

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"
)

func utf16Bytes(s string, bigEndian bool) []byte {
	var b bytes.Buffer
	if bigEndian {
		b.Write([]byte{0xFE, 0xFF})
	} else {
		b.Write([]byte{0xFF, 0xFE})
	}
	for _, u := range utf16.Encode([]rune(s)) {
		if bigEndian {
			b.Write([]byte{byte(u >> 8), byte(u)})
		} else {
			b.Write([]byte{byte(u), byte(u >> 8)})
		}
	}
	return b.Bytes()
}

func importErr(t *testing.T, err error) *ImportError {
	t.Helper()
	var ie *ImportError
	if !errors.As(err, &ie) {
		t.Fatalf("err = %v, want an *ImportError", err)
	}
	return ie
}

func TestInspectAcceptsUTF8(t *testing.T) {
	for _, data := range []string{"kana;meaning;lesson\nせんせい;teacher;1\nがくせい;student;1\nえき;station;2\n",
		"\ufeffkana;meaning;lesson\nせんせい;teacher;1\nがくせい;student;1\nえき;station;2\n"} {
		c, err := Inspect("Minna no Nihongo.csv", []byte(data))
		if err != nil {
			t.Fatal(err)
		}
		if c.Name != "Minna no Nihongo" || c.Ext != ".csv" || len(c.List.Words) != 3 || c.List.Name != "Minna no Nihongo" {
			t.Fatalf("candidate %+v", c)
		}
		if !bytes.Equal(c.Data, []byte(data)) {
			t.Fatal("UTF-8 data must be written as is")
		}
	}
}

func TestInspectIgnoresDirectories(t *testing.T) {
	c, err := Inspect("sub/../../x.csv", []byte("kana\nえき\n"))
	if err != nil || c.Name != "x" {
		t.Fatalf("name %q, err %v; a path must not escape the folder", c.Name, err)
	}
}

func TestInspectConvertsUTF16(t *testing.T) {
	text := "kana\tmeaning\nせんせい\tteacher\n"
	for _, be := range []bool{false, true} {
		c, err := Inspect("liste.txt", utf16Bytes(text, be))
		if err != nil {
			t.Fatal(err)
		}
		if c.Ext != ".tsv" || string(c.Data) != text || len(c.List.Words) != 1 {
			t.Fatalf("big endian %v: ext %q, data %q, words %d", be, c.Ext, c.Data, len(c.List.Words))
		}
	}
}

func TestInspectJSON(t *testing.T) {
	c, err := Inspect("a.JSON", []byte(`[{"kana":"せんせい","meaning":"teacher"}]`))
	if err != nil || c.Ext != ".json" || len(c.List.Words) != 1 {
		t.Fatalf("c %+v, err %v", c, err)
	}
}

func TestInspectRejects(t *testing.T) {
	big := "kana\n" + strings.Repeat("あ\n", MaxImportSize/4)
	cases := []struct {
		name, file string
		data       []byte
		reason     string // substring
		hint       string // substring, "" = no hint expected
	}{
		{"spreadsheet", "list.xlsx", []byte("PK"), "spreadsheet", "CSV UTF-8"},
		{"other extension", "list.pdf", []byte("x"), ".csv", ""},
		{"dot name", ".hidden.csv", []byte("kana\nあ\n"), `"."`, "Rename"},
		{"reserved", "JMdict.csv", []byte("kana\nあ\n"), "jmdict", "Rename"},
		{"too large", "big.csv", []byte(big), "5 MB", ""},
		{"not UTF-8", "ansi.csv", []byte("kana;meaning\n\x82\xa0;x\n"), "UTF-8", "CSV UTF-8"},
		{"question marks", "lost.csv", []byte("kana;meaning\n???;teacher\n"), "no kana", "CSV UTF-8"},
		{"no kana column", "cols.csv", []byte("word;meaning\nせんせい;teacher\n"), "kana", "kana;meaning;lesson"},
		{"empty", "empty.csv", nil, "empty", ""},
		{"UTF-16 without BOM", "nobom.csv", []byte("k\x00a\x00n\x00a\x00"), "UTF-8", "CSV UTF-8"},
		{"odd UTF-16", "odd.txt", []byte{0xFF, 0xFE, 0x41}, "UTF-8", "CSV UTF-8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Inspect(tc.file, tc.data)
			ie := importErr(t, err)
			if !strings.Contains(ie.Reason, tc.reason) {
				t.Fatalf("reason %q, want it to contain %q", ie.Reason, tc.reason)
			}
			if (tc.hint == "") != (ie.Hint == "") || !strings.Contains(ie.Hint, tc.hint) {
				t.Fatalf("hint %q, want %q", ie.Hint, tc.hint)
			}
		})
	}
}

func writeFile(t *testing.T, dir, name, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestImportNewList(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "words") // does not exist yet
	c, err := Inspect("mine.txt", []byte("kana\tmeaning\nえき\tstation\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Existing(dir, "mine"); len(got) != 0 {
		t.Fatal("Existing before the import")
	}
	if name, err := Import(dir, c); err != nil || name != "mine" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if got := dirNames(t, dir); len(got) != 1 || got[0] != "mine.tsv" {
		t.Fatalf("files = %v", got)
	}
	if got := Existing(dir, "MINE"); !slices.Equal(got, []string{"mine.tsv"}) {
		t.Fatalf("Existing ignores case and keeps the stored name: %v", got)
	}
	lists, err := LoadDir(dir)
	if err != nil || len(lists) != 1 || lists[0].Name != "mine" || len(lists[0].Words) != 1 {
		t.Fatalf("lists %+v, err %v", lists, err)
	}
}

func TestImportReplacesSameName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Mine.json", `[{"kana":"あお"}]`)
	writeFile(t, dir, "other.csv", "kana\nいえ\n")
	writeFile(t, dir, "notes.md", "kept")
	c, err := Inspect("mine.csv", []byte("kana\nえき\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Existing(dir, "mine"); !slices.Equal(got, []string{"Mine.json"}) {
		t.Fatal("Mine.json exists under the same name")
	}
	if name, err := Import(dir, c); err != nil || name != "Mine" {
		t.Fatalf("name %q, err %v; the list keeps its name", name, err)
	}
	got := dirNames(t, dir)
	if len(got) != 3 || !slices.Contains(got, "Mine.csv") || !slices.Contains(got, "other.csv") || !slices.Contains(got, "notes.md") {
		t.Fatalf("files = %v; want Mine.json replaced by Mine.csv", got)
	}
}

func TestImportSameFileOtherCase(t *testing.T) {
	// On a case-insensitive file system (macOS) Mine.csv and mine.csv are
	// one file; the replaced list must not be deleted afterwards.
	dir := t.TempDir()
	writeFile(t, dir, "Mine.csv", "kana\nあお\n")
	c, _ := Inspect("mine.csv", []byte("kana\nえき\n"))
	if _, err := Import(dir, c); err != nil {
		t.Fatal(err)
	}
	lists, _ := LoadDir(dir)
	if len(lists) != 1 || len(lists[0].Words) != 1 || lists[0].Words[0].Kana != "えき" {
		t.Fatalf("lists %+v", lists)
	}
}

func TestImportFailureKeepsTheOldList(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "mine.json", `[{"kana":"あお"}]`)
	// A directory where the new file should go makes the rename fail.
	if err := os.Mkdir(filepath.Join(dir, "mine.csv"), 0o755); err != nil {
		t.Fatal(err)
	}
	c, _ := Inspect("mine.csv", []byte("kana\nえき\n"))
	if _, err := Import(dir, c); err == nil {
		t.Fatal("want an error")
	}
	got := dirNames(t, dir)
	if !slices.Contains(got, "mine.json") || len(got) != 2 {
		t.Fatalf("files = %v; want the old list kept and no temporary file", got)
	}
}

func TestImportReplacesSeveralFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "genki.csv", "kana\nあお\n")
	writeFile(t, dir, "Genki.json", `[{"kana":"いえ"}]`)
	c, err := Inspect("genki.tsv", []byte("kana\nえき\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Existing(dir, "GENKI"); !slices.Equal(got, []string{"Genki.json", "genki.csv"}) {
		t.Fatalf("Existing = %v", got)
	}
	if name, err := Import(dir, c); err != nil || name != "Genki" {
		t.Fatalf("name %q, err %v; the first existing name wins", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"Genki.tsv"}) {
		t.Fatalf("files = %v; want only the new file", got)
	}
}
