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
	if len(chars) != 46 {
		t.Fatalf("expected 46 characters, got %d", len(chars))
	}
}

func TestAllKanaRowsCount(t *testing.T) {
	if len(AllKanaRows) != 11 {
		t.Fatalf("expected 11 rows, got %d", len(AllKanaRows))
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
