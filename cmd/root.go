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

package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/k1LoW/dirtop/aggregator"
	"github.com/k1LoW/dirtop/collector"
	"github.com/k1LoW/dirtop/tui"
	"github.com/k1LoW/dirtop/version"
)

var (
	flagSort     string
	flagTopProcs int
	flagDepth    int
	flagFullCmd  bool
	flagJSON     bool
	flagWatch    bool
	flagInterval time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "dirtop DIR [DIR...]",
	Short: "dirtop lists per-directory CPU and memory usage of running processes",
	Long: `dirtop lists per-directory CPU and memory usage of running processes.

Each argument is a directory to monitor. Processes whose current working directory
(cwd) is at or under one of the given directories are aggregated into a single row.
Processes whose cwd cannot be read (typically those owned by other users) are
silently dropped.`,
	Version:      version.Version,
	SilenceUsage: true,
	Args:         cobra.MinimumNArgs(1),
	RunE:         runRoot,
}

// Execute is the entry point used by main.go.
func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	log.SetOutput(io.Discard)
	if env := os.Getenv("DEBUG"); env != "" {
		log.SetOutput(os.Stderr)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	f := rootCmd.Flags()
	f.StringVar(&flagSort, "sort", aggregator.SortInput, "sort rows by: input | cpu | mem | pids")
	f.IntVar(&flagTopProcs, "top-procs", 0, "show top N processes per directory (0 = off)")
	f.IntVar(&flagDepth, "depth", 0, "expand each DIR into subdirectories exactly N levels below (0 = off)")
	f.BoolVar(&flagFullCmd, "full-cmd", false, "show full command line of nested top-procs rows")
	f.BoolVar(&flagJSON, "json", false, "output as JSON")
	f.BoolVarP(&flagWatch, "watch", "w", false, "continuously refresh (TUI)")
	f.DurationVar(&flagInterval, "interval", 500*time.Millisecond, "CPU sampling interval")
}

func runRoot(cmd *cobra.Command, args []string) error {
	targets, err := buildTargets(args)
	if err != nil {
		return err
	}
	if flagDepth > 0 {
		targets, err = expandTargets(targets, flagDepth)
		if err != nil {
			return err
		}
	}

	opts := aggregator.Options{
		Targets:  targets,
		SortKey:  flagSort,
		TopProcs: flagTopProcs,
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tableOpts := tui.TableOpts{FullCmd: flagFullCmd}

	if flagWatch {
		if flagJSON {
			return fmt.Errorf("--watch and --json cannot be combined")
		}
		return tui.RunWatch(ctx, collector.Collect, tui.WatchOptions{
			Interval:  flagInterval,
			AggOpts:   opts,
			TableOpts: tableOpts,
		})
	}

	samples, err := collector.Collect(ctx, flagInterval)
	if err != nil {
		return err
	}
	stats := aggregator.Aggregate(samples, opts)

	if flagJSON {
		return tui.WriteJSON(cmd.OutOrStdout(), stats)
	}
	return tui.WriteTable(cmd.OutOrStdout(), stats, tableOpts)
}

func buildTargets(args []string) ([]aggregator.Target, error) {
	targets := make([]aggregator.Target, 0, len(args))
	for _, a := range args {
		t, err := aggregator.NewTarget(a)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", a, err)
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// expandTargets replaces each target with its subdirectories exactly `depth`
// levels below it. Labels keep the original prefix so output stays readable.
// Unreadable subdirectories (permission denied, missing) are skipped.
func expandTargets(in []aggregator.Target, depth int) ([]aggregator.Target, error) {
	out := make([]aggregator.Target, 0, len(in))
	for _, t := range in {
		subs, err := findSubdirsAtDepth(t.AbsPath, depth)
		if err != nil {
			return nil, fmt.Errorf("expand %q: %w", t.Label, err)
		}
		for _, sub := range subs {
			rel, err := filepath.Rel(t.AbsPath, sub)
			if err != nil {
				rel = filepath.Base(sub)
			}
			out = append(out, aggregator.Target{
				Label:   filepath.Join(t.Label, rel),
				AbsPath: sub,
			})
		}
	}
	return out, nil
}

// findSubdirsAtDepth returns subdirectories exactly `depth` levels below root.
// Directories whose entries cannot be read are skipped silently.
func findSubdirsAtDepth(root string, depth int) ([]string, error) {
	if depth <= 0 {
		return []string{root}, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		child := filepath.Join(root, e.Name())
		if depth == 1 {
			out = append(out, child)
			continue
		}
		deeper, err := findSubdirsAtDepth(child, depth-1)
		if err != nil {
			continue
		}
		out = append(out, deeper...)
	}
	return out, nil
}

