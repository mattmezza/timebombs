// Package output renders Timebomb results in various formats.
package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/mattmezza/timebombs/internal/model"
)

// TextOptions controls text rendering.
type TextOptions struct {
	Now        time.Time
	NoColor    bool
	WithinDays int // used to decide "soon" highlighting; defaults to 30 if 0
}

// WriteText renders bombs as a grouped, colored report.
func WriteText(w io.Writer, bombs []model.Timebomb, opts TextOptions) error {
	if opts.WithinDays == 0 {
		opts.WithinDays = 30
	}
	if opts.NoColor {
		color.NoColor = true
	}

	// Group by file.
	byFile := map[string][]model.Timebomb{}
	for _, b := range bombs {
		byFile[b.File] = append(byFile[b.File], b)
	}
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	ticking, exploded := 0, 0
	for _, b := range bombs {
		if b.State(opts.Now) == model.StateExploded {
			exploded++
		} else {
			ticking++
		}
	}

	red := color.New(color.FgRed, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	fileColor := color.New(color.FgCyan, color.Bold).SprintFunc()

	first := true
	for _, file := range files {
		list := byFile[file]
		sort.Slice(list, func(i, j int) bool { return list[i].Line < list[j].Line })

		if !first {
			fmt.Fprintln(w)
		}
		first = false

		fmt.Fprintln(w, fileColor(file))

		// Compute column widths for this file group.
		maxLineW, maxIDW, maxDescW := 0, 0, 0
		descs := make([]string, len(list))
		for i, b := range list {
			lw := len(fmt.Sprintf("L%d", b.Line))
			if lw > maxLineW {
				maxLineW = lw
			}
			if len(b.ID) > maxIDW {
				maxIDW = len(b.ID)
			}
			descs[i] = firstLine(b.Description)
			if len(descs[i]) > maxDescW {
				maxDescW = len(descs[i])
			}
		}

		for i, b := range list {
			state := b.State(opts.Now)
			days := b.DaysRemaining(opts.Now)

			lineStr := fmt.Sprintf("L%d", b.Line)
			deadline := b.Deadline.Format("2006-01-02")

			var deadlineStr, badge string
			switch state {
			case model.StateExploded:
				deadlineStr = red(deadline)
				badge = red("[EXPLODED]")
			case model.StateTicking:
				if days <= opts.WithinDays {
					deadlineStr = yellow(deadline)
				} else {
					deadlineStr = dim(deadline)
				}
				badge = fmt.Sprintf("[ticking, %dd]", days)
				if days <= opts.WithinDays {
					badge = yellow(badge)
				} else {
					badge = dim(badge)
				}
			}

			fmt.Fprintf(w, "  %-*s  %s  %-*s  %-*s  %s\n",
				maxLineW, lineStr,
				deadlineStr,
				maxIDW, b.ID,
				maxDescW, descs[i],
				badge,
			)
		}
	}

	if len(bombs) > 0 {
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "%d timebomb%s: %d ticking, %d exploded\n",
		len(bombs), plural(len(bombs)), ticking, exploded)
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
