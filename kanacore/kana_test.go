package kanacore

import "testing"

func TestHiraganaGetRomaji(t *testing.T) {
	romaji, ok := Hiragana().GetRomaji("か")
	if !ok {
		t.Fatal("expected か to exist in Hiragana set")
	}
	if romaji != "ka" {
		t.Fatalf("expected romaji 'ka', got %q", romaji)
	}
}

func TestHiraganaGetCharacters(t *testing.T) {
	chars := Hiragana().GetCharacters()
	if len(chars) != 104 {
		t.Fatalf("expected 104 characters, got %d", len(chars))
	}
}

func TestAllKanaRowsCount(t *testing.T) {
	if len(AllKanaRows) != 27 {
		t.Fatalf("expected 27 rows, got %d", len(AllKanaRows))
	}
}

func TestCharToRow(t *testing.T) {
	rowID, ok := CharToRow["か"]
	if !ok {
		t.Fatal("expected か to be in CharToRow")
	}
	if rowID != "k" {
		t.Fatalf("expected row ID 'k', got %q", rowID)
	}
}

func TestDefaultRowIDs(t *testing.T) {
	ids := DefaultRowIDs()
	if len(ids) != 11 {
		t.Fatalf("expected 11 row IDs, got %d", len(ids))
	}
}

func TestMatchesBasic(t *testing.T) {
	cs := Hiragana()
	cases := []struct {
		char, input string
		want        bool
	}{
		{"か", "ka", true},
		{"か", "  KA ", true},
		{"し", "shi", true},
		{"し", "si", true},
		{"ち", "ti", true},
		{"つ", "tu", true},
		{"ふ", "hu", true},
		{"ん", "n", true},
		{"ん", "nn", true},
		{"を", "wo", true},
		{"を", "o", false},
		{"か", "ki", false},
		{"か", "", false},
		{"x", "ka", false},
	}
	for _, c := range cases {
		if got := cs.Matches(c.char, c.input); got != c.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", c.char, c.input, got, c.want)
		}
	}
}

func TestRowByID(t *testing.T) {
	row, ok := RowByID("k")
	if !ok || row.Label != "K-row (か)" {
		t.Fatalf("RowByID(k) = %+v, %v", row, ok)
	}
	if _, ok := RowByID("nope"); ok {
		t.Fatal("RowByID(nope) should not exist")
	}
}

func TestRowCharactersFollowEntries(t *testing.T) {
	row, _ := RowByID("vowels")
	got := row.Characters()
	want := []string{"あ", "い", "う", "え", "お"}
	if len(got) != len(want) {
		t.Fatalf("Characters() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Characters() = %v, want %v", got, want)
		}
	}
}

func TestAllKanaRowsHaveNoDuplicateCharacters(t *testing.T) {
	seen := make(map[string]string)
	total := 0
	for _, row := range AllKanaRows {
		for _, char := range row.Characters() {
			if other, dup := seen[char]; dup {
				t.Fatalf("%s appears in rows %s and %s", char, other, row.ID)
			}
			seen[char] = row.ID
			total++
		}
	}
	if total != 104 {
		t.Fatalf("expected 104 characters, got %d", total)
	}
}

func TestRowsAreOrderedByGroup(t *testing.T) {
	order := make(map[Group]int)
	for i, g := range Groups() {
		order[g.Group] = i
	}
	last := 0
	for _, row := range AllKanaRows {
		idx, ok := order[row.Group]
		if !ok {
			t.Fatalf("row %s has unknown group %q", row.ID, row.Group)
		}
		if idx < last {
			t.Fatalf("row %s (group %s) appears after a later group", row.ID, row.Group)
		}
		last = idx
	}
}

func TestBasicRowsUnchanged(t *testing.T) {
	rows := BasicRows()
	wantIDs := []string{"vowels", "k", "s", "t", "n", "h", "m", "y", "r", "w", "n-only"}
	if len(rows) != len(wantIDs) {
		t.Fatalf("BasicRows() has %d rows, want %d", len(rows), len(wantIDs))
	}
	chars := 0
	for i, row := range rows {
		if row.ID != wantIDs[i] {
			t.Errorf("BasicRows()[%d] = %s, want %s", i, row.ID, wantIDs[i])
		}
		chars += len(row.Entries)
	}
	if chars != 46 {
		t.Errorf("BasicRows() has %d characters, want 46", chars)
	}
}

func TestMatchesExtended(t *testing.T) {
	cs := Hiragana()
	cases := []struct {
		char, input string
		want        bool
	}{
		{"が", "ga", true},
		{"じ", "ji", true},
		{"じ", "zi", true},
		{"じ", "di", false},
		{"ぢ", "ji", true},
		{"ぢ", "di", true},
		{"ず", "zu", true},
		{"ず", "du", false},
		{"づ", "zu", true},
		{"づ", "du", true},
		{"ぱ", "pa", true},
		{"きゃ", "kya", true},
		{"しゃ", "sha", true},
		{"しゃ", "sya", true},
		{"ちゃ", "cha", true},
		{"ちゃ", "tya", true},
		{"ちゃ", "cya", true},
		{"じゃ", "ja", true},
		{"じゃ", "jya", true},
		{"じゃ", "zya", true},
		{"ぴょ", "pyo", true},
		{"きゃ", "ka", false},
	}
	for _, c := range cases {
		if got := cs.Matches(c.char, c.input); got != c.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", c.char, c.input, got, c.want)
		}
	}
}

func TestProgressionStepsCoverRowsInOrder(t *testing.T) {
	var flat []string
	for i, step := range ProgressionSteps {
		if len(step) == 0 {
			t.Fatalf("step %d is empty", i)
		}
		first, _ := RowByID(step[0])
		for _, id := range step {
			row, ok := RowByID(id)
			if !ok {
				t.Fatalf("step %d has unknown row %q", i, id)
			}
			if row.Group != first.Group {
				t.Fatalf("step %d crosses groups: %v", i, step)
			}
		}
		flat = append(flat, step...)
	}
	if len(flat) != len(AllKanaRows) {
		t.Fatalf("steps cover %d rows, want %d", len(flat), len(AllKanaRows))
	}
	for i, row := range AllKanaRows {
		if flat[i] != row.ID {
			t.Fatalf("step order position %d = %s, want %s", i, flat[i], row.ID)
		}
	}
	if len(ProgressionSteps[0]) != 1 || ProgressionSteps[0][0] != "vowels" {
		t.Fatalf("first step = %v, want [vowels]", ProgressionSteps[0])
	}
}
