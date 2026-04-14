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
