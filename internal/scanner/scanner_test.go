package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScan_Directory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.py"), "# TIMEBOMB(2025-01-01): exploded.\n")
	writeFile(t, filepath.Join(dir, "b.go"), "// TIMEBOMB(2099-01-01): future.\n")
	writeFile(t, filepath.Join(dir, "c.txt"), "nothing here.\n")

	bombs, err := Scan([]string{dir}, Options{UseGitignore: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 2 {
		t.Fatalf("got %d bombs, want 2: %+v", len(bombs), bombs)
	}
}

func TestScan_RespectsGitignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "ignored/\n*.skip\n")
	writeFile(t, filepath.Join(dir, "kept.go"), "// TIMEBOMB(2025-01-01): keep.\n")
	writeFile(t, filepath.Join(dir, "ignored", "x.go"), "// TIMEBOMB(2025-01-01): skip.\n")
	writeFile(t, filepath.Join(dir, "foo.skip"), "// TIMEBOMB(2025-01-01): skip.\n")

	bombs, err := Scan([]string{dir}, Options{UseGitignore: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 1 {
		t.Fatalf("got %d bombs, want 1: %+v", len(bombs), bombs)
	}
	if !filepath.IsAbs(bombs[0].File) && bombs[0].File == "" {
		t.Errorf("expected file path set")
	}
}

func TestScan_Exclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "keep.go"), "// TIMEBOMB(2025-01-01): keep.\n")
	writeFile(t, filepath.Join(dir, "skip", "x.go"), "// TIMEBOMB(2025-01-01): skip.\n")

	bombs, err := Scan([]string{dir}, Options{Exclude: []string{"skip/**"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 1 {
		t.Fatalf("got %d bombs, want 1", len(bombs))
	}
}

func TestScan_SkipsBinary(t *testing.T) {
	dir := t.TempDir()
	// File with a null byte = treated as binary.
	content := "// TIMEBOMB(2025-01-01): shouldnotappear.\n\x00\x01\x02binary"
	writeFile(t, filepath.Join(dir, "bin.dat"), content)

	bombs, err := Scan([]string{dir}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 0 {
		t.Fatalf("got %d bombs, want 0", len(bombs))
	}
}

func TestScan_SkipsAlwaysSkipDirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "node_modules", "x.js"), "// TIMEBOMB(2025-01-01): skip.\n")
	writeFile(t, filepath.Join(dir, "src", "x.js"), "// TIMEBOMB(2025-01-01): keep.\n")

	bombs, err := Scan([]string{dir}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 1 {
		t.Fatalf("got %d bombs, want 1", len(bombs))
	}
}

func TestScan_SingleFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.go")
	writeFile(t, p, "// TIMEBOMB(2025-01-01): only.\n")

	bombs, err := Scan([]string{p}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != 1 {
		t.Fatalf("got %d bombs, want 1", len(bombs))
	}
}

func TestScan_ParallelManyFiles(t *testing.T) {
	// Stress the worker pool with enough files to force multiple batches.
	dir := t.TempDir()
	const n = 200
	for i := 0; i < n; i++ {
		name := filepath.Join(dir, "pkg", "f", "f.go")
		if i%2 == 0 {
			name = filepath.Join(dir, "a", "file.go")
		}
		// Unique file per i so we actually write n files.
		p := filepath.Join(dir, "dir", "f"+itoa(i)+".go")
		_ = name
		writeFile(t, p, "// TIMEBOMB(2099-01-01): bomb "+itoa(i)+".\n")
	}

	bombs, err := Scan([]string{dir}, Options{Workers: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(bombs) != n {
		t.Fatalf("got %d bombs, want %d", len(bombs), n)
	}

	// Serial pass yields the same set (content-wise).
	serial, err := Scan([]string{dir}, Options{Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(serial) != n {
		t.Fatalf("serial got %d, want %d", len(serial), n)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want int
		err  bool
	}{
		{"30d", 30, false},
		{"2w", 14, false},
		{"3m", 90, false},
		{"1y", 365, false},
		{"0d", 0, false},
		{"abc", 0, true},
		{"30", 0, true},
		{"", 0, true},
		{"1h", 0, true},
	}
	for _, c := range cases {
		got, err := ParseDuration(c.in)
		if c.err {
			if err == nil {
				t.Errorf("ParseDuration(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDuration(%q): unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseDuration(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
