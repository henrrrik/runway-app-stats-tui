package stats

import (
	"testing"
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
