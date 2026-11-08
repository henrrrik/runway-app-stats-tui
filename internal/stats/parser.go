package stats

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Point is a single sample in a metric series. Timestamp is in milliseconds.
type Point struct {
	Timestamp int64   `json:"ts"`
	Value     float64 `json:"v"`
}

// Level classifies a value against a metric's warning thresholds.
type Level int

const (
	LevelOK Level = iota
	LevelWarn
	LevelCritical
)

// Metric is one series from `runway app stats -o json`, with its unit,
// plan limit and warning thresholds.
type Metric struct {
	Title         string  `json:"title"`
	Unit          string  `json:"unit"`
	Max           float64 `json:"max"`
	WarnAbove     float64 `json:"warnAbove"`
	CriticalAbove float64 `json:"criticalAbove"`
	Data          []Point `json:"data"`
}

// Latest returns the most recent value, or false if the series is empty.
func (m Metric) Latest() (float64, bool) {
	if len(m.Data) == 0 {
		return 0, false
	}
	return m.Data[len(m.Data)-1].Value, true
}

// Values returns the series values in order.
func (m Metric) Values() []float64 {
	values := make([]float64, len(m.Data))
	for i, p := range m.Data {
		values[i] = p.Value
	}
	return values
}

// Level reports whether v is above the metric's warning or critical threshold.
// A threshold of zero is treated as unset.
func (m Metric) Level(v float64) Level {
	switch {
	case m.CriticalAbove > 0 && v > m.CriticalAbove:
		return LevelCritical
	case m.WarnAbove > 0 && v > m.WarnAbove:
		return LevelWarn
	}
	return LevelOK
}

type StatsData struct {
	CPU    Metric
	RAM    Metric
	Disk   Metric
	NetIn  Metric
	NetOut Metric
}

const diskPrefix = "hdd_"

// Parse decodes the output of `runway app stats -o json` (runway CLI >= v1.49).
func Parse(data []byte) (StatsData, error) {
	var raw map[string]Metric
	if err := json.Unmarshal(data, &raw); err != nil {
		return StatsData{}, err
	}

	var out StatsData
	required := []struct {
		key string
		dst *Metric
	}{
		{"cpu", &out.CPU},
		{"ram", &out.RAM},
		{"net_in", &out.NetIn},
		{"net_out", &out.NetOut},
	}
	for _, r := range required {
		m, ok := raw[r.key]
		if !ok {
			return StatsData{}, missingMetric(r.key)
		}
		*r.dst = m
	}

	disk, ok := pickDisk(raw)
	if !ok {
		return StatsData{}, missingMetric(diskPrefix + "*")
	}
	out.Disk = disk
	return out, nil
}

func missingMetric(key string) error {
	return fmt.Errorf("unexpected runway stats format: missing %q metric (runway CLI v1.49+ required)", key)
}

// pickDisk prefers the app's ephemeral disk, falling back to the first
// other disk series (alphabetically) that has data, e.g. the worker disk
// reported for database apps.
func pickDisk(raw map[string]Metric) (Metric, bool) {
	const preferred = diskPrefix + "ephemeral_app"
	if m, ok := raw[preferred]; ok && len(m.Data) > 0 {
		return m, true
	}

	var keys []string
	for k := range raw {
		if strings.HasPrefix(k, diskPrefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		if len(raw[k].Data) > 0 {
			return raw[k], true
		}
	}

	m, ok := raw[preferred]
	if !ok && len(keys) > 0 {
		m, ok = raw[keys[0]], true
	}
	return m, ok
}
