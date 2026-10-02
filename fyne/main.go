package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"
	"kana/store"
)

func main() {
	path, err := store.DefaultPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error locating data directory: %v\n", err)
		os.Exit(1)
	}

	if moved, err := store.MigrateLegacy("kana.db", path); err != nil {
		fmt.Fprintf(os.Stderr, "Error moving existing database to %s: %v\n", path, err)
	} else if moved {
		fmt.Fprintf(os.Stderr, "Moved existing database to %s\n", path)
	}

	st, err := store.Open(path)
	if err != nil {
		fmt.Printf("Error opening store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	a := app.New()
	w := buildWindow(a, st)
	w.ShowAndRun()
}
