package scanner

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ChangedFiles returns the set of files that differ from base in the current
// working tree, plus any untracked files (respecting .gitignore).
//
// base may be a ref like "origin/main", "HEAD~1", or a commit SHA. The scope
// matches what a reviewer would see on a PR: committed, staged, unstaged, and
// untracked changes, compared against base's merge point.
//
// Returned paths are relative to repoRoot, using the OS path separator.
func ChangedFiles(repoRoot, base string) ([]string, error) {
	if base == "" {
		base = "origin/main"
	}

	// `git diff --name-only <base>` shows working-tree vs base (committed +
	// staged + unstaged). It does not show untracked files.
	diff, err := runGit(repoRoot, "diff", "--name-only", base)
	if err != nil {
		// Fall back to merge-base-style comparison if "origin/main" doesn't
		// exist locally; give a clearer error pointing at --base.
		return nil, fmt.Errorf("git diff vs %q failed (try --base <ref>): %w", base, err)
	}
	untracked, err := runGit(repoRoot, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	set := map[string]struct{}{}
	for _, line := range append(splitLines(diff), splitLines(untracked)...) {
		if line == "" {
			continue
		}
		set[filepath.FromSlash(line)] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	return out, nil
}

func splitLines(b []byte) []string {
	s := strings.TrimRight(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
