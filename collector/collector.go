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

// Package collector samples per-process CPU and memory usage and reports those
// whose current working directory could be obtained.
package collector

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sync/errgroup"
)

// ProcSample is the per-process measurement at one sampling round.
type ProcSample struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	Cmdline string  `json:"cmdline,omitempty"`
	Cwd     string  `json:"cwd"`
	CPU     float64 `json:"cpu_percent"`
	RSS     uint64  `json:"rss_bytes"`
}

// defaultParallel bounds concurrent syscalls so we don't open too many FDs at once.
const defaultParallel = 32

// Collect lists all running processes and samples each over the given interval.
// Processes whose cwd cannot be read are silently dropped.
func Collect(ctx context.Context, interval time.Duration) ([]ProcSample, error) { //nostyle:repetition
	return defaultSource.Collect(ctx, interval)
}

// source abstracts the process listing/sampling backend so tests can inject fakes.
type source interface { //nostyle:ifacenames
	Collect(ctx context.Context, interval time.Duration) ([]ProcSample, error)
}

var defaultSource source = &gopsutilSource{parallel: defaultParallel}

type gopsutilSource struct {
	parallel int
}

func (g *gopsutilSource) Collect(ctx context.Context, interval time.Duration) ([]ProcSample, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	parallel := g.parallel
	if parallel <= 0 {
		parallel = defaultParallel
	}

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(parallel)

	var (
		mu      sync.Mutex
		samples = make([]ProcSample, 0, len(procs))
	)

	for _, p := range procs {
		eg.Go(func() error {
			cwd, err := p.CwdWithContext(egCtx)
			if err != nil || cwd == "" {
				return nil
			}
			cpu, err := p.PercentWithContext(egCtx, interval)
			if err != nil {
				return nil
			}
			mem, err := p.MemoryInfoWithContext(egCtx)
			if err != nil || mem == nil {
				return nil
			}
			name, err := p.NameWithContext(egCtx)
			if err != nil {
				name = ""
			}
			cmdline, err := p.CmdlineWithContext(egCtx)
			if err != nil {
				cmdline = ""
			}
			sample := ProcSample{
				PID:     p.Pid,
				Name:    name,
				Cmdline: cmdline,
				Cwd:     cwd,
				CPU:     cpu,
				RSS:     mem.RSS,
			}
			mu.Lock()
			samples = append(samples, sample)
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return samples, nil
}
