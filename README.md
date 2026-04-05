# runway-stats

A terminal dashboard for monitoring [Runway](https://www.runway.horse) app resources. Displays CPU, RAM, Network I/O, and Disk usage with sparkline charts.

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

## Prerequisites

- Go 1.21+
- [Runway CLI](https://www.runway.horse/docs/cli/install/) installed and logged in

## Install

```
go build -o runway-stats .
```

## Usage

```
runway-stats [flags] <app-name> [app-name...]
```

### Examples

```bash
# Single app
runway-stats my-app

# Multiple apps side by side
runway-stats my-app my-other-app

# Custom time range and refresh interval
runway-stats --interval=6h --refresh=30s my-app
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--interval` | `1h` | Stats time range (e.g. `1h`, `6h`, `24h`) |
| `--refresh` | `60s` | How often to re-fetch data |

Press `q` or `Ctrl+C` to quit.
