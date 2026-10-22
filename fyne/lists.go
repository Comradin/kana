package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"kana/vocab"
)

// listInfo describes a private word list for the word list dialog and the
// status panel.
type listInfo struct {
	name     string   // the list's name, as in the source select
	path     string   // its file in the word list folder
	words    int      // rows read as words
	lessons  int      // distinct lessons; words without a lesson do not count
	skipped  int      // skipped rows
	problems []string // messages on skipped rows, at most 10
}

// newListInfo summarises a loaded list.
func newListInfo(l vocab.List) listInfo {
	lessons := map[string]bool{}
	for _, w := range l.Words {
		if w.Lesson != "" {
			lessons[w.Lesson] = true
		}
	}
	return listInfo{name: l.Name, path: l.Path, words: len(l.Words), lessons: len(lessons),
		skipped: l.Skipped, problems: slices.Clone(l.Problems)}
}

// fileBase is the list's file name without its extension: what a rename
// changes.
func (li listInfo) fileBase() string {
	file := filepath.Base(li.path)
	return strings.TrimSuffix(file, filepath.Ext(file))
}

// listByName returns the list named name. Must be called under lock.
func (gs *GameState) listByName(name string) (listInfo, bool) {
	for _, li := range gs.lists {
		if li.name == name {
			return li, true
		}
	}
	return listInfo{}, false
}

// errListGone says that a list to rename or delete is no longer there.
func errListGone(name string) error {
	return fmt.Errorf("The word list %s is no longer there.", prettyListName(name))
}

// ListInfos returns the private word lists, sorted by name. The messages
// are shared; do not modify them.
func (gs *GameState) ListInfos() []listInfo {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return slices.Clone(gs.lists)
}

// WordsDir returns the word list folder; "" if unknown.
func (gs *GameState) WordsDir() string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.wordsDir
}

// ReloadLists reads the word list folder again, so lists copied, edited or
// removed by hand come in (see reloadLists).
func (gs *GameState) ReloadLists() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.reloadLists()
}

// reloadLists reloads the lists after they changed, which invalidates the
// playable words, and drops word tiles that may no longer fall. A game
// without a word list folder (the tests' hand-built games) has nothing to
// reload and never reads the real folder. Must be called under lock.
func (gs *GameState) reloadLists() {
	if gs.wordsDir == "" {
		return
	}
	gs.loadLists()
	gs.dropInactiveTiles()
	gs.buildSnapshot()
}

// DeleteList removes the list's file for good and reloads the lists.
// Progress on its words is kept; its lesson filter is removed. When it was
// the source of new words, JLPT becomes the source and the newest package
// is cleared, as for any source change. A file already gone counts as
// deleted. After an error the lists are reloaded too.
func (gs *GameState) DeleteList(name string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	li, ok := gs.listByName(name)
	if !ok {
		return errListGone(name)
	}
	if err := vocab.DeleteList(li.path); err != nil {
		gs.reloadLists()
		return err
	}
	gs.setLessonFilter(name, nil)
	if gs.wordSource == listSourcePrefix+name {
		gs.wordSource = jlptSource
		gs.newestPackage = nil
		if gs.store != nil {
			_ = gs.store.SaveWordSource(jlptSource)
		}
		gs.saveNewestPackage()
	}
	gs.reloadLists()
	return nil
}

// RenameList renames the list's file (see vocab.RenameList), reloads the
// lists and returns the list's new name. Its lesson filter and its place as
// the source of new words follow it; the newest package holds readings and
// stays. After an error the lists are reloaded too.
func (gs *GameState) RenameList(name, newName string) (string, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	li, ok := gs.listByName(name)
	if !ok {
		return "", errListGone(name)
	}
	renamed, err := vocab.RenameList(li.path, newName)
	if err != nil {
		gs.reloadLists()
		return "", err
	}
	if renamed != name {
		gs.moveLessonFilter(name, renamed)
		if gs.wordSource == listSourcePrefix+name {
			gs.wordSource = listSourcePrefix + renamed
			if gs.store != nil {
				_ = gs.store.SaveWordSource(gs.wordSource)
			}
		}
	}
	gs.reloadLists()
	return renamed, nil
}

// CheckListName reports why the list name cannot be renamed to newName
// (see vocab.CheckListName); nil if it can.
func (gs *GameState) CheckListName(name, newName string) error {
	gs.mu.Lock()
	li, ok := gs.listByName(name)
	gs.mu.Unlock()
	if !ok {
		return errListGone(name)
	}
	_, err := vocab.CheckListName(li.path, newName)
	return err
}

// moveLessonFilter moves a list's lesson filter to its new name, in the game
// and in the store. Must be called under lock.
func (gs *GameState) moveLessonFilter(from, to string) {
	if gs.lessonFilters == nil {
		gs.lessonFilters = map[string][]string{}
	}
	if ls, ok := gs.lessonFilters[from]; ok {
		delete(gs.lessonFilters, from)
		gs.lessonFilters[to] = ls
	} else {
		delete(gs.lessonFilters, to)
	}
	if gs.store != nil {
		_ = gs.store.RenameLessonFilter(from, to)
	}
}
