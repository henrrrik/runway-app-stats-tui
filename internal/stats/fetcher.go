package stats

import (
	"fmt"
	"os/exec"
	"strings"
)

type Fetcher interface {
	Fetch() (StatsData, error)
}

type CLIFetcher struct {
	AppName  string
	Interval string
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
	args := f.buildArgs()
	cmd := exec.Command("runway", args...)
	out, err := cmd.Output()
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
