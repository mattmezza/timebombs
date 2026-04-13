// Package scanner parses source files for TIMEBOMB annotations.
package scanner

import (
	"regexp"
	"strings"
	"time"

	"github.com/mattmezza/timebombs/internal/model"
)

// bombRe matches a TIMEBOMB(deadline[, id]): description annotation.
// Description runs to end of line; multi-line handling is done above.
var bombRe = regexp.MustCompile(`TIMEBOMB\((\d{4}-\d{2}-\d{2})(?:\s*,\s*([^)]*?))?\s*\)\s*:\s*(.*)$`)

// Parse extracts timebombs from a single file's contents.
func Parse(content []byte) []model.Timebomb {
	lines := strings.Split(string(content), "\n")
	infos := classify(lines)

	var out []model.Timebomb
	i := 0
	for i < len(infos) {
		info := infos[i]
		if !info.isComment {
			i++
			continue
		}
		loc := strings.Index(info.inner, "TIMEBOMB(")
		if loc < 0 {
			i++
			continue
		}
		m := bombRe.FindStringSubmatch(info.inner[loc:])
		if m == nil {
			i++
			continue
		}
		deadline, err := time.Parse("2006-01-02", m[1])
		if err != nil {
			i++
			continue
		}
		id := strings.TrimSpace(m[2])
		desc := strings.TrimRight(strings.TrimSpace(m[3]), " \t")
		markerCol := info.innerCol + loc
		markerBlockID := info.blockID

		// Collect continuations.
		j := i + 1
		for j < len(infos) {
			c := infos[j]
			if !c.isComment {
				break
			}
			trimmed := strings.TrimLeft(c.inner, " \t")
			if strings.TrimSpace(trimmed) == "" {
				break
			}
			if markerBlockID > 0 {
				// Same block = continuation.
				if c.blockID != markerBlockID {
					break
				}
			} else {
				// Line-comment group: must be indented past marker and not a block.
				if c.blockID > 0 {
					break
				}
				contentCol := c.innerCol + (len(c.inner) - len(trimmed))
				if contentCol <= markerCol {
					break
				}
			}
			desc += "\n" + strings.TrimRight(trimmed, " \t")
			j++
		}

		out = append(out, model.Timebomb{
			Line:        i + 1,
			Deadline:    deadline,
			ID:          id,
			Description: desc,
		})
		i = j
	}
	return out
}

type lineInfo struct {
	isComment bool
	inner     string // content after comment prefix stripped
	innerCol  int    // column (0-indexed) in original line where inner begins
	blockID   int    // 0 if not in block comment; otherwise unique block id
}

// classify tags each line with comment info, tracking block-comment state.
func classify(lines []string) []lineInfo {
	out := make([]lineInfo, len(lines))
	inBlock := false
	blockClose := ""
	blockID := 0
	curBlock := 0

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			// Blank ends line-comment groups; still inside block if applicable
			// but treated as non-comment for continuation purposes.
			out[i] = lineInfo{}
			continue
		}
		if inBlock {
			t := strings.TrimLeft(line, " \t")
			lead := len(line) - len(t)
			inner := t
			innerCol := lead
			// Strip optional leading `*` (but not `*/`).
			if strings.HasPrefix(t, "*") && !strings.HasPrefix(t, "*/") {
				inner = t[1:]
				innerCol = lead + 1
			}
			if idx := strings.Index(inner, blockClose); idx >= 0 {
				inner = inner[:idx]
				out[i] = lineInfo{isComment: true, inner: inner, innerCol: innerCol, blockID: curBlock}
				inBlock = false
				curBlock = 0
				blockClose = ""
				continue
			}
			out[i] = lineInfo{isComment: true, inner: inner, innerCol: innerCol, blockID: curBlock}
			continue
		}

		trimmedL := strings.TrimLeft(line, " \t")
		leading := len(line) - len(trimmedL)
		low := strings.ToLower(trimmedL)

		var prefix string
		switch {
		case strings.HasPrefix(trimmedL, "//"):
			prefix = "//"
		case strings.HasPrefix(trimmedL, "/*"):
			prefix = "/*"
		case strings.HasPrefix(trimmedL, "{-"):
			prefix = "{-"
		case strings.HasPrefix(trimmedL, "--"):
			prefix = "--"
		case strings.HasPrefix(trimmedL, ";;"):
			prefix = ";;"
		case strings.HasPrefix(trimmedL, "#"):
			prefix = "#"
		case strings.HasPrefix(trimmedL, "%"):
			prefix = "%"
		case strings.HasPrefix(trimmedL, "'"):
			prefix = "'"
		case strings.HasPrefix(low, "rem ") || strings.HasPrefix(low, "rem\t"):
			prefix = trimmedL[:3]
		default:
			out[i] = lineInfo{}
			continue
		}

		inner := trimmedL[len(prefix):]
		innerCol := leading + len(prefix)

		if prefix == "/*" || prefix == "{-" {
			closeTok := "*/"
			if prefix == "{-" {
				closeTok = "-}"
			}
			blockID++
			thisBlock := blockID
			if idx := strings.Index(inner, closeTok); idx >= 0 {
				// Block opens and closes on the same line.
				inner = inner[:idx]
				out[i] = lineInfo{isComment: true, inner: inner, innerCol: innerCol, blockID: thisBlock}
				continue
			}
			inBlock = true
			curBlock = thisBlock
			blockClose = closeTok
			out[i] = lineInfo{isComment: true, inner: inner, innerCol: innerCol, blockID: thisBlock}
			continue
		}

		out[i] = lineInfo{isComment: true, inner: inner, innerCol: innerCol}
	}
	return out
}
