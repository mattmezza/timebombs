// Package model defines the core Timebomb type.
package model

import "time"

// State represents whether a timebomb is still ticking or has exploded.
type State string

const (
	StateTicking  State = "ticking"
	StateExploded State = "exploded"
)

// Timebomb is a single TIMEBOMB annotation found in source code.
type Timebomb struct {
	File        string    `json:"file"`
	Line        int       `json:"line"`
	Deadline    time.Time `json:"deadline"`
	ID          string    `json:"id,omitempty"`
	Description string    `json:"description"`
}

// State returns whether the bomb is ticking or exploded relative to now.
// A deadline that is today or in the past is exploded.
func (t Timebomb) State(now time.Time) State {
	d := dayOf(t.Deadline)
	n := dayOf(now)
	if !d.After(n) {
		return StateExploded
	}
	return StateTicking
}

// DaysRemaining returns days until deadline. Negative if exploded.
func (t Timebomb) DaysRemaining(now time.Time) int {
	d := dayOf(t.Deadline)
	n := dayOf(now)
	return int(d.Sub(n).Hours() / 24)
}

func dayOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
