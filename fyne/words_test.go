package main

import "testing"

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
