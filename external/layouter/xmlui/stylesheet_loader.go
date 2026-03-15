package xmlui

import (
	"io/fs"

	"github.com/kellydornhaus/layouter/layout"
)

// StylesheetOptions allows future extension for stylesheet loading helpers.
type StylesheetOptions struct{}

// LoadStylesheet loads a stylesheet from disk, preserving @import semantics.
func LoadStylesheet(_ *layout.Context, filename string, _ StylesheetOptions) (*Stylesheet, error) {
	return ParseStylesheetFile(filename)
}

// LoadStylesheetFS loads a stylesheet from an fs.FS, preserving @import semantics.
func LoadStylesheetFS(_ *layout.Context, fsys fs.FS, name string, _ StylesheetOptions) (*Stylesheet, error) {
	return ParseStylesheetFS(fsys, name)
}
