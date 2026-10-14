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

// spellingsOf returns the accepted spellings of a normal unit; ん also
// accepts n'.
func spellingsOf(u string) []string {
	sp := unitSpellings[u]
	if isN(u) {
		return append(append([]string(nil), sp...), "n'")
	}
	return sp
}

// lastVowel returns the final vowel of a spelling, or 0 if it ends in a
// consonant or apostrophe.
func lastVowel(s string) byte {
	if s == "" || !isVowel(s[len(s)-1]) {
		return 0
	}
	return s[len(s)-1]
}

// ReadingMatches reports whether input is a valid romaji reading of word:
// each unit by one of its spellings, っ as a doubled first letter (or t
// before ch), ー as the previous vowel or "-". Input is trimmed and
// lower-cased first, and a typographic apostrophe (’) counts as '.
func ReadingMatches(word, input string) bool {
	units, ok := KanaUnits(word)
	if !ok {
		return false
	}
	// Whitespace anywhere is ignored, so "shitsurei desu ga" matches too.
	in := strings.ReplaceAll(strings.Join(strings.Fields(strings.ToLower(input)), ""), "’", "'")
	if in == "" {
		return false
	}
	type state struct {
		unit, pos int
		vowel     byte
	}
	failed := make(map[state]bool)
	var match func(i, pos int, vowel byte) bool
	match = func(i, pos int, vowel byte) bool {
		if i == len(units) {
			return pos == len(in)
		}
		st := state{i, pos, vowel}
		if failed[st] {
			return false
		}
		rest := in[pos:]
		u := units[i]
		switch {
		case u == longMark:
			if strings.HasPrefix(rest, "-") && match(i+1, pos+1, vowel) {
				return true
			}
			if vowel != 0 && rest != "" && rest[0] == vowel && match(i+1, pos+1, vowel) {
				return true
			}
		case isSokuon(u):
			for _, s := range spellingsOf(units[i+1]) {
				prefixes := []string{s[:1]}
				if strings.HasPrefix(s, "ch") {
					prefixes = append(prefixes, "t")
				}
				for _, p := range prefixes {
					if strings.HasPrefix(rest, p+s) && match(i+2, pos+len(p)+len(s), lastVowel(s)) {
						return true
					}
				}
			}
		default:
			for _, s := range spellingsOf(u) {
				if strings.HasPrefix(rest, s) && match(i+1, pos+len(s), lastVowel(s)) {
					return true
				}
			}
		}
		failed[st] = true
		return false
	}
	return match(0, 0, 0)
}

// Romaji returns the canonical romaji of word, or "" if it cannot be split
// into units: っ doubles the next consonant (t before ch), ー repeats the
// previous vowel, ん is n' before a vowel or y.
func Romaji(word string) string {
	units, ok := KanaUnits(word)
	if !ok {
		return ""
	}
	var b strings.Builder
	var vowel byte
	for i, u := range units {
		switch {
		case u == longMark:
			b.WriteByte(vowel)
		case isSokuon(u):
			next := unitSpellings[units[i+1]][0]
			if strings.HasPrefix(next, "ch") {
				b.WriteByte('t')
			} else {
				b.WriteByte(next[0])
			}
		case isN(u):
			b.WriteString("n")
			if i+1 < len(units) {
				if sp := unitSpellings[units[i+1]]; sp != nil && (isVowel(sp[0][0]) || sp[0][0] == 'y') {
					b.WriteString("'")
				}
			}
			vowel = 0
		default:
			s := unitSpellings[u][0]
			b.WriteString(s)
			vowel = lastVowel(s)
		}
	}
	return b.String()
}
