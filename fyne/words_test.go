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
	var words []vocab.Entry
	for _, k := range []string{"せんせい", "きって", "せんせい", "コーヒー", "まっちゃ"} {
		words = append(words, vocab.Entry{Kana: k})
	}
	want := "きって (kitte), せんせい (sensei), まっちゃ (matcha)\nコーヒー (koohii)"
	if got := missedWordsText(words); got != want {
		t.Fatalf("missedWordsText = %q, want %q", got, want)
	}
	san := vocab.Entry{Kana: "さん", Display: "～さん"}
	if got := missedWordsText([]vocab.Entry{san, san}); got != "～さん (san)" {
		t.Fatalf("shown form = %q, want ～さん (san)", got)
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
	if got := hitFeedbackText(hai, "en", false); got != "はい — yes" {
		t.Errorf("hit = %q", got)
	}
	if got := missFeedbackText(hai, "en"); got != "はい · hai — yes" {
		t.Errorf("miss = %q", got)
	}
	bare := vocab.Entry{Kana: "はい"}
	if got := hitFeedbackText(bare, "en", false); got != "はい" {
		t.Errorf("hit without meanings = %q", got)
	}
	if got := missFeedbackText(bare, "en"); got != "はい · hai" {
		t.Errorf("miss without meanings = %q", got)
	}
	san := vocab.Entry{Kana: "さん", Display: "～さん", Meanings: vocab.Meanings{"en": {"Mr./Ms."}}}
	if got := hitFeedbackText(san, "en", false); got != "～さん — Mr./Ms." {
		t.Errorf("hit with notation = %q", got)
	}
	if got := hitFeedbackText(san, "en", true); got != "～さん · san — Mr./Ms." {
		t.Errorf("hit by meaning with notation = %q", got)
	}
	if got := missFeedbackText(san, "en"); got != "～さん · san — Mr./Ms." {
		t.Errorf("miss with notation = %q", got)
	}
}

func TestFormatWordsLine(t *testing.T) {
	cases := []struct {
		name string
		ws   WordsStatus
		want string
	}{
		{"JLPT counts", WordsStatus{Introduced: 25, Learned: 18, Source: "JLPT N5", Level: 5}, "Words: 25 introduced · 18 learned · JLPT N5"},
		{"list counts", WordsStatus{Introduced: 3, Learned: 1, Source: "minna no nihongo"}, "Words: 3 introduced · 1 learned · minna no nihongo"},
		{"waiting for kana", WordsStatus{Introduced: 18, Learned: 5, Waiting: 15, Source: "minna no nihongo"}, "Words: 18 introduced · 5 learned · 15 waiting for kana · minna no nihongo"},
		{"load failed", WordsStatus{LoadFailed: true, Err: "broken data", Introduced: 3}, "Words unavailable: could not load progress"},
		{"bundled error", WordsStatus{Err: "broken data", Introduced: 3, Level: 5}, "Words unavailable: broken data"},
		{"missing list", WordsStatus{Missing: true, Dir: "~/.local/share/kana/words", Source: "minna"}, "Words: add a list to ~/.local/share/kana/words or choose JLPT"},
		{"missing list, no folder", WordsStatus{Missing: true, Source: "minna"}, "Words: add a word list or choose JLPT"},
		{"needs kana", WordsStatus{Next: wordsNeedKana, Introduced: 5, Source: "JLPT N5", Level: 5}, "Words: learn more kana for new words"},
		{"list finished", WordsStatus{Next: wordsFinished, Introduced: 40, Source: "minna no nihongo"}, "Words: minna no nihongo finished"},
		{"skipped rows", WordsStatus{Introduced: 3, Learned: 1, Source: "minna", Skipped: 2}, "Words: 3 introduced · 1 learned · minna · 2 rows skipped"},
		{"skipped rows, finished", WordsStatus{Next: wordsFinished, Source: "minna", Skipped: 1}, "Words: minna finished · 1 row skipped"},
		{"level finished", WordsStatus{Next: wordsFinished, Source: "JLPT N5", Level: 5}, "Words: JLPT N5 finished"},
		{"JLPT finished", WordsStatus{Next: wordsFinished, Source: "JLPT N1", Level: 1}, "Words: JLPT finished"},
		{"words only, counts", WordsStatus{WordsOnly: true, Introduced: 3, Learned: 1, Source: "minna"}, "Words: 3 introduced · 1 learned · minna"},
		{"words only, load failed", WordsStatus{WordsOnly: true, KanaFallback: true, LoadFailed: true}, "Words only: could not load progress"},
		{"words only, bundled error", WordsStatus{WordsOnly: true, KanaFallback: true, Err: "broken data", Level: 5}, "Words only: words unavailable: broken data"},
		{"words only, missing list", WordsStatus{WordsOnly: true, KanaFallback: true, Missing: true, Dir: "~/w", Source: "minna"}, "Words only: add a list to ~/w or choose JLPT"},
		{"words only, missing list, no folder", WordsStatus{WordsOnly: true, KanaFallback: true, Missing: true, Source: "minna"}, "Words only: add a word list or choose JLPT"},
		{"words only, empty lessons", WordsStatus{WordsOnly: true, KanaFallback: true, Empty: true, Filtered: true, Source: "minna"}, "Words only: no words in the chosen lessons"},
		{"words only, empty source", WordsStatus{WordsOnly: true, KanaFallback: true, Empty: true, Source: "minna"}, "Words only: no words in minna"},
		{"words only, scripts", WordsStatus{WordsOnly: true, KanaFallback: true, Introduced: 2, Source: "minna"}, "Words only: no word fits the active scripts"},
		{"words only, new words need a script", WordsStatus{WordsOnly: true, Next: wordsNeedKana, Introduced: 3, Source: "minna"}, "Words only: new words need a script that is off"},
		{"mixed, empty lessons", WordsStatus{Empty: true, Filtered: true, Next: wordsFinished, Source: "minna"}, "Words: no words in the chosen lessons"},
		{"words only, no meanings", WordsStatus{WordsOnly: true, KanaFallback: true, NoMeanings: true, Introduced: 2, Source: "minna"}, "Words only: no word has a meaning to ask for"},
		{"meaning counts", WordsStatus{Meaning: true, Introduced: 25, Learned: 3, Source: "JLPT N5", Level: 5}, "Meanings: 25 introduced · 3 learned · JLPT N5"},
		{"meaning, waiting for kana", WordsStatus{Meaning: true, Introduced: 18, Learned: 5, Waiting: 15, Source: "minna"}, "Meanings: 18 introduced · 5 learned · 15 waiting for kana · minna"},
		{"meaning, needs kana", WordsStatus{Meaning: true, Next: wordsNeedKana, Introduced: 5, Source: "JLPT N5", Level: 5}, "Words: learn more kana for new words"},
	}
	for _, c := range cases {
		if got := formatWordsLine(c.ws); got != c.want {
			t.Errorf("%s: line = %q, want %q", c.name, got, c.want)
		}
	}
	long := formatWordsLine(WordsStatus{Err: strings.Repeat("x", 500)})
	if r := []rune(long); len(r) != maxWordsLineRunes || !strings.HasSuffix(long, "…") {
		t.Errorf("long error = %q (%d runes), want the %d-rune safety cap", long, len(r), maxWordsLineRunes)
	}
}

func TestResolveWordSource(t *testing.T) {
	cases := []struct {
		stored string
		lists  []string
		want   string
	}{
		{"", nil, "jlpt"},
		{"", []string{"a-list", "b-list"}, "list:a-list"},
		{"jlpt", []string{"a-list"}, "jlpt"},
		{"list:gone", nil, "list:gone"}, // a missing list stays selected
		{"list:", []string{"a"}, "list:a"},
		{"bogus", nil, "jlpt"},
	}
	for _, c := range cases {
		if got := resolveWordSource(c.stored, c.lists); got != c.want {
			t.Errorf("resolveWordSource(%q, %v) = %q, want %q", c.stored, c.lists, got, c.want)
		}
	}
}

func TestPrettyListName(t *testing.T) {
	if got := prettyListName("minna-no_nihongo"); got != "minna no nihongo" {
		t.Fatalf("prettyListName = %q", got)
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

func TestFormatWordsLineReadingPractice(t *testing.T) {
	cases := []struct {
		name string
		ws   WordsStatus
		want string
	}{
		{"no word read yet", WordsStatus{Reading: true, Readable: 12, Source: "JLPT N5", Level: 5}, "Reading practice: 12 words to read · – kana/min"},
		{"speed", WordsStatus{Reading: true, Readable: 1, ReadKana: 40, KanaPerMin: 35, Source: "JLPT N5", Level: 5}, "Reading practice: 1 word to read · 35 kana/min"},
		{"nothing readable", WordsStatus{Reading: true, Source: "JLPT N5", Level: 5}, noWordsToReadText},
		{"nothing readable, words only", WordsStatus{Reading: true, WordsOnly: true, KanaFallback: true, Source: "minna", Next: wordsNeedKana}, noWordsToReadText},
		{"list missing", WordsStatus{Reading: true, WordsOnly: true, KanaFallback: true, Missing: true, Dir: "~/words", Source: "minna"}, "Words only: add a list to ~/words or choose JLPT"},
		{"skipped rows", WordsStatus{Reading: true, Readable: 3, Skipped: 2, Source: "minna"}, "Reading practice: 3 words to read · – kana/min · 2 rows skipped"},
	}
	for _, c := range cases {
		if got := formatWordsLine(c.ws); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	if noWordsToReadText != "No words to read yet — learn more kana or pick another list" {
		t.Fatalf("reason %q", noWordsToReadText)
	}
}
