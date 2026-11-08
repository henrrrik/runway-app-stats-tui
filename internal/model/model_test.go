package model

import (
	"errors"
	"fmt"
	"runway-app-stats/internal/stats"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type stubFetcher struct {
	data stats.StatsData
	err  error
}

func (s stubFetcher) Fetch() (stats.StatsData, error) {
	return s.data, s.err
}

func metric(unit string, max, warn, crit float64, values ...float64) stats.Metric {
	m := stats.Metric{Unit: unit, Max: max, WarnAbove: warn, CriticalAbove: crit}
	for i, v := range values {
		m.Data = append(m.Data, stats.Point{Timestamp: int64(i) * 60000, Value: v})
	}
	return m
}

func sampleData() stats.StatsData {
	return stats.StatsData{
		CPU:    metric("cores", 0.5, 0.325, 0.4, 0.01, 0.02),
		RAM:    metric("MiB", 128, 83.2, 102.4, 12.5, 13.5),
		Disk:   metric("MiB", 2048, 1331.2, 1638.4, 19.1, 19.2),
		NetIn:  metric("MBit/s", 5, 3.25, 4, 0.0045),
		NetOut: metric("MBit/s", 5, 3.25, 4, 0.0024),
	}
}

func newSingleApp(name string, f stats.Fetcher, interval time.Duration) Model {
	return New([]AppConfig{{Name: name, Fetcher: f}}, interval)
}

func newTwoApps(f1, f2 stats.Fetcher, interval time.Duration) Model {
	return New([]AppConfig{
		{Name: "app1", Fetcher: f1},
		{Name: "app2", Fetcher: f2},
	}, interval)
}

func TestUpdate_StatsMsg(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	data := sampleData()
	updated, _ := m.Update(statsMsg{app: "myapp", data: data})
	model := updated.(Model)

	if len(model.apps[0].stats.RAM.Data) != 2 {
		t.Errorf("expected 2 RAM data points, got %d", len(model.apps[0].stats.RAM.Data))
	}
	if !model.apps[0].loaded {
		t.Error("expected app to be marked as loaded")
	}
	if model.apps[0].err != nil {
		t.Errorf("expected no error, got %v", model.apps[0].err)
	}
}

func TestUpdate_ErrMsg(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	updated, _ := m.Update(errMsg{app: "myapp", err: errors.New("fetch failed")})
	model := updated.(Model)

	if model.apps[0].err == nil {
		t.Fatal("expected error to be stored")
	}
	if model.apps[0].err.Error() != "fetch failed" {
		t.Errorf("expected 'fetch failed', got %q", model.apps[0].err.Error())
	}
}

func TestUpdate_QuitOnQ(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestUpdate_QuitOnCtrlC(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestUpdate_WindowSize(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(Model)

	if model.width != 120 || model.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", model.width, model.height)
	}
}

func TestUpdate_ErrMsg_SchedulesRetry(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	_, cmd := m.Update(errMsg{app: "myapp", err: errors.New("fetch failed")})
	if cmd == nil {
		t.Fatal("expected retry tick command after error")
	}
}

func TestView_ContainsMetricLabels(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	m.apps[0].stats = sampleData()
	m.width = 80
	m.height = 24

	view := m.View().Content

	for _, label := range []string{"CPU", "RAM", "Network", "Disk"} {
		if !strings.Contains(view, label) {
			t.Errorf("view should contain %q", label)
		}
	}
}

func TestView_ContainsAppName(t *testing.T) {
	m := newSingleApp("my-cool-app", stubFetcher{}, 60*time.Second)
	m.apps[0].stats = sampleData()
	m.width = 80
	m.height = 24

	view := m.View().Content
	if !strings.Contains(view, "my-cool-app") {
		t.Error("view should contain the app name")
	}
}

func TestView_MultipleApps(t *testing.T) {
	m := newTwoApps(stubFetcher{}, stubFetcher{}, 60*time.Second)
	m.apps[0].stats = sampleData()
	m.apps[1].stats = sampleData()
	m.width = 80
	m.height = 48

	view := m.View().Content
	if !strings.Contains(view, "app1") {
		t.Error("view should contain 'app1'")
	}
	if !strings.Contains(view, "app2") {
		t.Error("view should contain 'app2'")
	}
}

func TestView_ContainsDividerBetweenApps(t *testing.T) {
	m := newTwoApps(stubFetcher{}, stubFetcher{}, 60*time.Second)
	m.apps[0].stats = sampleData()
	m.apps[1].stats = sampleData()
	m.width = 80
	m.height = 48

	view := m.View().Content
	// Divider should be a run of at least 20 ─ characters (wider than any border segment)
	if !strings.Contains(view, strings.Repeat("─", 20)) {
		t.Error("view should contain a divider between apps")
	}
}

func TestView_ShowsError(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	m.apps[0].err = errors.New("connection failed")
	m.width = 80
	m.height = 24

	view := m.View().Content
	if !strings.Contains(view, "connection failed") {
		t.Error("view should show the error message")
	}
}

func TestView_ShowsValuesWithUnits(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	m.apps[0].stats = sampleData()
	m.apps[0].loaded = true
	m.width = 120
	m.height = 24

	view := m.View().Content
	for _, want := range []string{"4.0% of 0.5 cores", "13.5 / 128 MiB", "19.2 / 2048 MiB", "0.0045 MBit/s", "0.0024 MBit/s"} {
		if !strings.Contains(view, want) {
			t.Errorf("view should contain %q\n%s", want, view)
		}
	}
}

func TestView_KeepsDashboardOnErrorAfterLoad(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	m.apps[0].stats = sampleData()
	m.apps[0].loaded = true
	m.apps[0].err = errors.New("timeout")
	m.width = 120
	m.height = 24

	view := m.View().Content
	if !strings.Contains(view, "Warning: timeout") {
		t.Error("view should show the error as a warning")
	}
	if !strings.Contains(view, "13.5 / 128 MiB") {
		t.Error("view should keep showing the last stats")
	}
}

func TestFormat_NoData(t *testing.T) {
	empty := stats.Metric{Unit: "MiB", Max: 128}
	if got := formatUsage(empty); got != "no data" {
		t.Errorf("formatUsage on empty metric = %q, want %q", got, "no data")
	}
	if got := formatCPU(empty); got != "no data" {
		t.Errorf("formatCPU on empty metric = %q, want %q", got, "no data")
	}
	if got := formatRate(empty); got != "no data" {
		t.Errorf("formatRate on empty metric = %q, want %q", got, "no data")
	}
}

func TestMetricLevel(t *testing.T) {
	cases := []struct {
		name string
		m    stats.Metric
		want stats.Level
	}{
		{"ok", metric("MiB", 128, 83.2, 102.4, 90, 50), stats.LevelOK},
		{"warn", metric("MiB", 128, 83.2, 102.4, 50, 90), stats.LevelWarn},
		{"critical", metric("MiB", 128, 83.2, 102.4, 50, 110), stats.LevelCritical},
		{"empty", stats.Metric{WarnAbove: 1}, stats.LevelOK},
	}
	for _, c := range cases {
		if got := metricLevel(c.m); got != c.want {
			t.Errorf("%s: metricLevel = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestMaxLevel(t *testing.T) {
	if got := maxLevel(stats.LevelOK, stats.LevelCritical, stats.LevelWarn); got != stats.LevelCritical {
		t.Errorf("maxLevel = %v, want critical", got)
	}
	if got := maxLevel(); got != stats.LevelOK {
		t.Errorf("maxLevel() = %v, want ok", got)
	}
}

func TestLevelColor(t *testing.T) {
	normal := lipgloss.Color("15")
	if got := levelColor(stats.LevelOK, normal); got != normal {
		t.Errorf("ok: got %v, want %v", got, normal)
	}
	if got := levelColor(stats.LevelWarn, normal); got != warnColor {
		t.Errorf("warn: got %v, want %v", got, warnColor)
	}
	if got := levelColor(stats.LevelCritical, normal); got != criticalColor {
		t.Errorf("critical: got %v, want %v", got, criticalColor)
	}
}

func TestView_FitsTerminalWidth(t *testing.T) {
	for _, width := range []int{80, 81, 120, 121} {
		m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
		m.apps[0].stats = sampleData()
		m.apps[0].loaded = true
		m.width = width
		m.height = 30

		for i, line := range strings.Split(m.View().Content, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line %d is %d columns wide", width, i, w)
			}
		}
	}
}

func TestFormat_SmallValuesKeepPrecision(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"usage below 1", formatUsage(metric("MiB", 2048, 0, 0, 0.0234375)), "0.0234 / 2048 MiB"},
		{"usage above 1", formatUsage(metric("MiB", 128, 0, 0, 13.5)), "13.5 / 128 MiB"},
		{"usage large", formatUsage(metric("MiB", 5120, 0, 0, 1500.75)), "1500.8 / 5120 MiB"},
		{"usage zero", formatUsage(metric("MiB", 128, 0, 0, 0)), "0.0 / 128 MiB"},
		{"cpu small", formatCPU(metric("cores", 0.5, 0, 0, 0.000386)), "0.0772% of 0.5 cores"},
		{"cpu normal", formatCPU(metric("cores", 0.5, 0, 0, 0.0172)), "3.4% of 0.5 cores"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, c.got, c.want)
		}
	}
}

func TestView_UsesAltScreen(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	if !m.View().AltScreen {
		t.Error("loading view should use the alt screen")
	}
	m.width = 80
	m.height = 24
	if !m.View().AltScreen {
		t.Error("dashboard view should use the alt screen")
	}
}

func TestView_FitsTerminalHeight(t *testing.T) {
	for _, tc := range []struct{ apps, height int }{{1, 24}, {1, 40}, {2, 40}, {2, 60}} {
		configs := make([]AppConfig, tc.apps)
		for i := range configs {
			configs[i] = AppConfig{Name: fmt.Sprintf("app%d", i), Fetcher: stubFetcher{}}
		}
		m := New(configs, time.Minute)
		for i := range m.apps {
			m.apps[i].stats = sampleData()
			m.apps[i].loaded = true
		}
		m.width = 100
		m.height = tc.height

		if got := len(strings.Split(m.View().Content, "\n")); got > tc.height {
			t.Errorf("%d apps at height %d: view is %d lines", tc.apps, tc.height, got)
		}
	}
}

func TestChartHeight_GrowsWithTerminal(t *testing.T) {
	small := chartHeight(24, 1)
	large := chartHeight(50, 1)
	if large <= small {
		t.Errorf("chartHeight(50) = %d should exceed chartHeight(24) = %d", large, small)
	}
	if got := chartHeight(10, 3); got < minChartHeight {
		t.Errorf("chartHeight should not drop below %d, got %d", minChartHeight, got)
	}
	if got := chartHeight(500, 1); got > maxChartHeight {
		t.Errorf("chartHeight should not exceed %d, got %d", maxChartHeight, got)
	}
}

func TestView_DrawsLineCharts(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, time.Minute)
	m.apps[0].stats = sampleData()
	m.apps[0].loaded = true
	m.width = 100
	m.height = 30

	if !strings.ContainsAny(m.View().Content, "⠁⠂⠄⡀⠈⠐⠠⢀⣀⠉⠒⠤") {
		t.Error("view should contain braille line charts")
	}
}
