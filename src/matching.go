package main

type Match struct {
	Word       string
	Positions  [][2]int // [col, row] pairs
	Horizontal bool
}

// Section is a connected region of matched tiles, possibly containing multiple words.
type Section struct {
	Words           []string
	Tiles           [][2]int
	tileSet         map[[2]int]bool
	ColorIdx        int
	HasIntersection bool // true if any tile is shared by 2+ words
}

func tileAvailable(t *Tile) bool {
	return t != nil && (t.State == TileIdle || t.State == TileMatched)
}

// FindMatches scans the board for active target words only.
func FindMatches(b *Board) []Match {
	if b.targets == nil {
		return nil
	}

	targetWords := b.targets.ActiveWords()
	if len(targetWords) == 0 {
		return nil
	}

	var matches []Match

	for _, word := range targetWords {
		wordLen := len(word)
		runes := []rune(word)

		// Horizontal scan
		for row := 0; row < BoardRows; row++ {
			for startCol := 0; startCol <= BoardCols-wordLen; startCol++ {
				ok := true
				positions := make([][2]int, wordLen)
				for i := 0; i < wordLen; i++ {
					t := b.Tiles[startCol+i][row]
					if !tileAvailable(t) || t.Letter != runes[i] {
						ok = false
						break
					}
					positions[i] = [2]int{startCol + i, row}
				}
				if ok {
					matches = append(matches, Match{Word: word, Positions: positions, Horizontal: true})
				}
			}
		}

		// Vertical scan
		for col := 0; col < BoardCols; col++ {
			for startRow := 0; startRow <= BoardRows-wordLen; startRow++ {
				ok := true
				positions := make([][2]int, wordLen)
				for i := 0; i < wordLen; i++ {
					t := b.Tiles[col][startRow+i]
					if !tileAvailable(t) || t.Letter != runes[i] {
						ok = false
						break
					}
					positions[i] = [2]int{col, startRow + i}
				}
				if ok {
					matches = append(matches, Match{Word: word, Positions: positions, Horizontal: false})
				}
			}
		}
	}

	return matches
}

var neighbors = [4][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}

// MergeSections groups matches into connected components via tile adjacency.
// Two words are in the same section if their tiles touch (4-directional) or overlap.
func MergeSections(matches []Match) []Section {
	if len(matches) == 0 {
		return nil
	}

	// Collect all matched tile positions
	allTiles := map[[2]int]bool{}
	for _, m := range matches {
		for _, p := range m.Positions {
			allTiles[p] = true
		}
	}

	// Flood-fill connected components of matched tiles
	visited := map[[2]int]bool{}
	var components []map[[2]int]bool

	for tile := range allTiles {
		if visited[tile] {
			continue
		}
		comp := map[[2]int]bool{}
		queue := [][2]int{tile}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if visited[cur] {
				continue
			}
			visited[cur] = true
			comp[cur] = true
			for _, d := range neighbors {
				n := [2]int{cur[0] + d[0], cur[1] + d[1]}
				if allTiles[n] && !visited[n] {
					queue = append(queue, n)
				}
			}
		}
		components = append(components, comp)
	}

	// Map each component to its words and detect intersection
	var sections []Section
	for ci, comp := range components {
		var words []string
		tileWordCount := map[[2]int]int{}

		for _, m := range matches {
			belongs := false
			for _, p := range m.Positions {
				if comp[p] {
					belongs = true
					break
				}
			}
			if belongs {
				words = append(words, m.Word)
				for _, p := range m.Positions {
					tileWordCount[p]++
				}
			}
		}

		hasIntersection := false
		for _, c := range tileWordCount {
			if c >= 2 {
				hasIntersection = true
				break
			}
		}

		tiles := make([][2]int, 0, len(comp))
		for p := range comp {
			tiles = append(tiles, p)
		}

		sections = append(sections, Section{
			Words:           words,
			Tiles:           tiles,
			tileSet:         comp,
			ColorIdx:        ci,
			HasIntersection: hasIntersection,
		})
	}

	return sections
}
