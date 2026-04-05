package model

import (
	"fmt"
	"runway-app-stats/internal/sparkline"
	"runway-app-stats/internal/stats"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type statsMsg struct {
	app  string
	data stats.StatsData
}

type errMsg struct {
	app string
	err error
}

type tickMsg struct{}

type AppConfig struct {
	Name    string
	Fetcher stats.Fetcher
}

type appState struct {
	name    string
	fetcher stats.Fetcher
	stats   stats.StatsData
	err     error
}

type Model struct {
	apps     []appState
	width    int
	height   int
	interval time.Duration
}

func New(configs []AppConfig, interval time.Duration) Model {
	apps := make([]appState, len(configs))
	for i, c := range configs {
		apps[i] = appState{name: c.Name, fetcher: c.Fetcher}
	}
	return Model{
		apps:     apps,
		interval: interval,
	}
}

func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.apps)+1)
	for _, a := range m.apps {
		cmds = append(cmds, fetchCmd(a.name, a.fetcher))
	}
	cmds = append(cmds, tickCmd(m.interval))
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case statsMsg:
		for i := range m.apps {
			if m.apps[i].name == msg.app {
				m.apps[i].stats = msg.data
				m.apps[i].err = nil
				break
			}
		}
	case errMsg:
		for i := range m.apps {
			if m.apps[i].name == msg.app {
				m.apps[i].err = msg.err
				break
			}
		}
		return m, tickCmd(m.interval)
	case tickMsg:
		cmds := make([]tea.Cmd, len(m.apps))
		for i, a := range m.apps {
			cmds[i] = fetchCmd(a.name, a.fetcher)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	appNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	panelWidth := m.width/2 - 2
	sparkWidth := panelWidth - 4
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(panelWidth).
		Padding(0, 1)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	var sections []string

	for i, a := range m.apps {
		if i > 0 {
			sections = append(sections, dividerStyle.Render(strings.Repeat("─", m.width)))
		}
		header := appNameStyle.Render(a.name)

		if a.err != nil && len(a.stats.TimeSeries) == 0 {
			sections = append(sections, lipgloss.JoinVertical(lipgloss.Left,
				header,
				errStyle.Render(fmt.Sprintf("Error: %s", a.err.Error())),
			))
			continue
		}

		dashboard := renderAppDashboard(a, panelStyle, titleStyle, valueStyle, panelWidth, sparkWidth)

		var status string
		if a.err != nil {
			status = errStyle.Render(fmt.Sprintf("Warning: %s", a.err.Error()))
		}

		parts := []string{header, dashboard}
		if status != "" {
			parts = append(parts, status)
		}
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, parts...))
	}

	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Press q to quit")
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func renderAppDashboard(a appState, panelStyle, titleStyle, valueStyle lipgloss.Style, panelWidth, sparkWidth int) string {
	cpuColor := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	ramColor := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	netColor := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	diskColor := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	cpuValues := extractValues(a.stats.TimeSeries, func(p stats.MetricPoint) float64 { return p.CPULoad })
	ramValues := extractValues(a.stats.TimeSeries, func(p stats.MetricPoint) float64 { return float64(p.RAMUsed) })
	rxValues := extractValues(a.stats.TimeSeries, func(p stats.MetricPoint) float64 { return p.NetRx })
	txValues := extractValues(a.stats.TimeSeries, func(p stats.MetricPoint) float64 { return p.NetTx })
	diskValues := extractValues(a.stats.TimeSeries, func(p stats.MetricPoint) float64 { return float64(p.DiskUsed) })

	cpu := renderPanel(panelStyle, titleStyle, valueStyle,
		"CPU", cpuColor.Render(sparkline.Render(cpuValues, sparkWidth)),
		fmt.Sprintf("%.4f%%", a.stats.Latest.CPULoad*100))

	ram := renderPanel(panelStyle, titleStyle, valueStyle,
		"RAM", ramColor.Render(sparkline.Render(ramValues, sparkWidth)),
		humanize.IBytes(uint64(a.stats.Latest.RAMUsed)))

	net := renderPanel(panelStyle, titleStyle, valueStyle,
		"Network",
		fmt.Sprintf("rx %s\ntx %s",
			netColor.Render(sparkline.Render(rxValues, sparkWidth-3)),
			netColor.Render(sparkline.Render(txValues, sparkWidth-3))),
		fmt.Sprintf("rx %s/s  tx %s/s",
			humanize.IBytes(uint64(a.stats.Latest.NetRx)),
			humanize.IBytes(uint64(a.stats.Latest.NetTx))))

	disk := renderPanel(panelStyle, titleStyle, valueStyle,
		"Disk", "\n"+diskColor.Render(sparkline.Render(diskValues, sparkWidth)),
		humanize.IBytes(uint64(a.stats.Latest.DiskUsed)))

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cpu, " ", ram)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, net, " ", disk)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

func renderPanel(panelStyle, titleStyle, valueStyle lipgloss.Style, title, chart, value string) string {
	return panelStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(title),
			chart,
			valueStyle.Render(value),
		),
	)
}

func extractValues(points []stats.MetricPoint, fn func(stats.MetricPoint) float64) []float64 {
	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = fn(p)
	}
	return values
}

func fetchCmd(app string, fetcher stats.Fetcher) tea.Cmd {
	return func() tea.Msg {
		data, err := fetcher.Fetch()
		if err != nil {
			return errMsg{app: app, err: err}
		}
		return statsMsg{app: app, data: data}
	}
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
