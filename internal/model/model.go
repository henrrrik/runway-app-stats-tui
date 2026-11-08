package model

import (
	"fmt"
	"image/color"
	"math"
	"runway-app-stats/internal/chart"
	"runway-app-stats/internal/stats"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	borderColor   = lipgloss.Color("8")
	valueColor    = lipgloss.Color("15")
	warnColor     = lipgloss.Color("3")
	criticalColor = lipgloss.Color("1")
	cpuColor      = lipgloss.Color("4")
	ramColor      = lipgloss.Color("2")
	diskColor     = lipgloss.Color("6")
	netInColor    = lipgloss.Color("3")
	netOutColor   = lipgloss.Color("5")
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
	loaded  bool
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
	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case statsMsg:
		return m.handleStatsMsg(msg)
	case errMsg:
		return m.handleErrMsg(msg)
	case tickMsg:
		return m.handleTickMsg()
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleStatsMsg(msg statsMsg) (tea.Model, tea.Cmd) {
	if a := m.findApp(msg.app); a != nil {
		a.stats = msg.data
		a.loaded = true
		a.err = nil
	}
	return m, nil
}

func (m Model) handleErrMsg(msg errMsg) (tea.Model, tea.Cmd) {
	if a := m.findApp(msg.app); a != nil {
		a.err = msg.err
	}
	return m, tickCmd(m.interval)
}

func (m Model) handleTickMsg() (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.apps))
	for i, a := range m.apps {
		cmds[i] = fetchCmd(a.name, a.fetcher)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) findApp(name string) *appState {
	for i := range m.apps {
		if m.apps[i].name == name {
			return &m.apps[i]
		}
	}
	return nil
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	if m.width == 0 {
		return "Loading..."
	}

	appNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	// In Lip Gloss v2 Width() includes border and padding; two panels plus a
	// 1-column gap must fit. The chart gets what's left inside both.
	panelWidth := (m.width - 1) / 2
	chartWidth := panelWidth - 4
	chartH := chartHeight(m.height, len(m.apps))
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(panelWidth).
		Padding(0, 1)
	valueStyle := lipgloss.NewStyle().Foreground(valueColor)
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	var sections []string

	for i, a := range m.apps {
		if i > 0 {
			sections = append(sections, dividerStyle.Render(strings.Repeat("━", m.width)))
		}
		header := appNameStyle.Render(a.name)

		if a.err != nil && !a.loaded {
			sections = append(sections, lipgloss.JoinVertical(lipgloss.Left,
				header,
				errStyle.Render(fmt.Sprintf("Error: %s", a.err.Error())),
			))
			continue
		}

		dashboard := renderAppDashboard(a, panelStyle, titleStyle, valueStyle, chartWidth, chartH)

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

func renderAppDashboard(a appState, panelStyle, titleStyle, valueStyle lipgloss.Style, w, h int) string {
	s := a.stats
	netIn := lipgloss.NewStyle().Foreground(netInColor)
	netOut := lipgloss.NewStyle().Foreground(netOutColor)

	cpu := renderPanel(panelStyle, titleStyle, valueStyle, metricLevel(s.CPU),
		"CPU", chart.Render(w, h, chart.Series{Points: s.CPU.Data, Color: cpuColor}),
		formatCPU(s.CPU))

	ram := renderPanel(panelStyle, titleStyle, valueStyle, metricLevel(s.RAM),
		"RAM", chart.Render(w, h, chart.Series{Points: s.RAM.Data, Color: ramColor}),
		formatUsage(s.RAM))

	net := renderPanel(panelStyle, titleStyle, valueStyle, maxLevel(metricLevel(s.NetIn), metricLevel(s.NetOut)),
		"Network",
		chart.Render(w, h,
			chart.Series{Points: s.NetIn.Data, Color: netInColor},
			chart.Series{Points: s.NetOut.Data, Color: netOutColor}),
		fmt.Sprintf("%s %s  %s %s",
			netIn.Render("in"), formatRate(s.NetIn),
			netOut.Render("out"), formatRate(s.NetOut)))

	disk := renderPanel(panelStyle, titleStyle, valueStyle, metricLevel(s.Disk),
		"Disk", chart.Render(w, h, chart.Series{Points: s.Disk.Data, Color: diskColor}),
		formatUsage(s.Disk))

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cpu, " ", ram)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, net, " ", disk)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

const (
	minChartHeight = 5
	maxChartHeight = 16
	// Rows per app besides its two chart rows: the app header, plus border,
	// title and value line around each of the two panel rows.
	appChromeRows = 1 + 2*4
)

// chartHeight splits the terminal height between the apps, leaving room for
// panel chrome, the divider between apps and the footer.
func chartHeight(termHeight, apps int) int {
	if apps < 1 {
		apps = 1
	}
	perApp := termHeight/apps - 1 // divider or footer
	return min(max((perApp-appChromeRows)/2, minChartHeight), maxChartHeight)
}

// renderPanel draws a metric panel. Warning and critical levels recolour the
// border and the value text.
func renderPanel(panelStyle, titleStyle, valueStyle lipgloss.Style, level stats.Level, title, chart, value string) string {
	panelStyle = panelStyle.BorderForeground(levelColor(level, borderColor))
	valueStyle = valueStyle.Foreground(levelColor(level, valueColor))
	return panelStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(title),
			chart,
			valueStyle.Render(value),
		),
	)
}

const noData = "no data"

// formatCPU shows CPU usage as a share of the plan's core limit.
func formatCPU(m stats.Metric) string {
	v, ok := m.Latest()
	if !ok {
		return noData
	}
	if m.Max <= 0 {
		return fmt.Sprintf("%s %s", formatNumber(v), m.Unit)
	}
	return fmt.Sprintf("%s%% of %g %s", formatNumber(v/m.Max*100), m.Max, m.Unit)
}

// formatUsage shows the latest value against the plan limit, e.g. "13.5 / 128 MiB".
func formatUsage(m stats.Metric) string {
	v, ok := m.Latest()
	if !ok {
		return noData
	}
	return fmt.Sprintf("%s / %g %s", formatNumber(v), m.Max, m.Unit)
}

// formatNumber uses one decimal place, but keeps three significant figures
// for values below 1 so small non-zero values don't round to 0.0.
func formatNumber(v float64) string {
	if v != 0 && math.Abs(v) < 1 {
		return fmt.Sprintf("%.3g", v)
	}
	return fmt.Sprintf("%.1f", v)
}

// formatRate shows the latest value with its unit, e.g. "0.0045 MBit/s".
func formatRate(m stats.Metric) string {
	v, ok := m.Latest()
	if !ok {
		return noData
	}
	return fmt.Sprintf("%.3g %s", v, m.Unit)
}

func metricLevel(m stats.Metric) stats.Level {
	v, ok := m.Latest()
	if !ok {
		return stats.LevelOK
	}
	return m.Level(v)
}

func maxLevel(levels ...stats.Level) stats.Level {
	worst := stats.LevelOK
	for _, l := range levels {
		if l > worst {
			worst = l
		}
	}
	return worst
}

func levelColor(l stats.Level, normal color.Color) color.Color {
	switch l {
	case stats.LevelCritical:
		return criticalColor
	case stats.LevelWarn:
		return warnColor
	}
	return normal
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
