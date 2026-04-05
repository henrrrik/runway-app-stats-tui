package model

import (
	"errors"
	"runway-app-stats/internal/stats"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type stubFetcher struct {
	data stats.StatsData
	err  error
}

func (s stubFetcher) Fetch() (stats.StatsData, error) {
	return s.data, s.err
}

func sampleData() stats.StatsData {
	return stats.StatsData{
		TimeSeries: []stats.MetricPoint{
			{Timestamp: 1000, CPULoad: 0.01, RAMUsed: 1024000, DiskUsed: 0, NetRx: 100, NetTx: 50},
			{Timestamp: 1060, CPULoad: 0.02, RAMUsed: 2048000, DiskUsed: 512, NetRx: 200, NetTx: 100},
		},
		Latest: stats.MetricPoint{CPULoad: 0.02, RAMUsed: 2048000, DiskUsed: 512, NetRx: 200, NetTx: 100},
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

	if len(model.apps[0].stats.TimeSeries) != 2 {
		t.Errorf("expected 2 data points, got %d", len(model.apps[0].stats.TimeSeries))
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
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestUpdate_QuitOnCtrlC(t *testing.T) {
	m := newSingleApp("myapp", stubFetcher{}, 60*time.Second)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
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

	view := m.View()

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

	view := m.View()
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

	view := m.View()
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

	view := m.View()
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

	view := m.View()
	if !strings.Contains(view, "connection failed") {
		t.Error("view should show the error message")
	}
}
