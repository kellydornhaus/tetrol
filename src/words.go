package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

const (
	MinWordLen = 5
	MaxWordLen = 7
)

type Dictionary struct {
	words     map[string]bool
	targetsByLen [8][]string // index by word length (5, 6, 7)
}

func LoadDictionary() *Dictionary {
	d := &Dictionary{words: make(map[string]bool)}

	path := filepath.Join("assets", "dictionary.json")
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: cannot load dictionary: %v", err)
		return d
	}

	var wordList []string
	if err := json.Unmarshal(data, &wordList); err != nil {
		log.Printf("Warning: cannot parse dictionary: %v", err)
		return d
	}

	for _, word := range wordList {
		word = strings.ToUpper(strings.TrimSpace(word))
		n := len(word)
		if n >= MinWordLen && n <= MaxWordLen {
			d.words[word] = true
		}
	}

	log.Printf("Loaded %d words (%d-%d letters)", len(d.words), MinWordLen, MaxWordLen)

	// Load common target words, bucketed by length
	targetsPath := filepath.Join("assets", "targets.json")
	tdata, err := os.ReadFile(targetsPath)
	if err == nil {
		var tlist []string
		if err := json.Unmarshal(tdata, &tlist); err == nil {
			for _, w := range tlist {
				w = strings.ToUpper(strings.TrimSpace(w))
				n := len(w)
				if n >= MinWordLen && n <= MaxWordLen && d.words[w] {
					d.targetsByLen[n] = append(d.targetsByLen[n], w)
				}
			}
		}
	}
	for n := MinWordLen; n <= MaxWordLen; n++ {
		log.Printf("Loaded %d common %d-letter target words", len(d.targetsByLen[n]), n)
	}

	return d
}

func (d *Dictionary) IsValidWord(word string) bool {
	return d.words[strings.ToUpper(word)]
}

func (d *Dictionary) RandomWords(_ int) []string {
	// Fixed distribution: 5 fives, 3 sixes, 2 sevens
	quota := []struct{ length, count int }{
		{5, 5}, {6, 3}, {7, 2},
	}

	used := make(map[string]bool)
	var result []string

	for _, q := range quota {
		source := d.targetsByLen[q.length]
		if len(source) == 0 {
			continue
		}
		for picked := 0; picked < q.count; {
			w := source[rand.Intn(len(source))]
			if used[w] {
				continue
			}
			used[w] = true
			result = append(result, w)
			picked++
		}
	}

	// Shuffle so the order isn't always 5s then 6s then 7s
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})

	return result
}
