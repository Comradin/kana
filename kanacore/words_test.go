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
