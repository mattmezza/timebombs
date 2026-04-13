package scanner

import (
	"path/filepath"
	"strings"
	"testing"
)

// testdataRoot resolves the repo-root testdata/ dir from this package.
func testdataRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestFixtures_Python(t *testing.T) {
	bombs, err := Scan([]string{filepath.Join(testdataRoot(t), "python")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 2 {
		t.Fatalf("want 2, got %d: %+v", len(bombs), bombs)
	}
	// First bomb is multi-line.
	if !strings.Contains(bombs[0].Description, "Remove v1 endpoints") ||
		!strings.Contains(bombs[0].Description, "Blocked by") {
		t.Errorf("multi-line desc incomplete: %q", bombs[0].Description)
	}
	// Second has id PY-1.
	if bombs[1].ID != "PY-1" {
		t.Errorf("id: got %q want PY-1", bombs[1].ID)
	}
}

func TestFixtures_TypeScript(t *testing.T) {
	bombs, err := Scan([]string{filepath.Join(testdataRoot(t), "typescript")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 1 {
		t.Fatalf("want 1, got %d", len(bombs))
	}
	if bombs[0].ID != "JIRA-123" {
		t.Errorf("id: %q", bombs[0].ID)
	}
	if !strings.Contains(bombs[0].Description, "WebSocket") ||
		!strings.Contains(bombs[0].Description, "polling interval is 5s") {
		t.Errorf("desc: %q", bombs[0].Description)
	}
}

func TestFixtures_Go_BlockComment(t *testing.T) {
	bombs, err := Scan([]string{filepath.Join(testdataRoot(t), "go")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 1 {
		t.Fatalf("want 1, got %d", len(bombs))
	}
	if !strings.Contains(bombs[0].Description, "feature flag") ||
		!strings.Contains(bombs[0].Description, "checkout flow") {
		t.Errorf("desc: %q", bombs[0].Description)
	}
}

func TestFixtures_Ruby(t *testing.T) {
	bombs, err := Scan([]string{filepath.Join(testdataRoot(t), "ruby")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 1 || bombs[0].ID != "#317" {
		t.Fatalf("unexpected: %+v", bombs)
	}
}

func TestFixtures_SQL(t *testing.T) {
	bombs, err := Scan([]string{filepath.Join(testdataRoot(t), "sql")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 1 {
		t.Fatalf("want 1, got %d", len(bombs))
	}
	if !strings.Contains(bombs[0].Description, "legacy_users") {
		t.Errorf("desc: %q", bombs[0].Description)
	}
}

func TestFixtures_Mixed(t *testing.T) {
	bombs, err := Scan([]string{filepath.Join(testdataRoot(t), "mixed")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 3 {
		t.Fatalf("want 3, got %d: %+v", len(bombs), bombs)
	}
}

func TestFixtures_AllAtOnce(t *testing.T) {
	bombs, err := Scan([]string{testdataRoot(t)}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// 2 + 1 + 1 + 1 + 1 + 3 = 9
	if len(bombs) != 9 {
		t.Fatalf("want 9, got %d", len(bombs))
	}
}
