package main

import (
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2/lang"
	"kana/store"
	"kana/vocab"
)

// meaningLangs are the languages the bundled word list carries.
var meaningLangs = []string{"en", "de", "fr", "ru", "es", "hu", "nl", "sv", "sl"}

// loadBundled decodes the bundled word list once per process.
var loadBundled = sync.OnceValues(vocab.Bundled)

// bundledWords returns the bundled word list. Tests replace it to simulate
// a broken list.
var bundledWords = loadBundled

// bundledIndex is the index of the bundled word list, built once per
// process, so a new GameState only merges the private lists into it.
// loadWords uses it whenever bundledWords succeeds.
var bundledIndex = sync.OnceValue(func() *vocab.Index {
	words, _ := loadBundled()
	return vocab.NewIndex(words, nil)
})

// systemLanguage returns the system locale, e.g. "de-DE". Tests replace it.
var systemLanguage = func() string { return string(lang.SystemLocale()) }

// wordsDir is the private word-list folder next to the database.
func wordsDir() (string, error) {
	p, err := store.DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), "words"), nil
}

// resolveMeaningLang returns the stored meaning language, or, when none is
// stored, the system language if the word list has it, else English.
func resolveMeaningLang(stored, system string) string {
	if stored != "" {
		return stored
	}
	base := strings.ToLower(system)
	if i := strings.IndexAny(base, "-_."); i >= 0 {
		base = base[:i]
	}
	for _, l := range meaningLangs {
		if l == base {
			return l
		}
	}
	return "en"
}
