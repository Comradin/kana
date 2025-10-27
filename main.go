package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"kana/store"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	st, err := store.Open("kana.db")
	if err != nil {
		fmt.Printf("Error opening store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	rows, autoProgress, err := setupSettingsForm(st)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Setup cancelled. Goodbye!")
			return
		}
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	model := InitialModel(st)
	if len(rows) > 0 {
		model.SetSelectedRows(rows)
	}
	model.SetAutoProgress(autoProgress)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
