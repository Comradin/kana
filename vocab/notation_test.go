package vocab

import (
	"reflect"
	"testing"
)

func TestParseNotation(t *testing.T) {
	cases := []struct {
		field string
		want  notation
	}{
		{"せんせい", notation{reading: "せんせい"}},
		// placeholders
		{"～さん", notation{reading: "さん", display: "～さん"}},
		{"〜さん", notation{reading: "さん", display: "〜さん"}},
		{"~さん", notation{reading: "さん", display: "~さん"}},
		{"こちらは～さんです", notation{reading: "こちらはさんです", display: "こちらは～さんです"}},
		// punctuation and spaces
		{"おなまえは？", notation{reading: "おなまえは", display: "おなまえは？"}},
		{"はい、そうです。", notation{reading: "はいそうです", display: "はい、そうです。"}},
		{"どうぞ よろしく", notation{reading: "どうぞよろしく", display: "どうぞ よろしく"}},
		{"どうぞ　よろしく", notation{reading: "どうぞよろしく", display: "どうぞ　よろしく"}},
		// optional parts: the full and the short form, nothing in between
		{"（どうぞ）よろしく（おねがいします）", notation{reading: "どうぞよろしくおねがいします",
			display: "（どうぞ）よろしく（おねがいします）", alts: []string{"よろしく"}}},
		{"(どうぞ)よろしく", notation{reading: "どうぞよろしく", display: "(どうぞ)よろしく", alts: []string{"よろしく"}}},
		{"（どうぞ)よろしく", notation{reading: "どうぞよろしく", display: "（どうぞ)よろしく", alts: []string{"よろしく"}}},
		// a placeholder with an optional part; a placeholder in brackets
		{"（お）～さん", notation{reading: "おさん", display: "（お）～さん", alts: []string{"さん"}}},
		{"（～さん）", notation{reading: "さん", display: "（～さん）"}},
		// variants: the last one is shown and is the reading
		{"あめりか // アメリカ", notation{reading: "アメリカ", alts: []string{"あめりか"}}},
		{"あめりか//アメリカ", notation{reading: "アメリカ", alts: []string{"あめりか"}}},
		{"ろうそく // ローソク", notation{reading: "ローソク", alts: []string{"ろうそく"}}},
		{"～くん // ～クン", notation{reading: "クン", display: "～クン", alts: []string{"くん"}}},
		{"（どうぞ）よろしく // よろしく", notation{reading: "よろしく", alts: []string{"どうぞよろしく"}}},
		{"ねこ // ねこ", notation{reading: "ねこ"}},
	}
	for _, c := range cases {
		got, reason := parseNotation(c.field)
		if reason != "" || !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseNotation(%q) = %+v, %q; want %+v", c.field, got, reason, c.want)
		}
	}
}

func TestParseNotationSkips(t *testing.T) {
	cases := map[string]string{
		"":           "empty kana",
		"～":          "empty kana",
		"？ 。":        "empty kana",
		"（）":         "empty kana",
		"（どうぞよろしく":   "（どうぞよろしく has unpaired brackets",
		"どうぞ）よろしく":   "どうぞ）よろしく has unpaired brackets",
		"（どう（ぞ））":    "（どう（ぞ）） has unpaired brackets",
		"先生":         "先生 is not kana",
		"sensei":     "sensei is not kana",
		"せんせい // 先生": "せんせい // 先生 is not kana",
		"先生 // せんせい": "先生 // せんせい is not kana",
		"ん（い）ー":      "ん（い）ー is not kana", // the short form んー cannot be typed
		"あめりか //":    "あめりか // has an empty variant",
		"// アメリカ":    "// アメリカ has an empty variant",
	}
	for field, want := range cases {
		if got, reason := parseNotation(field); reason != want {
			t.Errorf("parseNotation(%q) = %+v, %q; want reason %q", field, got, reason, want)
		}
	}
}
