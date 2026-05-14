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

// Package aggregator groups ProcSamples by user-supplied target directories.
package aggregator

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/k1LoW/dirtop/collector"
)

// Target is a directory the user asked dirtop to monitor.
type Target struct {
	Label   string
	AbsPath string
}

// NewTarget normalizes the path while preserving the original user-supplied label.
func NewTarget(input string) (Target, error) {
	abs, err := filepath.Abs(input)
	if err != nil {
		return Target{}, err
	}
	return Target{Label: input, AbsPath: filepath.Clean(abs)}, nil
}

// DirStat is the aggregated metric for a single Target.
type DirStat struct {
	Target Target                `json:"-"`
	Dir    string                `json:"dir"`
	PIDs   int                   `json:"pids"`
	CPU    float64               `json:"cpu_percent"`
	RSS    uint64                `json:"rss_bytes"`
	Top    []collector.ProcSample `json:"top_procs,omitempty"`
}

// Sort keys.
const (
	SortInput = "input"
	SortCPU   = "cpu"
	SortMem   = "mem"
	SortPIDs  = "pids"
)

// Options controls aggregation behavior.
type Options struct {
	Targets  []Target
	SortKey  string
	TopProcs int
}

// Aggregate buckets samples into per-target DirStats. Samples that don't
// match any target are dropped; ones that match multiple targets are
// assigned to the longest-prefix match (no double counting).
func Aggregate(samples []collector.ProcSample, opts Options) []DirStat {
	statsByIdx := make([]DirStat, len(opts.Targets))
	bucketsByIdx := make([][]collector.ProcSample, len(opts.Targets))
	for i, t := range opts.Targets {
		statsByIdx[i] = DirStat{Target: t, Dir: t.Label}
	}

	for _, s := range samples {
		idx := matchLongest(s.Cwd, opts.Targets)
		if idx < 0 {
			continue
		}
		statsByIdx[idx].PIDs++
		statsByIdx[idx].CPU += s.CPU
		statsByIdx[idx].RSS += s.RSS
		bucketsByIdx[idx] = append(bucketsByIdx[idx], s)
	}

	if opts.TopProcs > 0 {
		for i := range statsByIdx {
			top := sortProcs(bucketsByIdx[i], opts.SortKey)
			if len(top) > opts.TopProcs {
				top = top[:opts.TopProcs]
			}
			statsByIdx[i].Top = top
		}
	}

	sortStats(statsByIdx, opts.SortKey)
	return statsByIdx
}

// matchLongest returns the index of the longest-prefix-matching target, or -1.
func matchLongest(cwd string, targets []Target) int {
	cwd = filepath.Clean(cwd)
	best := -1
	bestLen := -1
	for i, t := range targets {
		if !isUnder(cwd, t.AbsPath) {
			continue
		}
		if len(t.AbsPath) > bestLen {
			best = i
			bestLen = len(t.AbsPath)
		}
	}
	return best
}

// isUnder reports whether `cwd` is `base` or a descendant of `base`, using
// path-component boundaries so that "/foo" does not match "/foobar".
func isUnder(cwd, base string) bool {
	if cwd == base {
		return true
	}
	sep := string(filepath.Separator)
	if base == sep {
		return strings.HasPrefix(cwd, sep)
	}
	return strings.HasPrefix(cwd, base+sep)
}

func sortProcs(procs []collector.ProcSample, key string) []collector.ProcSample {
	out := make([]collector.ProcSample, len(procs))
	copy(out, procs)
	switch key {
	case SortMem:
		sort.SliceStable(out, func(i, j int) bool { return out[i].RSS > out[j].RSS })
	case SortPIDs, SortCPU, SortInput, "":
		// Within a directory, "pids" / "input" / default all fall back to CPU,
		// since per-process pid count is always 1.
		sort.SliceStable(out, func(i, j int) bool { return out[i].CPU > out[j].CPU })
	default:
		sort.SliceStable(out, func(i, j int) bool { return out[i].CPU > out[j].CPU })
	}
	return out
}

func sortStats(stats []DirStat, key string) {
	switch key {
	case SortCPU:
		sort.SliceStable(stats, func(i, j int) bool { return stats[i].CPU > stats[j].CPU })
	case SortMem:
		sort.SliceStable(stats, func(i, j int) bool { return stats[i].RSS > stats[j].RSS })
	case SortPIDs:
		sort.SliceStable(stats, func(i, j int) bool { return stats[i].PIDs > stats[j].PIDs })
	case SortInput, "":
		// Keep input order.
	default:
		// Unknown key falls back to input order.
	}
}
