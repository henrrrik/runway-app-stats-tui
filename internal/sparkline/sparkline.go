package sparkline

var blocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Render returns a sparkline string for the given values.
// Values are auto-scaled to the range [min, max].
// If there are more values than width, only the last `width` values are used.
func Render(values []float64, width int) string {
	if len(values) == 0 {
		return ""
	}

	// Truncate to width (keep latest values)
	if len(values) > width {
		values = values[len(values)-width:]
	}

	// Find min and max for auto-scaling
	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	runes := make([]rune, len(values))
	for i, v := range values {
		var idx int
		if max == min {
			idx = 0
		} else {
			normalized := (v - min) / (max - min)
			idx = int(normalized * float64(len(blocks)-1))
			if idx >= len(blocks) {
				idx = len(blocks) - 1
			}
		}
		runes[i] = blocks[idx]
	}

	return string(runes)
}
