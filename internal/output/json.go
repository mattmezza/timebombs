package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/mattmezza/timebombs/internal/model"
)

type jsonSummary struct {
	Total    int `json:"total"`
	Ticking  int `json:"ticking"`
	Exploded int `json:"exploded"`
}

type jsonBomb struct {
	File          string `json:"file"`
	Line          int    `json:"line"`
	Deadline      string `json:"deadline"`
	ID            string `json:"id,omitempty"`
	Description   string `json:"description"`
	State         string `json:"state"`
	DaysRemaining int    `json:"days_remaining"`
}

type jsonReport struct {
	ScannedAt string      `json:"scanned_at"`
	Summary   jsonSummary `json:"summary"`
	Timebombs []jsonBomb  `json:"timebombs"`
}

// WriteJSON renders bombs as JSON per the spec schema.
func WriteJSON(w io.Writer, bombs []model.Timebomb, now time.Time) error {
	report := jsonReport{
		ScannedAt: now.Format("2006-01-02"),
		Timebombs: make([]jsonBomb, 0, len(bombs)),
	}
	for _, b := range bombs {
		state := b.State(now)
		if state == model.StateExploded {
			report.Summary.Exploded++
		} else {
			report.Summary.Ticking++
		}
		report.Summary.Total++
		report.Timebombs = append(report.Timebombs, jsonBomb{
			File:          b.File,
			Line:          b.Line,
			Deadline:      b.Deadline.Format("2006-01-02"),
			ID:            b.ID,
			Description:   b.Description,
			State:         string(state),
			DaysRemaining: b.DaysRemaining(now),
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
