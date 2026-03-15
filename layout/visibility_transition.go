package layout

import (
	"strconv"
	"strings"
	"time"
)

// VisibilityTransitionMode describes the supported transition kinds.
type VisibilityTransitionMode int

const (
	// VisibilityTransitionNone disables transition effects.
	VisibilityTransitionNone VisibilityTransitionMode = iota
	// VisibilityTransitionSize scales components for both show and hide transitions.
	VisibilityTransitionSize
	// VisibilityTransitionSizeIn scales components when becoming visible.
	VisibilityTransitionSizeIn
	// VisibilityTransitionSizeOut scales components when hiding.
	VisibilityTransitionSizeOut
)

// VisibilityTransition configures animated visibility state changes.
type VisibilityTransition struct {
	Mode     VisibilityTransitionMode
	Duration time.Duration
	Scale    float64
}

// Enabled reports whether the transition should run.
func (t VisibilityTransition) Enabled() bool {
	return t.Mode != VisibilityTransitionNone && t.Duration > 0
}

// AnimateIn reports whether the transition should run when showing.
func (t VisibilityTransition) AnimateIn() bool {
	return t.Enabled() && (t.Mode == VisibilityTransitionSize || t.Mode == VisibilityTransitionSizeIn)
}

// AnimateOut reports whether the transition should run when hiding.
func (t VisibilityTransition) AnimateOut() bool {
	return t.Enabled() && (t.Mode == VisibilityTransitionSize || t.Mode == VisibilityTransitionSizeOut)
}

// ParseVisibilityTransition parses a transition string such as "size 1s 50%".
func ParseVisibilityTransition(raw string) (VisibilityTransition, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return VisibilityTransition{}, false
	}
	value = strings.ToLower(value)
	if value == "none" {
		return VisibilityTransition{Mode: VisibilityTransitionNone}, true
	}
	parts := strings.Fields(value)
	if len(parts) != 3 {
		return VisibilityTransition{}, false
	}
	var mode VisibilityTransitionMode
	switch parts[0] {
	case "size":
		mode = VisibilityTransitionSize
	case "size-in", "sizein", "size_in":
		mode = VisibilityTransitionSizeIn
	case "size-out", "sizeout", "size_out":
		mode = VisibilityTransitionSizeOut
	default:
		return VisibilityTransition{}, false
	}
	duration, err := time.ParseDuration(parts[1])
	if err != nil || duration <= 0 {
		return VisibilityTransition{}, false
	}
	scale, ok := parseTransitionPercent(parts[2])
	if !ok {
		return VisibilityTransition{}, false
	}
	scale = clampFraction(scale)
	return VisibilityTransition{
		Mode:     mode,
		Duration: duration,
		Scale:    scale,
	}, true
}

func parseTransitionPercent(token string) (float64, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, false
	}
	if strings.HasSuffix(token, "%") {
		raw := strings.TrimSpace(token[:len(token)-1])
		if raw == "" {
			return 0, false
		}
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, false
		}
		return val / 100.0, true
	}
	val, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return 0, false
	}
	// Treat whole numbers as percentages (e.g. "50" => 50%).
	if val > 1 {
		val = val / 100.0
	}
	return val, true
}

func clampFraction(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
