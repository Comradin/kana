package main

import (
	"math"
	"testing"
	"time"
)

func TestLevelByName(t *testing.T) {
	cases := map[string]struct {
		label string
		limit int
		every time.Duration
	}{
		"relaxed": {"Relaxed", 10, 4500 * time.Millisecond},
		"normal":  {"Normal", 10, 4 * time.Second},
		"fast":    {"Fast", 7, 2500 * time.Millisecond},
		"pro":     {"Pro", 5, 1600 * time.Millisecond},
		"":        {"Normal", 10, 4 * time.Second},
		"insane":  {"Normal", 10, 4 * time.Second},
	}
	for name, want := range cases {
		got := LevelByName(name)
		if got.Label != want.label || got.MissLimit != want.limit || got.SpawnEvery != want.every {
			t.Errorf("LevelByName(%q) = %+v", name, got)
		}
	}
	if len(Levels()) != 4 || Levels()[0].Level != LevelRelaxed || Levels()[3].Level != LevelPro {
		t.Fatal("Levels() must list relaxed, normal, fast, pro")
	}
}

func TestCharFallFactor(t *testing.T) {
	for correct, want := range map[int]float32{0: 1.2, 2: 1.0, 3: 0.7, 9: 0.7} {
		if got := charFallFactor(correct); got != want {
			t.Errorf("charFallFactor(%d) = %v, want %v", correct, got, want)
		}
	}
}

func TestTempoFactor(t *testing.T) {
	if tempoFactor(0) != 1 || tempoFactor(1) != float32(0.95) || tempoFactor(10) != 0.6 || tempoFactor(20) != 0.6 {
		t.Fatalf("tempoFactor = %v %v %v %v", tempoFactor(0), tempoFactor(1), tempoFactor(10), tempoFactor(20))
	}
}

func TestValidPoints(t *testing.T) {
	for in, want := range map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 5: 5, 4: 0, -1: 0, 7: 0} {
		if got := validPoints(in); got != want {
			t.Errorf("validPoints(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestFormatStatusAndMissReason(t *testing.T) {
	if got := formatStatus(240, 1000, "Fast", 1.15); got != "Score: 240/1000 · Fast · Tempo ×1.15" {
		t.Errorf("formatStatus = %q", got)
	}
	if got := formatStatus(240, 0, "Normal", 1); got != "Score: 240 · Normal · Tempo ×1.00" {
		t.Errorf("formatStatus endless = %q", got)
	}
	if got := missReason(5); got != "5 tiles slipped through." {
		t.Errorf("missReason = %q", got)
	}
}

func TestWordShare(t *testing.T) {
	cases := map[int]float64{0: 0, 5: 0.05, 10: 0.1, 30: 0.1, 500: 0.1}
	for n, want := range cases {
		if got := wordShare(n); math.Abs(got-want) > 1e-9 {
			t.Errorf("wordShare(%d) = %v, want %v", n, got, want)
		}
	}
}

func TestWordFallFactor(t *testing.T) {
	cases := map[int]float32{1: 1, 2: 1.2, 3: 1.4, 6: 2, 9: 2}
	for units, want := range cases {
		if got := wordFallFactor(units); got-want > 1e-5 || want-got > 1e-5 {
			t.Errorf("wordFallFactor(%d) = %v, want %v", units, got, want)
		}
	}
}
