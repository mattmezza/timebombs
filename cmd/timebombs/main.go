// Command timebombs scans a codebase for TIMEBOMB annotations.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/mattmezza/timebombs/internal/initcmd"
	"github.com/mattmezza/timebombs/internal/model"
	"github.com/mattmezza/timebombs/internal/output"
	"github.com/mattmezza/timebombs/internal/scanner"
)

// version is set via ldflags at release.
var version = "dev"

// errThresholdExceeded signals a non-zero exit due to --max-exploded.
var errThresholdExceeded = fmt.Errorf("threshold exceeded")

func main() {
	if err := newRootCmd().Execute(); err != nil {
		if err == errThresholdExceeded {
			os.Exit(1)
		}
		os.Exit(2)
	}
}

type scanFlags struct {
	format      string
	quiet       bool
	within      string
	atTime      string
	idPrefix    string
	exploded    bool
	ticking     bool
	maxExploded int
	exclude     []string
	include     []string
	noGitignore bool
	changedOnly bool
	base        string
	stdin       bool
	stdinName   string
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "timebombs [command] [path...]",
		Short:         "Scan a codebase for TIMEBOMB annotations.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(newScanCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newVersionCmd())

	// Default: if args look like paths and first isn't a known command,
	// fall through to `scan`. Cobra handles unknown subcommands, so we
	// register the scan command as the "default" via PersistentPreRunE
	// routing on the root.
	root.RunE = func(cmd *cobra.Command, args []string) error {
		return runScan(cmd, args, rootScanFlags)
	}
	attachScanFlags(root, &rootScanFlags)
	return root
}

var rootScanFlags scanFlags

func newScanCmd() *cobra.Command {
	var f scanFlags
	cmd := &cobra.Command{
		Use:   "scan [path...]",
		Short: "Scan paths for TIMEBOMB annotations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, f)
		},
	}
	attachScanFlags(cmd, &f)
	return cmd
}

func attachScanFlags(cmd *cobra.Command, f *scanFlags) {
	cmd.Flags().StringVar(&f.format, "format", "text", "Output format: text, json, sarif")
	cmd.Flags().BoolVar(&f.quiet, "quiet", false, "Suppress output; use exit code only")
	cmd.Flags().StringVar(&f.within, "within", "", "Only show bombs within a window (e.g. 30d, 2w, 3m, 1y)")
	cmd.Flags().StringVar(&f.atTime, "at-time", "", "Pretend now is this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.idPrefix, "id", "", "Filter by ID prefix")
	cmd.Flags().BoolVar(&f.exploded, "exploded", false, "Only show exploded bombs")
	cmd.Flags().BoolVar(&f.ticking, "ticking", false, "Only show ticking bombs")
	cmd.Flags().IntVar(&f.maxExploded, "max-exploded", -1, "Exit non-zero if more than N bombs are exploded")
	cmd.Flags().StringSliceVar(&f.exclude, "exclude", nil, "Exclude glob (doublestar; repeatable)")
	cmd.Flags().StringSliceVar(&f.include, "include", nil, "Only scan files matching this glob (doublestar; repeatable)")
	cmd.Flags().BoolVar(&f.noGitignore, "no-gitignore", false, "Ignore .gitignore when walking")
	cmd.Flags().BoolVar(&f.changedOnly, "changed-only", false, "Scan only files changed vs --base (committed, staged, unstaged, untracked)")
	cmd.Flags().StringVar(&f.base, "base", "origin/main", "Base ref for --changed-only")
	cmd.Flags().BoolVar(&f.stdin, "stdin", false, "Read a single file from stdin instead of walking paths")
	cmd.Flags().StringVar(&f.stdinName, "stdin-filename", "<stdin>", "Display name used for --stdin results")
}

type initFlags struct {
	agent string
	all   bool
	list  bool
	ci    string
	root  string
}

func newInitCmd() *cobra.Command {
	var f initFlags
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install the timebombs skill for AI coding agents in this repo.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, f)
		},
	}
	cmd.Flags().StringVar(&f.agent, "agent", "", "Install for a specific agent (see --list)")
	cmd.Flags().BoolVar(&f.all, "all", false, "Install for every supported agent")
	cmd.Flags().BoolVar(&f.list, "list", false, "List supported agents and their install targets")
	cmd.Flags().StringVar(&f.ci, "ci", "", "Also generate a CI workflow (github-actions)")
	cmd.Flags().StringVar(&f.root, "root", ".", "Repo root to install into")
	return cmd
}

func runInit(cmd *cobra.Command, f initFlags) error {
	out := cmd.OutOrStdout()

	if f.list {
		fmt.Fprintln(out, "Supported agents:")
		for _, a := range initcmd.ListAgents() {
			fmt.Fprintf(out, "  %-12s  %s\n    target: %s\n", a.ID, a.DisplayName, a.Target)
		}
		return nil
	}

	var targets []initcmd.Agent
	switch {
	case f.agent != "" && f.all:
		return fmt.Errorf("--agent and --all are mutually exclusive")
	case f.all:
		targets = initcmd.ListAgents()
	case f.agent != "":
		a, ok := initcmd.Lookup(f.agent)
		if !ok {
			return fmt.Errorf("unknown agent %q (run `timebombs init --list`)", f.agent)
		}
		targets = []initcmd.Agent{a}
	default:
		targets = initcmd.DetectInstalled(f.root)
		if len(targets) == 0 {
			fmt.Fprintln(out, "No agents auto-detected. Use --agent <id> or --all. See --list.")
		}
	}

	for _, a := range targets {
		res, err := initcmd.InstallForAgent(f.root, a)
		if err != nil {
			return fmt.Errorf("install %s: %w", a.ID, err)
		}
		switch {
		case res.Skipped:
			fmt.Fprintf(out, "  skipped  %s (already installed: %s)\n", a.DisplayName, res.Path)
		case res.Created:
			fmt.Fprintf(out, "  created  %s  %s\n", a.DisplayName, res.Path)
		default:
			fmt.Fprintf(out, "  appended %s  %s\n", a.DisplayName, res.Path)
		}
	}

	if f.ci != "" {
		res, err := initcmd.InstallCI(f.root, f.ci)
		if err != nil {
			return err
		}
		if res.Skipped {
			fmt.Fprintf(out, "  skipped  CI workflow (already present: %s)\n", res.Path)
		} else {
			fmt.Fprintf(out, "  created  CI workflow  %s\n", res.Path)
		}
	}
	return nil
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and exit.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}

func runScan(cmd *cobra.Command, args []string, f scanFlags) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	now := time.Now().UTC()
	if f.atTime != "" {
		t, err := time.Parse("2006-01-02", f.atTime)
		if err != nil {
			return fmt.Errorf("invalid --at-time %q: %w", f.atTime, err)
		}
		now = t
	}

	var withinDays int
	hasWithin := false
	if f.within != "" {
		d, err := scanner.ParseDuration(f.within)
		if err != nil {
			return fmt.Errorf("invalid --within %q: %w", f.within, err)
		}
		withinDays = d
		hasWithin = true
	}

	if f.exploded && f.ticking {
		return fmt.Errorf("--exploded and --ticking are mutually exclusive")
	}
	if f.stdin && f.changedOnly {
		return fmt.Errorf("--stdin and --changed-only are mutually exclusive")
	}

	opts := scanner.Options{
		Exclude:      f.exclude,
		Include:      f.include,
		UseGitignore: !f.noGitignore,
	}

	var bombs []model.Timebomb
	var err error
	switch {
	case f.stdin:
		bombs, err = scanStdin(cmd.InOrStdin(), f.stdinName)
	case f.changedOnly:
		bombs, err = scanChangedOnly(paths, f.base, opts)
	default:
		bombs, err = scanner.Scan(paths, opts)
	}
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	// Apply filters.
	filtered := make([]model.Timebomb, 0, len(bombs))
	explodedCount := 0
	for _, b := range bombs {
		state := b.State(now)
		if state == model.StateExploded {
			explodedCount++
		}
		if f.exploded && state != model.StateExploded {
			continue
		}
		if f.ticking && state != model.StateTicking {
			continue
		}
		if f.idPrefix != "" && !strings.HasPrefix(b.ID, f.idPrefix) {
			continue
		}
		if hasWithin && b.DaysRemaining(now) > withinDays {
			continue
		}
		filtered = append(filtered, b)
	}

	// Stable ordering.
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].File != filtered[j].File {
			return filtered[i].File < filtered[j].File
		}
		return filtered[i].Line < filtered[j].Line
	})

	// Render.
	if !f.quiet {
		w := cmd.OutOrStdout()
		if err := render(w, filtered, f.format, now); err != nil {
			return err
		}
	}

	// Gating.
	if f.maxExploded >= 0 && explodedCount > f.maxExploded {
		cmd.SilenceErrors = true
		return errThresholdExceeded
	}
	return nil
}

// scanStdin reads a single file's contents from r and parses it. filename is
// used as the display path in results.
func scanStdin(r io.Reader, filename string) ([]model.Timebomb, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	bombs := scanner.Parse(data)
	for i := range bombs {
		bombs[i].File = filename
	}
	return bombs, nil
}

// scanChangedOnly resolves the list of files changed vs base (from the first
// path arg that is a git repo; defaults to cwd) and scans each of them.
// Include/Exclude globs still apply; the globs are evaluated relative to the
// repo root.
func scanChangedOnly(paths []string, base string, opts scanner.Options) ([]model.Timebomb, error) {
	repoRoot := "."
	if len(paths) > 0 {
		repoRoot = paths[0]
	}
	files, err := scanner.ChangedFiles(repoRoot, base)
	if err != nil {
		return nil, err
	}
	var out []model.Timebomb
	for _, rel := range files {
		if scanner.MatchAny(opts.Exclude, rel) {
			continue
		}
		if len(opts.Include) > 0 && !scanner.MatchAny(opts.Include, rel) {
			continue
		}
		abs := filepath.Join(repoRoot, rel)
		if fi, err := os.Stat(abs); err != nil || !fi.Mode().IsRegular() {
			// Deleted / non-regular — skip.
			continue
		}
		bombs, err := scanner.Scan([]string{abs}, scanner.Options{
			// gitignore/include/exclude already applied above; keep defaults here.
			MaxFileSize: opts.MaxFileSize,
		})
		if err != nil {
			return nil, err
		}
		// Rewrite File to the repo-relative path for nicer output.
		for i := range bombs {
			bombs[i].File = filepath.ToSlash(rel)
		}
		out = append(out, bombs...)
	}
	return out, nil
}

func render(w io.Writer, bombs []model.Timebomb, format string, now time.Time) error {
	switch format {
	case "text", "":
		noColor := !isTerminal(w)
		return output.WriteText(w, bombs, output.TextOptions{Now: now, NoColor: noColor})
	case "json":
		return output.WriteJSON(w, bombs, now)
	case "sarif":
		return output.WriteSARIF(w, bombs, now, version)
	default:
		return fmt.Errorf("unknown --format %q (use text, json, sarif)", format)
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	if color.NoColor {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}
