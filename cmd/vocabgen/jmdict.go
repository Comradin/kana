package main

import (
	"compress/gzip"
	"encoding/xml"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
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
	glossNumber = regexp.MustCompile(`^(\d+\))?\s*:?\s*`)
	// glossForm is a marker for a grammatical form of the word itself
	// ("{～な} неясный": as a na-adjective, "unclear"), not a derived phrase.
	glossForm = regexp.MustCompile(`^\{～\[?(な|の|に|と|で|する|である|たる)\]?\}\s*`)
)

// cleanGloss tidies a non-English gloss. It strips a leading "1) " or ": ",
// leading "(…)" labels such as "(уст.)" and markers of the word's own
// grammatical form such as "{～な}". It rejects what is left if it is not a
// plain meaning: another "{…}" pattern (the meaning of a derived phrase such
// as "{～をする}"), an unbalanced "(", a "-" affix, text with kana, kanji or
// stray braces, or fewer than two characters.
func cleanGloss(s string) (string, bool) {
	s = glossNumber.ReplaceAllString(strings.TrimSpace(s), "")
	for {
		t := strings.TrimSpace(glossForm.ReplaceAllString(stripLabel(s), ""))
		if t == s {
			break
		}
		s = t
	}
	if utf8.RuneCountInString(s) < 2 || strings.HasPrefix(s, "-") || strings.HasPrefix(s, "(") || strings.ContainsAny(s, "{}") {
		return "", false
	}
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x30FF) || (r >= 0x4E00 && r <= 0x9FFF) {
			return "", false
		}
	}
	return s, true
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
// the first sense that has usable glosses in that language, at most three.
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
				// English glosses are clean; the others may carry markup.
				text := strings.TrimSpace(g.Text)
				if code != "en" {
					text, ok = cleanGloss(text)
				}
				if !ok || text == "" {
					continue
				}
				if _, taken := glosses[code]; taken && !fresh[code] {
					continue
				}
				fresh[code] = true
				if len(glosses[code]) < maxGlossesPerLang {
					glosses[code] = append(glosses[code], text)
				}
			}
		}
		out[e.Seq] = glosses
	}
}
