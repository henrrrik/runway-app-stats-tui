package stats

import (
	"strings"
	"testing"
)

// Shape of `runway app stats -o json` as of runway CLI v1.49.0.
const sampleJSON = `{
  "cpu": {
    "title": "CPU", "unit": "cores", "max": 0.5, "warnAbove": 0.325, "criticalAbove": 0.4,
    "interval": {"tickMilliseconds": 60000, "startTimestamp": 1794092105678, "endTimestamp": 1794095705678},
    "data": [{"ts": 1794092225678, "v": 0}, {"ts": 1794092285678, "v": 0.00038}]
  },
  "hdd_ephemeral_app": {
    "title": "HDD (ephemeral, app)", "unit": "MiB", "max": 2048, "warnAbove": 1331.2, "criticalAbove": 1638.4,
    "interval": {"tickMilliseconds": 60000, "startTimestamp": 1794092105678, "endTimestamp": 1794095705678},
    "data": [{"ts": 1794092105678, "v": 19.14}, {"ts": 1794092165678, "v": 19.15}]
  },
  "net_in": {
    "title": "Network (in)", "unit": "MBit/s", "max": 5, "warnAbove": 3.25, "criticalAbove": 4,
    "interval": {"tickMilliseconds": 60000, "startTimestamp": 1794092105678, "endTimestamp": 1794095705678},
    "data": [{"ts": 1794092225678, "v": 0.0045}]
  },
  "net_out": {
    "title": "Network (out)", "unit": "MBit/s", "max": 5, "warnAbove": 3.25, "criticalAbove": 4,
    "interval": {"tickMilliseconds": 60000, "startTimestamp": 1794092105678, "endTimestamp": 1794095705678},
    "data": [{"ts": 1794092225678, "v": 0.0024}]
  },
  "ram": {
    "title": "Memory", "unit": "MiB", "max": 128, "warnAbove": 83.2, "criticalAbove": 102.4,
    "interval": {"tickMilliseconds": 60000, "startTimestamp": 1794092105678, "endTimestamp": 1794095705678},
    "data": [{"ts": 1794092105678, "v": 12.87}, {"ts": 1794092165678, "v": 12.82}, {"ts": 1794092225678, "v": 13.5}]
  }
}`

// Database apps report disk usage on the worker; the app disk series is empty.
const dbSampleJSON = `{
  "cpu": {"unit": "cores", "max": 0.5, "warnAbove": 0.325, "criticalAbove": 0.4, "data": [{"ts": 1, "v": 0.01}]},
  "hdd_ephemeral_app": {"unit": "MiB", "max": 5120, "warnAbove": 3328, "criticalAbove": 4096, "data": []},
  "hdd_ephemeral_worker": {"unit": "MiB", "max": 5120, "warnAbove": 3328, "criticalAbove": 4096, "data": [{"ts": 1, "v": 300}]},
  "net_in": {"unit": "MBit/s", "max": 10, "warnAbove": 6.5, "criticalAbove": 8, "data": []},
  "net_out": {"unit": "MBit/s", "max": 10, "warnAbove": 6.5, "criticalAbove": 8, "data": []},
  "ram": {"unit": "MiB", "max": 256, "warnAbove": 166.4, "criticalAbove": 204.8, "data": [{"ts": 1, "v": 100}]}
}`

func mustParse(t *testing.T, input string) StatsData {
	t.Helper()
	data, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return data
}

func TestParse_RAM(t *testing.T) {
	ram := mustParse(t, sampleJSON).RAM
	if got := len(ram.Data); got != 3 {
		t.Fatalf("expected 3 RAM points, got %d", got)
	}
	if ram.Data[0] != (Point{Timestamp: 1794092105678, Value: 12.87}) {
		t.Errorf("unexpected first RAM point: %+v", ram.Data[0])
	}
	if ram.Unit != "MiB" || ram.Max != 128 || ram.WarnAbove != 83.2 || ram.CriticalAbove != 102.4 {
		t.Errorf("unexpected RAM metadata: %+v", ram)
	}
}

func TestParse_CPU(t *testing.T) {
	cpu := mustParse(t, sampleJSON).CPU
	if cpu.Unit != "cores" || cpu.Max != 0.5 || len(cpu.Data) != 2 {
		t.Errorf("unexpected CPU metric: %+v", cpu)
	}
}

func TestParse_Network(t *testing.T) {
	data := mustParse(t, sampleJSON)
	if len(data.NetIn.Data) != 1 || data.NetIn.Data[0].Value != 0.0045 {
		t.Errorf("unexpected net_in metric: %+v", data.NetIn)
	}
	if len(data.NetOut.Data) != 1 || data.NetOut.Data[0].Value != 0.0024 {
		t.Errorf("unexpected net_out metric: %+v", data.NetOut)
	}
}

func TestParse_Disk(t *testing.T) {
	disk := mustParse(t, sampleJSON).Disk
	if disk.Title != "HDD (ephemeral, app)" || len(disk.Data) != 2 {
		t.Errorf("unexpected disk metric: %+v", disk)
	}
}

func TestParse_DiskFallsBackToSeriesWithData(t *testing.T) {
	data := mustParse(t, dbSampleJSON)
	if len(data.Disk.Data) != 1 || data.Disk.Data[0].Value != 300 {
		t.Errorf("expected worker disk series, got %+v", data.Disk)
	}
}

func TestParse_MissingMetricIsError(t *testing.T) {
	_, err := Parse([]byte(`{"cpu": {"data": []}}`))
	if err == nil {
		t.Fatal("expected error for missing metrics")
	}
	if !strings.Contains(err.Error(), "ram") {
		t.Errorf("error should name the missing metric, got %q", err.Error())
	}
}

func TestParse_LegacyFormatIsError(t *testing.T) {
	legacy := `{"all": [], "latest": {"ram": {"used_bytes": 0}}}`
	if _, err := Parse([]byte(legacy)); err == nil {
		t.Fatal("expected error for pre-v1.49 stats format")
	}
}

func TestParse_MalformedJSON(t *testing.T) {
	_, err := Parse([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestMetric_Latest(t *testing.T) {
	m := Metric{Data: []Point{{1, 2}, {2, 5}}}
	v, ok := m.Latest()
	if !ok || v != 5 {
		t.Errorf("expected (5, true), got (%v, %v)", v, ok)
	}
	if _, ok := (Metric{}).Latest(); ok {
		t.Error("expected ok=false for empty metric")
	}
}

func TestMetric_Values(t *testing.T) {
	m := Metric{Data: []Point{{1, 2}, {2, 5}}}
	got := m.Values()
	if len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Errorf("unexpected values: %v", got)
	}
}

func TestMetric_Level(t *testing.T) {
	m := Metric{WarnAbove: 10, CriticalAbove: 20}
	cases := []struct {
		v    float64
		want Level
	}{
		{5, LevelOK},
		{10, LevelOK},
		{10.1, LevelWarn},
		{20, LevelWarn},
		{20.1, LevelCritical},
	}
	for _, c := range cases {
		if got := m.Level(c.v); got != c.want {
			t.Errorf("Level(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestMetric_LevelWithoutThresholds(t *testing.T) {
	if got := (Metric{}).Level(100); got != LevelOK {
		t.Errorf("expected LevelOK when no thresholds set, got %v", got)
	}
}
