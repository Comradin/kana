package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes a synthetic word list into a fresh temp dir and returns
// its path.
func writeFile(t *testing.T, name, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckGoodFile(t *testing.T) {
	path := writeFile(t, "good.csv", "kana;meaning;lesson\nせんせい;teacher;1\nがくせい;student;1\n")
	var out bytes.Buffer
	if !checkFile(&out, path, false) {
		t.Fatalf("a good file must pass:\n%s", out.String())
	}
	if want := path + ": word list \"good\", 2 words, 0 rows skipped\n"; out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
}

func TestCheckListsEverySkippedRow(t *testing.T) {
	path := writeFile(t, "many.csv", "kana;meaning\nあお;blue\n"+strings.Repeat("x;y\n", 150))
	var out bytes.Buffer
	if !checkFile(&out, path, false) {
		t.Fatalf("a file with words must pass:\n%s", out.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if want := path + ": word list \"many\", 1 word, 150 rows skipped"; lines[0] != want {
		t.Fatalf("first line %q, want %q", lines[0], want)
	}
	if len(lines) != 152 || lines[1] != "Skipped rows:" {
		t.Fatalf("%d lines, second %q; want a heading and all 150 messages", len(lines), lines[1])
	}
	if lines[2] != "  line 3: x is not kana" || lines[151] != "  line 152: x is not kana" {
		t.Fatalf("messages %q … %q", lines[2], lines[151])
	}
}

func TestCheckFileThatCannotBeImported(t *testing.T) {
	path := writeFile(t, "sheet.xlsx", "PK")
	var out bytes.Buffer
	if checkFile(&out, path, false) {
		t.Fatal("a spreadsheet cannot be imported")
	}
	want := path + ": cannot import: This is a spreadsheet file.\n" +
		"  Save the sheet as \"CSV UTF-8\" in your spreadsheet program and import the .csv file.\n"
	if out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}

	missing := filepath.Join(t.TempDir(), "gone.csv")
	out.Reset()
	if checkFile(&out, missing, false) || !strings.HasPrefix(out.String(), missing+": cannot read: ") {
		t.Fatalf("missing file: %q", out.String())
	}
}

func TestCheckWords(t *testing.T) {
	path := writeFile(t, "words.csv", "kana;meaning;lesson\nせんせい;teacher;1\nえき;station;\n")
	var out bytes.Buffer
	checkFile(&out, path, true)
	want := path + ": word list \"words\", 2 words, 0 rows skipped\n" +
		"Words:\n  せんせい · sensei · lesson 1\n  えき · eki\n"
	if out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
}

func TestRunExitStatus(t *testing.T) {
	good := writeFile(t, "good.csv", "kana\nえき\n")
	bad := writeFile(t, "bad.json", "[]")
	var out, errOut bytes.Buffer
	if got := run([]string{good}, &out, &errOut); got != 0 {
		t.Fatalf("good file: status %d", got)
	}
	out.Reset()
	if got := run([]string{bad, good}, &out, &errOut); got != 1 {
		t.Fatalf("one bad file: status %d, want 1", got)
	}
	if s := out.String(); !strings.Contains(s, bad+": cannot import: ") || !strings.Contains(s, "\n\n"+good+": word list") {
		t.Fatalf("both files must be reported, separated by a blank line:\n%s", s)
	}
	out.Reset()
	if got := run([]string{"-words", good}, &out, &errOut); got != 0 || !strings.Contains(out.String(), "  えき · eki\n") {
		t.Fatalf("-words: status %d, output %q", got, out.String())
	}
	if got := run(nil, &out, &errOut); got != 2 || !strings.Contains(errOut.String(), "usage: listcheck [-words] FILE...") {
		t.Fatalf("no file: status %d, stderr %q", got, errOut.String())
	}
	if got := run([]string{"-nope", good}, &out, &errOut); got != 2 {
		t.Fatalf("unknown flag: status %d", got)
	}
}
