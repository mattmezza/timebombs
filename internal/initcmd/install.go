// Package initcmd implements `timebombs init`: agent detection and skill
// installation.
package initcmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed templates/shared.md templates/claude-code/planting.md templates/claude-code/scanning.md templates/ci/github-actions.yml
var templatesFS embed.FS

// Mode controls how a skill is installed into an agent's file layout.
type Mode int

const (
	// ModeDedicatedFile writes a standalone file at Target.Path. If that file
	// already exists, the install is a no-op (we do not clobber user edits).
	ModeDedicatedFile Mode = iota
	// ModeAppend appends to Target.Path, creating it if missing. Idempotency
	// is enforced by scanning for the shared `## Timebombs` heading.
	ModeAppend
)

// Target is one file written for an agent.
type Target struct {
	Path     string // relative to repo root
	Template string // path within the embedded FS
	Mode     Mode
}

// Agent describes a supported AI coding agent.
type Agent struct {
	ID          string
	DisplayName string
	// DetectPaths are files/dirs that, if present, indicate this agent is in use.
	DetectPaths []string
	// Targets are the files written for this agent. Usually one; Claude Code
	// installs two skill files.
	Targets []Target
}

// Agents is the canonical list of supported agents.
var Agents = []Agent{
	{
		ID:          "claude-code",
		DisplayName: "Claude Code",
		DetectPaths: []string{".claude", "CLAUDE.md"},
		Targets: []Target{
			{
				Path:     ".claude/skills/timebombs-planting/SKILL.md",
				Template: "templates/claude-code/planting.md",
				Mode:     ModeDedicatedFile,
			},
			{
				Path:     ".claude/skills/timebombs-scanning/SKILL.md",
				Template: "templates/claude-code/scanning.md",
				Mode:     ModeDedicatedFile,
			},
		},
	},
	{
		ID:          "codex",
		DisplayName: "Codex CLI",
		DetectPaths: []string{"codex", "AGENTS.md"},
		Targets: []Target{{
			Path:     "AGENTS.md",
			Template: "templates/shared.md",
			Mode:     ModeAppend,
		}},
	},
	{
		ID:          "opencode",
		DisplayName: "OpenCode",
		DetectPaths: []string{".opencode"},
		Targets: []Target{{
			Path:     ".opencode/agents/timebombs.md",
			Template: "templates/shared.md",
			Mode:     ModeDedicatedFile,
		}},
	},
	{
		ID:          "cursor",
		DisplayName: "Cursor",
		DetectPaths: []string{".cursor", ".cursorrules"},
		Targets: []Target{{
			Path:     ".cursor/rules",
			Template: "templates/shared.md",
			Mode:     ModeAppend,
		}},
	},
	{
		ID:          "copilot",
		DisplayName: "GitHub Copilot",
		DetectPaths: []string{".github"},
		Targets: []Target{{
			Path:     ".github/copilot-instructions.md",
			Template: "templates/shared.md",
			Mode:     ModeAppend,
		}},
	},
}

// appendMarker is the heading used for idempotency in append-mode targets.
const appendMarker = "## Timebombs"

// loadTemplate returns the embedded template body.
func loadTemplate(path string) string {
	b, err := templatesFS.ReadFile(path)
	if err != nil {
		panic(err) // embedded; shouldn't happen
	}
	return string(b)
}

// DetectInstalled returns agents that appear to be in use in rootDir.
func DetectInstalled(rootDir string) []Agent {
	var out []Agent
	for _, a := range Agents {
		for _, p := range a.DetectPaths {
			if _, err := os.Stat(filepath.Join(rootDir, p)); err == nil {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

// Lookup finds an agent by ID.
func Lookup(id string) (Agent, bool) {
	for _, a := range Agents {
		if a.ID == id {
			return a, true
		}
	}
	return Agent{}, false
}

// TargetResult is one file's install outcome.
type TargetResult struct {
	Path    string
	Created bool
	Skipped bool
}

// InstallResult reports what happened for one agent.
type InstallResult struct {
	Agent   Agent
	Targets []TargetResult
}

// InstallForAgent installs every target for the given agent into rootDir.
func InstallForAgent(rootDir string, a Agent) (InstallResult, error) {
	res := InstallResult{Agent: a}
	for _, t := range a.Targets {
		tr, err := installTarget(rootDir, t)
		if err != nil {
			return res, err
		}
		res.Targets = append(res.Targets, tr)
	}
	return res, nil
}

func installTarget(rootDir string, t Target) (TargetResult, error) {
	target := filepath.Join(rootDir, t.Path)
	out := TargetResult{Path: target}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return out, err
	}
	content := loadTemplate(t.Template)

	switch t.Mode {
	case ModeDedicatedFile:
		// Never clobber user edits. If the file exists at all, skip.
		if _, err := os.Stat(target); err == nil {
			out.Skipped = true
			return out, nil
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return out, err
		}
		out.Created = true
		return out, nil

	case ModeAppend:
		existing, err := os.ReadFile(target)
		if err != nil && !os.IsNotExist(err) {
			return out, err
		}
		if err == nil && strings.Contains(string(existing), appendMarker) {
			out.Skipped = true
			return out, nil
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return out, err
		}
		defer f.Close()
		var prefix string
		switch {
		case len(existing) == 0:
			// new file
		case !strings.HasSuffix(string(existing), "\n"):
			prefix = "\n\n"
		default:
			prefix = "\n"
		}
		if _, err := f.WriteString(prefix + content); err != nil {
			return out, err
		}
		out.Created = len(existing) == 0
		return out, nil

	default:
		return out, fmt.Errorf("unknown install mode %d", t.Mode)
	}
}

// InstallCI writes a CI workflow for the given system into rootDir.
func InstallCI(rootDir, system string) (TargetResult, error) {
	switch system {
	case "github-actions":
		target := filepath.Join(rootDir, ".github/workflows/timebombs.yml")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return TargetResult{}, err
		}
		if _, err := os.Stat(target); err == nil {
			return TargetResult{Path: target, Skipped: true}, nil
		}
		b, err := templatesFS.ReadFile("templates/ci/github-actions.yml")
		if err != nil {
			return TargetResult{}, err
		}
		if err := os.WriteFile(target, b, 0o644); err != nil {
			return TargetResult{Path: target}, err
		}
		return TargetResult{Path: target, Created: true}, nil
	default:
		return TargetResult{}, fmt.Errorf("unsupported CI system %q (try: github-actions)", system)
	}
}

// ListAgents returns agents sorted by ID (for display).
func ListAgents() []Agent {
	out := append([]Agent(nil), Agents...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
