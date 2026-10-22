package main

import (
	"fmt"
	"path/filepath"
	"slices"
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

// bundledWords reports whether the bundled word list loaded. It is a test
// hook for error injection only: tests replace it to simulate a broken
// list. The words themselves always come from bundledIndex.
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

// languageNames are meaning languages by their own names.
var languageNames = map[string]string{
	"cs": "Čeština", "da": "Dansk", "de": "Deutsch", "en": "English", "es": "Español",
	"fi": "Suomi", "fr": "Français", "hu": "Magyar", "it": "Italiano", "ja": "日本語",
	"ko": "한국어", "nl": "Nederlands", "no": "Norsk", "pl": "Polski", "pt": "Português",
	"ru": "Русский", "sl": "Slovenščina", "sv": "Svenska", "tr": "Türkçe", "uk": "Українська",
	"zh": "中文",
}

// languageName names a meaning language in itself; an unknown code is
// shown as is.
func languageName(code string) string {
	if n, ok := languageNames[code]; ok {
		return n
	}
	return code
}

// meaningLangOptions are the "Meanings in" select's labels and values:
// "System" (value "") first, then the languages by code, each by its own
// name. A stored language the words no longer have stays as an option.
func meaningLangOptions(langs []string, stored string) (labels, values []string) {
	all := slices.Clone(langs)
	if stored != "" && !slices.Contains(all, stored) {
		all = append(all, stored)
		slices.Sort(all)
	}
	labels, values = []string{"System"}, []string{""}
	for _, l := range all {
		labels = append(labels, languageName(l))
		values = append(values, l)
	}
	return labels, values
}

// missedWordsText lists each missed word once, sorted by reading, in its
// shown form with the romaji of its reading, as "～さん (san)", three per
// line so the game-over dialog stays narrow. "" when no word was missed.
func missedWordsText(words []vocab.Entry) string {
	seen := map[string]bool{}
	var uniq []vocab.Entry
	for _, e := range words {
		if !seen[e.Kana] {
			seen[e.Kana] = true
			uniq = append(uniq, e)
		}
	}
	sort.Slice(uniq, func(i, j int) bool { return uniq[i].Kana < uniq[j].Kana })
	var lines []string
	for i := 0; i < len(uniq); i += 3 {
		end := min(i+3, len(uniq))
		parts := make([]string, 0, 3)
		for _, e := range uniq[i:end] {
			parts = append(parts, fmt.Sprintf("%s (%s)", e.Shown(), kanacore.Romaji(e.Kana)))
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

// hitFeedbackText is shown after a word hit, the word in its shown form:
// "はい — yes", or with the romaji of its reading, "はい · hai — yes", when
// the word was answered by its meaning.
func hitFeedbackText(e vocab.Entry, lang string, byMeaning bool) string {
	s := e.Shown()
	if byMeaning {
		s += " · " + kanacore.Romaji(e.Kana)
	}
	if m := meaningSnippet(e.Meanings, lang); m != "" {
		s += " — " + m
	}
	return s
}

// missFeedbackText is shown after a missed word, with the solution: the
// shown form and the romaji of the reading, "～さん · san — Mr./Ms.".
func missFeedbackText(e vocab.Entry, lang string) string {
	s := e.Shown() + " · " + kanacore.Romaji(e.Kana)
	if m := meaningSnippet(e.Meanings, lang); m != "" {
		s += " — " + m
	}
	return s
}

// maxWordsLineRunes is a safety cap on the words line; the status panel
// fits the line to its width (see StatsPanel.Update).
const maxWordsLineRunes = 200

// Word sources as stored in the word_source setting.
const (
	jlptSource       = "jlpt"  // the bundled JLPT words
	listSourcePrefix = "list:" // followed by a private list's name
)

// resolveWordSource returns the stored word source, or, when none (or no
// valid one) is stored, the first private list by name, else JLPT.
func resolveWordSource(stored string, lists []string) string {
	if stored == jlptSource || (strings.HasPrefix(stored, listSourcePrefix) && stored != listSourcePrefix) {
		return stored
	}
	if len(lists) > 0 {
		return listSourcePrefix + lists[0]
	}
	return jlptSource
}

// prettyListName shows a list name with spaces: "minna-no-nihongo" →
// "minna no nihongo".
func prettyListName(name string) string {
	return strings.NewReplacer("-", " ", "_", " ").Replace(name)
}

// sourceLabel names a source for display: "JLPT N5" or a pretty list name.
func sourceLabel(src vocab.Source) string {
	if src.List != "" {
		return prettyListName(src.List)
	}
	return fmt.Sprintf("JLPT N%d", src.Level)
}

// wordsNext says whether new words can be introduced from the source.
type wordsNext int

const (
	wordsMore     wordsNext = iota // the source has a word that can be introduced now
	wordsNeedKana                  // words are left, but none is playable yet
	wordsFinished                  // every word of the source is introduced
)

// WordsStatus is the status panel's words line.
type WordsStatus struct {
	LoadFailed bool      // the introduced words could not be loaded
	Err        string    // the bundled list failed to load (JLPT source only)
	Missing    bool      // the chosen list was not found
	Dir        string    // the private list folder for display; "" if unknown
	Source     string    // "JLPT N5" or a pretty list name
	Level      int       // the JLPT level of a JLPT source; 0 for a list
	Introduced int       // introduced readings in the word index
	Learned    int       // introduced readings at level 5
	Waiting    int       // introduced readings that cannot fall: a kana is not learned
	Next       wordsNext // whether new words can come from the source
	Skipped    int       // rows skipped in the chosen list; 0 for JLPT or a missing list

	WordsOnly    bool // words-only mode is on
	KanaFallback bool // words-only mode, but no word can fall and no package can come: kana fall
	Filtered     bool // a lesson filter limits the source
	Empty        bool // the source with its lesson filter has no word at all
}

// formatWordsLine is the status panel's words line: counts, or a hint when
// nothing can be introduced. In words-only mode a problem starts with
// "Words only:".
func formatWordsLine(ws WordsStatus) string {
	var s string
	switch {
	case ws.KanaFallback:
		s = "Words only: " + wordsOnlyReason(ws)
	case ws.LoadFailed:
		s = "Words unavailable: could not load progress"
	case ws.Err != "":
		s = "Words unavailable: " + ws.Err
	case ws.Missing && ws.Dir != "":
		s = "Words: add a list to " + ws.Dir + " or choose JLPT"
	case ws.Missing:
		s = "Words: add a word list or choose JLPT"
	case ws.Empty && ws.Filtered:
		s = "Words: no words in the chosen lessons"
	case ws.Next == wordsNeedKana && ws.WordsOnly:
		s = "Words only: new words need a script that is off"
	case ws.Next == wordsNeedKana:
		s = "Words: learn more kana for new words"
	case ws.Next == wordsFinished && ws.Level == 1:
		s = "Words: JLPT finished"
	case ws.Next == wordsFinished:
		s = "Words: " + ws.Source + " finished"
	case ws.Waiting > 0:
		s = fmt.Sprintf("Words: %d introduced · %d learned · %d waiting for kana · %s", ws.Introduced, ws.Learned, ws.Waiting, ws.Source)
	default:
		s = fmt.Sprintf("Words: %d introduced · %d learned · %s", ws.Introduced, ws.Learned, ws.Source)
	}
	if ws.Skipped > 0 {
		s += " · " + plural(ws.Skipped, "row") + " skipped"
	}
	return truncateRunes(s, maxWordsLineRunes)
}

// wordsOnlyReason says why no word can fall in words-only mode.
func wordsOnlyReason(ws WordsStatus) string {
	switch {
	case ws.LoadFailed:
		return "could not load progress"
	case ws.Err != "":
		return "words unavailable: " + ws.Err
	case ws.Missing && ws.Dir != "":
		return "add a list to " + ws.Dir + " or choose JLPT"
	case ws.Missing:
		return "add a word list or choose JLPT"
	case ws.Empty && ws.Filtered:
		return "no words in the chosen lessons"
	case ws.Empty:
		return "no words in " + ws.Source
	}
	return "no word fits the active scripts"
}

// displayDir shortens a folder inside home to "~/…" for display.
func displayDir(dir, home string) string {
	if home == "" {
		return dir
	}
	rel, err := filepath.Rel(home, dir)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return dir
	}
	return "~" + string(filepath.Separator) + rel
}
