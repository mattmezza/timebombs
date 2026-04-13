package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mattmezza/timebombs/internal/model"
)

func TestWriteJSON_Schema(t *testing.T) {
	now := mustDate("2026-04-13")
	bombs := []model.Timebomb{
		{File: "a.py", Line: 42, Deadline: mustDate("2025-05-22"), ID: "JIRA-123",
			Description: "Remove v1 endpoints.\nStill used."},
		{File: "b.go", Line: 10, Deadline: mustDate("2099-01-01"), Description: "future."},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, bombs, now); err != nil {
		t.Fatal(err)
	}

	var parsed struct {
		ScannedAt string `json:"scanned_at"`
		Summary   struct {
			Total    int `json:"total"`
			Ticking  int `json:"ticking"`
			Exploded int `json:"exploded"`
		}
		Timebombs []struct {
			File          string `json:"file"`
			Line          int    `json:"line"`
			Deadline      string `json:"deadline"`
			ID            string `json:"id"`
			Description   string `json:"description"`
			State         string `json:"state"`
			DaysRemaining int    `json:"days_remaining"`
		}
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if parsed.ScannedAt != "2026-04-13" {
		t.Errorf("scanned_at: %q", parsed.ScannedAt)
	}
	if parsed.Summary.Total != 2 || parsed.Summary.Ticking != 1 || parsed.Summary.Exploded != 1 {
		t.Errorf("summary: %+v", parsed.Summary)
	}
	if len(parsed.Timebombs) != 2 {
		t.Fatalf("expected 2 bombs")
	}
	b0 := parsed.Timebombs[0]
	if b0.Deadline != "2025-05-22" || b0.State != "exploded" || b0.DaysRemaining >= 0 {
		t.Errorf("bomb 0 bad: %+v", b0)
	}
	if b0.ID != "JIRA-123" {
		t.Errorf("id: %q", b0.ID)
	}
	if b0.Description != "Remove v1 endpoints.\nStill used." {
		t.Errorf("desc: %q", b0.Description)
	}
	b1 := parsed.Timebombs[1]
	if b1.State != "ticking" || b1.DaysRemaining <= 0 {
		t.Errorf("bomb 1 bad: %+v", b1)
	}
}

func TestWriteJSON_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, nil, mustDate("2026-04-13")); err != nil {
		t.Fatal(err)
	}
	// Timebombs is an empty array, not null.
	if !bytes.Contains(buf.Bytes(), []byte(`"timebombs": []`)) {
		t.Errorf("expected empty array, got:\n%s", buf.String())
	}
}
