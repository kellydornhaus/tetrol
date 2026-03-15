package layout

import "math"

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func clampSizeToConstraints(sz Size, cs Constraints) Size {
	return cs.clamp(sz)
}

func nearlyEqual(a, b float64) bool {
	const eps = 1e-6
	return math.Abs(a-b) < eps
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
