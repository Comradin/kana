package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"
	"kana/store"
)

func main() {
	st, err := store.Open("kana.db")
	if err != nil {
		fmt.Printf("Error opening store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	a := app.New()
	w := buildWindow(a, st)
	w.ShowAndRun()
}
