package model

import (
	"fmt"
	"github.com/henrrrik/runway-app-stats-tui/internal/chart"
	"github.com/henrrrik/runway-app-stats-tui/internal/stats"
	"image/color"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Colors are the terminal's numbered ANSI colors (or none), so the user's
// theme decides the actual shades.
var (
	// Lip Gloss can't make borders faint, so they use bright black.
	borderColor = lipgloss.Color("8")
	// Values use the theme's default foreground, readable on light and dark.
	valueColor    color.Color = lipgloss.NoColor{}
	warnColor                 = lipgloss.Color("3")
	criticalColor             = lipgloss.Color("1")
	cpuColor                  = lipgloss.Color("4")
	ramColor                  = lipgloss.Color("2")
	diskColor                 = lipgloss.Color("6")
	netInColor                = lipgloss.Color("4")
	netOutColor               = lipgloss.Color("5")

	appNameStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	footerStyle  = lipgloss.NewStyle().Faint(true)
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
	// fetching is set while a fetch is running, so a slow fetch is never
	// overlapped by the next tick's.
	fetching bool
}

type Model struct {
	apps     []appState
	width    int
	height   int
	interval time.Duration
	// viewport scrolls the dashboard when it is taller than the terminal.
	viewport viewport.Model
}

func New(configs []AppConfig, interval time.Duration) Model {
	apps := make([]appState, len(configs))
	for i, c := range configs {
		// Init starts a fetch for every app.
		apps[i] = appState{name: c.Name, fetcher: c.Fetcher, fetching: true}
	}
	return Model{
		apps:     apps,
		interval: interval,
		viewport: viewport.New(),
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
	m.viewport = m.syncedViewport()
	m, cmd := m.update(msg)
	m.viewport = m.syncedViewport()
	return m, cmd
}

// syncedViewport returns the viewport sized to the terminal (minus the
// footer) and holding the current dashboard, keeping the scroll position.
func (m Model) syncedViewport() viewport.Model {
	vp := m.viewport
	vp.SetWidth(m.width)
	vp.SetHeight(max(m.height-footerRows, 0))
	vp.SetContent(m.body())
	return vp
}

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg)
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
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

func (m Model) handleKeyMsg(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "home", "g":
		m.viewport.GotoTop()
		return m, nil
	case "end", "G":
		m.viewport.GotoBottom()
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleStatsMsg(msg statsMsg) (Model, tea.Cmd) {
	if a := m.findApp(msg.app); a != nil {
		a.stats = msg.data
		a.loaded = true
		a.err = nil
		a.fetching = false
	}
	return m, nil
}

func (m Model) handleErrMsg(msg errMsg) (Model, tea.Cmd) {
	if a := m.findApp(msg.app); a != nil {
		a.err = msg.err
		a.fetching = false
	}
	// The regular tick retries; scheduling one here would multiply timers.
	return m, nil
}

// handleTickMsg refreshes every app that isn't still fetching and schedules
// the next tick, so the dashboard keeps polling for as long as it runs.
func (m Model) handleTickMsg() (Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(m.apps)+1)
	for i := range m.apps {
		a := &m.apps[i]
		if a.fetching {
			continue
		}
		a.fetching = true
		cmds = append(cmds, fetchCmd(a.name, a.fetcher))
	}
	cmds = append(cmds, tickCmd(m.interval))
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

const footerRows = 1

func (m Model) View() tea.View {
	v := tea.NewView("Loading...")
	v.AltScreen = true
	if m.width == 0 {
		return v
	}
	vp := m.syncedViewport()
	scrollable := vp.TotalLineCount() > vp.Height()
	v.SetContent(lipgloss.JoinVertical(lipgloss.Left, vp.View(), footer(m.width, vp, scrollable)))
	if scrollable {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

func footer(width int, vp viewport.Model, scrollable bool) string {
	text := "Press q to quit"
	if scrollable {
		text += fmt.Sprintf(" · ↑/↓ to scroll (%.0f%%)", vp.ScrollPercent()*100)
	}
	return footerStyle.Render(ansi.Truncate(text, width, "…"))
}

// body renders every app: as chart panels when the terminal is tall enough
// for one app's charts (the rest scroll), otherwise as compact rows of values.
func (m Model) body() string {
	if m.width == 0 {
		return ""
	}
	if !chartsFit(m.width, m.height) {
		return m.renderCompact()
	}
	return m.renderCharts()
}

func (m Model) renderCharts() string {
	panelW := panelWidth(m.width)
	chartW := chartWidth(m.width)
	chartH := chartHeight(m.height, len(m.apps))
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(panelW).
		Padding(0, 1)
	valueStyle := lipgloss.NewStyle().Foreground(valueColor)

	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	var sections []string

	for i, a := range m.apps {
		if i > 0 {
			sections = append(sections, dividerStyle.Render(strings.Repeat("━", m.width)))
		}
		if a.err != nil && !a.loaded {
			sections = append(sections, appNameStyle.Render(a.name), errorLine(a.err))
			continue
		}
		sections = append(sections, appHeader(a),
			renderAppDashboard(a, panelStyle, titleStyle, valueStyle, chartW, chartH))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderCompact shows each app as its name and a line or two of values, for
// terminals too short for charts.
func (m Model) renderCompact() string {
	items := make([][]string, len(m.apps))
	for i, a := range m.apps {
		items[i] = compactItems(a.stats)
	}
	alignColumns(items)

	var lines []string
	for i, a := range m.apps {
		if a.err != nil && !a.loaded {
			lines = append(lines, appNameStyle.Render(a.name), errorLine(a.err))
			continue
		}
		lines = append(lines, appHeader(a), pack(items[i], m.width, "   "))
	}
	return strings.Join(lines, "\n")
}

func compactItems(s stats.StatsData) []string {
	item := func(label string, level stats.Level, value string) string {
		valueStyle := lipgloss.NewStyle().Foreground(levelColor(level, valueColor))
		return titleStyle.Render(label) + " " + valueStyle.Render(value)
	}
	return []string{
		item("CPU", metricLevel(s.CPU), formatCPU(s.CPU)),
		item("RAM", metricLevel(s.RAM), formatUsage(s.RAM)),
		item("Net", maxLevel(metricLevel(s.NetIn), metricLevel(s.NetOut)),
			fmt.Sprintf("in %s  out %s", formatRate(s.NetIn), formatRate(s.NetOut))),
		item("Disk", metricLevel(s.Disk), formatUsage(s.Disk)),
	}
}

// alignColumns pads each column to its widest cell, so rows line up and
// wrap at the same places.
func alignColumns(rows [][]string) {
	var widths []int
	for _, row := range rows {
		for c, cell := range row {
			if c == len(widths) {
				widths = append(widths, 0)
			}
			widths[c] = max(widths[c], lipgloss.Width(cell))
		}
	}
	for _, row := range rows {
		for c, cell := range row {
			row[c] = cell + strings.Repeat(" ", widths[c]-lipgloss.Width(cell))
		}
	}
}

// pack joins items with sep, starting a new line whenever the next item
// would make the line wider than width.
func pack(items []string, width int, sep string) string {
	var lines []string
	line := ""
	for _, it := range items {
		switch {
		case line == "":
			line = it
		case lipgloss.Width(line+sep+it) <= width:
			line += sep + it
		default:
			lines = append(lines, line)
			line = it
		}
	}
	lines = append(lines, line)
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n")
}

func errorLine(err error) string {
	return errStyle.Render(fmt.Sprintf("Error: %s", err.Error()))
}

// appHeader is the app name, followed by a warning when the latest refresh
// failed and the last good stats are still on screen.
func appHeader(a appState) string {
	header := appNameStyle.Render(a.name)
	if a.err != nil {
		header += "  " + errStyle.Render(fmt.Sprintf("Warning: %s", a.err.Error()))
	}
	return header
}

func renderAppDashboard(a appState, panelStyle, titleStyle, valueStyle lipgloss.Style, w, h int) string {
	s := a.stats
	netIn := lipgloss.NewStyle().Foreground(netInColor)
	netOut := lipgloss.NewStyle().Foreground(netOutColor)

	cpu := renderPanel(panelStyle, titleStyle, valueStyle, metricLevel(s.CPU),
		"CPU", chart.Render(w, h, s.CPU.Max, chart.Series{Points: s.CPU.Data, Color: cpuColor}),
		formatCPU(s.CPU))

	ram := renderPanel(panelStyle, titleStyle, valueStyle, metricLevel(s.RAM),
		"RAM", chart.Render(w, h, s.RAM.Max, chart.Series{Points: s.RAM.Data, Color: ramColor}),
		formatUsage(s.RAM))

	net := renderPanel(panelStyle, titleStyle, valueStyle, maxLevel(metricLevel(s.NetIn), metricLevel(s.NetOut)),
		"Network",
		chart.Render(w, h, max(s.NetIn.Max, s.NetOut.Max),
			chart.Series{Points: s.NetIn.Data, Color: netInColor},
			chart.Series{Points: s.NetOut.Data, Color: netOutColor}),
		fmt.Sprintf("%s %s  %s %s",
			netIn.Render("in"), formatRate(s.NetIn),
			netOut.Render("out"), formatRate(s.NetOut)))

	disk := renderPanel(panelStyle, titleStyle, valueStyle, metricLevel(s.Disk),
		"Disk", chart.Render(w, h, s.Disk.Max, chart.Series{Points: s.Disk.Data, Color: diskColor}),
		formatUsage(s.Disk))

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cpu, " ", ram)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, net, " ", disk)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

const (
	minChartHeight = 5
	// minChartWidth leaves room for the value axis plus a readable plot.
	minChartWidth  = 20
	maxChartHeight = 16
	// Rows per app besides its two chart rows: the app header, plus border,
	// title and value line around each of the two panel rows.
	appChromeRows = 1 + 2*4
)

// chartHeight splits the terminal height between the apps, leaving room for
// panel chrome, the divider between apps and the footer.
func chartHeight(termHeight, apps int) int {
	return min(max(fittingChartHeight(termHeight, apps), minChartHeight), maxChartHeight)
}

// chartsFit reports whether the terminal can show a whole app with charts of
// at least minChartWidth×minChartHeight. Further apps scroll; only terminals
// smaller than that fall back to compact rows.
func chartsFit(termWidth, termHeight int) bool {
	return chartWidth(termWidth) >= minChartWidth &&
		fittingChartHeight(termHeight, 1) >= minChartHeight
}

// panelWidth fits two panels and a 1-column gap side by side. In Lip Gloss v2
// Width() includes border and padding.
func panelWidth(termWidth int) int {
	return (termWidth - 1) / 2
}

// chartWidth is what's left inside a panel's border and padding.
func chartWidth(termWidth int) int {
	return panelWidth(termWidth) - 4
}

func fittingChartHeight(termHeight, apps int) int {
	if apps < 1 {
		apps = 1
	}
	perApp := termHeight/apps - 1 // divider or footer
	return (perApp - appChromeRows) / 2
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
