package model

import (
	"testing"
	"time"
)

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestState(t *testing.T) {
	now := mustDate("2026-04-13")
	cases := []struct {
		deadline string
		want     State
	}{
		{"2026-04-14", StateTicking},
		{"2026-04-13", StateExploded}, // today = exploded
		{"2026-04-12", StateExploded},
		{"2030-01-01", StateTicking},
	}
	for _, c := range cases {
		tb := Timebomb{Deadline: mustDate(c.deadline)}
		if got := tb.State(now); got != c.want {
			t.Errorf("deadline=%s: got %s want %s", c.deadline, got, c.want)
		}
	}
}

func TestDaysRemaining(t *testing.T) {
	now := mustDate("2026-04-13")
	cases := []struct {
		deadline string
		want     int
	}{
		{"2026-04-13", 0},
		{"2026-04-14", 1},
		{"2026-04-10", -3},
		{"2026-05-13", 30},
	}
	for _, c := range cases {
		tb := Timebomb{Deadline: mustDate(c.deadline)}
		if got := tb.DaysRemaining(now); got != c.want {
			t.Errorf("deadline=%s: got %d want %d", c.deadline, got, c.want)
		}
	}
}
