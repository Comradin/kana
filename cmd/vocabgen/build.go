package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"kana/kanacore"
	"kana/vocab"
)

// maxGlossesMerged caps the meanings per language of a merged word.
const maxGlossesMerged = 5

// Stats reports what the generator kept and dropped.
type Stats struct {
	Rows       int // JLPT rows read
	Words      int // words written
	Dropped    int // readings KanaUnits rejects
	MissingSeq int // JLPT sequence numbers not found in JMdict
	NoSeq      int // JLPT rows without a sequence number (skipped)
}

type outWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	R int                 `json:"r"` // frequency rank, lower = more common
	M map[string][]string `json:"m"`
}

// run builds the bundled word list from a JMdict file and a JLPT directory
// and writes it gzip-compressed to w.
func run(jmdictPath, jlptDir string, w io.Writer) (Stats, error) {
	var st Stats
	rows, err := readJLPT(jlptDir)
	if err != nil {
		return st, err
	}
	st.Rows = len(rows)
	wanted := map[string]bool{}
	for _, r := range rows {
		if r.Seq != "" {
			wanted[r.Seq] = true
		}
	}
	entries, err := readJMdict(jmdictPath, wanted)
	if err != nil {
		return st, err
	}
	missing := map[string]bool{}
	byKana := map[string]*outWord{}
	dropped := map[string]bool{}
	for _, r := range rows {
		if r.Seq == "" {
			st.NoSeq++
			continue
		}
		if _, ok := kanacore.KanaUnits(r.Kana); !ok {
			dropped[r.Kana] = true
			continue
		}
		var g map[string][]string
		rank := rankNone
		if e, ok := entries[r.Seq]; ok {
			g, rank = e.glosses, e.rank(r.Kana)
		} else {
			missing[r.Seq] = true
			g = map[string][]string{"en": wallerGlosses(r.Waller)}
		}
		ow, ok := byKana[r.Kana]
		if !ok {
			ow = &outWord{K: r.Kana, J: r.Level, R: rank, M: map[string][]string{}}
			byKana[r.Kana] = ow
		}
		if r.Level > ow.J {
			ow.J = r.Level
		}
		ow.R = min(ow.R, rank) // merged homophones take the best rank
		for lang, glosses := range g {
			for _, gl := range glosses {
				// Already split for JMdict; Waller's fallback may need it.
				for _, it := range vocab.SplitMeanings(gl) {
					if len(ow.M[lang]) < maxGlossesMerged && !has(ow.M[lang], it) {
						ow.M[lang] = append(ow.M[lang], it)
					}
				}
			}
		}
	}
	st.Dropped = len(dropped)
	st.MissingSeq = len(missing)
	out := make([]outWord, 0, len(byKana))
	for _, ow := range byKana {
		out = append(out, *ow)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].K < out[j].K })
	st.Words = len(out)
	zw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return st, err
	}
	// Zero header (no name, zero ModTime) keeps the output reproducible.
	if err := json.NewEncoder(zw).Encode(out); err != nil {
		return st, err
	}
	if err := zw.Close(); err != nil {
		return st, err
	}
	return st, nil
}

// wallerGlosses splits Waller's comma-separated definition.
func wallerGlosses(def string) []string {
	var out []string
	for _, p := range strings.Split(def, ",") {
		if p = strings.TrimSpace(p); p != "" && len(out) < 3 {
			out = append(out, p)
		}
	}
	return out
}

func has(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func (s Stats) String() string {
	return fmt.Sprintf("%d JLPT rows, %d words written, %d readings dropped (not typeable), %d sequence numbers missing in JMdict, %d rows without sequence number", s.Rows, s.Words, s.Dropped, s.MissingSeq, s.NoSeq)
}
