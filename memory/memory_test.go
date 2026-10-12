package memory

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)

func after(days float64) time.Time {
	return t0.Add(time.Duration(days * 24 * float64(time.Hour)))
}

func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestRetrievability(t *testing.T) {
	s := State{Stability: 2, LastSeen: t0}
	if r := Retrievability(s, t0); r != 1 {
		t.Fatalf("R at 0 = %v", r)
	}
	if r := Retrievability(s, after(2)); !near(r, 0.9, 1e-9) {
		t.Fatalf("R at S = %v", r)
	}
	if r := Retrievability(s, after(4)); !near(r, 0.81, 1e-9) {
		t.Fatalf("R at 2S = %v", r)
	}
}

func TestCorrectAndMiss(t *testing.T) {
	if s := Correct(State{}, false, t0); s.Stability != MinStability || !s.LastSeen.Equal(t0) {
		t.Fatalf("first correct = %+v", s)
	}
	s := State{Stability: 2, LastSeen: t0}
	if got := Correct(s, true, after(2)).Stability; !near(got, 2*1.7, 1e-9) {
		t.Fatalf("correct at R=0.9 → %v, want 3.4", got)
	}
	halfLife := 2 * math.Log(0.5) / math.Log(0.9) // Δt where R = 0.5
	if got := Correct(s, true, after(halfLife)).Stability; !near(got, 2*2.5, 1e-6) {
		t.Fatalf("correct at R=0.5 → %v, want 5", got)
	}
	if got := Correct(State{Stability: 300, LastSeen: t0}, true, after(300)).Stability; got != MaxStability {
		t.Fatalf("cap → %v", got)
	}
	missed := Miss(s, after(1))
	if !near(missed.Stability, 0.6, 1e-9) {
		t.Fatalf("miss → %v, want 0.6", missed.Stability)
	}
	if r := Retrievability(missed, after(1)); !near(r, 0.5, 1e-6) {
		t.Fatalf("R right after a miss = %v, want 0.5", r)
	}
	if got := Miss(State{Stability: MinStability, LastSeen: t0}, t0).Stability; got != MinStability {
		t.Fatalf("miss floor → %v", got)
	}
}

func TestWeightAndSeed(t *testing.T) {
	if w := Weight(State{Stability: 2, LastSeen: t0}, t0); w != WeightFloor {
		t.Fatalf("weight at R=1 = %v", w)
	}
	if w := Weight(State{Stability: 2, LastSeen: t0}, after(4)); !near(w, 0.19, 1e-9) {
		t.Fatalf("weight at R=0.81 = %v", w)
	}
	if s := Seed(3, t0); !near(s.Stability, MinStability*8, 1e-12) {
		t.Fatalf("Seed(3) = %v", s.Stability)
	}
	if s := Seed(10, t0); !near(s.Stability, 7.11, 0.01) {
		t.Fatalf("Seed(10) = %v days", s.Stability)
	}
	if s := Seed(50, t0); s.Stability > MaxStability {
		t.Fatalf("Seed must cap at MaxStability")
	}
}

func share(n int, f func() bool) float64 {
	hits := 0
	for i := 0; i < n; i++ {
		if f() {
			hits++
		}
	}
	return float64(hits) / float64(n)
}

func TestPickNewShare(t *testing.T) {
	rnd := rand.New(rand.NewSource(1)).Float64
	states := map[string]State{}
	var cands []string
	for i := 0; i < 20; i++ {
		c := string(rune('a' + i))
		cands = append(cands, c)
		if i >= 10 {
			states[c] = State{Stability: 10, LastSeen: t0}
		}
	}
	got := share(10000, func() bool { _, known := states[Pick(cands, states, t0, rnd)]; return !known })
	if !near(got, 0.40, 0.02) {
		t.Fatalf("new share with 10/10 = %v, want 0.40", got)
	}

	for i := 1; i < 10; i++ {
		states[cands[i]] = State{Stability: 10, LastSeen: t0}
	}
	got = share(10000, func() bool { _, known := states[Pick(cands, states, t0, rnd)]; return !known })
	if !near(got, 0.05, 0.015) {
		t.Fatalf("new share with 1/19 = %v, want ~0.05", got)
	}
}

func TestPickPrefersWeak(t *testing.T) {
	rnd := rand.New(rand.NewSource(2)).Float64
	halfLife := 2 * math.Log(0.5) / math.Log(0.9)
	states := map[string]State{
		"weak":   {Stability: 2, LastSeen: t0},
		"strong": {Stability: 2, LastSeen: after(halfLife)},
	}
	now := after(halfLife) // weak at R=0.5 (w 0.5), strong at R=1 (w 0.05)
	weak := share(11000, func() bool { return Pick([]string{"weak", "strong"}, states, now, rnd) == "weak" })
	ratio := weak / (1 - weak)
	if ratio < 8.5 || ratio > 11.5 {
		t.Fatalf("weak:strong = %v, want ~10", ratio)
	}
}

func TestPickEdges(t *testing.T) {
	rnd := rand.New(rand.NewSource(3)).Float64
	known := map[string]State{"k": {Stability: 1, LastSeen: t0}}
	if Pick(nil, known, t0, rnd) != "" {
		t.Fatal("empty candidates → empty string")
	}
	for i := 0; i < 100; i++ {
		if Pick([]string{"n1", "n2"}, known, t0, rnd) == "" {
			t.Fatal("only new candidates must still pick one")
		}
		if Pick([]string{"k"}, known, t0, rnd) != "k" {
			t.Fatal("only known candidates → known")
		}
	}
}
