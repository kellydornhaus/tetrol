//go:build js && wasm
// +build js,wasm

package screens

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed layouts
var embeddedLayouts embed.FS

func layoutFS() fs.FS {
	sub, err := fs.Sub(embeddedLayouts, "layouts")
	if err != nil {
		log.Printf("zoo: embedded layouts missing: %v", err)
		return nil
	}
	return sub
}
