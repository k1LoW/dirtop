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
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/k1LoW/dirtop/aggregator"
)

// buildTree creates the given paths (directories and files) under root.
// Paths ending with "/" are directories; otherwise empty files.
func buildTree(t *testing.T, root string, paths []string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(root, p)
		if filepath.Ext(p) != "" || (len(p) > 0 && p[len(p)-1] != '/') {
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindSubdirsAtDepth(t *testing.T) {
	root := t.TempDir()
	buildTree(t, root, []string{
		"a/",
		"b/c/",
		"b/d/e/",
		"file.txt",
	})

	cases := []struct {
		name  string
		depth int
		want  []string
	}{
		{"depth1", 1, []string{"a", "b"}},
		{"depth2", 2, []string{"b/c", "b/d"}},
		{"depth3", 3, []string{"b/d/e"}},
		{"depth4", 4, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := findSubdirsAtDepth(root, c.depth)
			if err != nil {
				t.Fatalf("findSubdirsAtDepth: %v", err)
			}
			// Convert to relative for stable comparison.
			rel := make([]string, len(got))
			for i, g := range got {
				r, err := filepath.Rel(root, g)
				if err != nil {
					t.Fatal(err)
				}
				rel[i] = r
			}
			sort.Strings(rel)
			sort.Strings(c.want)
			if len(rel) != len(c.want) {
				t.Fatalf("got %v, want %v", rel, c.want)
			}
			for i := range rel {
				if rel[i] != c.want[i] {
					t.Errorf("got %v, want %v", rel, c.want)
					break
				}
			}
		})
	}
}

func TestExpandTargets(t *testing.T) {
	root := t.TempDir()
	buildTree(t, root, []string{
		"foo/",
		"bar/",
		"baz/",
	})

	in := []aggregator.Target{{Label: root, AbsPath: root}}
	out, err := expandTargets(in, 1)
	if err != nil {
		t.Fatalf("expandTargets: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("want 3 targets, got %d (%+v)", len(out), out)
	}

	labels := map[string]bool{}
	for _, tt := range out {
		labels[filepath.Base(tt.Label)] = true
		// Label should include parent prefix.
		if tt.Label == filepath.Base(tt.Label) {
			t.Errorf("label %q lost parent prefix", tt.Label)
		}
		// AbsPath should be absolute and exist.
		if !filepath.IsAbs(tt.AbsPath) {
			t.Errorf("AbsPath not absolute: %q", tt.AbsPath)
		}
		if _, err := os.Stat(tt.AbsPath); err != nil {
			t.Errorf("AbsPath not exist: %q (%v)", tt.AbsPath, err)
		}
	}
	for _, want := range []string{"foo", "bar", "baz"} {
		if !labels[want] {
			t.Errorf("missing expanded target for %q in %v", want, labels)
		}
	}
}
