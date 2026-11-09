package stats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIFetcher_CommandArgs_WithAppName(t *testing.T) {
	f := CLIFetcher{AppName: "myapp", Interval: "1h"}
	args := f.buildArgs()

	expected := []string{"app", "stats", "--app", "myapp", "--interval=1h", "-o", "json"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, want := range expected {
		if args[i] != want {
			t.Errorf("arg[%d]: expected %q, got %q", i, want, args[i])
		}
	}
}

func TestCLIFetcher_CommandArgs_WithoutAppName(t *testing.T) {
	f := CLIFetcher{Interval: "6h"}
	args := f.buildArgs()

	expected := []string{"app", "stats", "--interval=6h", "-o", "json"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, want := range expected {
		if args[i] != want {
			t.Errorf("arg[%d]: expected %q, got %q", i, want, args[i])
		}
	}
}

// fakeRunway puts a "runway" script with the given body first on PATH.
func fakeRunway(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "runway"), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestCLIFetcher_TimesOutWhenRunwayHangs(t *testing.T) {
	fakeRunway(t, "sleep 30")
	f := CLIFetcher{AppName: "my-app", Interval: "1h", Timeout: 100 * time.Millisecond}

	start := time.Now()
	_, err := f.Fetch()
	if err == nil {
		t.Fatal("expected an error when runway hangs")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error should say it timed out, got %q", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Fetch took %v; it should give up soon after the timeout", elapsed)
	}
}

func TestCLIFetcher_ReportsRunwayStderr(t *testing.T) {
	fakeRunway(t, "echo 'app not found' >&2; exit 1")
	_, err := CLIFetcher{AppName: "nope", Interval: "1h"}.Fetch()
	if err == nil || !strings.Contains(err.Error(), "app not found") {
		t.Errorf("expected runway's error message, got %v", err)
	}
}
