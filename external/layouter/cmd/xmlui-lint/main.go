package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kellydornhaus/layouter/xmlui"
)

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(val string) error {
	*m = append(*m, val)
	return nil
}

func main() {
	var cssPaths multiFlag
	xmlPath := flag.String("xml", "", "Path to XML layout file (required)")
	flag.Var(&cssPaths, "css", "Additional CSS stylesheet (repeatable)")
	flag.Parse()

	if strings.TrimSpace(*xmlPath) == "" {
		fmt.Fprintln(os.Stderr, "xmlui-lint: -xml path is required")
		os.Exit(2)
	}

	report, err := xmlui.LintFile(*xmlPath, xmlui.LintFileOptions{ExtraCSS: cssPaths})
	if err != nil {
		log.Fatalf("xmlui-lint: %v", err)
	}
	if len(report.Issues) == 0 {
		fmt.Println("No lint issues found.")
		return
	}
	for _, issue := range report.Issues {
		fmt.Printf("%s: %s\n", issue.Kind, issue.Message)
	}
}
