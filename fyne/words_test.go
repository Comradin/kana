package main

import (
	"strings"
	"testing"

	"kana/vocab"
)

func TestResolveMeaningLang(t *testing.T) {
	cases := []struct{ stored, system, want string }{
		{"", "de-DE", "de"},
		{"", "sl", "sl"},
		{"", "pt_BR", "en"},
		{"", "ja-JP", "en"},
		{"", "", "en"},
		{"", "SV-se", "sv"},
		{"fr", "de-DE", "fr"},
	}
	for _, c := range cases {
		if got := resolveMeaningLang(c.stored, c.system); got != c.want {
			t.Errorf("resolveMeaningLang(%q, %q) = %q, want %q", c.stored, c.system, got, c.want)
		}
	}
}

func TestMissedWordsText(t *testing.T) {
	if got := missedWordsText(nil); got != "" {
		t.Fatalf("no words = %q", got)
	}
	got := missedWordsText([]string{"せんせい", "きって", "せんせい", "コーヒー", "まっちゃ"})
	want := "きって (kitte), せんせい (sensei), まっちゃ (matcha)\nコーヒー (koohii)"
	if got != want {
		t.Fatalf("missedWordsText = %q, want %q", got, want)
	}
}

func TestMeaningSnippet(t *testing.T) {
	m := vocab.Meanings{"en": {"yes", "that is correct", "right", "indeed"}, "de": {"ja"}}
	if got := meaningSnippet(m, "en"); got != "yes; that is correct; right" {
		t.Fatalf("en = %q, want the first three items", got)
	}
	if got := meaningSnippet(m, "de"); got != "ja" {
		t.Fatalf("de = %q", got)
	}
	if got := meaningSnippet(m, "fr"); got != "yes; that is correct; right" {
		t.Fatalf("fr = %q, want the English fallback", got)
	}
	long := vocab.Meanings{"en": {strings.Repeat("a", 40), strings.Repeat("b", 40)}}
	got := meaningSnippet(long, "en")
	if r := []rune(got); len(r) != maxMeaningRunes || !strings.HasSuffix(got, "…") {
		t.Fatalf("long = %q (%d runes), want %d runes ending in …", got, len(r), maxMeaningRunes)
	}
	if got := meaningSnippet(nil, "en"); got != "" {
		t.Fatalf("no meanings = %q", got)
	}
}

func TestFeedbackTexts(t *testing.T) {
	hai := vocab.Entry{Kana: "はい", Meanings: vocab.Meanings{"en": {"yes"}}}
	if got := hitFeedbackText(hai, "en"); got != "はい — yes" {
		t.Errorf("hit = %q", got)
	}
	if got := missFeedbackText(hai, "en"); got != "はい · hai — yes" {
		t.Errorf("miss = %q", got)
	}
	bare := vocab.Entry{Kana: "はい"}
	if got := hitFeedbackText(bare, "en"); got != "はい" {
		t.Errorf("hit without meanings = %q", got)
	}
	if got := missFeedbackText(bare, "en"); got != "はい · hai" {
		t.Errorf("miss without meanings = %q", got)
	}
}
