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

package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/k1LoW/dirtop/aggregator"
	"github.com/k1LoW/dirtop/collector"
)

// Sampler produces a fresh set of ProcSamples each tick.
type Sampler func(ctx context.Context, interval time.Duration) ([]collector.ProcSample, error)

// WatchOptions configures the watch loop.
type WatchOptions struct {
	Interval  time.Duration
	AggOpts   aggregator.Options
	TableOpts TableOpts
}

// RunWatch starts the TUI loop. Returns when the user quits (q / Ctrl-C / esc)
// or ctx is canceled.
func RunWatch(ctx context.Context, sampler Sampler, opts WatchOptions) error {
	m := &watchModel{
		ctx:     ctx,
		sampler: sampler,
		opts:    opts,
		// Render an empty table immediately so the layout doesn't jump on first tick.
		stats: aggregator.Aggregate(nil, opts.AggOpts),
	}
	p := tea.NewProgram(m, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

type watchModel struct {
	ctx     context.Context
	sampler Sampler
	opts    WatchOptions
	stats   []aggregator.DirStat
	err     error
}

type tickMsg struct {
	stats []aggregator.DirStat
	err   error
}

func (m *watchModel) Init() tea.Cmd {
	return m.tick()
}

func (m *watchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tickMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.stats = msg.stats
		}
		return m, m.tick()
	}
	return m, nil
}

func (m *watchModel) View() string {
	if m.err != nil {
		return "error: " + m.err.Error() + "\n"
	}
	return RenderTable(m.stats, m.opts.TableOpts)
}

func (m *watchModel) tick() tea.Cmd {
	return func() tea.Msg {
		samples, err := m.sampler(m.ctx, m.opts.Interval)
		if err != nil {
			return tickMsg{err: err}
		}
		return tickMsg{stats: aggregator.Aggregate(samples, m.opts.AggOpts)}
	}
}
