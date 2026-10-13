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
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	type file struct{ path, base, ext string }
	var files []file
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
		files = append(files, file{filepath.Join(dir, e.Name()), base, ext})
		count[strings.ToLower(base)]++
	}
	used := map[string]bool{}
	var lists []List
	for _, f := range files {
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
		lists = append(lists, loadFile(name, f.path, f.ext))
	}
	sort.Slice(lists, func(i, j int) bool { return lists[i].Name < lists[j].Name })
	return lists, nil
}

// loadFile reads one list file; an open error becomes the list's problem.
func loadFile(name, path, ext string) List {
	fh, err := os.Open(path)
	if err != nil {
		l := List{Name: name}
		l.problem(err.Error())
		return l
	}
	defer fh.Close()
	if ext == ".json" {
		return readJSON(name, fh)
	}
	return readCSV(name, fh)
}
