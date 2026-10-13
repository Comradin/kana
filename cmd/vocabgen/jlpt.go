package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// jlptRow is one row of the JLPT lists (yomitan-jlpt-vocab original_data).
type jlptRow struct {
	Seq    string
	Kana   string
	Level  int
	Waller string
}

// readJLPT reads every n1.csv … n5.csv in dir, easiest level (n5) first and
// each file in its own order, so the easiest level's meanings come first
// when words are merged. It fails if dir has no such file.
func readJLPT(dir string) ([]jlptRow, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(dir, "n[1-5].csv"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%s: no n1.csv … n5.csv", dir)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths))) // n5 … n1
	var rows []jlptRow
	for _, path := range paths {
		level := int(filepath.Base(path)[1] - '0')
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		recs, err := csv.NewReader(f).ReadAll()
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		for i, r := range recs {
			if i == 0 || len(r) < 4 {
				continue // header or short row
			}
			rows = append(rows, jlptRow{Seq: strings.TrimSpace(r[0]), Kana: strings.TrimSpace(r[1]), Level: level, Waller: r[3]})
		}
	}
	return rows, nil
}
