package xmlui

import (
	"strings"
	"testing"
)

func TestParseStylesheetLongLine(t *testing.T) {
	long := strings.Repeat("a", 70*1024)
	css := ".a { padding: 1; }" + long
	sheet, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatalf("ParseStylesheet error: %v", err)
	}
	if sheet == nil || len(sheet.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %v", len(sheet.Rules))
	}
}
