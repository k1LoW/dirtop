/*
Copyright © 2026 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package tui renders DirStats as a table (one-shot) or as a periodically
// redrawn watch view. Both modes share the same table format.
package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/k1LoW/dirtop/aggregator"
	"github.com/k1LoW/dirtop/collector"
)

// TableOpts controls how the directory table is rendered.
type TableOpts struct {
	// FullCmd makes nested top-procs rows show the full command line
	// (`Cmdline`) instead of the short process name.
	FullCmd bool
}

// WriteTable writes the directory-summary table to w. Same format used by --watch.
//
// Layout: DIR | PID | COMMAND | CPU% | MEM(RSS).
// Parent rows hold the directory path and a process count rendered as `(N)`
// in the PID column. Nested top-procs rows place a tree marker (`├─` /
// `└─`) in the DIR column and fill in PID (right-aligned by digit width)
// and COMMAND (process name, or full cmdline when --full-cmd is set).
func WriteTable(w io.Writer, stats []aggregator.DirStat, opts TableOpts) error {
	pidWidth := maxPIDWidth(stats)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "DIR\tPID(S)\tCOMMAND\tCPU%\tMEM(RSS)"); err != nil {
		return err
	}
	for _, s := range stats {
		if _, err := fmt.Fprintf(tw, "%s\t(%d)\t\t%.1f\t%s\n", s.Dir, s.PIDs, s.CPU, HumanBytes(s.RSS)); err != nil {
			return err
		}
		for i, p := range s.Top {
			marker := treeBranch
			if i == len(s.Top)-1 {
				marker = treeLast
			}
			pidCell := fmt.Sprintf("%*d", pidWidth, p.PID)
			cmdCell := fmt.Sprintf("%s (%s)", procDisplay(p, opts), relCwd(s.Target.AbsPath, p.Cwd))
			if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%.1f\t%s\n", marker, pidCell, cmdCell, p.CPU, HumanBytes(p.RSS)); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}

// RenderTable is a convenience wrapper returning the table as a string.
// Used by --watch to feed the same renderer through the TUI library.
func RenderTable(stats []aggregator.DirStat, opts TableOpts) string {
	var sb strings.Builder
	_ = WriteTable(&sb, stats, opts)
	return sb.String()
}

// relCwd returns the process cwd relative to the target directory, with a
// trailing separator so it visibly reads as a directory. Falls back to the
// absolute cwd when targetAbs is empty or filepath.Rel fails.
func relCwd(targetAbs, cwd string) string {
	rel := cwd
	if targetAbs != "" {
		if r, err := filepath.Rel(targetAbs, cwd); err == nil {
			rel = r
		}
	}
	sep := string(filepath.Separator)
	if !strings.HasSuffix(rel, sep) {
		rel += sep
	}
	return rel
}

// procDisplay returns what to show as the human-friendly identifier of a process.
// Falls back to Name if Cmdline is empty (e.g. permission denied).
func procDisplay(p collector.ProcSample, opts TableOpts) string {
	if opts.FullCmd && p.Cmdline != "" {
		return p.Cmdline
	}
	if p.Name != "" {
		return p.Name
	}
	return "?"
}

// maxPIDWidth returns the column width needed to display all PIDs in the
// nested top-procs rows. Falls back to 1 when there are no nested rows.
func maxPIDWidth(stats []aggregator.DirStat) int {
	w := 1
	for _, s := range stats {
		for _, p := range s.Top {
			if n := len(strconv.FormatInt(int64(p.PID), 10)); n > w {
				w = n
			}
		}
	}
	return w
}

// Tree markers for nested rows.
const (
	treeBranch = "├─"
	treeLast   = "└─"
)
