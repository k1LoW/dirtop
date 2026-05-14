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
	"bytes"
	"strings"
	"testing"

	"github.com/k1LoW/dirtop/aggregator"
	"github.com/k1LoW/dirtop/collector"
)

func TestWriteTable_Basic(t *testing.T) {
	stats := []aggregator.DirStat{
		{Dir: "~/src/foo", PIDs: 12, CPU: 245.3, RSS: 1288490188},
		{Dir: "~/src/bar", PIDs: 4, CPU: 12.1, RSS: 537919488},
	}
	var buf bytes.Buffer
	if err := WriteTable(&buf, stats, TableOpts{}); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"DIR", "PID", "COMMAND", "CPU%", "MEM(RSS)", "~/src/foo", "(12)", "245.3", "1.2GiB"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteTable_WithTopProcs(t *testing.T) {
	stats := []aggregator.DirStat{
		{
			Dir: "~/src/foo", PIDs: 2, CPU: 200, RSS: 1024 * 1024 * 100,
			Top: []collector.ProcSample{
				{PID: 1234, Name: "node", CPU: 120, RSS: 1024 * 1024 * 50},
				{PID: 5678, Name: "go", CPU: 80, RSS: 1024 * 1024 * 50},
			},
		},
	}
	var buf bytes.Buffer
	if err := WriteTable(&buf, stats, TableOpts{}); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"node", "go", "1234", "5678", "120.0", "80.0"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteTable_PIDAlignment(t *testing.T) {
	stats := []aggregator.DirStat{
		{
			Dir: "/x",
			Top: []collector.ProcSample{
				{PID: 7, Name: "a", CPU: 1, RSS: 1},
				{PID: 19363, Name: "b", CPU: 1, RSS: 1},
			},
		},
	}
	var buf bytes.Buffer
	if err := WriteTable(&buf, stats, TableOpts{}); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	got := buf.String()
	// PID 7 should be right-padded to align under 19363's digit width (5).
	if !strings.Contains(got, "    7") {
		t.Errorf("expected PID 7 right-aligned to digit width 5, got:\n%s", got)
	}
	if !strings.Contains(got, "19363") {
		t.Errorf("expected PID 19363 in output, got:\n%s", got)
	}
}

func TestWriteTable_RelativeCwd(t *testing.T) {
	stats := []aggregator.DirStat{
		{
			Dir:    "/x",
			Target: aggregator.Target{Label: "/x", AbsPath: "/x"},
			Top: []collector.ProcSample{
				{PID: 1, Name: "p1", Cwd: "/x", CPU: 1, RSS: 1},
				{PID: 2, Name: "p2", Cwd: "/x/sub", CPU: 1, RSS: 1},
				{PID: 3, Name: "p3", Cwd: "/x/sub/deep", CPU: 1, RSS: 1},
			},
		},
	}
	var buf bytes.Buffer
	if err := WriteTable(&buf, stats, TableOpts{}); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"p1 (./)", "p2 (sub/)", "p3 (sub/deep/)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteTable_FullCmd(t *testing.T) {
	stats := []aggregator.DirStat{
		{
			Dir: "/x",
			Top: []collector.ProcSample{
				{PID: 1, Name: "main", Cmdline: "/var/folders/.../main . --top-procs 5"},
			},
		},
	}
	var buf bytes.Buffer
	if err := WriteTable(&buf, stats, TableOpts{FullCmd: true}); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "--top-procs 5") {
		t.Errorf("expected cmdline shown with FullCmd=true, got:\n%s", got)
	}

	buf.Reset()
	if err := WriteTable(&buf, stats, TableOpts{FullCmd: false}); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	got = buf.String()
	if strings.Contains(got, "--top-procs 5") {
		t.Errorf("expected name only with FullCmd=false, got:\n%s", got)
	}
	if !strings.Contains(got, "main") {
		t.Errorf("expected name 'main' with FullCmd=false, got:\n%s", got)
	}
}

func TestWriteJSON_Basic(t *testing.T) {
	stats := []aggregator.DirStat{
		{Dir: "/a", PIDs: 1, CPU: 10, RSS: 100},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, stats); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	got := buf.String()
	for _, want := range []string{`"dir": "/a"`, `"pids": 1`, `"cpu_percent": 10`, `"rss_bytes": 100`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// top_procs is omitempty when empty.
	if strings.Contains(got, "top_procs") {
		t.Errorf("expected top_procs omitted, got:\n%s", got)
	}
}
