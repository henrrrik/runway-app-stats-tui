package sparkline

import "testing"

func TestRender_Empty(t *testing.T) {
	result := Render(nil, 10)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestRender_SingleValue(t *testing.T) {
	result := Render([]float64{0.5}, 10)
	if len([]rune(result)) != 1 {
		t.Errorf("expected 1 character, got %d: %q", len([]rune(result)), result)
	}
}

func TestRender_AllZero(t *testing.T) {
	result := Render([]float64{0, 0, 0}, 10)
	expected := "▁▁▁"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRender_AllSame(t *testing.T) {
	// When all values are the same, there's no range — all render as lowest block
	result := Render([]float64{1.0, 1.0, 1.0}, 10)
	expected := "▁▁▁"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRender_WidthTruncation(t *testing.T) {
	values := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	result := Render(values, 3)
	runes := []rune(result)
	if len(runes) != 3 {
		t.Errorf("expected 3 characters, got %d: %q", len(runes), result)
	}
}

func TestRender_Ascending(t *testing.T) {
	values := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
	result := Render(values, 10)
	runes := []rune(result)
	for i := 1; i < len(runes); i++ {
		if runes[i] < runes[i-1] {
			t.Errorf("expected ascending blocks, got %q", result)
			break
		}
	}
}

func TestRender_AutoScales(t *testing.T) {
	// When max value is 50, a value of 50 should render as full block
	values := []float64{0, 25, 50}
	result := Render(values, 10)
	runes := []rune(result)
	if runes[2] != '█' {
		t.Errorf("expected max value to render as full block, got %q", result)
	}
	if runes[0] != '▁' {
		t.Errorf("expected min value to render as lowest block, got %q", result)
	}
}
