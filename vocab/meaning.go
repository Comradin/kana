package vocab

import (
	"strings"
	"unicode"
)

// leadingWords may be left out or typed in front of a meaning: the English
// infinitive marker and the English and German articles.
var leadingWords = map[string]bool{
	"to": true, "the": true, "a": true, "an": true,
	"der": true, "die": true, "das": true, "ein": true, "eine": true,
}

// umlauts spells ä, ö, ü and ß as ae, oe, ue and ss, so either spelling
// matches.
var umlauts = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")

// MeaningMatches reports whether input is one of the meanings shown for lang
// (see For) or one of the English meanings. Case, text in parentheses or
// brackets, punctuation, extra spaces, the spelling of umlauts and a leading
// "to" or article do not matter; typos do.
func (m Meanings) MeaningMatches(lang, input string) bool {
	in := normalizeMeaning(input)
	if in == "" {
		return false
	}
	for _, items := range [][]string{m.For(lang), m["en"]} {
		for _, it := range items {
			if normalizeMeaning(it) == in {
				return true
			}
		}
	}
	return false
}

// normalizeMeaning reduces a meaning to the form MeaningMatches compares.
func normalizeMeaning(s string) string {
	s = umlauts.Replace(strings.ToLower(s))
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '(' || r == '[':
			depth++
		case r == ')' || r == ']':
			if depth > 0 {
				depth--
			}
		case depth > 0:
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	words := strings.Fields(b.String())
	if len(words) > 1 && leadingWords[words[0]] {
		words = words[1:]
	}
	return strings.Join(words, " ")
}
