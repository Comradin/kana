package main

import (
	"compress/gzip"
	"encoding/xml"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"kana/vocab"
)

// langCodes maps JMdict's ISO 639-2 codes to ISO 639-1.
var langCodes = map[string]string{
	"eng": "en", "ger": "de", "fre": "fr", "rus": "ru", "spa": "es",
	"hun": "hu", "dut": "nl", "swe": "sv", "slv": "sl",
}

const maxGlossesPerLang = 3

type xmlGloss struct {
	Lang string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	Text string `xml:",chardata"`
}

type xmlEntry struct {
	Seq    string `xml:"ent_seq"`
	Senses []struct {
		Glosses []xmlGloss `xml:"gloss"`
	} `xml:"sense"`
}

var (
	// glossNumber is a leading "1) ", "1. ", "а) " (Cyrillic) or ": ".
	glossNumber = regexp.MustCompile(`^(?:\d+[.)]|[а-я]\))?\s*:?\s*`)
	// glossForm is a marker for a grammatical form of the word itself
	// ("{～な} неясный": as a na-adjective, "unclear"), not a derived phrase.
	glossForm    = regexp.MustCompile(`^\{～\[?(な|の|に|と|で|する|である|たる)\]?\}\s*`)
	glossFormAny = regexp.MustCompile(`\{～\[?(な|の|に|と|で|する|である|たる)\]?\}`)
	// glossBraces is a "{…}" group; one without "～" or "~" is a
	// clarification such as "{formalno}", one with them a derived phrase.
	glossBraces = regexp.MustCompile(`\{([^{}]*)\}`)
	// glossSourceLang is a trailing note on the word's origin: "(eng: scene)".
	glossSourceLang = regexp.MustCompile(`\s*\((?:eng|por|ger|fre|ita|lat|dut|spa|rus|chi|kor|san|ara|gre|heb|hin|mon|swe|fin|nor|dan|pol|tur|haw|may|tha|vie|ind)(?::[^)]*)?\)$`)
)

// cleanGloss tidies a non-English gloss, gently: it strips what is clearly
// markup and rejects a gloss it cannot make plain rather than guess.
//
// It strips, in any order and repeated, a leading "1) ", "1. ", "а) " or
// ": ", markers of the word's own grammatical form such as "{～な}" and
// "(…)" labels such as "(уст.)"; then a trailing source-language note such
// as "(eng: scene)". Other "{…}" clarifications become "(…)" first:
// "moški {formalno}" → "moški (formalno)".
//
// It rejects what is left if it is not a plain meaning: a "~" or "～" (the
// meaning of a derived phrase such as "{～をする} работать"), unbalanced
// parentheses, a leading "(" or "-", stray braces, kana or kanji, or fewer
// than two characters.
func cleanGloss(s string) (string, bool) {
	s = strings.TrimSpace(s)
	s = glossBraces.ReplaceAllStringFunc(s, func(g string) string {
		if glossForm.MatchString(g) || strings.ContainsAny(g, "~～") {
			return g // a form marker (stripped below) or a phrase (rejected)
		}
		return "(" + g[1:len(g)-1] + ")"
	})
	for {
		t := glossNumber.ReplaceAllString(s, "")
		t = strings.TrimSpace(glossForm.ReplaceAllString(stripLabel(t), ""))
		if t == s {
			break
		}
		s = t
	}
	s = strings.TrimSpace(glossSourceLang.ReplaceAllString(s, ""))
	if utf8.RuneCountInString(s) < 2 || strings.HasPrefix(s, "-") || strings.HasPrefix(s, "(") ||
		strings.ContainsAny(s, "{}~～") || strings.Count(s, "(") != strings.Count(s, ")") {
		return "", false
	}
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x30FF) || (r >= 0x4E00 && r <= 0x9FFF) {
			return "", false
		}
	}
	return s, true
}

// glossItems turns a gloss into meaning items. A meaning item never
// contains ";", the separator of private lists, so the gloss is split like
// they are. English glosses are clean and only split. The others may carry
// markup: a gloss with a "~" or "～" outside a form marker is the meaning of
// a derived phrase and rejected as a whole; otherwise each item is cleaned
// on its own, as each may start with its own label or numbering.
func glossItems(lang, gloss string) []string {
	gloss = strings.TrimSpace(gloss)
	if lang == "en" {
		return splitGloss(gloss)
	}
	if strings.ContainsAny(glossFormAny.ReplaceAllString(gloss, ""), "~～") {
		return nil
	}
	var out []string
	for _, it := range splitGloss(gloss) {
		if it, ok := cleanGloss(it); ok {
			out = append(out, it)
		}
	}
	return out
}

// splitGloss splits a gloss into meaning items like vocab.SplitMeanings. A
// ";" inside parentheses separates parts of a note, not meanings, and
// becomes ",": "y (junto con; al igual que)" stays one item.
func splitGloss(s string) []string {
	if !strings.Contains(s, ";") {
		return vocab.SplitMeanings(s)
	}
	b := []rune(s)
	depth := 0
	for i, r := range b {
		switch {
		case r == '(':
			depth++
		case r == ')' && depth > 0:
			depth--
		case r == ';' && depth > 0:
			b[i] = ','
		}
	}
	return vocab.SplitMeanings(string(b))
}

// stripLabel removes one leading parenthesised group, nested ones included.
// An unbalanced group is left in place.
func stripLabel(s string) string {
	if !strings.HasPrefix(s, "(") {
		return s
	}
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[i+1:]
			}
		}
	}
	return s
}

// readJMdict streams the JMdict XML (plain or .gz) and returns, for every
// entry whose sequence number is wanted, the glosses per language: those of
// the first sense that has usable glosses in that language, split on ";",
// at most three items.
func readJMdict(path string, wanted map[string]bool) (map[string]map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		zr, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		r = zr
	}
	dec := xml.NewDecoder(r)
	dec.Strict = false // JMdict uses DTD entities (&n; …); their text is not needed
	out := map[string]map[string][]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "entry" {
			continue
		}
		var e xmlEntry
		if err := dec.DecodeElement(&e, &se); err != nil {
			return nil, err
		}
		if !wanted[e.Seq] {
			continue
		}
		glosses := map[string][]string{}
		for _, sense := range e.Senses {
			// A language is taken only from the first sense that has
			// usable glosses in it.
			fresh := map[string]bool{}
			for _, g := range sense.Glosses {
				lang := g.Lang
				if lang == "" {
					lang = "eng"
				}
				code, ok := langCodes[lang]
				if !ok {
					continue
				}
				items := glossItems(code, g.Text)
				if len(items) == 0 {
					continue
				}
				if _, taken := glosses[code]; taken && !fresh[code] {
					continue
				}
				fresh[code] = true
				for _, it := range items {
					if len(glosses[code]) < maxGlossesPerLang && !has(glosses[code], it) {
						glosses[code] = append(glosses[code], it)
					}
				}
			}
		}
		out[e.Seq] = glosses
	}
}
