// Package vocab loads word lists and selects words a learner can read.
package vocab

import (
	"sort"
	"strconv"
	"strings"
)

// Meanings maps an ISO 639-1 language to its meanings in first-seen order.
// The key "" holds meanings given without a language; they apply to every
// language that has none of its own.
type Meanings map[string][]string

// Word is one entry of a word list.
type Word struct {
	Kana     string   // reading, e.g. "せんせい"
	Meanings Meanings // language → alternatives
	JLPT     int      // 5…1, 0 = none
	Source   string   // "jmdict" or a list name
	Lesson   string   // optional, private lists only
}

// List is one private word list file.
type List struct {
	Name     string
	Words    []Word
	Problems []string // at most maxProblems messages
	Skipped  int      // rows skipped, including those without a message
}

// BundledSource is the Source of bundled words and a reserved list name.
const BundledSource = "jmdict"

const maxProblems = 10

// SplitMeanings splits a field on ";", trims items and drops empty and
// duplicate ones, keeping first-seen order.
func SplitMeanings(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(s, ";") {
		p := strings.TrimSpace(part)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// Add appends items for lang, skipping ones already present.
func (m Meanings) Add(lang string, items ...string) {
	for _, it := range items {
		dup := false
		for _, have := range m[lang] {
			if have == it {
				dup = true
				break
			}
		}
		if !dup {
			m[lang] = append(m[lang], it)
		}
	}
}

// Merge adds every language of other to m.
func (m Meanings) Merge(other Meanings) {
	for lang, items := range other {
		m.Add(lang, items...)
	}
}

// For returns the meanings to show for lang: lang itself, then the
// language-less meanings, then English, then the first language in
// alphabetical order. Nil if there are none.
func (m Meanings) For(lang string) []string {
	for _, k := range []string{lang, "", "en"} {
		if len(m[k]) > 0 {
			return m[k]
		}
	}
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if len(v) > 0 {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	return m[keys[0]]
}

// Display joins For(lang) with "; ".
func (m Meanings) Display(lang string) string {
	return strings.Join(m.For(lang), "; ")
}

// parseLevel reads a JLPT level written as "5" or "N5"; anything else is 0.
func parseLevel(s string) int {
	s = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(s)), "N")
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 5 {
		return 0
	}
	return n
}
