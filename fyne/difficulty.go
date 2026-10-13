package main

import (
	"fmt"
	"math"
	"time"
)

// Level names a difficulty level; the value is what the store keeps.
type Level string

const (
	LevelRelaxed Level = "relaxed"
	LevelNormal  Level = "normal"
	LevelFast    Level = "fast"
	LevelPro     Level = "pro"
)

// LevelInfo is what a level changes about the game.
type LevelInfo struct {
	Level        Level
	Label        string
	MinFall      float32 // seconds a tile takes to fall, before factors
	MaxFall      float32
	SpawnEvery   time.Duration
	MissLimit    int
	PointsFactor int // used when the points setting is Auto
}

var levels = []LevelInfo{
	{LevelRelaxed, "Relaxed", 18, 26, 4500 * time.Millisecond, 10, 1},
	{LevelNormal, "Normal", 16, 24, 4 * time.Second, 10, 1},
	{LevelFast, "Fast", 10, 15, 2500 * time.Millisecond, 7, 2},
	{LevelPro, "Pro", 6, 9, 1600 * time.Millisecond, 5, 3},
}

// Levels returns all levels in display order.
func Levels() []LevelInfo {
	return append([]LevelInfo(nil), levels...)
}

// LevelByName returns the named level; unknown or empty names give Normal,
// which matches the game before levels existed.
func LevelByName(name string) LevelInfo {
	for _, l := range levels {
		if string(l.Level) == name {
			return l
		}
	}
	return levels[1]
}

const (
	pointsAuto     = 0  // stored "Auto": use the level's points factor
	maxTempoStep   = 10 // 0.95^10 ≈ 0.6, where the tempo floor is reached
	streakPerStep  = 5  // correct answers in a row per tempo step
	missTempoSteps = 2  // tempo steps a miss takes back
)

// pointsChoices are the selectable points settings, Auto first.
var pointsChoices = []int{pointsAuto, 1, 2, 3, 5}

// validPoints keeps a selectable points setting and maps anything else to Auto.
func validPoints(p int) int {
	for _, c := range pointsChoices {
		if c == p {
			return p
		}
	}
	return pointsAuto
}

// pointsLabel is the settings label of a points setting.
func pointsLabel(p int) string {
	if p == pointsAuto {
		return "Auto (by level)"
	}
	return fmt.Sprintf("×%d", p)
}

// charFallFactor gives new characters more time and mastered ones less.
func charFallFactor(correct int) float32 {
	switch stateFor(correct) {
	case cellMastered:
		return 0.7
	case cellLearning:
		return 1.0
	default:
		return 1.2
	}
}

// tempoFactor scales fall time and spawn interval for a tempo step, never
// below 0.6.
func tempoFactor(step int) float32 {
	f := float32(math.Pow(0.95, float64(step)))
	if f < 0.6 {
		return 0.6
	}
	return f
}

// formatStatus is the input bar's score line.
func formatStatus(score, limit int, level string, tempo float32) string {
	s := fmt.Sprintf("Score: %d", score)
	if limit > 0 {
		s = fmt.Sprintf("Score: %d/%d", score, limit)
	}
	return fmt.Sprintf("%s · %s · Tempo ×%.2f", s, level, tempo)
}

// missReason is the game-over reason after too many misses (kana and words).
func missReason(missed int) string {
	return fmt.Sprintf("%d tiles slipped through.", missed)
}
