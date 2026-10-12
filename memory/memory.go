// Package memory models how well a learner remembers an item, so practice
// can favour what is about to be forgotten.
package memory

import (
	"math"
	"time"
)

// State is what the model knows about one item.
type State struct {
	Stability float64 // days until recall drops to 90 %
	LastSeen  time.Time
}

const (
	MinStability = 10.0 / (24 * 60) // 10 minutes, in days
	MaxStability = 365.0
	NewCap       = 0.4  // highest share of new items in Pick
	WeightFloor  = 0.05 // lowest pick weight of a known item
)

func elapsedDays(s State, now time.Time) float64 {
	return now.Sub(s.LastSeen).Hours() / 24
}

func clamp(v float64) float64 {
	if math.IsNaN(v) {
		return MinStability
	}
	return math.Min(math.Max(v, MinStability), MaxStability)
}

// Retrievability is the estimated chance of recalling the item now.
func Retrievability(s State, now time.Time) float64 {
	dt := elapsedDays(s, now)
	if dt <= 0 {
		return 1
	}
	return math.Pow(0.9, dt/clamp(s.Stability))
}

// Correct updates a state after a correct answer. An item that was not known
// starts at MinStability; a known one grows more the harder the recall was.
func Correct(s State, known bool, now time.Time) State {
	if !known {
		return State{Stability: MinStability, LastSeen: now}
	}
	r := Retrievability(s, now)
	return State{Stability: clamp(s.Stability * (1.5 + 2*(1-r))), LastSeen: now}
}

// Miss updates a known item's state after it was not recognised. The
// stability shrinks, and the item is treated as half forgotten (R = 0.5 at
// now), so it is picked often again soon after the miss.
func Miss(s State, now time.Time) State {
	st := clamp(s.Stability * 0.3)
	back := st * math.Log(0.5) / math.Log(0.9) // days until R = 0.5
	return State{Stability: st, LastSeen: now.Add(-time.Duration(back * 24 * float64(time.Hour)))}
}

// Weight is how strongly Pick favours a known item: its chance of being
// forgotten, but never below WeightFloor.
func Weight(s State, now time.Time) float64 {
	return math.Max(1-Retrievability(s, now), WeightFloor)
}

// Seed turns a count of past correct answers into a starting state.
func Seed(correct int, now time.Time) State {
	n := math.Min(math.Max(float64(correct), 0), 10)
	return State{Stability: clamp(MinStability * math.Pow(2, n)), LastSeen: now}
}

// Pick chooses one candidate. Candidates missing from states are new: they
// are chosen with probability min(NewCap, their share), uniformly. Otherwise
// a known candidate is chosen, weighted by Weight. rnd returns [0, 1).
func Pick(candidates []string, states map[string]State, now time.Time, rnd func() float64) string {
	if len(candidates) == 0 {
		return ""
	}
	var fresh, known []string
	for _, c := range candidates {
		if _, ok := states[c]; ok {
			known = append(known, c)
		} else {
			fresh = append(fresh, c)
		}
	}
	pNew := math.Min(NewCap, float64(len(fresh))/float64(len(candidates)))
	switch {
	case len(known) == 0:
		pNew = 1
	case len(fresh) == 0:
		pNew = 0
	}
	if rnd() < pNew {
		i := int(rnd() * float64(len(fresh)))
		if i >= len(fresh) {
			i = len(fresh) - 1
		}
		return fresh[i]
	}
	weights := make([]float64, len(known))
	total := 0.0
	for i, c := range known {
		weights[i] = Weight(states[c], now)
		total += weights[i]
	}
	x := rnd() * total
	for i, w := range weights {
		if x < w {
			return known[i]
		}
		x -= w
	}
	return known[len(known)-1]
}
