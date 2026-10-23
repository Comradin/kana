package vocab

import "testing"

func TestMeaningMatches(t *testing.T) {
	kasa := Meanings{
		"de": {"Regenschirm", "Schirm"},
		"en": {"umbrella", "parasol"},
		"es": {"paraguas"},
	}
	sensei := Meanings{"de": {"Lehrer", "Dozent (wird nicht für den eigenen Beruf verwendet)"}}
	kiru := Meanings{"de": {"anziehen"}, "en": {"to wear (on the upper body)"}}
	daigaku := Meanings{"de": {"Universität"}}
	plain := Meanings{"": {"die Person dort drüben"}}
	cases := []struct {
		name  string
		m     Meanings
		lang  string
		input string
		want  bool
	}{
		{"shown language", kasa, "de", "Regenschirm", true},
		{"other alternative, case", kasa, "de", "  SCHIRM ", true},
		{"English too", kasa, "de", "umbrella", true},
		{"other languages not", kasa, "de", "paraguas", false},
		{"typo", kasa, "de", "Regenschrim", false},
		{"part of a meaning", kasa, "de", "Regen", false},
		{"parentheses ignored", sensei, "de", "dozent", true},
		{"leading to optional", kiru, "de", "wear", true},
		{"leading to typed", kiru, "de", "to wear", true},
		{"umlaut spelled out", daigaku, "de", "universitaet", true},
		{"umlaut typed", daigaku, "de", "Universität", true},
		{"article optional", plain, "de", "Person dort drüben", true},
		{"article typed", plain, "de", "die person dort drueben", true},
		{"blank", kasa, "de", "  ", false},
		{"a meaning of one leading word is kept", Meanings{"en": {"a"}}, "en", "a", true},
		{"no meanings", Meanings{}, "de", "x", false},
	}
	for _, c := range cases {
		if got := c.m.MeaningMatches(c.lang, c.input); got != c.want {
			t.Errorf("%s: MeaningMatches(%q, %q) = %v, want %v", c.name, c.lang, c.input, got, c.want)
		}
	}
}

func TestAnswerable(t *testing.T) {
	cases := []struct {
		name string
		m    Meanings
		lang string
		want bool
	}{
		{"nil", nil, "de", false},
		{"empty", Meanings{}, "de", false},
		{"meaning language", Meanings{"de": {"Regenschirm"}}, "de", true},
		{"English only", Meanings{"en": {"umbrella"}}, "de", true},
		{"language-less", Meanings{"": {"die Person dort drüben"}}, "fr", true},
		{"only brackets", Meanings{"en": {"(polite)", "[obsolete]"}}, "en", false},
		{"another language, as For falls back to it", Meanings{"hu": {"esernyő"}}, "de", true},
	}
	for _, c := range cases {
		if got := c.m.Answerable(c.lang); got != c.want {
			t.Errorf("%s: Answerable(%q) = %v, want %v", c.name, c.lang, got, c.want)
		}
	}
}
