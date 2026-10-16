package vocab

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/words.json.gz
var bundledData []byte

type bundledWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	R int                 `json:"r"`
	M map[string][]string `json:"m"`
}

// Bundled returns the word list shipped with the app (JMdict meanings, JLPT
// levels from Jonathan Waller's lists; see data/ATTRIBUTION.md).
func Bundled() ([]Word, error) {
	return decodeBundled(bundledData)
}

func decodeBundled(data []byte) ([]Word, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("bundled words: %w", err)
	}
	defer zr.Close()
	var raw []bundledWord
	if err := json.NewDecoder(zr).Decode(&raw); err != nil {
		return nil, fmt.Errorf("bundled words: %w", err)
	}
	words := make([]Word, len(raw))
	for i, r := range raw {
		m := Meanings(r.M)
		if m == nil {
			m = Meanings{}
		}
		words[i] = Word{Kana: r.K, JLPT: r.J, Rank: r.R, Meanings: m, Source: BundledSource}
	}
	return words, nil
}
