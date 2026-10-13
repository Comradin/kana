package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"fyne.io/fyne/v2/lang"
	"kana/kanacore"
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

// missedWordsText lists each missed word once, sorted, as
// "せんせい (sensei)", three per line so the game-over dialog stays narrow.
// "" when no word was missed.
func missedWordsText(words []string) string {
	seen := map[string]bool{}
	var uniq []string
	for _, w := range words {
		if !seen[w] {
			seen[w] = true
			uniq = append(uniq, w)
		}
	}
	sort.Strings(uniq)
	var lines []string
	for i := 0; i < len(uniq); i += 3 {
		end := min(i+3, len(uniq))
		parts := make([]string, 0, 3)
		for _, w := range uniq[i:end] {
			parts = append(parts, fmt.Sprintf("%s (%s)", w, kanacore.Romaji(w)))
		}
		lines = append(lines, strings.Join(parts, ", "))
	}
	return strings.Join(lines, "\n")
}

const (
	maxMeaningItems = 3  // meanings shown after a hit or miss
	maxMeaningRunes = 60 // longer meaning text ends in an ellipsis
)

// truncateRunes shortens s to at most n runes, ending in "…" when cut.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimRight(string(r[:n-1]), " ;,") + "…"
}

// meaningSnippet is the meaning text for feedback: the first three meanings
// in lang (with vocab's fallback to English and any language), cut to about
// 60 characters. "" when the word has no meanings.
func meaningSnippet(m vocab.Meanings, lang string) string {
	items := m.For(lang)
	if len(items) > maxMeaningItems {
		items = items[:maxMeaningItems]
	}
	return truncateRunes(strings.Join(items, "; "), maxMeaningRunes)
}

// hitFeedbackText is shown after a word hit: "はい — yes".
func hitFeedbackText(e vocab.Entry, lang string) string {
	if m := meaningSnippet(e.Meanings, lang); m != "" {
		return e.Kana + " — " + m
	}
	return e.Kana
}

// missFeedbackText is shown after a missed word, with the solution:
// "はい · hai — yes".
func missFeedbackText(e vocab.Entry, lang string) string {
	s := e.Kana + " · " + kanacore.Romaji(e.Kana)
	if m := meaningSnippet(e.Meanings, lang); m != "" {
		s += " — " + m
	}
	return s
}
