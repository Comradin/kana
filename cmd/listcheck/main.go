// Command listcheck checks word list files the way the game's import does,
// without importing them, and prints every skipped row.
//
//	go run ./cmd/listcheck [-words] FILE...
//
// For each file it prints the word list's name with its number of words and
// skipped rows, with -words every word (shown form, romaji, other accepted
// readings, lesson), then every skipped row's message (all of them; the
// game keeps the first 100), or why the file cannot be imported, what to do
// about it and every skipped row's message. The exit status is 1 when a file
// cannot be read or imported, 2 for a usage error.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"kana/kanacore"
	"kana/vocab"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run checks the files named in args, reporting to stdout, and returns the
// exit status. Usage errors go to stderr.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("listcheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	words := flags.Bool("words", false, "also print every word: shown form, romaji, other readings, lesson")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: listcheck [-words] FILE...")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() == 0 {
		flags.Usage()
		return 2
	}
	status := 0
	for i, path := range flags.Args() {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		if !checkFile(stdout, path, *words) {
			status = 1
		}
	}
	return status
}

// checkFile checks one file as the import does and reports on it to w: the
// word list's name and counts, with words every word, then every message;
// for a file that cannot be imported why, the hint and every message. It
// returns false when the file cannot be read or imported.
func checkFile(w io.Writer, path string, words bool) bool {
	data, err := readFile(path)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err // the path is printed already
		}
		fmt.Fprintf(w, "%s: cannot read: %v\n", path, err)
		return false
	}
	c, err := vocab.InspectAll(path, data)
	if err != nil {
		fmt.Fprintf(w, "%s: cannot import: %v\n", path, err)
		var ie *vocab.ImportError
		if errors.As(err, &ie) {
			if ie.Hint != "" {
				fmt.Fprintf(w, "  %s\n", ie.Hint)
			}
			printProblems(w, ie.List.Problems)
		}
		return false
	}
	l := c.List
	fmt.Fprintf(w, "%s: word list %q, %s, %s skipped\n", path, c.Name,
		plural(len(l.Words), "word"), plural(l.Skipped, "row"))
	if words {
		fmt.Fprintln(w, "Words:")
		for _, word := range l.Words {
			fmt.Fprintf(w, "  %s\n", wordLine(word))
		}
	}
	printProblems(w, l.Problems)
	return true
}

// printProblems prints every skipped row's message under a heading, if any.
func printProblems(w io.Writer, problems []string) {
	if len(problems) == 0 {
		return
	}
	fmt.Fprintln(w, "Skipped rows:")
	for _, p := range problems {
		fmt.Fprintf(w, "  %s\n", p)
	}
}

// readFile reads at most one byte more than an import accepts, so that
// Inspect can tell a file is too large.
func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, vocab.MaxImportSize+1))
}

// wordLine describes a word for -words: its shown form, the romaji of its
// reading, its other accepted readings and its lesson, as
// "（どうぞ）よろしく · douzoyoroshiku · also よろしく (yoroshiku) · lesson 1".
// Other readings are given in kana, since variants often share the romaji.
func wordLine(w vocab.Word) string {
	parts := []string{w.Shown(), kanacore.Romaji(w.Kana)}
	if len(w.Alts) > 0 {
		alts := make([]string, len(w.Alts))
		for i, a := range w.Alts {
			alts[i] = fmt.Sprintf("%s (%s)", a, kanacore.Romaji(a))
		}
		parts = append(parts, "also "+strings.Join(alts, ", "))
	}
	if w.Lesson != "" {
		parts = append(parts, "lesson "+w.Lesson)
	}
	return strings.Join(parts, " · ")
}

// plural is "1 word" or "4 words".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
