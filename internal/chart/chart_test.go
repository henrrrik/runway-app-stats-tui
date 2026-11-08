package chart

import (
	"math"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"runway-app-stats/internal/stats"
)

func points(values ...float64) []stats.Point {
	ps := make([]stats.Point, len(values))
	for i, v := range values {
		ps[i] = stats.Point{Timestamp: 1_700_000_000_000 + int64(i)*60_000, Value: v}
	}
	return ps
}

func hasBraille(s string) bool {
	for _, r := range s {
		if r > 0x2800 && r <= 0x28FF {
			return true
		}
	}
	return false
}

func checkSize(t *testing.T, out string, width, height int) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != height {
		t.Errorf("expected %d lines, got %d:\n%s", height, len(lines), out)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != width {
			t.Errorf("line %d is %d columns wide, want %d", i, w, width)
		}
	}
}

func TestRender_DrawsLineWithinSize(t *testing.T) {
	out := Render(30, 6, Series{Points: points(1, 3, 2, 5, 4)})
	checkSize(t, out, 30, 6)
	if !hasBraille(out) {
		t.Errorf("expected braille line in chart:\n%s", out)
	}
}

func TestRender_LabelsTimeAxis(t *testing.T) {
	out := Render(30, 6, Series{Points: points(1, 3, 2, 5, 4)})
	if !strings.Contains(ansi.Strip(out), ":") {
		t.Errorf("expected HH:MM time labels:\n%s", out)
	}
}

func TestRender_LabelsValueAxisAbovePeak(t *testing.T) {
	out := Render(30, 6, Series{Points: points(10, 30, 20)})
	first := ansi.Strip(strings.Split(out, "\n")[0])
	if !strings.Contains(first, "40") {
		t.Errorf("expected top label rounded up above the peak (40):\n%s", out)
	}
}

func TestRender_MultipleSeries(t *testing.T) {
	out := Render(30, 6,
		Series{Points: points(1, 2, 3)},
		Series{Points: points(3, 2, 1)},
	)
	checkSize(t, out, 30, 6)
	if !hasBraille(out) {
		t.Errorf("expected braille lines:\n%s", out)
	}
}

func TestRender_NoDataKeepsSize(t *testing.T) {
	out := Render(30, 6, Series{})
	checkSize(t, out, 30, 6)
	if !strings.Contains(ansi.Strip(out), "no data") {
		t.Errorf("expected 'no data' placeholder:\n%s", out)
	}
}

func TestRender_FlatZeroSeries(t *testing.T) {
	out := Render(30, 6, Series{Points: points(0, 0, 0)})
	checkSize(t, out, 30, 6)
}

func TestFormatValue(t *testing.T) {
	cases := map[float64]string{
		0:      "0",
		0.0045: "0.0045",
		0.5:    "0.5",
		7.5:    "7.5",
		36:     "36",
		2048:   "2048",
	}
	for v, want := range cases {
		if got := formatValue(v); got != want {
			t.Errorf("formatValue(%g) = %q, want %q", v, got, want)
		}
	}
}

func TestNiceCeil(t *testing.T) {
	cases := map[float64]float64{
		0.0048: 0.005,
		0.024:  0.025,
		0.6:    0.6,
		0.7:    0.8,
		16:     20,
		23:     25,
		36:     40,
		100:    100,
		1300:   2000,
	}
	for v, want := range cases {
		if got := niceCeil(v); math.Abs(got-want) > 1e-9 {
			t.Errorf("niceCeil(%g) = %g, want %g", v, got, want)
		}
	}
}
