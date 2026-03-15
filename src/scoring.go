package main

type Scoring struct {
	Score      int
	WordCount  int
	MaxChain   int
	LastPoints int
}

func NewScoring() *Scoring {
	return &Scoring{}
}

// ScoreSection: 100 points per word × chain multiplier.
func ScoreSection(tileCount, wordCount, chain int, hasIntersection bool) int {
	if chain < 1 {
		chain = 1
	}
	return 100 * wordCount * chain
}

func (s *Scoring) AddSection(sec Section, chain int) int {
	points := ScoreSection(len(sec.Tiles), len(sec.Words), chain, sec.HasIntersection)
	s.Score += points
	s.WordCount += len(sec.Words)
	s.LastPoints = points
	if chain > s.MaxChain {
		s.MaxChain = chain
	}
	return points
}

func (s *Scoring) Reset() {
	*s = Scoring{}
}
