package layout

import (
	"fmt"
	"strings"
	"testing"
)

func TestFrameLogDedup(t *testing.T) {
	t.Helper()
	ctx := &Context{
		Debug: DebugOptions{
			LogLayoutDecisions: true,
		},
	}
	var flushed []string
	ctx.Debug.Logger = func(format string, args ...interface{}) {
		flushed = append(flushed, fmt.Sprintf(format, args...))
	}

	ctx.BeginFrameLog(1)
	ctx.debugLogf("layout", "first")
	ctx.EndFrameLog()
	if len(flushed) != 1 || !strings.Contains(flushed[0], "frame 1") || !strings.Contains(flushed[0], "[layout] first") {
		t.Fatalf("unexpected first frame flush: %+v", flushed)
	}

	ctx.BeginFrameLog(2)
	ctx.debugLogf("layout", "first")
	ctx.EndFrameLog()
	if len(flushed) != 1 {
		t.Fatalf("expected no flush on identical frame, got %d: %+v", len(flushed), flushed)
	}

	ctx.BeginFrameLog(3)
	ctx.debugLogf("layout", "second")
	ctx.EndFrameLog()
	if len(flushed) != 2 || !strings.Contains(flushed[1], "frame 3") || !strings.Contains(flushed[1], "[layout] second") {
		t.Fatalf("unexpected third frame flush: %+v", flushed)
	}
}
