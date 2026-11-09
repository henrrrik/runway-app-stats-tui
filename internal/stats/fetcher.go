package stats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Fetcher interface {
	Fetch() (StatsData, error)
}

// DefaultTimeout is how long a fetch may take when CLIFetcher.Timeout is zero.
const DefaultTimeout = 30 * time.Second

// waitDelay bounds how long Fetch waits for output after killing a timed-out
// runway, in case a child process still holds its stdout open.
const waitDelay = time.Second

type CLIFetcher struct {
	AppName  string
	Interval string
	// Timeout kills a runway call that takes longer; zero means DefaultTimeout.
	Timeout time.Duration
}

func (f CLIFetcher) buildArgs() []string {
	args := []string{"app", "stats"}
	if f.AppName != "" {
		args = append(args, "--app", f.AppName)
	}
	args = append(args, fmt.Sprintf("--interval=%s", f.Interval), "-o", "json")
	return args
}

func (f CLIFetcher) Fetch() (StatsData, error) {
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "runway", f.buildArgs()...)
	cmd.WaitDelay = waitDelay
	out, err := cmd.Output()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return StatsData{}, fmt.Errorf("runway timed out after %s", timeout)
	}
	if msg := cliError(out); msg != "" {
		return StatsData{}, fmt.Errorf("runway: %s", msg)
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return StatsData{}, fmt.Errorf("runway: %s", stderr)
			}
		}
		return StatsData{}, fmt.Errorf("runway command failed: %w", err)
	}
	return Parse(out)
}

// cliError returns the message from runway's JSON error output, such as
// {"status": "error", "error": "Unauthorized: Please login first"}, which it
// prints on stdout rather than stderr. It returns "" for any other output.
func cliError(out []byte) string {
	var resp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if json.Unmarshal(out, &resp) != nil || resp.Status != "error" {
		return ""
	}
	return strings.TrimSpace(resp.Error)
}
