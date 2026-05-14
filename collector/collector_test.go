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

package collector

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestCollect_FindsSelf is a smoke test: the running test binary itself has
// a readable cwd, so the live collector must return at least one sample
// matching our PID with a non-empty Cwd.
func TestCollect_FindsSelf(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live collector test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	samples, err := Collect(ctx, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(samples) == 0 {
		t.Fatal("expected at least one sample, got 0")
	}

	myPID := os.Getpid()
	for _, s := range samples {
		if int(s.PID) == myPID {
			if s.Cwd == "" {
				t.Fatal("self sample has empty Cwd")
			}
			return
		}
	}
	t.Fatalf("self PID %d not found in samples", myPID)
}
