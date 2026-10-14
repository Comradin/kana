package main

import (
	"path/filepath"
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

func TestFormatWordsLine(t *testing.T) {
	cases := []struct {
		name string
		ws   WordsStatus
		want string
	}{
		{"bundled", WordsStatus{Bundled: true, Playable: 42, Secure: 17, Level: 5}, "Words: 42 playable · 17 secure · up to N5"},
		{"bundled error", WordsStatus{Bundled: true, Err: "broken data", Playable: 3, Level: 5}, "Words unavailable: broken data"},
		{"own lists", WordsStatus{HasLists: true, Playable: 23, Secure: 5, Level: 5}, "Words: 23 playable · 5 secure · own lists"},
		{"own lists, bundled error", WordsStatus{HasLists: true, Err: "broken data", Playable: 23, Secure: 5}, "Words: 23 playable · 5 secure · own lists"},
		{"no lists", WordsStatus{Dir: "~/.local/share/kana/words"}, "Words: add a list to ~/.local/share/kana/words"},
		{"no lists, no folder", WordsStatus{}, "Words: add a word list"},
	}
	for _, c := range cases {
		if got := formatWordsLine(c.ws); got != c.want {
			t.Errorf("%s: line = %q, want %q", c.name, got, c.want)
		}
	}
	long := formatWordsLine(WordsStatus{Bundled: true, Err: strings.Repeat("x", 500)})
	if r := []rune(long); len(r) != maxWordsLineRunes || !strings.HasSuffix(long, "…") {
		t.Errorf("long error = %q (%d runes), want the %d-rune safety cap", long, len(r), maxWordsLineRunes)
	}
}

func TestDisplayDir(t *testing.T) {
	p := filepath.FromSlash
	home := p("/home/me")
	cases := map[string]string{
		p("/home/me/.local/share/kana/words"): "~" + p("/.local/share/kana/words"),
		p("/home/me"):                         p("/home/me"),
		p("/home/mex/words"):                  p("/home/mex/words"),
		p("/data/kana/words"):                 p("/data/kana/words"),
	}
	for dir, want := range cases {
		if got := displayDir(dir, home); got != want {
			t.Errorf("displayDir(%q) = %q, want %q", dir, got, want)
		}
	}
	if got := displayDir(p("/data/words"), ""); got != p("/data/words") {
		t.Errorf("without a home: %q", got)
	}
}
