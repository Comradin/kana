package kanacore

import (
	"reflect"
	"testing"
)

func TestKanaUnits(t *testing.T) {
	ok := []struct {
		word string
		want []string
	}{
		{"きゃく", []string{"きゃ", "く"}},
		{"パーティー", []string{"パ", "ー", "ティ", "ー"}},
		{"ヴァイオリン", []string{"ヴァ", "イ", "オ", "リ", "ン"}},
		{"きって", []string{"き", "っ", "て"}},
		{"まっちゃ", []string{"ま", "っ", "ちゃ"}},
		{"こーー", []string{"こ", "ー", "ー"}},
		{"すげー", []string{"す", "げ", "ー"}},
	}
	for _, c := range ok {
		got, valid := KanaUnits(c.word)
		if !valid || !reflect.DeepEqual(got, c.want) {
			t.Errorf("KanaUnits(%s) = %v, %v; want %v", c.word, got, valid, c.want)
		}
	}
	bad := []string{
		"",     // empty
		"先生",   // kanji
		"ゐ",    // unsupported kana
		"ァ",    // lone small kana
		"ーあ",   // leading long mark
		"あっ",   // trailing sokuon
		"あっあ",  // sokuon before a vowel
		"あっん",  // sokuon before ん
		"あっー",  // sokuon before ー (also covers ー after っ)
		"っか",   // leading sokuon
		"かんー",  // long mark after ん
		"kana", // latin
	}
	for _, w := range bad {
		if _, valid := KanaUnits(w); valid {
			t.Errorf("KanaUnits(%q) should fail", w)
		}
	}
}

func TestReadingMatches(t *testing.T) {
	yes := map[string][]string{
		"きって":    {"kitte", " KITTE "},
		"まっちゃ":   {"matcha", "maccha", "mattya"},
		"コーヒー":   {"koohii", "ko-hi-", "ko-hii"},
		"でんしゃ":   {"densha", "densya", "dennsha", "den'sha"},
		"きんえん":   {"kinen", "kin'en", "kinnen"},
		"パーティー":  {"paatii", "paathii", "pa-ti-"},
		"とうきょう":  {"toukyou"},
		"ふじさん":   {"fujisan", "huzisan"},
		"こーー":    {"kooo", "ko--", "koo-"},
		"ヴァイオリン": {"vaiorin"},
		"はい":     {"hai"},
		"バッティング": {"battingu", "batthingu"},
		"きゃっきゃ":  {"kyakkya"},
		"ほんや":    {"honya", "hon'ya", "honnya", "hon’ya"},
		"こんにゃく":  {"konnyaku", "kon'nyaku"},
		"ウォッカ":   {"wokka", "whokka"},
	}
	for word, inputs := range yes {
		for _, in := range inputs {
			if !ReadingMatches(word, in) {
				t.Errorf("ReadingMatches(%s, %q) = false, want true", word, in)
			}
		}
	}
	no := map[string][]string{
		"とうきょう": {"tokyo", "toukyo"},
		"きって":   {"kite", "kitt", "kittee", ""},
		"はい":    {"ha", "haii", "hi"},
		"コーヒー":  {"kohi", "kouhii"},
		"先生":    {"sensei"},
	}
	for word, inputs := range no {
		for _, in := range inputs {
			if ReadingMatches(word, in) {
				t.Errorf("ReadingMatches(%s, %q) = true, want false", word, in)
			}
		}
	}
}

func TestRomaji(t *testing.T) {
	cases := map[string]string{
		"きって":    "kitte",
		"まっちゃ":   "matcha",
		"コーヒー":   "koohii",
		"きんえん":   "kin'en",
		"きんよう":   "kin'you",
		"でんしゃ":   "densha",
		"パーティー":  "paatii",
		"とうきょう":  "toukyou",
		"ヴァイオリン": "vaiorin",
		"バッティング": "battingu",
		"ほんや":    "hon'ya",
		"先生":     "",
	}
	for word, want := range cases {
		if got := Romaji(word); got != want {
			t.Errorf("Romaji(%s) = %q, want %q", word, got, want)
		}
	}
}
