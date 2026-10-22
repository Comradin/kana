package vocab

import (
	"strings"
	"unicode"

	"kana/kanacore"
)

// variantSep separates the variants of a word in a kana field.
const variantSep = "//"

// notation is what a word list's kana field says: the reading, how the word
// is shown, and the other readings that count as answers.
type notation struct {
	reading string   // the last variant's full form
	display string   // the last variant as written; "" when it is the reading
	alts    []string // the other variants' and forms' readings, unique, in field order
	phrase  bool     // a variant has a placeholder, a bracket, punctuation or a space
}

// isPlaceholder reports whether r stands for something the learner fills
// in: shown, not typed.
func isPlaceholder(r rune) bool { return r == '～' || r == '〜' || r == '~' }

// isShownOnly reports whether r is punctuation or a space: shown, not typed.
func isShownOnly(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("？！。、・?!.,", r)
}

// isPhraseMark reports whether r makes a field a phrase: a placeholder,
// punctuation, a space or a bracket.
func isPhraseMark(r rune) bool {
	return isPlaceholder(r) || isShownOnly(r) || isOpenBracket(r) || isCloseBracket(r)
}

// isOpenBracket reports whether r opens an optional part.
func isOpenBracket(r rune) bool { return r == '（' || r == '(' }

// isCloseBracket reports whether r closes an optional part; either closing
// bracket closes either opening one.
func isCloseBracket(r rune) bool { return r == '）' || r == ')' }

// parseNotation reads a trimmed kana field: placeholders, punctuation and
// spaces are shown but not typed, text in brackets is optional (the full and
// the short form count), and "//" separates variants, of which the last is
// shown and gives the reading. A field whose variants use any of this
// except "//" is a phrase. It returns the notation, or why the row is
// skipped: "empty kana", or the field as written followed by "is not kana",
// "has unpaired brackets" or "has an empty variant". Every accepted reading
// must pass kanacore.KanaUnits.
func parseNotation(field string) (notation, string) {
	var n notation
	var readings []string
	variants := strings.Split(field, variantSep)
	for i, v := range variants {
		v = strings.TrimSpace(v)
		if strings.IndexFunc(v, isPhraseMark) >= 0 {
			n.phrase = true
		}
		full, short, ok := variantForms(v)
		if !ok {
			return notation{}, field + " has unpaired brackets"
		}
		if full == "" {
			if len(variants) == 1 {
				return notation{}, "empty kana"
			}
			return notation{}, field + " has an empty variant"
		}
		for _, r := range []string{full, short} {
			if r == "" {
				continue // all of the variant is optional: only the full form counts
			}
			if _, ok := kanacore.KanaUnits(r); !ok {
				return notation{}, field + " is not kana"
			}
			if !contains(readings, r) {
				readings = append(readings, r)
			}
		}
		if i == len(variants)-1 {
			n.reading = full
			if v != full {
				n.display = v
			}
		}
	}
	for _, r := range readings {
		if r != n.reading {
			n.alts = append(n.alts, r)
		}
	}
	return n, ""
}

// variantForms returns a variant's full form (bracket characters removed,
// their contents kept) and its short form (bracketed parts removed), both
// without placeholders, punctuation and spaces. It reports false for an
// unpaired or nested bracket.
func variantForms(v string) (full, short string, ok bool) {
	var f, s strings.Builder
	open := false
	for _, r := range v {
		switch {
		case isOpenBracket(r):
			if open {
				return "", "", false
			}
			open = true
		case isCloseBracket(r):
			if !open {
				return "", "", false
			}
			open = false
		case isPlaceholder(r) || isShownOnly(r):
		default:
			f.WriteRune(r)
			if !open {
				s.WriteRune(r)
			}
		}
	}
	if open {
		return "", "", false
	}
	return f.String(), s.String(), true
}
