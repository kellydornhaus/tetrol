//go:build !(js && wasm)
// +build !js !wasm

package screens

import "io/fs"

func layoutFS() fs.FS { return nil }
