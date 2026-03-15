package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Action int

const (
	ActionNone Action = iota
	ActionMoveLeft
	ActionMoveRight
	ActionMoveUp
	ActionMoveDown
	ActionSwap
	ActionToggle
)

const (
	repeatDelay = 15
	repeatRate  = 4
)

type Input struct {
	holdTicks map[ebiten.Key]int
}

func NewInput() *Input {
	return &Input{holdTicks: make(map[ebiten.Key]int)}
}

func (inp *Input) Update() Action {
	for _, key := range []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyArrowUp, ebiten.KeyArrowDown} {
		if ebiten.IsKeyPressed(key) {
			inp.holdTicks[key]++
		} else {
			inp.holdTicks[key] = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		return ActionToggle
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return ActionSwap
	}

	if inp.shouldFire(ebiten.KeyArrowLeft) {
		return ActionMoveLeft
	}
	if inp.shouldFire(ebiten.KeyArrowRight) {
		return ActionMoveRight
	}
	if inp.shouldFire(ebiten.KeyArrowUp) {
		return ActionMoveUp
	}
	if inp.shouldFire(ebiten.KeyArrowDown) {
		return ActionMoveDown
	}

	return ActionNone
}

func (inp *Input) shouldFire(key ebiten.Key) bool {
	if inpututil.IsKeyJustPressed(key) {
		return true
	}
	ticks := inp.holdTicks[key]
	return ticks >= repeatDelay && (ticks-repeatDelay)%repeatRate == 0
}
