package main

import "kana/kanacore"

// StepState is how far one learning-path step has come.
type StepState int

const (
	StepOpen     StepState = iota // not every row of the step is selected
	StepLearning                  // all rows selected, not all mastered
	StepMastered                  // all rows selected and mastered
)

// PathStatus summarises one script's learning path for the status panel.
type PathStatus struct {
	Steps    []StepState
	Unlocked int      // steps whose rows are all selected, in any order
	Next     []string // rows the next unlock would add; nil when complete
}

// pathStatus computes the learning-path status of a script. Must be called
// under lock.
func (gs *GameState) pathStatus(s kanacore.Script) PathStatus {
	steps := kanacore.ProgressionStepsFor(s)
	ps := PathStatus{Steps: make([]StepState, len(steps))}
	for i, step := range steps {
		selected, mastered := true, true
		for _, id := range step {
			if !gs.selectedRows[id] {
				selected = false
				break
			}
			if row, ok := kanacore.RowByID(id); !ok || !gs.isRowMastered(row) {
				mastered = false
			}
		}
		switch {
		case !selected:
			ps.Steps[i] = StepOpen
		case mastered:
			ps.Steps[i] = StepMastered
			ps.Unlocked++
		default:
			ps.Steps[i] = StepLearning
			ps.Unlocked++
		}
	}
	ps.Next = gs.nextStepRows(s)
	return ps
}

// totalCorrect returns stored plus session correct answers per character.
// Must be called under lock.
func (gs *GameState) totalCorrect() map[string]int {
	total := make(map[string]int, len(gs.overallStats))
	for char, st := range gs.overallStats {
		total[char] += st.CorrectCount
	}
	for char, st := range gs.sessionStats {
		total[char] += st.CorrectCount
	}
	return total
}
