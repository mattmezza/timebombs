package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.py"),
		"# TIMEBOMB(2025-05-22, JIRA-123): Remove v1 endpoints.\n"+
			"#   still used by legacy clients.\n"+
			"def f(): pass\n"+
			"# TIMEBOMB(2030-01-01): future.\n")
	mustWrite(t, filepath.Join(dir, "b.go"),
		"/* TIMEBOMB(2026-05-01, FLR-42): Flag cleanup.\n"+
			"   still branching. */\n")
	return dir
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCLIStdin(t, nil, args...)
}

func runCLIStdin(t *testing.T, stdin []byte, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := newRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	if stdin != nil {
		root.SetIn(bytes.NewReader(stdin))
	}
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestCLI_ScanDefault(t *testing.T) {
	dir := setupFixture(t)
	out, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out, "3 timebombs") {
		t.Errorf("want 3 timebombs; got:\n%s", out)
	}
	if !strings.Contains(out, "[EXPLODED]") {
		t.Errorf("want EXPLODED badge")
	}
}

func TestCLI_Exploded(t *testing.T) {
	dir := setupFixture(t)
	out, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13", "--exploded")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1 timebomb: 0 ticking, 1 exploded") {
		t.Errorf("wrong filter result:\n%s", out)
	}
}

func TestCLI_IDPrefix(t *testing.T) {
	dir := setupFixture(t)
	out, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13", "--id", "FLR-")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "FLR-42") {
		t.Errorf("missing FLR-42:\n%s", out)
	}
	if strings.Contains(out, "JIRA-") {
		t.Errorf("should not include JIRA-:\n%s", out)
	}
}

func TestCLI_Within(t *testing.T) {
	dir := setupFixture(t)
	out, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13", "--within", "60d")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "future.") {
		t.Errorf("distant bomb should not appear with --within 60d:\n%s", out)
	}
}

func TestCLI_MaxExplodedFail(t *testing.T) {
	dir := setupFixture(t)
	_, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13", "--max-exploded", "0", "--quiet")
	if err != errThresholdExceeded {
		t.Errorf("expected threshold-exceeded error, got %v", err)
	}
}

func TestCLI_MaxExplodedOK(t *testing.T) {
	dir := setupFixture(t)
	_, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13", "--max-exploded", "5", "--quiet")
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestCLI_Version(t *testing.T) {
	out, _, err := runCLI(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) == "" {
		t.Errorf("version empty")
	}
}

func TestCLI_MutuallyExclusiveFilters(t *testing.T) {
	dir := setupFixture(t)
	_, _, err := runCLI(t, "scan", dir, "--exploded", "--ticking")
	if err == nil {
		t.Errorf("expected error on --exploded + --ticking")
	}
}

func TestCLI_BadAtTime(t *testing.T) {
	_, _, err := runCLI(t, "scan", ".", "--at-time", "not-a-date")
	if err == nil {
		t.Errorf("expected error on bad --at-time")
	}
}

func TestCLI_JSONFormat(t *testing.T) {
	dir := setupFixture(t)
	out, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"scanned_at": "2026-04-13"`) {
		t.Errorf("missing scanned_at: %s", out)
	}
	if !strings.Contains(out, `"state": "exploded"`) {
		t.Errorf("missing state: %s", out)
	}
}

func TestCLI_SARIFFormat(t *testing.T) {
	dir := setupFixture(t)
	out, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13", "--format", "sarif")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"version": "2.1.0"`) {
		t.Errorf("missing sarif version")
	}
}

func TestCLI_InitList(t *testing.T) {
	out, _, err := runCLI(t, "init", "--list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "claude-code") || !strings.Contains(out, "codex") {
		t.Errorf("missing agents:\n%s", out)
	}
}

func TestCLI_InitAgent(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runCLI(t, "init", "--agent", "claude-code", "--root", dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		".claude/skills/timebombs-planting/SKILL.md",
		".claude/skills/timebombs-scanning/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected skill file %s: %v", rel, err)
		}
	}
}

func TestCLI_InitUnknownAgent(t *testing.T) {
	_, _, err := runCLI(t, "init", "--agent", "not-a-thing")
	if err == nil {
		t.Errorf("expected error for unknown agent")
	}
}

func TestCLI_Stdin(t *testing.T) {
	in := []byte("# TIMEBOMB(2099-01-01): stdin bomb.\n")
	out, _, err := runCLIStdin(t, in, "scan", "--stdin", "--at-time", "2026-04-13", "--stdin-filename", "pipe.py")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "stdin bomb.") {
		t.Errorf("stdin bomb not found:\n%s", out)
	}
	if !strings.Contains(out, "pipe.py") {
		t.Errorf("stdin-filename not used:\n%s", out)
	}
}

func TestCLI_StdinConflictsWithChangedOnly(t *testing.T) {
	_, _, err := runCLIStdin(t, []byte(""), "scan", "--stdin", "--changed-only")
	if err == nil {
		t.Errorf("expected mutual-exclusion error")
	}
}

func TestCLI_Include(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "keep.py"), "# TIMEBOMB(2099-01-01): keep.\n")
	mustWrite(t, filepath.Join(dir, "skip.go"), "// TIMEBOMB(2099-01-01): skip.\n")

	out, _, err := runCLI(t, "scan", dir, "--at-time", "2026-04-13", "--include", "**/*.py")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "skip.") {
		t.Errorf("skip.go should not appear with include=*.py:\n%s", out)
	}
	if !strings.Contains(out, "keep.") {
		t.Errorf("keep.py should appear:\n%s", out)
	}
}

func TestCLI_ChangedOnly(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("config", "user.email", "t@t.test")
	git("config", "user.name", "T")
	git("commit", "--allow-empty", "-q", "-m", "initial")

	mustWrite(t, filepath.Join(dir, "old.py"), "# TIMEBOMB(2099-01-01): old.\n")
	git("add", "old.py")
	git("commit", "-q", "-m", "old")

	baseSHA, _ := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	base := strings.TrimSpace(string(baseSHA))

	// Add a new file after base — this is the only "changed" file.
	mustWrite(t, filepath.Join(dir, "new.py"), "# TIMEBOMB(2099-01-01): new.\n")

	out, _, err := runCLI(t, "scan", dir,
		"--at-time", "2026-04-13",
		"--changed-only",
		"--base", base)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "new.") {
		t.Errorf("new.py should appear:\n%s", out)
	}
	if strings.Contains(out, "old.") {
		t.Errorf("old.py (unchanged vs base) should NOT appear:\n%s", out)
	}
}
