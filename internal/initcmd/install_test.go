package initcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectInstalled_None(t *testing.T) {
	dir := t.TempDir()
	if got := DetectInstalled(dir); len(got) != 0 {
		t.Errorf("expected no agents, got %+v", got)
	}
}

func TestDetectInstalled_ClaudeCode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	agents := DetectInstalled(dir)
	if len(agents) != 1 || agents[0].ID != "claude-code" {
		t.Errorf("want [claude-code], got %+v", agents)
	}
}

func TestDetectInstalled_Multiple(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".cursor"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".github"), 0o755)

	agents := DetectInstalled(dir)
	ids := map[string]bool{}
	for _, a := range agents {
		ids[a.ID] = true
	}
	for _, want := range []string{"claude-code", "cursor", "copilot"} {
		if !ids[want] {
			t.Errorf("missing %s in %+v", want, ids)
		}
	}
}

func TestInstall_DedicatedFile(t *testing.T) {
	dir := t.TempDir()
	a, _ := Lookup("claude-code")
	res, err := InstallForAgent(dir, a)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created {
		t.Errorf("expected Created=true, got %+v", res)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".claude/timebombs.md"))
	if err != nil {
		t.Fatalf("target file not written: %v", err)
	}
	if !strings.Contains(string(data), "## Timebombs") {
		t.Errorf("missing delimiter heading")
	}
}

func TestInstall_DedicatedFile_Idempotent(t *testing.T) {
	dir := t.TempDir()
	a, _ := Lookup("claude-code")
	if _, err := InstallForAgent(dir, a); err != nil {
		t.Fatal(err)
	}
	res, err := InstallForAgent(dir, a)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped {
		t.Errorf("second install should be skipped, got %+v", res)
	}
}

func TestInstall_AppendNewFile(t *testing.T) {
	dir := t.TempDir()
	a, _ := Lookup("copilot")
	res, err := InstallForAgent(dir, a)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created {
		t.Errorf("expected Created=true, got %+v", res)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".github/copilot-instructions.md"))
	if !strings.Contains(string(data), "## Timebombs") {
		t.Errorf("missing heading:\n%s", data)
	}
}

func TestInstall_AppendExistingFile_Idempotent(t *testing.T) {
	dir := t.TempDir()
	a, _ := Lookup("copilot")
	target := filepath.Join(dir, ".github/copilot-instructions.md")
	os.MkdirAll(filepath.Dir(target), 0o755)
	os.WriteFile(target, []byte("# Project rules\n\nDo things carefully.\n"), 0o644)

	res, err := InstallForAgent(dir, a)
	if err != nil {
		t.Fatal(err)
	}
	if res.Created || res.Skipped {
		t.Errorf("want append-to-existing (not created, not skipped): %+v", res)
	}
	data, _ := os.ReadFile(target)
	if !strings.Contains(string(data), "Do things carefully") {
		t.Errorf("clobbered existing content:\n%s", data)
	}
	if !strings.Contains(string(data), "## Timebombs") {
		t.Errorf("did not append skill:\n%s", data)
	}

	// Second run is a no-op.
	res2, _ := InstallForAgent(dir, a)
	if !res2.Skipped {
		t.Errorf("second install should skip: %+v", res2)
	}
	// Content should not have been appended again.
	data2, _ := os.ReadFile(target)
	if strings.Count(string(data2), "## Timebombs") != 1 {
		t.Errorf("heading duplicated on re-run:\n%s", data2)
	}
}

func TestInstallCI_GitHubActions(t *testing.T) {
	dir := t.TempDir()
	res, err := InstallCI(dir, "github-actions")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created {
		t.Errorf("expected Created=true: %+v", res)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".github/workflows/timebombs.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "timebombs scan") {
		t.Errorf("workflow missing scan cmd:\n%s", data)
	}

	// Idempotent.
	res2, _ := InstallCI(dir, "github-actions")
	if !res2.Skipped {
		t.Errorf("second CI install should skip: %+v", res2)
	}
}

func TestInstallCI_Unknown(t *testing.T) {
	_, err := InstallCI(t.TempDir(), "bitbucket")
	if err == nil {
		t.Errorf("expected error for unknown CI system")
	}
}
