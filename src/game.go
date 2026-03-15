package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	text "github.com/kellydornhaus/layouter/adapters/etxt"
	"github.com/kellydornhaus/layouter/layout"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/sfnt"
)

type GameState int

const (
	StatePlaying GameState = iota
	StateGameOver
	StateWin
)

type FloatingText struct {
	Text  string
	X, Y  float64
	Timer int
	Max   int
	Color color.NRGBA
}

type Game struct {
	board    *Board
	targets  *TargetManager
	renderer *Renderer
	input    *Input
	scoring  *Scoring

	ctx       *layout.Context
	txtEngine *text.EtxtEngine

	state  GameState
	floats []FloatingText
}

func NewGame() *Game {
	rnd := adp.NewRenderer()
	scale := adp.ScaleProvider{}
	txt := text.New(scale)

	fontsDir := filepath.Join("assets", "fonts")
	tileFontData, scoreFontData := loadGameFonts(txt, fontsDir)
	ctx := layout.NewContext(scale, rnd, txt)

	dict := LoadDictionary()
	scoring := NewScoring()
	targets := NewTargetManager(dict)
	board := NewBoard(dict, scoring, targets)
	renderer := NewRenderer(tileFontData, scoreFontData, ctx)
	input := NewInput()

	return &Game{
		board:     board,
		targets:   targets,
		renderer:  renderer,
		input:     input,
		scoring:   scoring,
		ctx:       ctx,
		txtEngine: txt,
		state:     StatePlaying,
	}
}

func loadGameFonts(txt *text.EtxtEngine, fontsDir string) (tileFontData, scoreFontData []byte) {
	entries, err := os.ReadDir(fontsDir)
	if err != nil {
		return nil, nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".ttf") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fontsDir, e.Name()))
		if err != nil {
			continue
		}
		font, err := sfnt.Parse(data)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		txt.RegisterFont(name, font)

		if name == "Montserrat-Bold" {
			tileFontData = data
		}
		if name == "RobotoMono-Regular" {
			scoreFontData = data
		}
	}
	return
}

func (g *Game) Update() error {
	alive := g.floats[:0]
	for i := range g.floats {
		g.floats[i].Timer--
		g.floats[i].Y -= 0.8
		if g.floats[i].Timer > 0 {
			alive = append(alive, g.floats[i])
		}
	}
	g.floats = alive

	switch g.state {
	case StatePlaying:
		action := g.input.Update()
		switch action {
		case ActionMoveLeft:
			g.board.MoveCursor(-1, 0)
		case ActionMoveRight:
			g.board.MoveCursor(1, 0)
		case ActionMoveUp:
			g.board.MoveCursor(0, -1)
		case ActionMoveDown:
			g.board.MoveCursor(0, 1)
		case ActionToggle:
			g.board.ToggleCursorDirection()
		case ActionSwap:
			g.board.Swap()
		}

		g.board.Update()

		// Mark cleared target words
		for _, word := range g.board.PendingClears {
			g.targets.MarkCleared(word)
		}
		g.board.PendingClears = nil

		// Tick target animations
		g.targets.Update()

		// Drain auto-clear popups from board
		if len(g.board.PendingPopups) > 0 {
			g.spawnSectionPopups(g.board.PendingPopups)
			g.board.PendingPopups = nil
		}

		if g.targets.AllDone() {
			g.state = StateWin
		} else if g.board.IsGameOver() {
			g.state = StateGameOver
		}

	case StateGameOver:
		if g.input.Update() == ActionSwap {
			g.targets = NewTargetManager(g.board.dict)
			g.board = NewBoard(g.board.dict, g.scoring, g.targets)
			g.scoring.Reset()
			g.state = StatePlaying
			g.floats = nil
		}

	case StateWin:
		if g.input.Update() == ActionSwap {
			g.targets = NewTargetManager(g.board.dict)
			g.board = NewBoard(g.board.dict, g.scoring, g.targets)
			g.scoring.Reset()
			g.state = StatePlaying
			g.floats = nil
		}
	}

	return nil
}

func (g *Game) spawnSectionPopups(sections []Section) {
	for _, sec := range sections {
		cx, cy := g.renderer.SectionCenter(sec, g.board)
		points := ScoreSection(len(sec.Tiles), len(sec.Words), g.board.ChainCount, sec.HasIntersection)

		label := fmt.Sprintf("+%d", points)
		if len(sec.Words) >= 2 {
			label = fmt.Sprintf("%dw +%d", len(sec.Words), points)
		}
		if sec.HasIntersection {
			label += " x2!"
		}
		if g.board.ChainCount > 1 {
			label += fmt.Sprintf(" CHAIN x%d", g.board.ChainCount)
		}

		g.floats = append(g.floats, FloatingText{
			Text:  label,
			X:     cx,
			Y:     cy,
			Timer: 100,
			Max:   100,
			Color: color.NRGBA{255, 255, 255, 255},
		})
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 18, 28, 255})
	g.renderer.Draw(screen, g.board, g.scoring, g.targets, g.state, g.floats)
}

func (g *Game) Layout(_, _ int) (int, int) {
	return 800, 600
}

func (g *Game) LayoutF(logicW, logicH float64) (float64, float64) {
	scale := adp.ScaleProvider{}.DeviceScaleFactor()
	if scale <= 0 {
		scale = 1
	}
	g.ctx.Scale = scale
	return logicW * scale, logicH * scale
}
