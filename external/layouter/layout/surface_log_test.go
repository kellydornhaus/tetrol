package layout

import (
	"fmt"
	"strings"
	"testing"
)

func TestSurfaceLogging(t *testing.T) {
	t.Helper()
	rnd := &fakeRenderer{}
	ctx := &Context{
		Scale:    1,
		Renderer: rnd,
		Debug: DebugOptions{
			LogSurfaceAllocations: true,
			LogSurfaceUsage:       true,
		},
	}
	var logs []string
	ctx.Debug.Logger = func(format string, args ...interface{}) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}

	comp := &stubComp{size: Size{W: 10, H: 8}}
	comp.Layout(ctx, nil, Rect{W: 10, H: 8})
	dst := &fakeSurface{w: 20, h: 20}

	comp.DrawTo(ctx, dst) // alloc + render
	comp.DrawTo(ctx, dst) // cache hit

	if len(logs) < 3 {
		t.Fatalf("expected surface logs, got %d entries: %+v", len(logs), logs)
	}
	foundAlloc := false
	foundRender := false
	foundCache := false
	for _, line := range logs {
		if strings.Contains(line, "alloc") {
			foundAlloc = true
		}
		if strings.Contains(line, "render") {
			foundRender = true
		}
		if strings.Contains(line, "cache-hit") {
			foundCache = true
		}
	}
	if !foundAlloc || !foundRender || !foundCache {
		t.Fatalf("missing surface log entries alloc=%t render=%t cache=%t in logs: %+v", foundAlloc, foundRender, foundCache, logs)
	}
	last := logs[len(logs)-1]
	if !strings.Contains(last, "draws=2") || !strings.Contains(last, "cache-hits=1") {
		t.Fatalf("expected aggregated counts in last log, got: %s", last)
	}
}

func TestSurfaceMemoryLoggingPerFrame(t *testing.T) {
	t.Helper()
	rnd := &fakeRenderer{}
	ctx := &Context{
		Scale:    1,
		Renderer: rnd,
		Debug: DebugOptions{
			LogSurfaceAllocations: true,
		},
	}
	var logs []string
	ctx.Debug.Logger = func(format string, args ...interface{}) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}

	comp := &stubComp{size: Size{W: 10, H: 10}}
	comp.Layout(ctx, nil, Rect{W: 10, H: 10})
	dst := &fakeSurface{w: 20, h: 20}

	memoryLogCount := func() int {
		count := 0
		for _, line := range logs {
			if strings.Contains(line, "[surface] memory") {
				count++
			}
		}
		return count
	}

	ctx.BeginFrameLog(1)
	comp.DrawTo(ctx, dst)
	ctx.EndFrameLog()
	if got := memoryLogCount(); got != 1 {
		t.Fatalf("expected one memory log after first frame, got %d entries: %+v", got, logs)
	}

	ctx.BeginFrameLog(2)
	comp.DrawTo(ctx, dst)
	ctx.EndFrameLog()
	if got := memoryLogCount(); got != 1 {
		t.Fatalf("expected no additional memory log when usage unchanged, got %d entries: %+v", got, logs)
	}

	comp.Layout(ctx, nil, Rect{W: 12, H: 10})
	ctx.BeginFrameLog(3)
	comp.DrawTo(ctx, dst)
	ctx.EndFrameLog()
	if got := memoryLogCount(); got != 2 {
		t.Fatalf("expected second memory log after size change, got %d entries: %+v", got, logs)
	}
}
