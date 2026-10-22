package vocab

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// LoadDir reads every *.csv, *.tsv and *.json file in dir as a list, except
// files whose name starts with ".". A missing directory gives no lists and
// no error. Lists are sorted by name.
//
// A list is named after its file without the extension. When several files
// share that name (ignoring case), or the name is the reserved jmdict, the
// extension is added: "a (csv)". Names that still collide get " 2", " 3" …
// in directory order.
func LoadDir(dir string) ([]List, error) {
	files, err := scanDir(dir)
	if err != nil {
		return nil, err
	}
	var lists []List
	for _, f := range files {
		lists = append(lists, loadFile(f.name, f.path, f.ext))
	}
	sort.Slice(lists, func(i, j int) bool { return lists[i].Name < lists[j].Name })
	return lists, nil
}

// listEntry is a word list file found by scanDir: the name LoadDir gives its
// list, its path, its file name without extension and its lower-case
// extension.
type listEntry struct{ name, path, base, ext string }

// scanDir finds the word list files in dir and names their lists as LoadDir
// does, in directory order, without reading them. A missing directory gives
// none and no error.
func scanDir(dir string) ([]listEntry, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []listEntry
	count := map[string]int{}
	for _, e := range entries {
		// Dot files are hidden: ".csv" would be a list without a name and
		// macOS writes "._x.csv" resource forks next to copied files.
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".csv" && ext != ".tsv" && ext != ".json" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		files = append(files, listEntry{path: filepath.Join(dir, e.Name()), base: base, ext: ext})
		count[strings.ToLower(base)]++
	}
	used := map[string]bool{}
	for i, f := range files {
		key := strings.ToLower(f.base)
		name := f.base
		if count[key] > 1 || key == BundledSource {
			name = f.base + " (" + strings.TrimPrefix(f.ext, ".") + ")"
		}
		if used[name] {
			n := 2
			for used[name+" "+strconv.Itoa(n)] {
				n++
			}
			name += " " + strconv.Itoa(n)
		}
		used[name] = true
		files[i].name = name
	}
	return files, nil
}

// loadFile reads one list file; an open error becomes the list's problem.
// The list remembers its path.
func loadFile(name, path, ext string) List {
	var l List
	fh, err := os.Open(path)
	switch {
	case err != nil:
		l = List{Name: name}
		l.problem(err.Error())
	case ext == ".json":
		l = readJSON(name, fh, maxProblems)
	default:
		l = readCSV(name, fh, maxProblems)
	}
	if fh != nil {
		fh.Close()
	}
	l.Path = path
	return l
}
