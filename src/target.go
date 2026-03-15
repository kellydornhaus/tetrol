package main

import (
	"log"
	"math/rand"
)

type TargetState int

const (
	TargetActive   TargetState = iota
	TargetClearing             // flash/fade animation
	TargetGone
)

const TargetClearTicks = 40

type TargetWord struct {
	Word     string
	Jumbled  string
	State    TargetState
	AnimTick int
}

type TargetManager struct {
	AllWords [10]string
	Visible  []*TargetWord // up to 3 active
	NextIdx  int           // next word index from AllWords to add
	Cleared  int
}

func NewTargetManager(dict *Dictionary) *TargetManager {
	words := dict.RandomWords(10)
	log.Printf("Target words: %v", words)
	tm := &TargetManager{}
	for i := 0; i < 10 && i < len(words); i++ {
		tm.AllWords[i] = words[i]
	}
	// Start with first 3 visible
	for i := 0; i < 3 && i < 10; i++ {
		tm.Visible = append(tm.Visible, &TargetWord{
			Word:    tm.AllWords[i],
			Jumbled: jumble(tm.AllWords[i]),
			State:   TargetActive,
		})
	}
	tm.NextIdx = 3
	return tm
}

func jumble(word string) string {
	runes := []rune(word)
	for {
		// Fisher-Yates shuffle
		for i := len(runes) - 1; i > 0; i-- {
			j := rand.Intn(i + 1)
			runes[i], runes[j] = runes[j], runes[i]
		}
		if string(runes) != word {
			return string(runes)
		}
	}
}

// ActiveWords returns the words currently on screen that are still active (not clearing/gone).
func (tm *TargetManager) ActiveWords() []string {
	var out []string
	for _, tw := range tm.Visible {
		if tw.State == TargetActive {
			out = append(out, tw.Word)
		}
	}
	return out
}

// IsTarget returns true if word is one of the currently active visible targets.
func (tm *TargetManager) IsTarget(word string) bool {
	for _, tw := range tm.Visible {
		if tw.State == TargetActive && tw.Word == word {
			return true
		}
	}
	return false
}

// MarkCleared starts the clearing animation for the given target word.
func (tm *TargetManager) MarkCleared(word string) {
	for _, tw := range tm.Visible {
		if tw.State == TargetActive && tw.Word == word {
			tw.State = TargetClearing
			tw.AnimTick = TargetClearTicks
			tm.Cleared++
			log.Printf("[target] MarkCleared: %s (%d/10 cleared)", word, tm.Cleared)
			return
		}
	}
	log.Printf("[target] MarkCleared: %s NOT FOUND in visible (visible: %v)", word, tm.ActiveWords())
}

// Update ticks clearing animations; when done, removes the word and adds the next from the queue.
func (tm *TargetManager) Update() {
	for i := 0; i < len(tm.Visible); i++ {
		tw := tm.Visible[i]
		if tw.State == TargetClearing {
			tw.AnimTick--
			if tw.AnimTick <= 0 {
				tw.State = TargetGone
				// Remove this entry, slide remaining up
				tm.Visible = append(tm.Visible[:i], tm.Visible[i+1:]...)
				i-- // re-check this index

				// Add next word from queue if available
				if tm.NextIdx < 10 {
					tm.Visible = append(tm.Visible, &TargetWord{
						Word:    tm.AllWords[tm.NextIdx],
						Jumbled: jumble(tm.AllWords[tm.NextIdx]),
						State:   TargetActive,
					})
					tm.NextIdx++
				}
			}
		}
	}
}

// RevealAll does one adjacent swap in each active jumbled word,
// moving each one step closer to the actual word.
func (tm *TargetManager) RevealAll() {
	for _, tw := range tm.Visible {
		if tw.State == TargetActive && tw.Jumbled != tw.Word {
			tw.revealStep()
		}
	}
}

// revealStep finds the leftmost wrong letter and swaps an adjacent pair
// to move the correct letter one position closer.
func (tw *TargetWord) revealStep() {
	runes := []rune(tw.Jumbled)
	target := []rune(tw.Word)

	for i := 0; i < len(runes); i++ {
		if runes[i] != target[i] {
			// Find where target[i] currently sits
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == target[i] {
					// Adjacent swap: move it one step left
					runes[j], runes[j-1] = runes[j-1], runes[j]
					tw.Jumbled = string(runes)
					return
				}
			}
		}
	}
}

// AllDone returns true when all 10 target words have been cleared.
func (tm *TargetManager) AllDone() bool {
	return tm.Cleared >= 10
}
