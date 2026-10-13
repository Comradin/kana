package vocab

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"kana/kanacore"
)

type jsonWord struct {
	Kana    string          `json:"kana"`
	Meaning json.RawMessage `json:"meaning"`
	Lesson  json.RawMessage `json:"lesson"`
	JLPT    json.RawMessage `json:"jlpt"`
}

// readJSON reads a word list: an array of objects with kana, meaning
// (string, array of strings, or an object of language → string or array),
// lesson and jlpt (number or string).
func readJSON(name string, r io.Reader) List {
	l := List{Name: name}
	var raw []jsonWord
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		l.problem(fmt.Sprintf("invalid JSON: %v", err))
		return l
	}
	for i, jw := range raw {
		kana := strings.TrimSpace(jw.Kana)
		if _, ok := kanacore.KanaUnits(kana); !ok {
			l.skip(fmt.Sprintf("entry %d: %s is not kana", i+1, kana))
			continue
		}
		w := Word{Kana: kana, Meanings: Meanings{}, Source: name,
			Lesson: scalar(jw.Lesson), JLPT: parseLevel(scalar(jw.JLPT))}
		addJSONMeaning(w.Meanings, "", jw.Meaning)
		l.Words = append(l.Words, w)
	}
	return l
}

// scalar renders a JSON string or number as text; anything else is "".
func scalar(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

func addJSONMeaning(m Meanings, lang string, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		m.Add(lang, SplitMeanings(s)...)
		return
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		for _, item := range list {
			m.Add(lang, SplitMeanings(item)...)
		}
		return
	}
	if lang != "" {
		return // no nested objects
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			addJSONMeaning(m, strings.ToLower(k), obj[k])
		}
	}
}
