//go:build !(js && wasm)
// +build !js !wasm

package assets

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// LoadImage loads an image from disk (PNG/JPEG/SVG) and caches the resulting ebiten.Image.
func LoadImage(path string) (*ebiten.Image, error) {
	key := path
	if abs, err := filepath.Abs(path); err == nil {
		key = abs
	}
	if v, ok := imageCache.Load(key); ok {
		return v.(*ebiten.Image), nil
	}
	file, err := os.Open(key)
	if err != nil {
		return nil, fmt.Errorf("open image %s: %w", path, err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read image %s: %w", path, err)
	}
	img, err := decodeByExt(key, data)
	if err != nil {
		return nil, err
	}
	eb := ebiten.NewImageFromImage(img)
	imageCache.Store(key, eb)
	return eb, nil
}

var errWalkLimit = errors.New("image walk limit")

// ListImagePaths walks the directory and returns up to limit image paths matching the provided extensions.
func ListImagePaths(root string, extensions []string, limit int) ([]string, error) {
	extSet := map[string]struct{}{}
	for _, ext := range extensions {
		if ext == "" {
			continue
		}
		e := strings.ToLower(ext)
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extSet[e] = struct{}{}
	}
	var paths []string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if len(extSet) > 0 {
			if _, ok := extSet[ext]; !ok {
				return nil
			}
		}
		paths = append(paths, path)
		if limit > 0 && len(paths) >= limit {
			return errWalkLimit
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errWalkLimit) {
		return nil, walkErr
	}
	sort.Strings(paths)
	if limit > 0 && len(paths) > limit {
		paths = paths[:limit]
	}
	return paths, nil
}
