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

//go:embed templates/skill.md templates/ci/github-actions.yml
var templatesFS embed.FS

// Mode controls how a skill is installed into an agent's file layout.
type Mode int

const (
	// ModeDedicatedFile writes a standalone file at Target.
	ModeDedicatedFile Mode = iota
	// ModeAppend appends to Target, creating it if missing.
	ModeAppend
)

// Agent describes a supported AI coding agent.
type Agent struct {
	// ID is the stable, user-facing identifier (e.g., "claude-code").
	ID string
	// DisplayName is the human label (e.g., "Claude Code").
	DisplayName string
	// DetectPaths are files/dirs that, if present, indicate this agent is in use.
	DetectPaths []string
	// Target is the path (relative to the repo root) where the skill is installed.
	Target string
	// Mode is how the skill content is written into the target.
	Mode Mode
}

// Agents is the canonical list of supported agents.
var Agents = []Agent{
	{
		ID:          "claude-code",
		DisplayName: "Claude Code",
		DetectPaths: []string{".claude", "CLAUDE.md"},
		Target:      ".claude/timebombs.md",
		Mode:        ModeDedicatedFile,
	},
	{
		ID:          "codex",
		DisplayName: "Codex CLI",
		DetectPaths: []string{"codex", "AGENTS.md"},
		Target:      "AGENTS.md",
		Mode:        ModeAppend,
	},
	{
		ID:          "opencode",
		DisplayName: "OpenCode",
		DetectPaths: []string{".opencode"},
		Target:      ".opencode/agents/timebombs.md",
		Mode:        ModeDedicatedFile,
	},
	{
		ID:          "cursor",
		DisplayName: "Cursor",
		DetectPaths: []string{".cursor", ".cursorrules"},
		Target:      ".cursor/rules",
		Mode:        ModeAppend,
	},
	{
		ID:          "copilot",
		DisplayName: "GitHub Copilot",
		DetectPaths: []string{".github"},
		Target:      ".github/copilot-instructions.md",
		Mode:        ModeAppend,
	},
}

// delimiterHeading marks the skill block within append-mode files. Used for
// idempotency checks.
const delimiterHeading = "## Timebombs"

// SkillContent returns the embedded skill template body.
func SkillContent() string {
	b, err := templatesFS.ReadFile("templates/skill.md")
	if err != nil {
		// Embedded, should never happen.
		panic(err)
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

// InstallResult reports what happened for one agent.
type InstallResult struct {
	Agent   Agent
	Path    string
	Created bool // file newly created
	Skipped bool // content already present (idempotent no-op)
}

// InstallForAgent installs the skill for a single agent into rootDir.
func InstallForAgent(rootDir string, a Agent) (InstallResult, error) {
	target := filepath.Join(rootDir, a.Target)
	res := InstallResult{Agent: a, Path: target}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return res, err
	}

	content := SkillContent()

	switch a.Mode {
	case ModeDedicatedFile:
		// Idempotent: if file exists and already contains the skill, skip.
		if existing, err := os.ReadFile(target); err == nil {
			if strings.Contains(string(existing), delimiterHeading) {
				res.Skipped = true
				return res, nil
			}
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return res, err
		}
		res.Created = true
		return res, nil

	case ModeAppend:
		existing, err := os.ReadFile(target)
		if err != nil && !os.IsNotExist(err) {
			return res, err
		}
		if err == nil && strings.Contains(string(existing), delimiterHeading) {
			res.Skipped = true
			return res, nil
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return res, err
		}
		defer f.Close()
		var prefix string
		if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
			prefix = "\n\n"
		} else if len(existing) > 0 {
			prefix = "\n"
		}
		if _, err := f.WriteString(prefix + content); err != nil {
			return res, err
		}
		res.Created = len(existing) == 0
		return res, nil

	default:
		return res, fmt.Errorf("unknown install mode %d", a.Mode)
	}
}

// InstallCI writes a CI workflow for the given system into rootDir.
// Currently only "github-actions" is supported.
func InstallCI(rootDir, system string) (InstallResult, error) {
	switch system {
	case "github-actions":
		target := filepath.Join(rootDir, ".github/workflows/timebombs.yml")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return InstallResult{}, err
		}
		if _, err := os.Stat(target); err == nil {
			return InstallResult{Path: target, Skipped: true}, nil
		}
		b, err := templatesFS.ReadFile("templates/ci/github-actions.yml")
		if err != nil {
			return InstallResult{}, err
		}
		if err := os.WriteFile(target, b, 0o644); err != nil {
			return InstallResult{Path: target}, err
		}
		return InstallResult{Path: target, Created: true}, nil
	default:
		return InstallResult{}, fmt.Errorf("unsupported CI system %q (try: github-actions)", system)
	}
}

// ListAgents returns agents sorted by ID (for display).
func ListAgents() []Agent {
	out := append([]Agent(nil), Agents...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
