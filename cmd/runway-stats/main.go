package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/henrrrik/runway-app-stats-tui/internal/model"
	"github.com/henrrrik/runway-app-stats-tui/internal/stats"
)

func main() {
	interval := flag.String("interval", "1h", "Stats time interval (e.g. 1h, 6h, 24h)")
	refresh := flag.Duration("refresh", 60*time.Second, "Auto-refresh interval")
	flag.Parse()

	appNames := flag.Args()
	if len(appNames) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: runway-stats [flags] <app-name> [app-name...]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	configs := make([]model.AppConfig, len(appNames))
	for i, name := range appNames {
		configs[i] = model.AppConfig{
			Name:    name,
			Fetcher: stats.CLIFetcher{AppName: name, Interval: *interval},
		}
	}

	m := model.New(configs, *refresh)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
