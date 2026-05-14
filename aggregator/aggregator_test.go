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

package aggregator

import (
	"testing"

	"github.com/k1LoW/dirtop/collector"
)

func TestIsUnder(t *testing.T) {
	cases := []struct {
		name string
		cwd  string
		base string
		want bool
	}{
		{"equal", "/foo", "/foo", true},
		{"child", "/foo/bar", "/foo", true},
		{"deep", "/foo/bar/baz", "/foo", true},
		{"sibling", "/foo/bar", "/foo/baz", false},
		{"prefix-only-no-sep", "/foobar", "/foo", false},
		{"root-base", "/anything", "/", true},
		{"unrelated", "/bar", "/foo", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isUnder(c.cwd, c.base); got != c.want {
				t.Errorf("isUnder(%q,%q)=%v, want %v", c.cwd, c.base, got, c.want)
			}
		})
	}
}

func TestMatchLongest(t *testing.T) {
	targets := []Target{
		{Label: "src", AbsPath: "/Users/k/src"},
		{Label: "src/foo", AbsPath: "/Users/k/src/foo"},
		{Label: "var", AbsPath: "/var"},
	}
	cases := []struct {
		name string
		cwd  string
		want int
	}{
		{"longest-prefix-wins", "/Users/k/src/foo/internal", 1},
		{"shallow-bucket", "/Users/k/src/bar", 0},
		{"no-match", "/etc", -1},
		{"exact-shallow", "/Users/k/src", 0},
		{"exact-deep", "/Users/k/src/foo", 1},
		{"var-match", "/var/log/something", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := matchLongest(c.cwd, targets); got != c.want {
				t.Errorf("matchLongest(%q)=%d, want %d", c.cwd, got, c.want)
			}
		})
	}
}

func TestAggregate_BasicSumAndInputOrder(t *testing.T) {
	targets := []Target{
		{Label: "a", AbsPath: "/a"},
		{Label: "b", AbsPath: "/b"},
	}
	samples := []collector.ProcSample{
		{PID: 1, Cwd: "/a", CPU: 10, RSS: 100},
		{PID: 2, Cwd: "/a/x", CPU: 5, RSS: 50},
		{PID: 3, Cwd: "/b", CPU: 1, RSS: 1000},
		{PID: 4, Cwd: "/elsewhere", CPU: 999, RSS: 999},
	}
	got := Aggregate(samples, Options{Targets: targets, SortKey: SortInput})
	if len(got) != 2 {
		t.Fatalf("want 2 rows, got %d", len(got))
	}
	// Input order preserved.
	if got[0].Dir != "a" || got[1].Dir != "b" {
		t.Errorf("input order broken: %v", []string{got[0].Dir, got[1].Dir})
	}
	if got[0].PIDs != 2 || got[0].CPU != 15 || got[0].RSS != 150 {
		t.Errorf("/a wrong: %+v", got[0])
	}
	if got[1].PIDs != 1 || got[1].CPU != 1 || got[1].RSS != 1000 {
		t.Errorf("/b wrong: %+v", got[1])
	}
}

func TestAggregate_LongestPrefixNoDoubleCount(t *testing.T) {
	targets := []Target{
		{Label: "shallow", AbsPath: "/x"},
		{Label: "deep", AbsPath: "/x/y"},
	}
	samples := []collector.ProcSample{
		{PID: 1, Cwd: "/x/y/z", CPU: 10, RSS: 100},
		{PID: 2, Cwd: "/x/other", CPU: 5, RSS: 50},
	}
	got := Aggregate(samples, Options{Targets: targets, SortKey: SortInput})
	if got[0].PIDs != 1 || got[0].CPU != 5 {
		t.Errorf("shallow should only have /x/other, got %+v", got[0])
	}
	if got[1].PIDs != 1 || got[1].CPU != 10 {
		t.Errorf("deep should have /x/y/z, got %+v", got[1])
	}
}

func TestAggregate_SortKeys(t *testing.T) {
	targets := []Target{
		{Label: "a", AbsPath: "/a"},
		{Label: "b", AbsPath: "/b"},
		{Label: "c", AbsPath: "/c"},
	}
	samples := []collector.ProcSample{
		{PID: 1, Cwd: "/a", CPU: 5, RSS: 1000},
		{PID: 2, Cwd: "/b", CPU: 100, RSS: 10},
		{PID: 3, Cwd: "/c", CPU: 1, RSS: 500},
		{PID: 4, Cwd: "/c", CPU: 1, RSS: 500},
		{PID: 5, Cwd: "/c", CPU: 1, RSS: 500},
	}
	cases := []struct {
		key  string
		want []string
	}{
		{SortCPU, []string{"b", "a", "c"}},
		{SortMem, []string{"c", "a", "b"}},
		{SortPIDs, []string{"c", "a", "b"}},
		{SortInput, []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			got := Aggregate(samples, Options{Targets: targets, SortKey: tc.key})
			gotLabels := []string{got[0].Dir, got[1].Dir, got[2].Dir}
			for i := range tc.want {
				if gotLabels[i] != tc.want[i] {
					t.Errorf("sort %s: got %v, want %v", tc.key, gotLabels, tc.want)
					break
				}
			}
		})
	}
}

func TestAggregate_TopProcs(t *testing.T) {
	targets := []Target{{Label: "a", AbsPath: "/a"}}
	samples := []collector.ProcSample{
		{PID: 1, Cwd: "/a", CPU: 10, RSS: 100, Name: "p1"},
		{PID: 2, Cwd: "/a", CPU: 30, RSS: 50, Name: "p2"},
		{PID: 3, Cwd: "/a", CPU: 20, RSS: 200, Name: "p3"},
		{PID: 4, Cwd: "/a", CPU: 1, RSS: 10, Name: "p4"},
	}

	got := Aggregate(samples, Options{Targets: targets, SortKey: SortCPU, TopProcs: 2})
	if len(got[0].Top) != 2 {
		t.Fatalf("want 2 top procs, got %d", len(got[0].Top))
	}
	if got[0].Top[0].Name != "p2" || got[0].Top[1].Name != "p3" {
		t.Errorf("CPU top procs wrong: %+v", got[0].Top)
	}

	got = Aggregate(samples, Options{Targets: targets, SortKey: SortMem, TopProcs: 2})
	if got[0].Top[0].Name != "p3" || got[0].Top[1].Name != "p1" {
		t.Errorf("Mem top procs wrong: %+v", got[0].Top)
	}

	got = Aggregate(samples, Options{Targets: targets, SortKey: SortCPU, TopProcs: 0})
	if got[0].Top != nil {
		t.Errorf("TopProcs=0 should leave Top nil, got %+v", got[0].Top)
	}
}
