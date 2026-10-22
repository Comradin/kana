package vocab

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// writeLists writes a one-word list file per name into dir.
func writeLists(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, name := range names {
		content := "kana\nねこ\n"
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			content = `[{"kana":"ねこ"}]`
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeleteList(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv", "other.csv")
	path := filepath.Join(dir, "mine.csv")
	if err := DeleteList(path); err != nil {
		t.Fatal(err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"other.csv"}) {
		t.Fatalf("files = %v, want only other.csv", got)
	}
	if err := DeleteList(path); err != nil {
		t.Fatalf("a file already gone counts as deleted: %v", err)
	}
}

func TestCheckListName(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv", "other.csv", "dup.csv", "dup.json")
	path := filepath.Join(dir, "mine.csv")
	for _, tc := range []struct {
		in   string
		want error
	}{
		{"", ErrNameEmpty},
		{"   ", ErrNameEmpty},
		{"a/b", ErrNameSeparator},
		{`a\b`, ErrNameSeparator},
		{".hidden", ErrNameDot},
		{"JMdict", ErrNameReserved},
		{"other", ErrNameTaken},
		{"OTHER", ErrNameTaken},
		{"Dup", ErrNameTaken},       // the file name of two other lists
		{"dup (CSV)", ErrNameTaken}, // the list name of another list
	} {
		if _, err := CheckListName(path, tc.in); !errors.Is(err, tc.want) {
			t.Errorf("CheckListName(%q) = %v, want %v", tc.in, err, tc.want)
		}
	}
	for in, want := range map[string]string{"mine": "mine", "Mine": "Mine", "  ours ": "ours"} {
		if got, err := CheckListName(path, in); err != nil || got != want {
			t.Errorf("CheckListName(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestRenameListKeepsExtension(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.TSV")
	name, err := RenameList(filepath.Join(dir, "mine.TSV"), " ours ")
	if err != nil || name != "ours" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"ours.TSV"}) {
		t.Fatalf("files = %v, want ours.TSV", got)
	}
	lists, err := LoadDir(dir)
	if err != nil || len(lists) != 1 || lists[0].Name != "ours" || lists[0].Path != filepath.Join(dir, "ours.TSV") {
		t.Fatalf("lists %+v, err %v", lists, err)
	}
}

func TestRenameListRejectsWithoutTouching(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv", "other.json")
	if _, err := RenameList(filepath.Join(dir, "mine.csv"), "Other"); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("err = %v, want ErrNameTaken", err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"mine.csv", "other.json"}) {
		t.Fatalf("files = %v, want both unchanged", got)
	}
}

func TestRenameListCaseOnly(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv")
	name, err := RenameList(filepath.Join(dir, "mine.csv"), "Mine")
	if err != nil || name != "Mine" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"Mine.csv"}) {
		t.Fatalf("files = %v, want Mine.csv only", got)
	}
	if name, err := RenameList(filepath.Join(dir, "Mine.csv"), "Mine"); err != nil || name != "Mine" {
		t.Fatalf("the same name: %q, %v; want no change", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"Mine.csv"}) {
		t.Fatalf("files = %v after renaming to the same name", got)
	}
}

func TestRenameListOfDuplicateName(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "dup.csv", "dup.json")
	lists, _ := LoadDir(dir)
	var js List
	for _, l := range lists {
		if l.Name == "dup (json)" {
			js = l
		}
	}
	name, err := RenameList(js.Path, "dup2")
	if err != nil || name != "dup2" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"dup.csv", "dup2.json"}) {
		t.Fatalf("files = %v; only the list's own file is renamed", got)
	}
	lists, _ = LoadDir(dir)
	if len(lists) != 2 || lists[0].Name != "dup" || lists[1].Name != "dup2" {
		t.Fatalf("lists = %+v, want dup and dup2", lists)
	}
}

func TestRenameListNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv")
	if err := os.Mkdir(filepath.Join(dir, "ours.csv"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := RenameList(filepath.Join(dir, "mine.csv"), "ours"); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("err = %v, want ErrNameTaken for an existing file", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); err != nil {
		t.Fatal("the list must be untouched")
	}
}

func TestRenameListGone(t *testing.T) {
	dir := t.TempDir()
	if _, err := RenameList(filepath.Join(dir, "gone.csv"), "ours"); err == nil {
		t.Fatal("renaming a missing file must fail")
	}
}
