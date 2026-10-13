package main

import (
	"fmt"
	"os"
	"testing"
)

// TestMain points XDG_DATA_HOME at an empty directory, so NewGameState never
// reads the developer's own private word lists.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "kana-fyne-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
