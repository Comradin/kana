package kanacore

import "strings"

// Units that only occur inside words.
const (
	sokuonHira = "っ"
	sokuonKata = "ッ"
	longMark   = "ー"
)

// unitSpellings maps every kana known to a row to its canonical romaji
// followed by its alternatives.
var unitSpellings map[string][]string

func init() {
	unitSpellings = make(map[string][]string)
	for _, row := range AllKanaRows {
		for _, e := range row.Entries {
			unitSpellings[e.Char] = append([]string{e.Romaji}, e.Alt...)
		}
	}
}

func isSokuon(u string) bool { return u == sokuonHira || u == sokuonKata }
func isN(u string) bool      { return u == "ん" || u == "ン" }

func isVowel(b byte) bool { return strings.IndexByte("aiueo", b) >= 0 }

// KanaUnits splits a word into units: two-character combinations known to a
// row (きゃ, ティ), single kana, っ/ッ and ー. It reports false when a
// character cannot be placed or the word contains a sequence that cannot be
// typed (a leading, trailing or misplaced っ, a ー with no vowel before it).
func KanaUnits(word string) ([]string, bool) {
	runes := []rune(word)
	if len(runes) == 0 {
		return nil, false
	}
	var units []string
	for i := 0; i < len(runes); {
		if i+1 < len(runes) {
			if pair := string(runes[i : i+2]); unitSpellings[pair] != nil {
				units = append(units, pair)
				i += 2
				continue
			}
		}
		single := string(runes[i])
		if unitSpellings[single] == nil && !isSokuon(single) && single != longMark {
			return nil, false
		}
		units = append(units, single)
		i++
	}
	for i, u := range units {
		switch {
		case isSokuon(u):
			if i == 0 || i+1 >= len(units) {
				return nil, false
			}
			next := units[i+1]
			sp := unitSpellings[next]
			if sp == nil || isN(next) || isVowel(sp[0][0]) {
				return nil, false
			}
		case u == longMark:
			// ー after っ is already rejected by the っ rule above.
			if i == 0 || isN(units[i-1]) {
				return nil, false
			}
		}
	}
	return units, true
}
