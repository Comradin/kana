package vocab

import (
	"sort"
	"strconv"
	"strings"
)

// Source names where new words come from.
type Source struct {
	List    string   // private list name; "" = JLPT (the bundled list)
	Level   int      // JLPT: lowest level number allowed (5 = N5 only); ignored for lists
	Lessons []string // list: the chosen lessons, "" for words without a lesson; empty = all lessons. Ignored for JLPT
}

// Has reports whether e belongs to the source: for JLPT a bundled entry
// with JLPT >= Level, for a list an entry of that list in one of the chosen
// lessons (or without a lesson, if "" is chosen).
func (s Source) Has(e Entry) bool {
	if s.List == "" {
		return e.Bundled && e.JLPT >= s.Level
	}
	lessons, ok := e.Lists[s.List]
	if !ok {
		return false
	}
	if len(s.Lessons) == 0 {
		return true
	}
	if len(lessons) == 0 {
		return contains(s.Lessons, "")
	}
	for _, l := range lessons {
		if contains(s.Lessons, l) {
			return true
		}
	}
	return false
}

// NextPackage returns up to n entries from src that are not introduced and
// whose units are all known, in learning order. Entries that are not
// playable yet are skipped, not waited for: they come once their kana are
// known. A list package holds words of a single lesson: the earliest chosen
// lesson (src.Lessons, all when empty) with a word to introduce; an entry in
// several lessons counts for its first chosen one.
//
// JLPT order: level from N5 down to src.Level, then frequency rank, then
// fewer units, then the reading. List order: by lesson (numeric if every
// lesson of the list is an integer, else by first appearance; an entry in
// several lessons counts for its first), then file order; entries without
// a lesson last, in file order.
func NextPackage(ix *Index, src Source, introduced map[string]bool, n int, known func(string) bool) []Entry {
	return NextPackageWhere(ix, src, introduced, n, known, nil)
}

// NextPackageWhere is NextPackage limited to the entries keep accepts (nil
// accepts all), e.g. to refill a package within its lesson or levels.
func NextPackageWhere(ix *Index, src Source, introduced map[string]bool, n int, known func(string) bool, keep func(Entry) bool) []Entry {
	if ix == nil || n <= 0 {
		return nil
	}
	eligible := func(e Entry) bool {
		return src.Has(e) && !introduced[e.Kana] && allKnown(e.Units, known) && (keep == nil || keep(e))
	}
	order := ix.order(src.List)
	var out []Entry
	if src.List == "" {
		for _, i := range order {
			if e := ix.entries[i]; eligible(e) {
				out = append(out, e)
				if len(out) == n {
					break
				}
			}
		}
		return out
	}
	// A list package holds words of one lesson only: the earliest (chosen)
	// lesson with an eligible word, so earlier lessons' unreadable words do
	// not hold it back. Words without a lesson come last.
	ranks := ix.lessonRanks(src.List)
	var lesson string
	var rank int
	found, hasLesson := false, false
	for _, i := range order {
		e := ix.entries[i]
		if !eligible(e) {
			continue
		}
		l, r, ok := firstLesson(e, src, ranks)
		if !found || (ok && (!hasLesson || r < rank)) {
			lesson, rank, hasLesson, found = l, r, ok, true
		}
	}
	if !found {
		return nil
	}
	for _, i := range order {
		e := ix.entries[i]
		if l, _, _ := firstLesson(e, src, ranks); l == lesson && eligible(e) {
			out = append(out, e)
			if len(out) == n {
				break
			}
		}
	}
	return out
}

// Lesson returns e's first lesson in the list's lesson order, "" if it has
// none in that list.
func (ix *Index) Lesson(e Entry, list string) string {
	return ix.LessonIn(e, Source{List: list})
}

// LessonIn returns e's first lesson in the lesson order of src's list,
// among src.Lessons when set; "" if it has none there or src is JLPT.
func (ix *Index) LessonIn(e Entry, src Source) string {
	if src.List == "" {
		return ""
	}
	lesson, _, _ := firstLesson(e, src, ix.lessonRanks(src.List))
	return lesson
}

// order returns the indices of a source's entries in learning order (list
// name, "" for JLPT; levels are filtered by the caller). It is computed
// once per index and source, then cached; the entries never change after
// the index is built.
func (ix *Index) order(list string) []int {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if o, ok := ix.orders[list]; ok {
		return o
	}
	var idx []int
	for i, e := range ix.entries {
		if (list == "" && e.Bundled) || (list != "" && hasList(e, list)) {
			idx = append(idx, i)
		}
	}
	if list == "" {
		sort.SliceStable(idx, func(a, b int) bool {
			x, y := ix.entries[idx[a]], ix.entries[idx[b]]
			switch {
			case x.JLPT != y.JLPT:
				return x.JLPT > y.JLPT
			case rankKey(x.Rank) != rankKey(y.Rank):
				return rankKey(x.Rank) < rankKey(y.Rank)
			case len(x.Units) != len(y.Units):
				return len(x.Units) < len(y.Units)
			}
			return x.Kana < y.Kana
		})
	} else {
		ranks := ix.lessonRanks(list)
		type key struct {
			hasLesson bool
			lesson    int
			pos       int
		}
		keys := make(map[int]key, len(idx))
		for _, i := range idx {
			e := ix.entries[i]
			_, r, ok := firstLesson(e, Source{List: list}, ranks)
			keys[i] = key{hasLesson: ok, lesson: r, pos: e.ListPos[list]}
		}
		sort.SliceStable(idx, func(a, b int) bool {
			x, y := keys[idx[a]], keys[idx[b]]
			switch {
			case x.hasLesson != y.hasLesson:
				return x.hasLesson
			case x.lesson != y.lesson:
				return x.lesson < y.lesson
			}
			return x.pos < y.pos
		})
	}
	if ix.orders == nil {
		ix.orders = map[string][]int{}
	}
	ix.orders[list] = idx
	return idx
}

// lessonRanks ranks a list's lessons: by their number if every lesson is
// an integer, else by first appearance in the list.
func (ix *Index) lessonRanks(list string) map[string]int {
	lessons := ix.lessons[list]
	numbers := make([]int, len(lessons))
	numeric := true
	for i, l := range lessons {
		n, err := strconv.Atoi(strings.TrimSpace(l))
		if err != nil {
			numeric = false
			break
		}
		numbers[i] = n
	}
	ranks := make(map[string]int, len(lessons))
	for i, l := range lessons {
		if numeric {
			ranks[l] = numbers[i]
		} else {
			ranks[l] = i
		}
	}
	return ranks
}

// firstLesson returns e's lesson in src's list that comes first by ranks,
// among src.Lessons when set, and its rank; ok is false when e has none
// there.
func firstLesson(e Entry, src Source, ranks map[string]int) (lesson string, rank int, ok bool) {
	for _, l := range e.Lists[src.List] {
		if len(src.Lessons) > 0 && !contains(src.Lessons, l) {
			continue
		}
		if r, known := ranks[l]; known && (!ok || r < rank) {
			lesson, rank, ok = l, r, true
		}
	}
	return lesson, rank, ok
}

func hasList(e Entry, list string) bool {
	_, ok := e.Lists[list]
	return ok
}
