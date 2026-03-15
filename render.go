package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kellydornhaus/layouter/layout"
)

// Logical (unscaled) constants — multiply by ctx.Scale when drawing.
const (
	TileSize    = 40
	TilePadding = 2
	BoardLeft   = 55
	BoardTop    = 55
)

var (
	BoardPixelW = BoardCols*TileSize + (BoardCols-1)*TilePadding
	BoardPixelH = BoardRows*TileSize + (BoardRows-1)*TilePadding
	PanelLeft   = BoardLeft + BoardPixelW + 30
)

// boardMetrics holds pre-scaled layout values used by every draw method.
type boardMetrics struct {
	s      float64 // device scale factor
	ts, tp float64 // tile size, tile padding (scaled)
	step   float64 // ts + tp
	bl, bt float64 // board left, board top (scaled)
	bw, bh float64 // board pixel width, height (scaled)
	pl     float64 // panel left (scaled)
}

type Renderer struct {
	tileFace  *text.GoTextFaceSource
	scoreFace *text.GoTextFaceSource
	ctx       *layout.Context
	frame     int
}

func NewRenderer(tileFontData, scoreFontData []byte, ctx *layout.Context) *Renderer {
	r := &Renderer{ctx: ctx}
	if tileFontData != nil {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(tileFontData))
		if err != nil {
			log.Printf("Warning: tile font: %v", err)
		} else {
			r.tileFace = src
		}
	}
	if scoreFontData != nil {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(scoreFontData))
		if err != nil {
			log.Printf("Warning: score font: %v", err)
		} else {
			r.scoreFace = src
		}
	}
	return r
}

func (r *Renderer) metrics() boardMetrics {
	s := r.ctx.Scale
	if s <= 0 {
		s = 1
	}
	ts := float64(TileSize) * s
	tp := float64(TilePadding) * s
	return boardMetrics{
		s: s, ts: ts, tp: tp, step: ts + tp,
		bl: float64(BoardLeft) * s,
		bt: float64(BoardTop) * s,
		bw: float64(BoardPixelW) * s,
		bh: float64(BoardPixelH) * s,
		pl: float64(PanelLeft) * s,
	}
}

// SectionCenter returns the logical (unscaled) center of a section's tiles.
func (r *Renderer) SectionCenter(sec Section, b *Board) (float64, float64) {
	rise := b.RiseOffset * float64(TileSize+TilePadding)
	var sx, sy float64
	for _, p := range sec.Tiles {
		sx += float64(BoardLeft + p[0]*(TileSize+TilePadding) + TileSize/2)
		sy += float64(BoardTop+p[1]*(TileSize+TilePadding)+TileSize/2) - rise
	}
	n := float64(len(sec.Tiles))
	return sx / n, sy / n
}

func (r *Renderer) Draw(screen *ebiten.Image, board *Board, scoring *Scoring, targets *TargetManager, state GameState, floats []FloatingText) {
	r.frame++

	r.drawBoardBackground(screen)
	r.drawTiles(screen, board)
	r.drawCursor(screen, board)
	r.drawHUD(screen, scoring, board, targets)
	r.drawTargetPanel(screen, targets)
	r.drawFloatingTexts(screen, floats)

	if state == StateGameOver {
		r.drawGameOver(screen, scoring)
	}
	if state == StateWin {
		r.drawWinScreen(screen, scoring)
	}
}

func (r *Renderer) drawBoardBackground(screen *ebiten.Image) {
	m := r.metrics()

	// Outer border
	vector.DrawFilledRect(screen,
		float32(m.bl-6*m.s), float32(m.bt-6*m.s),
		float32(m.bw+12*m.s), float32(m.bh+12*m.s),
		color.RGBA{8, 8, 16, 255}, false)
	vector.StrokeRect(screen,
		float32(m.bl-6*m.s), float32(m.bt-6*m.s),
		float32(m.bw+12*m.s), float32(m.bh+12*m.s),
		float32(1.5*m.s), color.RGBA{50, 50, 80, 255}, false)

	// Grid lines (subtle)
	gridColor := color.NRGBA{30, 30, 45, 100}
	for col := 0; col <= BoardCols; col++ {
		x := float32(m.bl + float64(col)*m.step - m.tp/2)
		vector.StrokeLine(screen, x, float32(m.bt), x, float32(m.bt+m.bh), float32(0.5*m.s), gridColor, false)
	}
	for row := 0; row <= BoardRows; row++ {
		y := float32(m.bt + float64(row)*m.step - m.tp/2)
		vector.StrokeLine(screen, float32(m.bl), y, float32(m.bl+m.bw), y, float32(0.5*m.s), gridColor, false)
	}
}

func (r *Renderer) drawTiles(screen *ebiten.Image, board *Board) {
	m := r.metrics()
	rise := board.RiseOffset * m.step

	for col := 0; col < BoardCols; col++ {
		for row := 0; row < BoardRows; row++ {
			t := board.Tiles[col][row]
			if t == nil {
				continue
			}

			x := m.bl + float64(col)*m.step
			y := m.bt + float64(row)*m.step - rise

			if t.State == TileFalling {
				y += t.FallOffset * m.step
			}
			if t.State == TileSwapping {
				if t.SwapVertical {
					y += t.SwapOffset * m.step
				} else {
					x += t.SwapOffset * m.step
				}
			}

			r.drawTile(screen, t, x, y, m.ts)
		}
	}
}

func (r *Renderer) drawTile(screen *ebiten.Image, t *Tile, x, y, size float64) {
	r.drawTileAlpha(screen, t, x, y, size, 1.0)
}

func (r *Renderer) drawTileAlpha(screen *ebiten.Image, t *Tile, x, y, size float64, alphaOverride float32) {
	s := r.ctx.Scale
	alpha := alphaOverride

	if t.State == TileClearing {
		progress := 1.0 - float64(t.AnimTick)/float64(ClearAnimTicks)
		shrink := 1.0 - progress*0.5
		offset := size * (1 - shrink) / 2
		x += offset
		y += offset
		size *= shrink
		alpha = float32(1.0 - progress)

		// White flash
		if progress < 0.3 {
			fa := float32((0.3 - progress) / 0.3)
			vector.DrawFilledRect(screen, float32(x)-float32(2*s), float32(y)-float32(2*s), float32(size)+float32(4*s), float32(size)+float32(4*s),
				color.NRGBA{255, 255, 255, uint8(fa * 180)}, false)
		}
	}

	// Background color
	bg := letterColor(t.Letter)

	// Brighten matched tiles
	if t.State == TileMatched {
		bg.R = clampAdd(bg.R, 30)
		bg.G = clampAdd(bg.G, 30)
		bg.B = clampAdd(bg.B, 30)
	}

	bg.A = uint8(float32(bg.A) * alpha)
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(size), float32(size), bg, false)

	// Top-edge highlight for depth
	highlight := color.NRGBA{255, 255, 255, uint8(25 * alpha)}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(size), float32(2*s), highlight, false)

	// Border
	borderColor := color.NRGBA{255, 255, 255, uint8(20 * alpha)}
	vector.StrokeRect(screen, float32(x), float32(y), float32(size), float32(size), float32(0.5*s), borderColor, false)

	// Letter
	if r.tileFace != nil {
		fontSize := size * 0.55
		face := &text.GoTextFace{Source: r.tileFace, Size: fontSize}
		str := string(t.Letter)
		w, h := text.Measure(str, face, 0)
		tx := x + (size-w)/2
		ty := y + (size-h)/2

		// Shadow
		if alpha > 0.5 {
			opS := &text.DrawOptions{}
			opS.GeoM.Translate(tx+1*s, ty+1*s)
			opS.ColorScale.Scale(0, 0, 0, alpha*0.5)
			text.Draw(screen, str, face, opS)
		}

		op := &text.DrawOptions{}
		op.GeoM.Translate(tx, ty)
		op.ColorScale.Scale(1, 1, 1, alpha)
		text.Draw(screen, str, face, op)
	}
}


func (r *Renderer) drawCursor(screen *ebiten.Image, board *Board) {
	m := r.metrics()
	col := board.CursorCol
	row := board.CursorRow
	rise := board.RiseOffset * m.step

	pad := float32(4 * m.s)
	x := float32(m.bl+float64(col)*m.step) - pad
	y := float32(m.bt+float64(row)*m.step-rise) - pad

	var w, h float32
	if board.CursorVertical {
		w = float32(m.ts) + pad*2
		h = float32(m.ts*2+m.tp) + pad*2
	} else {
		w = float32(m.ts*2+m.tp) + pad*2
		h = float32(m.ts) + pad*2
	}

	pulse := float32(math.Sin(float64(r.frame)*0.12)*0.2 + 0.8)
	ca := uint8(255 * pulse)

	bracketLen := float32(6 * m.s)
	thick := float32(2.5 * m.s)
	cc := color.NRGBA{255, 220, 50, ca}

	// Top-left bracket
	vector.StrokeLine(screen, x, y, x+bracketLen, y, thick, cc, false)
	vector.StrokeLine(screen, x, y, x, y+bracketLen, thick, cc, false)

	// Top-right bracket
	rx := x + w
	vector.StrokeLine(screen, rx-bracketLen, y, rx, y, thick, cc, false)
	vector.StrokeLine(screen, rx, y, rx, y+bracketLen, thick, cc, false)

	// Bottom-left bracket
	by := y + h
	vector.StrokeLine(screen, x, by, x+bracketLen, by, thick, cc, false)
	vector.StrokeLine(screen, x, by-bracketLen, x, by, thick, cc, false)

	// Bottom-right bracket
	vector.StrokeLine(screen, rx-bracketLen, by, rx, by, thick, cc, false)
	vector.StrokeLine(screen, rx, by-bracketLen, rx, by, thick, cc, false)

	// Center divider
	divColor := color.NRGBA{255, 220, 50, uint8(float32(80) * pulse)}
	if board.CursorVertical {
		cy := float32(m.bt+float64(row+1)*m.step-rise) - float32(m.tp)/2
		cx1 := float32(m.bl + float64(col)*m.step)
		vector.StrokeLine(screen, cx1, cy, cx1+float32(m.ts), cy, float32(1*m.s), divColor, false)
	} else {
		cx := float32(m.bl+float64(col+1)*m.step) - float32(m.tp)/2
		cy1 := float32(m.bt + float64(row)*m.step - rise)
		vector.StrokeLine(screen, cx, cy1, cx, cy1+float32(m.ts), float32(1*m.s), divColor, false)
	}
}

func (r *Renderer) drawHUD(screen *ebiten.Image, scoring *Scoring, board *Board, targets *TargetManager) {
	if r.scoreFace == nil {
		return
	}

	m := r.metrics()
	face := &text.GoTextFace{Source: r.scoreFace, Size: 18 * m.s}

	// Score - top left
	scoreStr := fmt.Sprintf("SCORE  %d", scoring.Score)
	op := &text.DrawOptions{}
	op.GeoM.Translate(m.bl, 18*m.s)
	op.ColorScale.ScaleWithColor(color.NRGBA{220, 220, 230, 255})
	text.Draw(screen, scoreStr, face, op)

	// Progress - top right of board
	progressStr := fmt.Sprintf("%d / 10", targets.Cleared)
	ww, _ := text.Measure(progressStr, face, 0)
	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(m.bl+m.bw-ww, 18*m.s)
	op2.ColorScale.ScaleWithColor(color.NRGBA{220, 220, 230, 255})
	text.Draw(screen, progressStr, face, op2)

	// Chain indicator
	if board.ChainCount > 1 {
		chainFace := &text.GoTextFace{Source: r.tileFace, Size: 28 * m.s}
		if r.tileFace != nil {
			chainStr := fmt.Sprintf("CHAIN x%d!", board.ChainCount)
			cw, _ := text.Measure(chainStr, chainFace, 0)
			cx := m.bl + m.bw/2 - cw/2
			cy := m.bt + m.bh + 15*m.s
			op3 := &text.DrawOptions{}
			op3.GeoM.Translate(cx, cy)
			pulse := math.Sin(float64(r.frame)*0.15)*0.3 + 0.7
			op3.ColorScale.ScaleWithColor(color.NRGBA{255, 200, 50, uint8(255 * pulse)})
			text.Draw(screen, chainStr, chainFace, op3)
		}
	}

	// Controls hint at bottom — screen bounds are already in canvas (scaled) pixels
	hintFace := &text.GoTextFace{Source: r.scoreFace, Size: 11 * m.s}
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()
	hint := "ARROWS move  |  SPACE swap  |  TAB toggle H/V"
	hw, _ := text.Measure(hint, hintFace, 0)
	op4 := &text.DrawOptions{}
	op4.GeoM.Translate((float64(sw)-hw)/2, float64(sh)-18*m.s)
	op4.ColorScale.ScaleWithColor(color.NRGBA{80, 80, 100, 255})
	text.Draw(screen, hint, hintFace, op4)
}

func (r *Renderer) drawTargetPanel(screen *ebiten.Image, targets *TargetManager) {
	if r.scoreFace == nil {
		return
	}

	m := r.metrics()
	px := m.pl
	py := m.bt

	headerFace := &text.GoTextFace{Source: r.scoreFace, Size: 14 * m.s}

	// Header
	op := &text.DrawOptions{}
	op.GeoM.Translate(px, py)
	op.ColorScale.ScaleWithColor(color.NRGBA{180, 180, 200, 255})
	text.Draw(screen, "TARGET WORDS", headerFace, op)

	// Progress
	progressStr := fmt.Sprintf("%d/10 CLEARED", targets.Cleared)
	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(px, py+18*m.s)
	op2.ColorScale.ScaleWithColor(color.NRGBA{100, 200, 100, 255})
	text.Draw(screen, progressStr, headerFace, op2)

	// Show visible target words as small tiles
	cy := py + 46*m.s
	smallTile := m.ts * 0.7
	smallPad := m.tp * 0.7

	for _, tw := range targets.Visible {
		alpha := float32(1.0)
		if tw.State == TargetClearing {
			progress := 1.0 - float64(tw.AnimTick)/float64(TargetClearTicks)
			alpha = float32(1.0 - progress)
			if int(progress*10)%2 == 0 {
				alpha *= 0.3
			}
		}

		for j, ch := range tw.Jumbled {
			tx := px + float64(j)*(smallTile+smallPad)
			t := &Tile{Letter: ch, State: TileIdle}
			r.drawTileAlpha(screen, t, tx, cy, smallTile, alpha)
		}

		cy += smallTile + 8*m.s
	}
}

func (r *Renderer) drawFloatingTexts(screen *ebiten.Image, floats []FloatingText) {
	if r.tileFace == nil {
		return
	}

	m := r.metrics()

	for _, ft := range floats {
		progress := 1.0 - float64(ft.Timer)/float64(ft.Max)
		alpha := 1.0 - progress
		if alpha < 0 {
			alpha = 0
		}

		// Scale up then shrink
		animScale := 1.0
		if progress < 0.1 {
			animScale = 0.5 + progress*5
		} else if progress > 0.7 {
			animScale = 1.0 - (progress-0.7)/0.3*0.5
		}

		fontSize := 22.0 * animScale * m.s
		face := &text.GoTextFace{Source: r.tileFace, Size: fontSize}
		w, h := text.Measure(ft.Text, face, 0)

		// ft.X and ft.Y are logical coords — scale to canvas
		fx := ft.X * m.s
		fy := ft.Y * m.s

		// Shadow
		opS := &text.DrawOptions{}
		opS.GeoM.Translate(fx-w/2+1*m.s, fy-h/2+1*m.s)
		opS.ColorScale.Scale(0, 0, 0, float32(alpha*0.6))
		text.Draw(screen, ft.Text, face, opS)

		// Text
		op := &text.DrawOptions{}
		op.GeoM.Translate(fx-w/2, fy-h/2)
		op.ColorScale.Scale(1, 1, 1, float32(alpha))
		text.Draw(screen, ft.Text, face, op)
	}
}

func (r *Renderer) drawGameOver(screen *ebiten.Image, scoring *Scoring) {
	m := r.metrics()
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Dim overlay
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), color.NRGBA{0, 0, 0, 180}, false)

	if r.tileFace != nil {
		face := &text.GoTextFace{Source: r.tileFace, Size: 48 * m.s}
		msg := "GAME OVER"
		w, _ := text.Measure(msg, face, 0)

		// Shadow
		opS := &text.DrawOptions{}
		opS.GeoM.Translate((float64(sw)-w)/2+2*m.s, float64(sh)/2-38*m.s)
		opS.ColorScale.Scale(0, 0, 0, 0.6)
		text.Draw(screen, msg, face, opS)

		op := &text.DrawOptions{}
		op.GeoM.Translate((float64(sw)-w)/2, float64(sh)/2-40*m.s)
		text.Draw(screen, msg, face, op)
	}

	if r.scoreFace != nil {
		scoreFace := &text.GoTextFace{Source: r.scoreFace, Size: 22 * m.s}
		scoreMsg := fmt.Sprintf("Score: %d   Words: %d", scoring.Score, scoring.WordCount)
		sw2, _ := text.Measure(scoreMsg, scoreFace, 0)
		op2 := &text.DrawOptions{}
		op2.GeoM.Translate((float64(sw)-sw2)/2, float64(sh)/2+20*m.s)
		op2.ColorScale.ScaleWithColor(color.NRGBA{200, 200, 210, 255})
		text.Draw(screen, scoreMsg, scoreFace, op2)

		if scoring.MaxChain > 1 {
			chainMsg := fmt.Sprintf("Best Chain: x%d", scoring.MaxChain)
			cw, _ := text.Measure(chainMsg, scoreFace, 0)
			op3 := &text.DrawOptions{}
			op3.GeoM.Translate((float64(sw)-cw)/2, float64(sh)/2+50*m.s)
			op3.ColorScale.ScaleWithColor(color.NRGBA{255, 200, 50, 255})
			text.Draw(screen, chainMsg, scoreFace, op3)
		}

		restartFace := &text.GoTextFace{Source: r.scoreFace, Size: 16 * m.s}
		restartMsg := "Press SPACE to restart"
		rw, _ := text.Measure(restartMsg, restartFace, 0)
		op4 := &text.DrawOptions{}
		op4.GeoM.Translate((float64(sw)-rw)/2, float64(sh)/2+90*m.s)
		op4.ColorScale.ScaleWithColor(color.NRGBA{100, 100, 120, 255})
		text.Draw(screen, restartMsg, restartFace, op4)
	}
}

func (r *Renderer) drawWinScreen(screen *ebiten.Image, scoring *Scoring) {
	m := r.metrics()
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Dim overlay
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), color.NRGBA{0, 0, 0, 180}, false)

	if r.tileFace != nil {
		face := &text.GoTextFace{Source: r.tileFace, Size: 48 * m.s}
		msg := "YOU WIN!"
		w, _ := text.Measure(msg, face, 0)

		// Shadow
		opS := &text.DrawOptions{}
		opS.GeoM.Translate((float64(sw)-w)/2+2*m.s, float64(sh)/2-38*m.s)
		opS.ColorScale.Scale(0, 0, 0, 0.6)
		text.Draw(screen, msg, face, opS)

		op := &text.DrawOptions{}
		op.GeoM.Translate((float64(sw)-w)/2, float64(sh)/2-40*m.s)
		op.ColorScale.ScaleWithColor(color.NRGBA{100, 255, 100, 255})
		text.Draw(screen, msg, face, op)
	}

	if r.scoreFace != nil {
		scoreFace := &text.GoTextFace{Source: r.scoreFace, Size: 22 * m.s}
		scoreMsg := fmt.Sprintf("Score: %d", scoring.Score)
		sw2, _ := text.Measure(scoreMsg, scoreFace, 0)
		op2 := &text.DrawOptions{}
		op2.GeoM.Translate((float64(sw)-sw2)/2, float64(sh)/2+20*m.s)
		op2.ColorScale.ScaleWithColor(color.NRGBA{200, 200, 210, 255})
		text.Draw(screen, scoreMsg, scoreFace, op2)

		if scoring.MaxChain > 1 {
			chainMsg := fmt.Sprintf("Best Chain: x%d", scoring.MaxChain)
			cw, _ := text.Measure(chainMsg, scoreFace, 0)
			op3 := &text.DrawOptions{}
			op3.GeoM.Translate((float64(sw)-cw)/2, float64(sh)/2+50*m.s)
			op3.ColorScale.ScaleWithColor(color.NRGBA{255, 200, 50, 255})
			text.Draw(screen, chainMsg, scoreFace, op3)
		}

		restartFace := &text.GoTextFace{Source: r.scoreFace, Size: 16 * m.s}
		restartMsg := "Press SPACE to play again"
		rw, _ := text.Measure(restartMsg, restartFace, 0)
		op4 := &text.DrawOptions{}
		op4.GeoM.Translate((float64(sw)-rw)/2, float64(sh)/2+90*m.s)
		op4.ColorScale.ScaleWithColor(color.NRGBA{100, 100, 120, 255})
		text.Draw(screen, restartMsg, restartFace, op4)
	}
}

func letterColor(letter rune) color.NRGBA {
	switch letter {
	case 'A', 'E', 'I', 'O', 'U':
		return color.NRGBA{190, 65, 55, 240}
	case 'B', 'C', 'D', 'F', 'G':
		return color.NRGBA{50, 85, 170, 240}
	case 'H', 'K', 'L', 'M', 'N':
		return color.NRGBA{40, 140, 125, 240}
	case 'P', 'R', 'S', 'T':
		return color.NRGBA{115, 60, 160, 240}
	case 'V', 'W', 'Y':
		return color.NRGBA{60, 140, 60, 240}
	default:
		return color.NRGBA{80, 80, 80, 240}
	}
}

func clampAdd(v uint8, add uint8) uint8 {
	r := int(v) + int(add)
	if r > 255 {
		return 255
	}
	return uint8(r)
}
