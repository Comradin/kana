package vocab

import (
	"maps"
	"slices"
	"sort"

	"kana/kanacore"
)

// Entry is one reading with everything known about it across sources.
type Entry struct {
	Kana     string
	Units    []string // kanacore.KanaUnits, computed once
	Meanings Meanings
	JLPT     int                 // easiest bundled level, 0 if not bundled
	Bundled  bool                // the bundled list contains it
	Lists    map[string][]string // private list name → lessons
}

// Index holds one entry per reading.
type Index struct {
	entries []Entry
	byKana  map[string]int

	// While WithLists merges: entries below shared are still those of the
	// base index unless owned marks them as copied.
	shared int
	owned  map[int]bool
}

// NewIndex merges bundled words and private lists into one entry per
// reading. Words whose reading cannot be split into units are dropped.
func NewIndex(bundled []Word, lists []List) *Index {
	ix := &Index{byKana: map[string]int{}}
	for _, w := range bundled {
		ix.add(w, "")
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
	}
	out.addLists(lists)
	out.shared = 0
	out.sort()
	return out
}

func (ix *Index) addLists(lists []List) {
	for _, l := range lists {
		for _, w := range l.Words {
			ix.add(w, l.Name)
		}
	}
}

// add merges one word into its entry; list is "" for the bundled list.
func (ix *Index) add(w Word, list string) {
	i, ok := ix.byKana[w.Kana]
	if !ok {
		units, valid := kanacore.KanaUnits(w.Kana)
		if !valid {
			return
		}
		ix.byKana[w.Kana] = len(ix.entries)
		ix.entries = append(ix.entries, Entry{Kana: w.Kana, Units: units, Meanings: Meanings{}, Lists: map[string][]string{}})
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
	if list == "" {
		if !e.Bundled || w.JLPT > e.JLPT {
			e.JLPT = w.JLPT
		}
		e.Bundled = true
		return
	}
	lessons := e.Lists[list]
	if w.Lesson != "" && !contains(lessons, w.Lesson) {
		lessons = append(lessons, w.Lesson)
	}
	e.Lists[list] = lessons
}

// sort orders the entries by reading and rebuilds byKana.
func (ix *Index) sort() {
	sort.Slice(ix.entries, func(a, b int) bool { return ix.entries[a].Kana < ix.entries[b].Kana })
	for i, e := range ix.entries {
		ix.byKana[e.Kana] = i
	}
	ix.owned = nil
}

// clone copies the entry's Meanings and Lists, so merging into the copy
// leaves the original untouched. Units is never modified and stays shared.
func (e Entry) clone() Entry {
	m := make(Meanings, len(e.Meanings))
	for lang, items := range e.Meanings {
		m[lang] = slices.Clone(items)
	}
	l := make(map[string][]string, len(e.Lists))
	for name, lessons := range e.Lists {
		l[name] = slices.Clone(lessons)
	}
	e.Meanings, e.Lists = m, l
	return e
}

// Entries returns all entries sorted by reading. The slice and the entries'
// Units, Meanings and Lists are shared with the index; do not modify them.
func (ix *Index) Entries() []Entry { return ix.entries }

// Lookup returns the entry for a reading. Its Units, Meanings and Lists are
// shared with the index; do not modify them.
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
	// NoBundled drops entries that come only from the bundled list. An entry
	// that is also in a private list stays, without the level gate.
	NoBundled bool
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
	bundledOK := !f.NoBundled && e.Bundled && e.JLPT >= f.MaxLevel
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
