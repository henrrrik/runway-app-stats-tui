package stats

import "encoding/json"

type MetricPoint struct {
	Timestamp int64
	CPULoad   float64
	RAMUsed   int64
	DiskUsed  int64
	NetRx     float64
	NetTx     float64
}

type StatsData struct {
	TimeSeries []MetricPoint
	Latest     MetricPoint
}

type rawEntry struct {
	Ts  int64 `json:"ts"`
	Val rawVal `json:"val"`
}

type rawVal struct {
	RAM struct {
		UsedBytes int64 `json:"used_bytes"`
	} `json:"ram"`
	CPU struct {
		Load float64 `json:"load"`
	} `json:"cpu"`
	HDD struct {
		UsedBytes int64 `json:"used_bytes"`
	} `json:"hdd"`
	Net struct {
		ReceivedBytes    float64 `json:"received_bytes"`
		TransmittedBytes float64 `json:"transmitted_bytes"`
	} `json:"net"`
}

type rawResponse struct {
	All    []rawEntry `json:"all"`
	Latest rawVal     `json:"latest"`
}

func valToPoint(ts int64, v rawVal) MetricPoint {
	return MetricPoint{
		Timestamp: ts,
		CPULoad:   v.CPU.Load,
		RAMUsed:   v.RAM.UsedBytes,
		DiskUsed:  v.HDD.UsedBytes,
		NetRx:     v.Net.ReceivedBytes,
		NetTx:     v.Net.TransmittedBytes,
	}
}

func Parse(data []byte) (StatsData, error) {
	var raw rawResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return StatsData{}, err
	}

	points := make([]MetricPoint, len(raw.All))
	for i, entry := range raw.All {
		points[i] = valToPoint(entry.Ts, entry.Val)
	}

	return StatsData{
		TimeSeries: points,
		Latest:     valToPoint(0, raw.Latest),
	}, nil
}
