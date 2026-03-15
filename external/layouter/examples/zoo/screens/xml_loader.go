package screens

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/kellydornhaus/layouter/layout"
	"github.com/kellydornhaus/layouter/xmlui"
)

func buildLayout(ctx *layout.Context, reg *xmlui.Registry, xmlPath string, fsys fs.FS) (xmlui.Result, error) {
	opts := xmlui.Options{
		ImageLoader: xmlui.ImageLoaderFunc(loadLayoutImage),
	}
	if fsys != nil {
		opts.ResolveImagePath = resolveEmbeddedImagePath
		name := trimLayoutPath(xmlPath)
		return xmlui.BuildFS(ctx, fsys, name, reg, opts)
	}

	baseDir := filepath.Dir(xmlPath)
	opts.ResolveImagePath = func(ref string) (string, error) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return "", nil
		}
		if filepath.IsAbs(ref) {
			return filepath.Clean(ref), nil
		}
		if strings.HasPrefix(ref, "assets/") || strings.HasPrefix(ref, "assets\\") {
			return filepath.Clean(ref), nil
		}
		return filepath.Clean(filepath.Join(baseDir, ref)), nil
	}
	return xmlui.BuildFile(ctx, xmlPath, reg, opts)
}

func trimLayoutPath(p string) string {
	clean := strings.ReplaceAll(p, "\\", "/")
	clean = strings.TrimPrefix(clean, "./")
	clean = strings.TrimPrefix(clean, "/")
	if strings.HasPrefix(clean, "screens/layouts/") {
		clean = strings.TrimPrefix(clean, "screens/layouts/")
	}
	if strings.HasPrefix(clean, "layouts/") {
		clean = strings.TrimPrefix(clean, "layouts/")
	}
	return clean
}

func resolveEmbeddedImagePath(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	clean := path.Clean(strings.ReplaceAll(ref, "\\", "/"))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid image path")
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("image path escapes layouts fs: %s", ref)
	}
	return clean, nil
}
