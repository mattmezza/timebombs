package scanner

import (
	"strings"
	"testing"
)

func TestParse_SingleLine(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		wantLine int
		wantDate string
		wantID   string
		wantDesc string
	}{
		{
			name:     "python hash",
			src:      `# TIMEBOMB(2025-09-01): Remove v1 endpoints.`,
			wantLine: 1, wantDate: "2025-09-01", wantID: "", wantDesc: "Remove v1 endpoints.",
		},
		{
			name:     "typescript slash with id",
			src:      `// TIMEBOMB(2025-09-01, JIRA-123): Replace polling.`,
			wantLine: 1, wantDate: "2025-09-01", wantID: "JIRA-123", wantDesc: "Replace polling.",
		},
		{
			name:     "sql double dash",
			src:      `-- TIMEBOMB(2025-10-01): Drop legacy_users.`,
			wantLine: 1, wantDate: "2025-10-01", wantDesc: "Drop legacy_users.",
		},
		{
			name:     "ruby hash with hash id",
			src:      `# TIMEBOMB(2025-08-01, #317): Remove unsafe workaround.`,
			wantLine: 1, wantDate: "2025-08-01", wantID: "#317", wantDesc: "Remove unsafe workaround.",
		},
		{
			name:     "inline C block",
			src:      `/* TIMEBOMB(2025-11-15): Quick note. */`,
			wantLine: 1, wantDate: "2025-11-15", wantDesc: "Quick note.",
		},
		{
			name:     "haskell block",
			src:      `{- TIMEBOMB(2025-12-01): Refactor state monad. -}`,
			wantLine: 1, wantDate: "2025-12-01", wantDesc: "Refactor state monad.",
		},
		{
			name:     "erlang percent",
			src:      `% TIMEBOMB(2025-07-10): rewrite gen_server.`,
			wantLine: 1, wantDate: "2025-07-10", wantDesc: "rewrite gen_server.",
		},
		{
			name:     "lisp double semicolon",
			src:      `;; TIMEBOMB(2025-06-01): fix macro hygiene.`,
			wantLine: 1, wantDate: "2025-06-01", wantDesc: "fix macro hygiene.",
		},
		{
			name:     "batch rem",
			src:      `REM TIMEBOMB(2025-05-01): migrate build script.`,
			wantLine: 1, wantDate: "2025-05-01", wantDesc: "migrate build script.",
		},
		{
			name:     "vb apostrophe",
			src:      `' TIMEBOMB(2025-04-01): drop VB6 legacy.`,
			wantLine: 1, wantDate: "2025-04-01", wantDesc: "drop VB6 legacy.",
		},
		{
			name:     "indented marker",
			src:      "    // TIMEBOMB(2025-09-01): indented.",
			wantLine: 1, wantDate: "2025-09-01", wantDesc: "indented.",
		},
		{
			name:     "trailing whitespace in id",
			src:      `// TIMEBOMB(2025-09-01,  FLR-42 ): spaces around id.`,
			wantLine: 1, wantDate: "2025-09-01", wantID: "FLR-42", wantDesc: "spaces around id.",
		},
		{
			name:     "code then comment",
			src:      "foo()\n// TIMEBOMB(2026-01-01): second line.",
			wantLine: 2, wantDate: "2026-01-01", wantDesc: "second line.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Parse([]byte(c.src))
			if len(got) != 1 {
				t.Fatalf("got %d bombs, want 1: %+v", len(got), got)
			}
			tb := got[0]
			if tb.Line != c.wantLine {
				t.Errorf("line: got %d want %d", tb.Line, c.wantLine)
			}
			if tb.Deadline.Format("2006-01-02") != c.wantDate {
				t.Errorf("deadline: got %s want %s", tb.Deadline.Format("2006-01-02"), c.wantDate)
			}
			if tb.ID != c.wantID {
				t.Errorf("id: got %q want %q", tb.ID, c.wantID)
			}
			if tb.Description != c.wantDesc {
				t.Errorf("desc: got %q want %q", tb.Description, c.wantDesc)
			}
		})
	}
}

func TestParse_MultiLine_LineComments(t *testing.T) {
	src := `# TIMEBOMB(2025-09-01): Remove v1 endpoints after migration complete.
#   The new v2 endpoints are already serving 90% of traffic.
#   Blocked by: mobile app rollout to force-update v1 clients.
`
	got := Parse([]byte(src))
	if len(got) != 1 {
		t.Fatalf("got %d bombs", len(got))
	}
	want := "Remove v1 endpoints after migration complete.\nThe new v2 endpoints are already serving 90% of traffic.\nBlocked by: mobile app rollout to force-update v1 clients."
	if got[0].Description != want {
		t.Errorf("desc mismatch:\ngot  %q\nwant %q", got[0].Description, want)
	}
}

func TestParse_MultiLine_BlockComment(t *testing.T) {
	src := `/* TIMEBOMB(2025-11-15): Rip out the feature flag scaffolding.
   We shipped the experiment, the flag is always-on now, but there's
   still branching logic everywhere in the checkout flow. */
`
	got := Parse([]byte(src))
	if len(got) != 1 {
		t.Fatalf("got %d bombs: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Description, "Rip out") ||
		!strings.Contains(got[0].Description, "checkout flow") {
		t.Errorf("desc missing content: %q", got[0].Description)
	}
}

func TestParse_ContinuationStopsAtNonIndented(t *testing.T) {
	src := `// TIMEBOMB(2025-09-01): first.
// not a continuation (same indent).
`
	got := Parse([]byte(src))
	if len(got) != 1 {
		t.Fatalf("got %d bombs", len(got))
	}
	if got[0].Description != "first." {
		t.Errorf("desc should be single-line: %q", got[0].Description)
	}
}

func TestParse_ContinuationStopsAtBlank(t *testing.T) {
	src := `// TIMEBOMB(2025-09-01): first.
//   continuation.

//   not a continuation anymore.
`
	got := Parse([]byte(src))
	if len(got) != 1 {
		t.Fatalf("got %d bombs", len(got))
	}
	if got[0].Description != "first.\ncontinuation." {
		t.Errorf("desc: %q", got[0].Description)
	}
}

func TestParse_ContinuationStopsAtCode(t *testing.T) {
	src := `// TIMEBOMB(2025-09-01): first.
//   continuation.
foo()
`
	got := Parse([]byte(src))
	if got[0].Description != "first.\ncontinuation." {
		t.Errorf("desc: %q", got[0].Description)
	}
}

func TestParse_MultipleBombs(t *testing.T) {
	src := `// TIMEBOMB(2025-01-01): one.
// unrelated.
// TIMEBOMB(2025-02-02, X-1): two.
`
	got := Parse([]byte(src))
	if len(got) != 2 {
		t.Fatalf("got %d bombs", len(got))
	}
	if got[0].Line != 1 || got[1].Line != 3 {
		t.Errorf("lines: %d %d", got[0].Line, got[1].Line)
	}
	if got[1].ID != "X-1" {
		t.Errorf("id: %q", got[1].ID)
	}
}

func TestParse_IgnoresTodoFixme(t *testing.T) {
	src := `// TODO: not a bomb.
// FIXME: also not.
// HACK: nope.
// "timebomb" mentioned but no marker.
`
	got := Parse([]byte(src))
	if len(got) != 0 {
		t.Fatalf("expected 0 bombs, got %d", len(got))
	}
}

func TestParse_MalformedDateIgnored(t *testing.T) {
	src := `// TIMEBOMB(2025-13-99): bad date.
// TIMEBOMB(not-a-date): bad.
`
	got := Parse([]byte(src))
	if len(got) != 0 {
		t.Fatalf("expected 0 bombs, got %d: %+v", len(got), got)
	}
}

func TestParse_MissingColonIgnored(t *testing.T) {
	src := `// TIMEBOMB(2025-09-01) no colon here.
`
	got := Parse([]byte(src))
	if len(got) != 0 {
		t.Fatalf("expected 0 bombs, got %d", len(got))
	}
}

func TestParse_BlockStarStripped(t *testing.T) {
	src := `/*
 * TIMEBOMB(2025-10-10): javadoc style.
 *   continuation indented.
 */
`
	got := Parse([]byte(src))
	if len(got) != 1 {
		t.Fatalf("got %d bombs", len(got))
	}
	if !strings.Contains(got[0].Description, "javadoc style") ||
		!strings.Contains(got[0].Description, "continuation") {
		t.Errorf("desc: %q", got[0].Description)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	if got := Parse([]byte("")); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}
