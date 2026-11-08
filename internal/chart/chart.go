// Package chart renders metric time series as braille line charts.
package chart

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"time"

	"charm.land/lipgloss/v2"
	tslc "github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"

	"github.com/henrrrik/runway-app-stats-tui/internal/stats"
)

// headroom keeps the line from touching the top of the chart.
const headroom = 1.2

var axisColor = lipgloss.Color("8")

// Series is one line on a chart.
type Series struct {
	Points []stats.Point
	Color  color.Color
}

// Render draws the series as a width×height chart with a value axis on the
// left and a time axis along the bottom. The value axis runs from zero to the
// plan limit, or is fitted to the data when the limit is unset (<= 0) or
// exceeded.
func Render(width, height int, limit float64, series ...Series) string {
	peak, first, last, ok := bounds(series)
	if !ok {
		return placeholder(width, height)
	}
	top := limit
	if limit <= 0 || peak > limit {
		top = niceCeil(peak * headroom)
	}
	if last == first {
		last = first + 1
	}

	axis := lipgloss.NewStyle().Foreground(axisColor)
	c := tslc.New(width, height,
		tslc.WithXYSteps(2, 2),
		tslc.WithAxesStyles(axis, axis),
		tslc.WithXLabelFormatter(func(_ int, v float64) string {
			return time.Unix(int64(v), 0).Format("15:04")
		}),
		tslc.WithYLabelFormatter(func(_ int, v float64) string { return formatValue(v) }),
		tslc.WithTimeRange(time.UnixMilli(first), time.UnixMilli(last)),
		tslc.WithYRange(0, top),
	)
	c.SetViewTimeAndYRange(time.UnixMilli(first), time.UnixMilli(last), 0, top)

	for i, s := range series {
		name := strconv.Itoa(i)
		c.SetDataSetStyle(name, lipgloss.NewStyle().Foreground(s.Color))
		for _, p := range s.Points {
			c.PushDataSet(name, tslc.TimePoint{Time: time.UnixMilli(p.Timestamp), Value: p.Value})
		}
	}
	c.DrawBrailleAll()
	return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(c.View())
}

// bounds returns the highest value and the time span across all series.
func bounds(series []Series) (peak float64, first, last int64, ok bool) {
	first, last = math.MaxInt64, math.MinInt64
	for _, s := range series {
		for _, p := range s.Points {
			ok = true
			peak = math.Max(peak, p.Value)
			first = min(first, p.Timestamp)
			last = max(last, p.Timestamp)
		}
	}
	return peak, first, last, ok
}

func placeholder(width, height int) string {
	return lipgloss.NewStyle().
		Width(width).Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(axisColor).
		Render("no data")
}

// formatValue keeps axis labels short: two significant figures for small
// values, whole numbers otherwise.
func formatValue(v float64) string {
	if v != 0 && math.Abs(v) < 10 {
		return strconv.FormatFloat(v, 'g', 2, 64)
	}
	return fmt.Sprintf("%.0f", v)
}

// niceCeil rounds v up to 1, 2, 2.5, 3, 4, 5, 6 or 8 times a power of ten, so the top of
// the value axis reads as a round number. Non-positive values become 1.
func niceCeil(v float64) float64 {
	if v <= 0 {
		return 1
	}
	exp := math.Pow(10, math.Floor(math.Log10(v)))
	for _, step := range []float64{1, 2, 2.5, 3, 4, 5, 6, 8, 10} {
		// small tolerance so 0.005 isn't pushed to 0.01 by float error
		if n := step * exp; n >= v*(1-1e-9) {
			return n
		}
	}
	return 10 * exp
}
