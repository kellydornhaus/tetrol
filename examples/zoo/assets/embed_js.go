//go:build js && wasm
// +build js,wasm

package assets

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed images
var embeddedAssets embed.FS

var assetsFS fs.FS = embeddedAssets

func normalizeAssetPath(p string) (string, error) {
	trimmed := strings.TrimSpace(p)
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	trimmed = strings.TrimPrefix(trimmed, "./")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if strings.HasPrefix(trimmed, "assets/") {
		trimmed = strings.TrimPrefix(trimmed, "assets/")
	}
	clean := path.Clean(trimmed)
	if clean == "." || clean == "" {
		return "", fmt.Errorf("assets: empty path")
	}
	if strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("assets: path escapes root: %s", p)
	}
	return clean, nil
}

// LoadImage reads an embedded asset (PNG/JPEG/SVG) and caches the resulting ebiten.Image.
func LoadImage(p string) (*ebiten.Image, error) {
	rel, err := normalizeAssetPath(p)
	if err != nil {
		return nil, err
	}
	if v, ok := imageCache.Load(rel); ok {
		return v.(*ebiten.Image), nil
	}
	data, err := fs.ReadFile(assetsFS, rel)
	if err != nil {
		return nil, fmt.Errorf("read embedded image %s: %w", p, err)
	}
	img, err := decodeByExt(rel, data)
	if err != nil {
		return nil, err
	}
	eb := ebiten.NewImageFromImage(img)
	imageCache.Store(rel, eb)
	return eb, nil
}

var errWalkLimit = errors.New("image walk limit")

// ListImagePaths walks the embedded assets and returns up to limit image paths matching the provided extensions.
func ListImagePaths(root string, extensions []string, limit int) ([]string, error) {
	relRoot := strings.TrimSpace(root)
	if relRoot == "" {
		relRoot = "."
	}
	relRoot = strings.ReplaceAll(relRoot, "\\", "/")
	relRoot = strings.TrimPrefix(relRoot, "./")
	if relRoot == "." {
		relRoot = ""
	}
	if relRoot != "" {
		trimmed, err := normalizeAssetPath(relRoot)
		if err != nil {
			return nil, err
		}
		relRoot = trimmed
	}
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
	walkRoot := relRoot
	if walkRoot == "" {
		walkRoot = "."
	}
	walkErr := fs.WalkDir(assetsFS, walkRoot, func(p string, d fs.DirEntry, err error) error {
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
		ext := strings.ToLower(path.Ext(p))
		if len(extSet) > 0 {
			if _, ok := extSet[ext]; !ok {
				return nil
			}
		}
		fullPath := path.Join("assets", p)
		paths = append(paths, fullPath)
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
