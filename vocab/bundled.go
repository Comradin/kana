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
	M map[string][]string `json:"m"`
}

// Bundled returns the word list shipped with the app (JMdict meanings, JLPT
// levels from Jonathan Waller's lists; see data/ATTRIBUTION.md).
func Bundled() ([]Word, error) {
	zr, err := gzip.NewReader(bytes.NewReader(bundledData))
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
		words[i] = Word{Kana: r.K, JLPT: r.J, Meanings: Meanings(r.M), Source: BundledSource}
	}
	return words, nil
}
