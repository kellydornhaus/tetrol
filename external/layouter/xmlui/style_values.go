package xmlui

import (
	"log"
	"strconv"
	"strings"

	"github.com/kellydornhaus/layouter/layout"
)

// ParseFloat converts a numeric CSS/XML attribute (optionally suffixed with "dp") to float64.
// When parsing fails or the input is blank, def is returned.
func ParseFloat(val string, def float64) float64 {
	if val == "" {
		return def
	}
	val = strings.TrimSpace(val)
	val = strings.TrimSuffix(val, "dp")
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return def
	}
	return f
}

// ParseBool converts a boolean attribute to bool, returning def when parsing fails.
func ParseBool(val string, def bool) bool {
	if val == "" {
		return def
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return def
	}
	return b
}

// ParseInsets interprets comma-separated padding values.
// Returns false when the input is empty or malformed.
func ParseInsets(val string) (layout.EdgeInsets, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return layout.EdgeInsets{}, false
	}
	parts := strings.Split(val, ",")
	nums := make([]float64, len(parts))
	for i := range parts {
		nums[i] = ParseFloat(strings.TrimSpace(parts[i]), 0)
	}
	switch len(nums) {
	case 0:
		return layout.EdgeInsets{}, false
	case 1:
		return layout.Insets(nums[0]), true
	case 2:
		return layout.EdgeInsets{Top: nums[0], Bottom: nums[0], Left: nums[1], Right: nums[1]}, true
	default:
		if len(nums) < 4 {
			return layout.EdgeInsets{}, false
		}
		return layout.EdgeInsets{Top: nums[0], Right: nums[1], Bottom: nums[2], Left: nums[3]}, true
	}
}

// ParseLength converts a CSS length string into a layout.Length.
// Supports dp, %, vw, vh, vmin, vmax, calc(), min(), max(), clamp(), and auto (empty/invalid returns false).
func ParseLength(val string) (layout.Length, bool) {
	return parseLengthString(val)
}

func parseLengthString(val string) (layout.Length, bool) {
	raw := strings.TrimSpace(val)
	if raw == "" {
		return layout.Length{}, false
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "calc(") || strings.HasPrefix(lower, "min(") || strings.HasPrefix(lower, "max(") || strings.HasPrefix(lower, "clamp(") {
		if computed, ok := parseCalcLength(raw); ok {
			return computed, true
		}
	}
	switch {
	case strings.HasSuffix(lower, "vw"):
		numeric := strings.TrimSpace(raw[:len(raw)-2])
		f, err := strconv.ParseFloat(numeric, 64)
		if err != nil {
			log.Printf("xmlui: invalid viewport width length %q: %v", val, err)
			return layout.Length{}, false
		}
		if f > 1 {
			f = f / 100
		}
		return layout.LengthVW(f), true
	case strings.HasSuffix(lower, "vh"):
		numeric := strings.TrimSpace(raw[:len(raw)-2])
		f, err := strconv.ParseFloat(numeric, 64)
		if err != nil {
			log.Printf("xmlui: invalid viewport height length %q: %v", val, err)
			return layout.Length{}, false
		}
		if f > 1 {
			f = f / 100
		}
		return layout.LengthVH(f), true
	case strings.HasSuffix(lower, "vmin"):
		numeric := strings.TrimSpace(raw[:len(raw)-4])
		f, err := strconv.ParseFloat(numeric, 64)
		if err != nil {
			log.Printf("xmlui: invalid viewport min length %q: %v", val, err)
			return layout.Length{}, false
		}
		if f > 1 {
			f = f / 100
		}
		return layout.LengthVMin(f), true
	case strings.HasSuffix(lower, "vmax"):
		numeric := strings.TrimSpace(raw[:len(raw)-4])
		f, err := strconv.ParseFloat(numeric, 64)
		if err != nil {
			log.Printf("xmlui: invalid viewport max length %q: %v", val, err)
			return layout.Length{}, false
		}
		if f > 1 {
			f = f / 100
		}
		return layout.LengthVMax(f), true
	case strings.HasSuffix(lower, "%"):
		numeric := strings.TrimSpace(raw[:len(raw)-1])
		f, err := strconv.ParseFloat(numeric, 64)
		if err != nil {
			log.Printf("xmlui: invalid percent length %q: %v", val, err)
			return layout.Length{}, false
		}
		return layout.LengthPercent(f / 100), true
	case strings.HasSuffix(lower, "dp"):
		numeric := strings.TrimSpace(raw[:len(raw)-2])
		f, err := strconv.ParseFloat(numeric, 64)
		if err != nil {
			log.Printf("xmlui: invalid dp length %q: %v", val, err)
			return layout.Length{}, false
		}
		return layout.LengthDP(f), true
	default:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			log.Printf("xmlui: invalid length %q: %v", val, err)
			return layout.Length{}, false
		}
		if f >= 0 && f <= 1 {
			return layout.LengthPercent(f), true
		}
		return layout.LengthDP(f), true
	}
}

// ParseColor interprets named, hex, rgb()/rgba() colors with optional alpha, returning def when empty or invalid.
func ParseColor(s string, def layout.Color) layout.Color {
	return parseColorInternal(s, def)
}
