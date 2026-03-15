package xmlui

import (
	"strings"
	"testing"
)

func TestLintFindsUnusedAndShadowedProperties(t *testing.T) {
	xmlSrc := `<VStack foo="bar" padding="4">
  <Panel id="p1" width="100" style="color: #fff; unknown-prop: 1"/>
</VStack>`
	cssSrc := `Panel { width: 200dp; bogus: 1; color: #111; }`

	root, _, err := parseXMLWithStyles(strings.NewReader(xmlSrc))
	if err != nil {
		t.Fatalf("parse xml: %v", err)
	}
	sheet, err := ParseStylesheet(strings.NewReader(cssSrc))
	if err != nil {
		t.Fatalf("parse css: %v", err)
	}

	report := Lint(root, sheet)

	assertHas := func(kind IssueKind, substr string) {
		t.Helper()
		for _, iss := range report.Issues {
			if iss.Kind == kind && strings.Contains(iss.Message, substr) {
				return
			}
		}
		t.Fatalf("expected issue %s containing %q, got %+v", kind, substr, report.Issues)
	}

	assertHas(IssueUnusedXMLAttr, "foo")
	assertHas(IssueUnusedCSSProp, "unknown-prop")
	assertHas(IssueUnusedCSSProp, "bogus")
	assertHas(IssueXMLShadowsCSS, "width")
}
