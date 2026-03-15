package layout

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"
)

// DebugOptions controls optional debug logging for layout and CSS resolution.
type DebugOptions struct {
	// LogLayoutDecisions logs measurement/layout results.
	LogLayoutDecisions bool
	// LogCSSQueries logs CSS attribute maps resolved for components.
	LogCSSQueries bool
	// LogSurfaceAllocations logs cache surface creation/resizing/release.
	LogSurfaceAllocations bool
	// LogSurfaceUsage logs whether cached surfaces were reused or redrawn.
	LogSurfaceUsage bool
	// Logger overrides the logging sink (defaults to log.Printf).
	Logger func(format string, args ...interface{})
}

// DebugLayoutEnabled reports whether layout logging is active.
func (c *Context) DebugLayoutEnabled() bool {
	return c != nil && c.Debug.LogLayoutDecisions
}

// DebugCSSEnabled reports whether CSS logging is active.
func (c *Context) DebugCSSEnabled() bool {
	return c != nil && c.Debug.LogCSSQueries
}

// DebugSurfaceAllocEnabled reports whether surface allocation logging is active.
func (c *Context) DebugSurfaceAllocEnabled() bool {
	return c != nil && c.Debug.LogSurfaceAllocations
}

// DebugSurfaceUsageEnabled reports whether surface usage logging is active.
func (c *Context) DebugSurfaceUsageEnabled() bool {
	return c != nil && c.Debug.LogSurfaceUsage
}

func (c *Context) DebugSurfacesEnabled() bool {
	return c != nil && (c.Debug.LogSurfaceAllocations || c.Debug.LogSurfaceUsage)
}

// DebugLayoutf logs a formatted layout message when enabled.
func (c *Context) DebugLayoutf(format string, args ...interface{}) {
	if c == nil || !c.DebugLayoutEnabled() {
		return
	}
	c.debugLogf("layout", format, args...)
}

// DebugCSSf logs a formatted CSS message when enabled.
func (c *Context) DebugCSSf(format string, args ...interface{}) {
	if c == nil || !c.DebugCSSEnabled() {
		return
	}
	c.debugLogf("css", format, args...)
}

func (c *Context) debugLogf(kind string, format string, args ...interface{}) {
	line := "[" + kind + "] " + fmt.Sprintf(format, args...)
	if c != nil && c.frameLog.active {
		c.frameLog.append(line)
		return
	}
	logger := c.logSink()
	logger("%s", line)
}

func (c *Context) logSink() func(format string, args ...interface{}) {
	if c != nil && c.Debug.Logger != nil {
		return c.Debug.Logger
	}
	return log.Printf
}

type surfaceLogEntry struct {
	label       string
	lastSize    PxSize
	allocations int
	renders     int
	cacheHits   int
	sizes       map[PxSize]int
}

type surfaceLogState struct {
	entries        map[*Base]*surfaceLogEntry
	lastTotalBytes int64
	hasLastTotal   bool
}

func (s *surfaceLogState) entry(base *Base, owner Component) *surfaceLogEntry {
	if base == nil {
		return nil
	}
	if s.entries == nil {
		s.entries = make(map[*Base]*surfaceLogEntry)
	}
	entry := s.entries[base]
	label := surfaceLabel(owner, base)
	if entry == nil {
		entry = &surfaceLogEntry{
			label: label,
			sizes: make(map[PxSize]int),
		}
		s.entries[base] = entry
		return entry
	}
	if owner != nil {
		if entry.label == "" || strings.HasPrefix(entry.label, "surface@") {
			entry.label = label
		}
	}
	return entry
}

func surfaceLabel(owner Component, base *Base) string {
	if owner != nil {
		return componentLabel(owner)
	}
	if base != nil && base.parent != nil {
		return componentLabel(base.parent)
	}
	return fmt.Sprintf("surface@%p", base)
}

func (s *surfaceLogState) memoryUsage() (totalBytes int64, surfaces int) {
	if s == nil || len(s.entries) == 0 {
		return 0, 0
	}
	for base := range s.entries {
		if base == nil || base.cache == nil {
			continue
		}
		w, h := base.cache.SizePx()
		if w <= 0 || h <= 0 {
			continue
		}
		surfaces++
		totalBytes += int64(w) * int64(h) * 4
	}
	return totalBytes, surfaces
}

func formatSizeCounts(counts map[PxSize]int) string {
	if len(counts) == 0 {
		return ""
	}
	sizes := make([]PxSize, 0, len(counts))
	for sz := range counts {
		sizes = append(sizes, sz)
	}
	sort.Slice(sizes, func(i, j int) bool {
		if sizes[i].W == sizes[j].W {
			return sizes[i].H < sizes[j].H
		}
		return sizes[i].W < sizes[j].W
	})
	parts := make([]string, 0, len(sizes))
	for _, sz := range sizes {
		parts = append(parts, fmt.Sprintf("%dx%d(%d)", sz.W, sz.H, counts[sz]))
	}
	return strings.Join(parts, " ")
}

func (c *Context) logSurfaceAllocation(base *Base, owner Component, size PxSize, prev PxSize) {
	if c == nil || !c.DebugSurfacesEnabled() {
		return
	}
	entry := c.surfaceLog.entry(base, owner)
	if entry == nil {
		return
	}
	entry.allocations++
	entry.sizes[size]++
	entry.lastSize = size
	if !c.DebugSurfaceAllocEnabled() {
		return
	}
	var notes []string
	if prev.W > 0 || prev.H > 0 {
		notes = append(notes, fmt.Sprintf("prev=%dx%d", prev.W, prev.H))
	}
	if caller := firstNonLayoutCaller(); caller != "" {
		notes = append(notes, "caller="+caller)
	}
	suffix := ""
	if len(notes) > 0 {
		suffix = " (" + strings.Join(notes, "; ") + ")"
	}
	c.debugLogf("surface", "alloc %s size=%dx%d scale=%.2f allocs=%d sizes=[%s]%s",
		entry.label, size.W, size.H, c.Scale, entry.allocations, formatSizeCounts(entry.sizes), suffix)
}

func (c *Context) logSurfaceRelease(base *Base, owner Component, reason string) {
	if c == nil || !c.DebugSurfaceAllocEnabled() {
		return
	}
	entry := c.surfaceLog.entry(base, owner)
	if entry == nil {
		return
	}
	c.debugLogf("surface", "release %s reason=%s last-size=%dx%d allocs=%d renders=%d cache-hits=%d",
		entry.label, reason, entry.lastSize.W, entry.lastSize.H, entry.allocations, entry.renders, entry.cacheHits)
}

func (c *Context) logSurfaceDraw(base *Base, owner Component, size PxSize, rendered bool, note string) {
	if c == nil || !c.DebugSurfaceUsageEnabled() {
		return
	}
	entry := c.surfaceLog.entry(base, owner)
	if entry == nil {
		return
	}
	entry.lastSize = size
	if rendered {
		entry.renders++
	} else {
		entry.cacheHits++
	}
	kind := "cache-hit"
	if rendered {
		kind = "render"
	}
	draws := entry.renders + entry.cacheHits
	if note != "" {
		note = " " + note
	}
	c.debugLogf("surface", "%s %s size=%dx%d draws=%d renders=%d cache-hits=%d allocs=%d%s",
		kind, entry.label, size.W, size.H, draws, entry.renders, entry.cacheHits, entry.allocations, note)
}

func (c *Context) logSurfaceMemoryIfChanged() {
	if c == nil || !c.DebugSurfaceAllocEnabled() {
		return
	}
	totalBytes, surfaces := c.surfaceLog.memoryUsage()
	if c.surfaceLog.hasLastTotal && totalBytes == c.surfaceLog.lastTotalBytes {
		return
	}
	c.surfaceLog.lastTotalBytes = totalBytes
	c.surfaceLog.hasLastTotal = true
	c.debugLogf("surface", "memory total=%s (%d bytes across %d surfaces)", formatBytes(totalBytes), totalBytes, surfaces)
}

type frameLogState struct {
	active bool
	frame  int64
	start  time.Time
	lines  []string
	prev   []string
}

func (s *frameLogState) begin(frame int64) {
	s.active = true
	s.frame = frame
	s.start = time.Now()
	s.lines = s.lines[:0]
}

func (s *frameLogState) append(line string) {
	if !s.active {
		return
	}
	s.lines = append(s.lines, line)
}

func (s *frameLogState) end(ctx *Context) {
	if ctx == nil || !s.active {
		s.active = false
		return
	}
	s.active = false
	if equalStringSlices(s.lines, s.prev) {
		return
	}
	snapshot := append([]string(nil), s.lines...)
	header := fmt.Sprintf("frame %d @ %s (elapsed %s, %d lines)",
		s.frame, s.start.Format("15:04:05.000"), time.Since(s.start), len(snapshot))
	if len(snapshot) == 0 {
		ctx.logSink()("%s", header)
	} else {
		ctx.logSink()("%s\n%s", header, strings.Join(snapshot, "\n"))
	}
	s.prev = snapshot
	s.lines = s.lines[:0]
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func componentLabel(comp Component) string {
	if comp == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s@%p", reflect.TypeOf(comp), comp)
}

func formatBytes(b int64) string {
	const unit = int64(1024)
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div := unit
	exp := 0
	for n := b / unit; n >= unit && exp < len("KMGTPE")-1; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func measureComponent(ctx *Context, owner Component, target Component, cs Constraints) Size {
	if target == nil {
		return Size{}
	}
	size := target.Measure(ctx, cs)
	if ctx != nil && ctx.DebugLayoutEnabled() {
		if owner != nil {
			ctx.DebugLayoutf("measure %s via %s constraints=%+v -> size=%+v", componentLabel(target), componentLabel(owner), cs, size)
		} else {
			ctx.DebugLayoutf("measure %s constraints=%+v -> size=%+v", componentLabel(target), cs, size)
		}
	}
	return size
}

func logSelfMeasure(ctx *Context, comp Component, cs Constraints, size Size) {
	if ctx == nil || !ctx.DebugLayoutEnabled() {
		return
	}
	ctx.DebugLayoutf("measure %s constraints=%+v -> size=%+v", componentLabel(comp), cs, size)
}

func logLayoutBounds(ctx *Context, comp Component, parent Component, bounds Rect) {
	if ctx == nil || !ctx.DebugLayoutEnabled() {
		return
	}
	ctx.DebugLayoutf("layout %s parent=%s bounds=%+v", componentLabel(comp), componentLabel(parent), bounds)
}

func logChildLayout(ctx *Context, parent Component, child Component, bounds Rect, note string) {
	if ctx == nil || !ctx.DebugLayoutEnabled() {
		return
	}
	if note != "" {
		ctx.DebugLayoutf("layout child %s in %s (%s) bounds=%+v", componentLabel(child), componentLabel(parent), note, bounds)
		return
	}
	ctx.DebugLayoutf("layout child %s in %s bounds=%+v", componentLabel(child), componentLabel(parent), bounds)
}

func envDebugEnabled(key string) bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch val {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNonLayoutCaller() string {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(3, pcs) // skip Callers + firstNonLayoutCaller + logSurfaceAllocation
	if n == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "" {
			if !more {
				break
			}
			continue
		}
		if !strings.Contains(frame.Function, "layouter/layout") {
			return fmt.Sprintf("%s:%d", filepath.Base(frame.File), frame.Line)
		}
		if !more {
			break
		}
	}
	return ""
}
