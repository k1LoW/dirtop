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
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/k1LoW/dirtop/aggregator"
)

func newTestModel() *watchModel {
	return &watchModel{
		opts: WatchOptions{
			AggOpts: aggregator.Options{
				Targets: []aggregator.Target{{Label: "/a", AbsPath: "/a"}},
				SortKey: aggregator.SortInput,
			},
		},
		stats: []aggregator.DirStat{{Dir: "/a"}},
	}
}

func TestWatchModel_QuitKeys(t *testing.T) {
	for _, key := range []string{"q", "ctrl+c", "esc"} {
		t.Run(key, func(t *testing.T) {
			m := newTestModel()
			var msg tea.KeyMsg
			switch key {
			case "q":
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
			case "ctrl+c":
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			case "esc":
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			}
			_, cmd := m.Update(msg)
			if cmd == nil {
				t.Fatalf("expected quit cmd for %s, got nil", key)
			}
			got := cmd()
			if _, ok := got.(tea.QuitMsg); !ok {
				t.Errorf("expected QuitMsg for %s, got %T", key, got)
			}
		})
	}
}

func TestWatchModel_TickUpdatesStats(t *testing.T) {
	m := newTestModel()
	newStats := []aggregator.DirStat{{Dir: "/a", PIDs: 5, CPU: 99}}
	_, _ = m.Update(tickMsg{stats: newStats})
	if len(m.stats) != 1 || m.stats[0].PIDs != 5 {
		t.Errorf("stats not updated: %+v", m.stats)
	}
	if m.err != nil {
		t.Errorf("err should be nil on success, got %v", m.err)
	}
}

func TestWatchModel_TickError(t *testing.T) {
	m := newTestModel()
	want := errors.New("boom")
	_, _ = m.Update(tickMsg{err: want})
	if !errors.Is(m.err, want) {
		t.Errorf("err not stored: %v", m.err)
	}
	if !strings.Contains(m.View(), "boom") {
		t.Errorf("View should show error; got %q", m.View())
	}
}
