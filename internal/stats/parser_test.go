package stats

import (
	"testing"
)

const sampleJSON = `{
  "all": [
    {
      "ts": 1775387643,
      "val": {
        "ram": { "used_bytes": 7446528 },
        "cpu": { "load": 0.0035187643796348225 },
        "hdd": { "used_bytes": 0 },
        "net": { "received_bytes": 1139.52, "transmitted_bytes": 610.53 }
      }
    },
    {
      "ts": 1775387703,
      "val": {
        "ram": { "used_bytes": 7450624 },
        "cpu": { "load": 0.00009422407727243648 },
        "hdd": { "used_bytes": 0 },
        "net": { "received_bytes": 339.47, "transmitted_bytes": 184.15 }
      }
    }
  ],
  "latest": {
    "ram": { "used_bytes": 8536064 },
    "cpu": { "load": 0.00006028016194331971 },
    "hdd": { "used_bytes": 0 },
    "net": { "received_bytes": 345.48, "transmitted_bytes": 183.92 }
  }
}`

func TestParse_ValidJSON(t *testing.T) {
	data, err := Parse([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data.TimeSeries) != 2 {
		t.Fatalf("expected 2 time series points, got %d", len(data.TimeSeries))
	}

	first := data.TimeSeries[0]
	if first.Timestamp != 1775387643 {
		t.Errorf("expected timestamp 1775387643, got %d", first.Timestamp)
	}
	if first.RAMUsed != 7446528 {
		t.Errorf("expected RAM 7446528, got %d", first.RAMUsed)
	}
	if first.CPULoad != 0.0035187643796348225 {
		t.Errorf("expected CPU load 0.00352, got %f", first.CPULoad)
	}
	if first.DiskUsed != 0 {
		t.Errorf("expected disk 0, got %d", first.DiskUsed)
	}
	if first.NetRx != 1139.52 {
		t.Errorf("expected NetRx 1139.52, got %f", first.NetRx)
	}
	if first.NetTx != 610.53 {
		t.Errorf("expected NetTx 610.53, got %f", first.NetTx)
	}
}

func TestParse_LatestField(t *testing.T) {
	data, err := Parse([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Latest.RAMUsed != 8536064 {
		t.Errorf("expected latest RAM 8536064, got %d", data.Latest.RAMUsed)
	}
	if data.Latest.CPULoad != 0.00006028016194331971 {
		t.Errorf("expected latest CPU load, got %f", data.Latest.CPULoad)
	}
}

func TestParse_EmptyAll(t *testing.T) {
	input := `{"all": [], "latest": {"ram": {"used_bytes": 0}, "cpu": {"load": 0}, "hdd": {"used_bytes": 0}, "net": {"received_bytes": 0, "transmitted_bytes": 0}}}`
	data, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data.TimeSeries) != 0 {
		t.Errorf("expected 0 time series, got %d", len(data.TimeSeries))
	}
}

func TestParse_MalformedJSON(t *testing.T) {
	_, err := Parse([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
