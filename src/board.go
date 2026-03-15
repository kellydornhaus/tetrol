package main

import (
	"log"
	"math/rand"
	"sort"
)

const (
	BoardCols = 7
	BoardRows = 12

	InitialRiseInterval = 1800
	MinRiseInterval     = 600
	RiseSpeedUpEvery    = 5
)

type Board struct {
	Tiles [BoardCols][BoardRows]*Tile

	CursorCol      int
	CursorRow      int
	CursorVertical bool

	RiseTimer    int
	RiseInterval int
	RiseOffset   float64
	RowsPushed   int

	// Word matching
	PendingClears []string  // cleared target word strings — Game picks these up
	PendingPopups []Section // sections just cleared — Game picks these up for floating text
	ChainCount    int       // current chain depth (0 = no chain active)
	settleWait    int

	targets  *TargetManager
	dict     *Dictionary
	scoring  *Scoring
	gameOver bool
}

func NewBoard(dict *Dictionary, scoring *Scoring, targets *TargetManager) *Board {
	b := &Board{
		CursorCol:    BoardCols/2 - 1,
		CursorRow:    BoardRows / 2,
		RiseInterval: InitialRiseInterval,
		dict:         dict,
		scoring:      scoring,
		targets:      targets,
	}

	for row := BoardRows - 6; row < BoardRows; row++ {
		b.generateRow(row)
	}

	return b
}

// generateRow fills a row with tiles: plain frequencies, inject 2 underrepresented
// target letters, then reject-and-retry if any target word is completed.
func (b *Board) generateRow(row int) {
	for attempt := 0; attempt < 100; attempt++ {
		// Step 1: plain frequency letters
		for col := 0; col < BoardCols; col++ {
			b.Tiles[col][row] = NewTile(RandomLetter())
		}

		// Step 2: inject 2 underrepresented target-pool letters
		b.injectUnderrepresented(row)

		// Step 3: reject if any target word is now formed on the board
		if len(FindMatches(b)) == 0 {
			return
		}
	}
	// Fallback: keep last attempt (extremely unlikely to reach here)
}

// injectUnderrepresented finds the 2 least-occurring letters from the visible
// target word pool on the board, and replaces 2 random tiles in the given row.
func (b *Board) injectUnderrepresented(row int) {
	// Collect pool of unique letters from active visible target words
	pool := make(map[rune]bool)
	for _, tw := range b.targets.Visible {
		if tw.State == TargetActive {
			for _, ch := range tw.Word {
				pool[ch] = true
			}
		}
	}
	if len(pool) < 2 {
		return
	}

	// Count occurrences of each pool letter on the entire board
	counts := make(map[rune]int)
	for ch := range pool {
		counts[ch] = 0
	}
	for col := 0; col < BoardCols; col++ {
		for r := 0; r < BoardRows; r++ {
			if t := b.Tiles[col][r]; t != nil && pool[t.Letter] {
				counts[t.Letter]++
			}
		}
	}

	// Sort pool letters by count ascending to find 2 least represented
	type lc struct {
		letter rune
		count  int
	}
	sorted := make([]lc, 0, len(counts))
	for ch, c := range counts {
		sorted = append(sorted, lc{ch, c})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count < sorted[j].count
	})

	// Replace 2 random positions in the row
	positions := rand.Perm(BoardCols)
	for i := 0; i < 2 && i < len(sorted); i++ {
		b.Tiles[positions[i]][row] = NewTile(sorted[i].letter)
	}
}

func (b *Board) MoveCursor(dx, dy int) {
	c := b.CursorCol + dx
	r := b.CursorRow + dy

	maxCol := BoardCols - 1
	maxRow := BoardRows - 1
	if !b.CursorVertical {
		maxCol = BoardCols - 2
	} else {
		maxRow = BoardRows - 2
	}

	if c >= 0 && c <= maxCol {
		b.CursorCol = c
	}
	if r >= 0 && r <= maxRow {
		b.CursorRow = r
	}
}

func (b *Board) ToggleCursorDirection() {
	b.CursorVertical = !b.CursorVertical
	if !b.CursorVertical && b.CursorCol > BoardCols-2 {
		b.CursorCol = BoardCols - 2
	}
	if b.CursorVertical && b.CursorRow > BoardRows-2 {
		b.CursorRow = BoardRows - 2
	}
}

func (b *Board) Swap() {
	var c1, r1, c2, r2 int
	if b.CursorVertical {
		c1, r1 = b.CursorCol, b.CursorRow
		c2, r2 = b.CursorCol, b.CursorRow+1
	} else {
		c1, r1 = b.CursorCol, b.CursorRow
		c2, r2 = b.CursorCol+1, b.CursorRow
	}

	t1 := b.Tiles[c1][r1]
	t2 := b.Tiles[c2][r2]

	if (t1 != nil && (t1.State == TileClearing || t1.State == TileFalling)) ||
		(t2 != nil && (t2.State == TileClearing || t2.State == TileFalling)) {
		return
	}

	b.Tiles[c1][r1] = t2
	b.Tiles[c2][r2] = t1

	if t1 != nil {
		t1.State = TileSwapping
		t1.AnimTick = SwapAnimTicks
		t1.SwapDir = 1
		t1.SwapVertical = b.CursorVertical
	}
	if t2 != nil {
		t2.State = TileSwapping
		t2.AnimTick = SwapAnimTicks
		t2.SwapDir = -1
		t2.SwapVertical = b.CursorVertical
	}
}

func (b *Board) Update() {
	if b.gameOver {
		return
	}

	b.updateAnimations()
	b.applyGravity()

	if b.hasFallingTiles() || b.hasSwappingTiles() || b.hasClearingTiles() {
		return
	}

	if b.settleWait > 0 {
		b.settleWait--
		return
	}

	b.scanAndClear()
	b.updateRise()
}

// scanAndClear finds target words and immediately clears them.
func (b *Board) scanAndClear() {
	matches := FindMatches(b)
	if len(matches) == 0 {
		b.ChainCount = 0
		return
	}

	var foundWords []string
	for _, m := range matches {
		foundWords = append(foundWords, m.Word)
	}
	log.Printf("[scan] FOUND matches: %v (active targets: %v)", foundWords, b.targets.ActiveWords())

	sections := MergeSections(matches)
	b.ChainCount++

	for _, sec := range sections {
		b.scoring.AddSection(sec, b.ChainCount)
	}
	for _, sec := range sections {
		for _, pos := range sec.Tiles {
			if t := b.Tiles[pos[0]][pos[1]]; t != nil {
				t.State = TileClearing
				t.AnimTick = ClearAnimTicks
			}
		}
	}

	// Collect cleared word strings for game.go to pick up
	for _, m := range matches {
		b.PendingClears = append(b.PendingClears, m.Word)
	}
	b.PendingPopups = append(b.PendingPopups, sections...)
	b.settleWait = ClearAnimTicks + 8
}

func (b *Board) updateAnimations() {
	for col := 0; col < BoardCols; col++ {
		for row := 0; row < BoardRows; row++ {
			t := b.Tiles[col][row]
			if t == nil {
				continue
			}
			switch t.State {
			case TileSwapping:
				t.AnimTick--
				if t.AnimTick <= 0 {
					t.State = TileIdle
					t.SwapOffset = 0
				} else {
					progress := 1.0 - float64(t.AnimTick)/float64(SwapAnimTicks)
					t.SwapOffset = float64(-t.SwapDir) * (1.0 - progress)
				}
			case TileClearing:
				t.AnimTick--
				if t.AnimTick <= 0 {
					b.Tiles[col][row] = nil
				}
			}
		}
	}
}

func (b *Board) applyGravity() {
	for col := 0; col < BoardCols; col++ {
		for row := BoardRows - 2; row >= 0; row-- {
			t := b.Tiles[col][row]
			if t == nil || t.State == TileClearing {
				continue
			}
			dest := row
			for r := row + 1; r < BoardRows; r++ {
				if b.Tiles[col][r] == nil {
					dest = r
				} else {
					break
				}
			}
			if dest != row {
				b.Tiles[col][dest] = t
				b.Tiles[col][row] = nil
				t.State = TileFalling
				t.FallOffset = float64(row - dest)
			}
		}
	}

	for col := 0; col < BoardCols; col++ {
		for row := 0; row < BoardRows; row++ {
			t := b.Tiles[col][row]
			if t == nil || t.State != TileFalling {
				continue
			}
			if t.FallOffset < -0.05 {
				t.FallOffset += 0.25
				if t.FallOffset > 0 {
					t.FallOffset = 0
				}
			} else {
				t.FallOffset = 0
				t.State = TileIdle
			}
		}
	}
}

func (b *Board) updateRise() {
	if b.hasClearingTiles() || b.hasFallingTiles() {
		return
	}

	b.RiseTimer++
	b.RiseOffset = float64(b.RiseTimer) / float64(b.RiseInterval)

	if b.RiseTimer >= b.RiseInterval {
		b.pushNewRow()
		b.targets.RevealAll()
		b.RiseTimer = 0
		b.RiseOffset = 0
		b.RowsPushed++

		if b.RowsPushed%RiseSpeedUpEvery == 0 && b.RiseInterval > MinRiseInterval {
			b.RiseInterval -= 30
			if b.RiseInterval < MinRiseInterval {
				b.RiseInterval = MinRiseInterval
			}
		}
	}
}

func (b *Board) pushNewRow() {
	for col := 0; col < BoardCols; col++ {
		if b.Tiles[col][0] != nil {
			b.gameOver = true
			return
		}
	}

	// Shift everything up
	for col := 0; col < BoardCols; col++ {
		for row := 0; row < BoardRows-1; row++ {
			b.Tiles[col][row] = b.Tiles[col][row+1]
		}
		b.Tiles[col][BoardRows-1] = nil
	}

	// Generate the new bottom row using the unified rule
	b.generateRow(BoardRows - 1)

	if b.CursorRow > 0 {
		b.CursorRow--
	}
}

func (b *Board) IsGameOver() bool { return b.gameOver }

func (b *Board) hasFallingTiles() bool {
	for col := 0; col < BoardCols; col++ {
		for row := 0; row < BoardRows; row++ {
			if t := b.Tiles[col][row]; t != nil && t.State == TileFalling {
				return true
			}
		}
	}
	return false
}

func (b *Board) hasSwappingTiles() bool {
	for col := 0; col < BoardCols; col++ {
		for row := 0; row < BoardRows; row++ {
			if t := b.Tiles[col][row]; t != nil && t.State == TileSwapping {
				return true
			}
		}
	}
	return false
}

func (b *Board) hasClearingTiles() bool {
	for col := 0; col < BoardCols; col++ {
		for row := 0; row < BoardRows; row++ {
			if t := b.Tiles[col][row]; t != nil && t.State == TileClearing {
				return true
			}
		}
	}
	return false
}
