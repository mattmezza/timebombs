package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitInit initializes a git repo in dir with an initial commit, and returns
// the first commit's SHA.
func gitInit(t *testing.T, dir string) string {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "t@t.test")
	run("config", "user.name", "T")
	run("commit", "--allow-empty", "-q", "-m", "initial")

	sha, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return string(sha[:len(sha)-1])
}

func TestChangedFiles_CommittedVsBase(t *testing.T) {
	dir := t.TempDir()
	base := gitInit(t, dir)

	// Commit a new file on top of base.
	if err := os.WriteFile(filepath.Join(dir, "added.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "added.go"},
		{"commit", "-q", "-m", "add"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	got, err := ChangedFiles(dir, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "added.go" {
		t.Errorf("got %v want [added.go]", got)
	}
}

func TestChangedFiles_UncommittedAndUntracked(t *testing.T) {
	dir := t.TempDir()
	base := gitInit(t, dir)

	// Commit a tracked file at the base so we can modify it later.
	if err := os.WriteFile(filepath.Join(dir, "tracked.go"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "tracked.go"},
		{"commit", "-q", "-m", "tracked"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// Unstaged modification + new untracked file.
	if err := os.WriteFile(filepath.Join(dir, "tracked.go"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "brand_new.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ChangedFiles(dir, base)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"tracked.go": false, "brand_new.go": false}
	for _, p := range got {
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}
	for p, seen := range want {
		if !seen {
			t.Errorf("missing %s in %v", p, got)
		}
	}
}

func TestChangedFiles_BadBase(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)
	_, err := ChangedFiles(dir, "definitely-not-a-ref")
	if err == nil {
		t.Errorf("expected error for bad base")
	}
}
