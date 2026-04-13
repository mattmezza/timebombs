package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mattmezza/timebombs/internal/model"
)

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestWriteText_GroupsByFileAndSummary(t *testing.T) {
	now := mustDate("2026-04-13")
	bombs := []model.Timebomb{
		{File: "src/a.go", Line: 10, Deadline: mustDate("2025-01-01"), ID: "X-1", Description: "old."},
		{File: "src/a.go", Line: 5, Deadline: mustDate("2099-01-01"), Description: "future."},
		{File: "src/b.py", Line: 42, Deadline: mustDate("2026-04-20"), Description: "soon."},
	}
	var buf bytes.Buffer
	if err := WriteText(&buf, bombs, TextOptions{Now: now, NoColor: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Files appear in sorted order.
	ai := strings.Index(out, "src/a.go")
	bi := strings.Index(out, "src/b.py")
	if ai < 0 || bi < 0 || ai > bi {
		t.Errorf("file order wrong:\n%s", out)
	}
	// Within src/a.go, line 5 should come before line 10.
	l5 := strings.Index(out, "L5")
	l10 := strings.Index(out, "L10")
	if l5 < 0 || l10 < 0 || l5 > l10 {
		t.Errorf("line order wrong:\n%s", out)
	}
	if !strings.Contains(out, "[EXPLODED]") {
		t.Errorf("expected EXPLODED badge:\n%s", out)
	}
	if !strings.Contains(out, "ticking") {
		t.Errorf("expected ticking badge:\n%s", out)
	}
	if !strings.Contains(out, "3 timebombs: 2 ticking, 1 exploded") {
		t.Errorf("summary wrong:\n%s", out)
	}
}

func TestWriteText_EmptySummary(t *testing.T) {
	var buf bytes.Buffer
	_ = WriteText(&buf, nil, TextOptions{Now: mustDate("2026-04-13"), NoColor: true})
	if !strings.Contains(buf.String(), "0 timebombs") {
		t.Errorf("expected empty summary, got: %s", buf.String())
	}
}

func TestWriteText_MultilineDescShowsFirstLine(t *testing.T) {
	bombs := []model.Timebomb{
		{File: "x.go", Line: 1, Deadline: mustDate("2099-01-01"), Description: "first line.\nsecond line."},
	}
	var buf bytes.Buffer
	_ = WriteText(&buf, bombs, TextOptions{Now: mustDate("2026-04-13"), NoColor: true})
	out := buf.String()
	if !strings.Contains(out, "first line.") {
		t.Errorf("missing first line:\n%s", out)
	}
	if strings.Contains(out, "second line.") {
		t.Errorf("should not include second line:\n%s", out)
	}
}
