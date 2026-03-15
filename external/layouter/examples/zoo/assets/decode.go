package assets

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"path/filepath"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func decodeByExt(path string, data []byte) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".svg":
		return rasterizeSVG(data)
	default:
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("decode image %s: %w", path, err)
		}
		return img, nil
	}
}

func rasterizeSVG(data []byte) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data), oksvg.WarnErrorMode)
	if err != nil {
		return nil, fmt.Errorf("parse svg: %w", err)
	}
	baseW := icon.ViewBox.W
	baseH := icon.ViewBox.H
	if baseW <= 0 || baseH <= 0 {
		baseW, baseH = 128, 128
	}
	scale := 1.0
	maxDim := math.Max(baseW, baseH)
	if maxDim > 0 && maxDim < 256 {
		scale = 256.0 / maxDim
	}
	w := int(math.Round(baseW * scale))
	h := int(math.Round(baseH * scale))
	if w <= 0 {
		w = 128
	}
	if h <= 0 {
		h = 128
	}
	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(w, h, scanner)
	icon.Draw(raster, 1.0)
	return rgba, nil
}
