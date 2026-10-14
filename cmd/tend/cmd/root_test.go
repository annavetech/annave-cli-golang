// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/annavetech/tend/internal/log/port"
)

// result is the captured outcome of one CLI run.
type result struct {
	stdout, stderr string
	code           int
}

// runCLI runs the CLI in process with stdin as input and captures both streams.
func runCLI(t *testing.T, ctx context.Context, stdin string, args ...string) result {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "stdin", stdin)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	in, err := root.Open("stdin")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := errors.Join(in.Close(), root.Close()); err != nil {
			t.Error(err)
		}
	})
	finish := captureStreams(t, in)
	code := execute(ctx, args)
	stdout, stderr := finish()
	return result{stdout: stdout, stderr: stderr, code: code}
}

// captureStreams points os.Stdin at in and os.Stdout and os.Stderr at temp files.
// The returned func restores the streams, resets flags and returns the captured output.
func captureStreams(t *testing.T, in *os.File) func() (stdout, stderr string) {
	t.Helper()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	outF, err := root.Create("stdout")
	if err != nil {
		t.Fatal(err)
	}
	errF, err := root.Create("stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldIn, oldOut, oldErr := os.Stdin, os.Stdout, os.Stderr
	os.Stdin, os.Stdout, os.Stderr = in, outF, errF
	return func() (string, string) {
		os.Stdin, os.Stdout, os.Stderr = oldIn, oldOut, oldErr
		for _, f := range []*os.File{outF, errF} {
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}
		resetFlags(t, rootCmd)
		return readFile(t, root, "stdout"), readFile(t, root, "stderr")
	}
}

// resetFlags restores every flag to its default so runs do not leak state.
func resetFlags(t *testing.T, c *cobra.Command) {
	t.Helper()
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			if err := sv.Replace(nil); err != nil {
				t.Fatal(err)
			}
		} else if err := f.Value.Set(f.DefValue); err != nil {
			t.Fatal(err)
		}
		f.Changed = false
	})
	for _, child := range c.Commands() {
		resetFlags(t, child)
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, root *os.Root, name string) string {
	t.Helper()
	b, err := root.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func lines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

const sampleLog = `2026-05-01T10:00:00Z INFO service started
2026-05-01T11:00:00Z WARN cache is slow
2026-05-01T12:00:00Z ERROR request failed
`

func TestRuntimeErrorPrintsOneLineWithoutUsage(t *testing.T) {
	file := writeFile(t, t.TempDir(), "app.log", sampleLog)
	r := runCLI(t, t.Context(), "", "log", "analyze", file, "--format", "xml")
	if r.code != 1 {
		t.Fatalf("exit code = %d, want 1", r.code)
	}
	if got := lines(r.stderr); len(got) != 1 || !strings.HasPrefix(got[0], "error: unknown format") {
		t.Fatalf("stderr = %q, want one error line", r.stderr)
	}
	if r.stdout != "" {
		t.Fatalf("stdout = %q, want empty", r.stdout)
	}
}

func TestFlagErrorPointsAtHelp(t *testing.T) {
	r := runCLI(t, t.Context(), "", "log", "analyze", "--bogus")
	want := []string{"error: unknown flag: --bogus", "Run 'tend log analyze --help' for usage."}
	if got := lines(r.stderr); r.code != 1 || len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("code %d, stderr %q, want 1 and %q", r.code, r.stderr, want)
	}
}

func TestArgErrorPointsAtHelp(t *testing.T) {
	r := runCLI(t, t.Context(), "", "doc", "search")
	if got := lines(r.stderr); r.code != 1 || len(got) != 2 || got[1] != "Run 'tend doc search --help' for usage." {
		t.Fatalf("code %d, stderr %q, want 1 and a --help line", r.code, r.stderr)
	}
}

func TestUnknownCommandPointsAtHelp(t *testing.T) {
	r := runCLI(t, t.Context(), "", "bogus")
	if got := lines(r.stderr); r.code != 1 || len(got) != 2 || got[1] != "Run 'tend --help' for usage." {
		t.Fatalf("code %d, stderr %q, want 1 and a --help line", r.code, r.stderr)
	}
}

// listen returns a TCP target that accepts connections until the test ends.
func listen(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	})
	return "tcp://" + l.Addr().String()
}

// closedTarget returns a TCP target with nothing listening.
func closedTarget(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return "tcp://" + addr
}

func TestHealthCheckExitCodes(t *testing.T) {
	up := listen(t)
	if r := runCLI(t, t.Context(), "", "health", "check", up, "--timeout", "2s"); r.code != 0 {
		t.Fatalf("all up: exit code = %d, want 0 (stderr %q)", r.code, r.stderr)
	}
	r := runCLI(t, t.Context(), "", "health", "check", up, closedTarget(t), "--timeout", "2s")
	if r.code != 2 || r.stderr != "" {
		t.Fatalf("one down: exit code = %d, stderr %q, want 2 and no error", r.code, r.stderr)
	}
}

func TestHealthCheckStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if r := runCLI(t, ctx, "", "health", "check", listen(t), "--timeout", "2s"); r.code != 2 {
		t.Fatalf("exit code = %d, want 2 for a check run under a cancelled context", r.code)
	}
}

func TestDocSearchNoResultsJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "readme.md", "# Hello\n\nnothing here\n")
	r := runCLI(t, t.Context(), "", "doc", "search", "zzzzz", "--path", dir, "--format", "json")
	if r.code != 0 || strings.TrimSpace(r.stdout) != "[]" {
		t.Fatalf("code %d, stdout %q, want 0 and []", r.code, r.stdout)
	}
}

func TestCostScanNotImplementedJSONError(t *testing.T) {
	for _, provider := range []string{"gcp", "azure"} {
		r := runCLI(t, t.Context(), "", "cost", "scan", "--provider", provider, "--format", "json")
		var got struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(r.stdout), &got); err != nil {
			t.Fatalf("%s: stdout %q is not JSON: %v", provider, r.stdout, err)
		}
		if r.code != 1 || !strings.Contains(got.Error, "not implemented yet. Use --provider aws.") || r.stderr != "" {
			t.Fatalf("%s: code %d, error %q, stderr %q, want 1, a not-implemented error, no stderr",
				provider, r.code, got.Error, r.stderr)
		}
	}
}

func TestLogLevelWarningFiltersAsWarn(t *testing.T) {
	file := writeFile(t, t.TempDir(), "app.log", sampleLog)
	all := runCLI(t, t.Context(), "", "log", "analyze", file, "--format", "json")
	warn := runCLI(t, t.Context(), "", "log", "analyze", file, "--format", "json", "--level", "warn")
	warning := runCLI(t, t.Context(), "", "log", "analyze", file, "--format", "json", "--level", "warning")
	if warning.stdout != warn.stdout || warn.stdout == all.stdout {
		t.Fatalf("--level warning must filter like warn:\nwarn    %s\nwarning %s", warn.stdout, warning.stdout)
	}
}

func TestLogLevelUnknownErrors(t *testing.T) {
	file := writeFile(t, t.TempDir(), "app.log", sampleLog)
	r := runCLI(t, t.Context(), "", "log", "analyze", file, "--level", "bogus")
	if r.code != 1 || !strings.Contains(r.stderr, `unknown level "bogus"`) {
		t.Fatalf("code %d, stderr %q, want 1 and an unknown level error", r.code, r.stderr)
	}
}

func TestLogAnalyzeNeedsInput(t *testing.T) {
	r := runCLI(t, t.Context(), sampleLog, "log", "analyze")
	if r.code != 1 || !strings.Contains(r.stderr, "no input") || !strings.Contains(r.stderr, "--help") {
		t.Fatalf("code %d, stderr %q, want 1 and a usage error", r.code, r.stderr)
	}
	for _, args := range [][]string{{"-"}, {"--stdin"}} {
		r := runCLI(t, t.Context(), sampleLog, append([]string{"log", "analyze", "--format", "json"}, args...)...)
		if r.code == 1 || !strings.Contains(r.stdout, `stdin`) {
			t.Fatalf("%v: code %d, stdout %q, want stdin to be read", args, r.code, r.stdout)
		}
	}
}

func TestLogAnalyzeFindingsExit2(t *testing.T) {
	log := strings.Repeat("2026-05-01T12:00:00Z ERROR request failed\n", 5)
	file := writeFile(t, t.TempDir(), "app.log", log)
	if r := runCLI(t, t.Context(), "", "log", "analyze", file); r.code != 2 {
		t.Fatalf("exit code = %d, want 2 when findings exist (stderr %q)", r.code, r.stderr)
	}
}

func TestLogAnalyzeCancelledExits1(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	file := writeFile(t, t.TempDir(), "app.log", sampleLog)
	if r := runCLI(t, ctx, "", "log", "analyze", file); r.code != 1 || !strings.Contains(r.stderr, "context canceled") {
		t.Fatalf("code %d, stderr %q, want 1 and a cancel error", r.code, r.stderr)
	}
}

func TestLogAnalyzeCancelStopsBlockedRead(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pr.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	finish := captureStreams(t, pr)
	done := make(chan int, 1)
	go func() { done <- execute(ctx, []string{"log", "analyze", "--stdin"}) }()
	var code int
	select {
	case code = <-done:
	case <-time.After(5 * time.Second):
		t.Error("log analyze kept reading after the context ended")
	}
	// Closing the writer ends a read that ignored the context.
	if err := pw.Close(); err != nil {
		t.Fatal(err)
	}
	if t.Failed() {
		code = <-done
	}
	_, stderr := finish()
	if code != 1 || !strings.Contains(stderr, "context deadline exceeded") {
		t.Fatalf("code %d, stderr %q, want 1 and the context error", code, stderr)
	}
}

// blockingReader blocks every read until unblock is closed, like an idle terminal.
type blockingReader struct{ unblock chan struct{} }

func (r blockingReader) Read([]byte) (int, error) {
	<-r.unblock
	return 0, io.EOF
}

func TestAnalyzeInputCancelStopsReadWithoutDeadline(t *testing.T) {
	r := blockingReader{unblock: make(chan struct{})}
	defer close(r.unblock)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := analyzeInput(ctx, r, "<stdin>", port.AnalyzeOptions{})
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want the context error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("analysis kept reading after the context ended")
	}
}
