package screens

import (
	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	"github.com/kellydornhaus/layouter/examples/zoo/assets"
	"github.com/kellydornhaus/layouter/layout"
)

func loadLayoutImage(path string) (layout.Image, error) {
	img, err := assets.LoadImage(path)
	if err != nil {
		return nil, err
	}
	return adp.WrapImage(img), nil
}
