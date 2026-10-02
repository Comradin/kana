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
		// Keep using the old database this time; the move is retried next start.
		fmt.Fprintf(os.Stderr, "Error moving existing database to %s: %v; using ./kana.db for now\n", path, err)
		path = "kana.db"
	} else if moved {
		fmt.Fprintf(os.Stderr, "Moved existing database to %s\n", path)
	}

	st, err := store.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	a := app.New()
	w := buildWindow(a, st)
	w.ShowAndRun()
}
