// Command timebombs scans a codebase for TIMEBOMB annotations.
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

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
	noGitignore bool
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "timebombs [command] [path...]",
		Short:         "Scan a codebase for TIMEBOMB annotations.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(newScanCmd())
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
	cmd.Flags().BoolVar(&f.noGitignore, "no-gitignore", false, "Ignore .gitignore when walking")
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

	bombs, err := scanner.Scan(paths, scanner.Options{
		Exclude:      f.exclude,
		UseGitignore: !f.noGitignore,
	})
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
