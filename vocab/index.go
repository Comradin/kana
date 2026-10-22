package vocab

import (
	"maps"
	"slices"
	"sort"
	"sync"

	"kana/kanacore"
)

// Entry is one reading with everything known about it across sources.
type Entry struct {
	Kana     string
	Display  string   // shown form from the first private list row of this reading; "" = Kana
	Alts     []string // other accepted readings across sources, unique, in first-seen order
	Phrase   bool     // any source row is a phrase: は may be wa, へ e
	Units    []string // kanacore.KanaUnits of Kana, computed once
	Meanings Meanings
	JLPT     int                 // easiest bundled level, 0 if not bundled
	Rank     int                 // best bundled frequency rank, lower = more common; 0 = unknown
	Bundled  bool                // the bundled list contains it
	Lists    map[string][]string // private list name → lessons
	ListPos  map[string]int      // private list name → index of its first row in that list
}

// Shown returns how the entry is shown to the learner: its Display, else
// its reading.
func (e Entry) Shown() string {
	if e.Display != "" {
		return e.Display
	}
	return e.Kana
}

// ReadingMatches reports whether input is a romaji reading of the entry's
// reading or of one of its other accepted readings (see
// kanacore.ReadingMatches; kanacore.PhraseReadingMatches for a phrase).
func (e Entry) ReadingMatches(input string) bool {
	matches := kanacore.ReadingMatches
	if e.Phrase {
		matches = kanacore.PhraseReadingMatches
	}
	if matches(e.Kana, input) {
		return true
	}
	for _, a := range e.Alts {
		if matches(a, input) {
			return true
		}
	}
	return false
}

// Index holds one entry per reading.
type Index struct {
	entries []Entry
	byKana  map[string]int

	// While WithLists merges: entries below shared are still those of the
	// base index unless owned marks them as copied.
	shared int
	owned  map[int]bool

	lessons map[string][]string // private list name → its lessons in first-appearance order

	mu     sync.Mutex
	orders map[string][]int // learning order per source (list name, "" = JLPT), see order
}

// NewIndex merges bundled words and private lists into one entry per
// reading. Words whose reading cannot be split into units are dropped.
func NewIndex(bundled []Word, lists []List) *Index {
	ix := &Index{byKana: map[string]int{}, lessons: map[string][]string{}}
	for _, w := range bundled {
		ix.add(w, "", 0)
	}
	ix.addLists(lists)
	ix.sort()
	return ix
}

// WithLists returns a new index with the private lists merged into ix's
// entries, the same as NewIndex(bundled, lists) for an ix built from
// NewIndex(bundled, nil), without splitting the readings again. ix is not
// changed: an entry a list touches is copied before the merge.
func (ix *Index) WithLists(lists []List) *Index {
	out := &Index{
		entries: slices.Clone(ix.entries),
		byKana:  maps.Clone(ix.byKana),
		shared:  len(ix.entries),
		lessons: maps.Clone(ix.lessons),
	}
	out.addLists(lists)
	out.shared = 0
	out.sort()
	return out
}

func (ix *Index) addLists(lists []List) {
	for _, l := range lists {
		for i, w := range l.Words {
			if ix.add(w, l.Name, i) && w.Lesson != "" && !contains(ix.lessons[l.Name], w.Lesson) {
				ix.lessons[l.Name] = append(ix.lessons[l.Name], w.Lesson)
			}
		}
	}
}

// add merges one word into its entry; list is "" for the bundled list, and
// pos is the word's row index in its list. It reports whether the word was
// added: a reading that cannot be split into units is dropped. The first
// private list row gives the shown form; other readings merge like meanings,
// and the entry is a phrase if any of its rows is.
func (ix *Index) add(w Word, list string, pos int) bool {
	i, ok := ix.byKana[w.Kana]
	if !ok {
		units, valid := kanacore.KanaUnits(w.Kana)
		if !valid {
			return false
		}
		ix.byKana[w.Kana] = len(ix.entries)
		ix.entries = append(ix.entries, Entry{Kana: w.Kana, Units: units, Meanings: Meanings{}, Lists: map[string][]string{}, ListPos: map[string]int{}})
		i = len(ix.entries) - 1
	} else if i < ix.shared && !ix.owned[i] {
		ix.entries[i] = ix.entries[i].clone()
		if ix.owned == nil {
			ix.owned = map[int]bool{}
		}
		ix.owned[i] = true
	}
	e := &ix.entries[i]
	e.Meanings.Merge(w.Meanings)
	e.Phrase = e.Phrase || w.Phrase
	for _, a := range w.Alts {
		if !contains(e.Alts, a) {
			e.Alts = append(e.Alts, a)
		}
	}
	if list == "" {
		if !e.Bundled || w.JLPT > e.JLPT {
			e.JLPT = w.JLPT
		}
		if !e.Bundled || rankKey(w.Rank) < rankKey(e.Rank) {
			e.Rank = w.Rank
		}
		e.Bundled = true
		return true
	}
	// The first private list row of this reading decides how it is shown,
	// even when it has no notation.
	if len(e.ListPos) == 0 {
		e.Display = w.Display
	}
	if _, ok := e.ListPos[list]; !ok {
		e.ListPos[list] = pos
	}
	lessons := e.Lists[list]
	if w.Lesson != "" && !contains(lessons, w.Lesson) {
		lessons = append(lessons, w.Lesson)
	}
	e.Lists[list] = lessons
	return true
}

// sort orders the entries by reading and rebuilds byKana.
func (ix *Index) sort() {
	sort.Slice(ix.entries, func(a, b int) bool { return ix.entries[a].Kana < ix.entries[b].Kana })
	for i, e := range ix.entries {
		ix.byKana[e.Kana] = i
	}
	ix.owned = nil
}

// rankKey orders frequency ranks: an unknown rank (0) sorts as 99, after
// every word with a priority tag.
func rankKey(r int) int {
	if r <= 0 {
		return 99
	}
	return r
}

// clone copies the entry's Meanings, Alts, Lists and ListPos, so merging
// into the copy leaves the original untouched. Units is never modified and
// stays shared.
func (e Entry) clone() Entry {
	m := make(Meanings, len(e.Meanings))
	for lang, items := range e.Meanings {
		m[lang] = slices.Clone(items)
	}
	l := make(map[string][]string, len(e.Lists))
	for name, lessons := range e.Lists {
		l[name] = slices.Clone(lessons)
	}
	e.Meanings, e.Lists, e.ListPos, e.Alts = m, l, maps.Clone(e.ListPos), slices.Clone(e.Alts)
	return e
}

// Entries returns all entries sorted by reading. The slice and the entries'
// Units, Alts, Meanings and Lists are shared with the index; do not modify
// them.
func (ix *Index) Entries() []Entry { return ix.entries }

// Lookup returns the entry for a reading. Its Units, Alts, Meanings and
// Lists are shared with the index; do not modify them.
func (ix *Index) Lookup(kana string) (Entry, bool) {
	i, ok := ix.byKana[kana]
	if !ok {
		return Entry{}, false
	}
	return ix.entries[i], true
}

// Filter selects the source of playable entries.
type Filter struct {
	MaxLevel int    // bundled entries need JLPT >= MaxLevel (5 = N5 only)
	List     string // "" = bundled up to MaxLevel plus every private list;
	// BundledSource = bundled only, up to MaxLevel; else only this private list
	Lesson string // with a private List: only this lesson ("" = all)
}

// Lessons returns a list's lessons in learning order (by number if every
// lesson is an integer, else by first appearance) and whether the list has
// words without a lesson. Nil and false for an unknown list.
func (ix *Index) Lessons(list string) (lessons []string, noLesson bool) {
	ranks := ix.lessonRanks(list)
	lessons = slices.Clone(ix.lessons[list])
	sort.SliceStable(lessons, func(a, b int) bool { return ranks[lessons[a]] < ranks[lessons[b]] })
	for _, i := range ix.order(list) {
		if len(ix.entries[i].Lists[list]) == 0 {
			noLesson = true
			break
		}
	}
	return lessons, noLesson
}

// Languages returns the meaning languages of the index's entries, sorted by
// code. Meanings without a language do not count.
func (ix *Index) Languages() []string {
	seen := map[string]bool{}
	for _, e := range ix.entries {
		for lang, items := range e.Meanings {
			if lang != "" && len(items) > 0 {
				seen[lang] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(seen))
}

// Playable returns the entries whose units all satisfy known and which pass
// the filter, in index order.
func Playable(entries []Entry, known func(unit string) bool, f Filter) []Entry {
	var out []Entry
	for _, e := range entries {
		if !f.passes(e) || !allKnown(e.Units, known) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (f Filter) passes(e Entry) bool {
	bundledOK := e.Bundled && e.JLPT >= f.MaxLevel
	switch f.List {
	case "":
		return bundledOK || len(e.Lists) > 0
	case BundledSource:
		return bundledOK
	default:
		lessons, ok := e.Lists[f.List]
		return ok && (f.Lesson == "" || contains(lessons, f.Lesson))
	}
}

func allKnown(units []string, known func(string) bool) bool {
	for _, u := range units {
		if !known(u) {
			return false
		}
	}
	return true
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
