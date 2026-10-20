package vocab

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"kana/kanacore"
)

// problemNoKanaColumn is the problem of a header without a kana column.
const problemNoKanaColumn = `no "kana" column in the header`

// headerAliases maps other (lower-case) column names to the canonical ones,
// so lists exported from other tools load without renaming.
var headerAliases = map[string]string{
	"japanisch":          "kana",
	"japanische schrift": "kana",
	"reading":            "kana",
	"deutsch":            "meaning_de",
	"german":             "meaning_de",
	"englisch":           "meaning_en",
	"english":            "meaning_en",
	"lektion":            "lesson",
}

// readCSV reads a word list with a header row. The separator (tab,
// semicolon or comma) is taken from the header line.
func readCSV(name string, r io.Reader) List {
	l := List{Name: name}
	br := bufio.NewReader(r)
	header, err := br.ReadString('\n')
	if err != nil && header == "" {
		l.problem("file is empty")
		return l
	}
	header = strings.TrimPrefix(header, "\ufeff")
	sep := ','
	switch {
	case strings.Contains(header, "\t"):
		sep = '\t'
	case strings.Contains(header, ";"):
		sep = ';'
	}
	cr := csv.NewReader(io.MultiReader(strings.NewReader(header), br))
	cr.Comma = sep
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cols, err := cr.Read()
	if err != nil {
		l.problem(fmt.Sprintf("header: %v", err))
		return l
	}
	// Column names are case-insensitive; of duplicate names the first wins.
	for i, c := range cols {
		cols[i] = strings.ToLower(strings.TrimSpace(c))
	}
	// An alias stands for its canonical name unless that column exists.
	present := map[string]bool{}
	for _, c := range cols {
		present[c] = true
	}
	for i, c := range cols {
		if canon, ok := headerAliases[c]; ok && !present[canon] {
			cols[i] = canon
		}
	}
	idx := map[string]int{}
	for i, c := range cols {
		if _, dup := idx[c]; !dup {
			idx[c] = i
		}
	}
	kanaCol, ok := idx["kana"]
	if !ok {
		l.problem(problemNoKanaColumn)
		return l
	}
	field := func(rec []string, col string) string {
		if i, ok := idx[col]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				l.skip(fmt.Sprintf("line %d: %v", pe.StartLine, pe.Err))
				continue
			}
			l.problem(fmt.Sprintf("read: %v", err))
			break
		}
		line, _ := cr.FieldPos(0)
		kana := ""
		if kanaCol < len(rec) {
			kana = strings.TrimSpace(rec[kanaCol])
		}
		if kana == "" {
			l.skip(fmt.Sprintf("line %d: empty kana", line))
			continue
		}
		if _, ok := kanacore.KanaUnits(kana); !ok {
			l.skip(fmt.Sprintf("line %d: %s is not kana", line, kana))
			continue
		}
		w := Word{Kana: kana, Meanings: Meanings{}, Source: name,
			Lesson: field(rec, "lesson"), JLPT: parseLevel(field(rec, "jlpt"))}
		// Meanings in header order, so that the result is deterministic.
		for i, col := range cols {
			if i >= len(rec) || idx[col] != i {
				continue
			}
			switch {
			case col == "meaning":
				w.Meanings.Add("", SplitMeanings(rec[i])...)
			case strings.HasPrefix(col, "meaning_") && col != "meaning_":
				lang := strings.TrimPrefix(col, "meaning_")
				w.Meanings.Add(lang, SplitMeanings(rec[i])...)
			}
		}
		l.Words = append(l.Words, w)
	}
	return l
}

// problem records a message without counting a skipped row.
func (l *List) problem(msg string) {
	if len(l.Problems) < maxProblems {
		l.Problems = append(l.Problems, msg)
	}
}

// skip counts a skipped row and records its message.
func (l *List) skip(msg string) {
	l.Skipped++
	l.problem(msg)
}
