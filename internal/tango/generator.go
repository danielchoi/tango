package tango

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
)

type Generator struct {
	size int
	rng  *rand.Rand
	seed int64
}

type clueKind uint8

const (
	clueGiven clueKind = iota + 1
	clueRelation
)

type clue struct {
	kind     clueKind
	given    Given
	relation Relation
}

func NewGenerator(size int, seed int64) *Generator {
	return &Generator{
		size: size,
		rng:  rand.New(rand.NewSource(seed)),
		seed: seed,
	}
}

func (g *Generator) Generate(id string) (Puzzle, error) {
	solution, err := g.generateSolution()
	if err != nil {
		return Puzzle{}, err
	}

	pool := g.buildCluePool(solution)
	active := make([]bool, len(pool))
	for i := range active {
		active[i] = true
	}

	order := g.rng.Perm(len(pool))
	for _, idx := range order {
		active[idx] = false
		candidate := g.buildPuzzle(id, solution, pool, active)
		count, err := CountSolutions(candidate, 2)
		if err != nil || count != 1 {
			active[idx] = true
		}
	}

	puzzle := g.buildPuzzle(id, solution, pool, active)
	report, err := AnalyzeDifficulty(puzzle)
	if err != nil {
		return Puzzle{}, err
	}
	puzzle.Difficulty = classifyDifficulty(report, g.seed)
	return puzzle, nil
}

func WritePuzzle(path string, puzzle Puzzle) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(puzzle, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func (g *Generator) generateSolution() ([][]CellValue, error) {
	board := make([][]CellValue, g.size)
	for r := range board {
		board[r] = make([]CellValue, g.size)
	}
	rowSums := make([]int, g.size)
	colSums := make([]int, g.size)
	target := g.size / 2

	var fill func(pos int) bool
	fill = func(pos int) bool {
		if pos == g.size*g.size {
			return true
		}
		r := pos / g.size
		c := pos % g.size
		values := []CellValue{Sun, Moon}
		if g.rng.Intn(2) == 0 {
			slices.Reverse(values)
		}
		for _, value := range values {
			rowCount := rowSums[r]
			colCount := colSums[c]
			if value == Moon {
				rowCount = (c + 1) - rowCount
				colCount = (r + 1) - colCount
			}
			if rowCount > target || colCount > target {
				continue
			}

			board[r][c] = value
			if value == Sun {
				rowSums[r]++
				colSums[c]++
			}

			if g.partialBoardValid(board, r, c, rowSums, colSums, target) && fill(pos+1) {
				return true
			}

			if value == Sun {
				rowSums[r]--
				colSums[c]--
			}
			board[r][c] = Unknown
		}
		return false
	}

	if !fill(0) {
		return nil, fmt.Errorf("failed to generate solution for %dx%d", g.size, g.size)
	}
	return board, nil
}

func (g *Generator) partialBoardValid(board [][]CellValue, r, c int, rowSums, colSums []int, target int) bool {
	if c >= 2 {
		a := board[r][c]
		if a != Unknown && a == board[r][c-1] && a == board[r][c-2] {
			return false
		}
	}
	if r >= 2 {
		a := board[r][c]
		if a != Unknown && a == board[r-1][c] && a == board[r-2][c] {
			return false
		}
	}

	if c == g.size-1 {
		rowSun := rowSums[r]
		rowMoon := g.size - rowSun
		if rowSun != target || rowMoon != target {
			return false
		}
	}
	if r == g.size-1 {
		colSun := colSums[c]
		colMoon := g.size - colSun
		if colSun != target || colMoon != target {
			return false
		}
	}

	return true
}

func (g *Generator) buildCluePool(solution [][]CellValue) []clue {
	pool := make([]clue, 0, g.size*g.size*3)
	for r := 0; r < g.size; r++ {
		for c := 0; c < g.size; c++ {
			pool = append(pool, clue{
				kind: clueGiven,
				given: Given{
					R:     r,
					C:     c,
					Value: solution[r][c].Name(),
				},
			})

			if c+1 < g.size {
				pool = append(pool, clue{
					kind: clueRelation,
					relation: Relation{
						R:    r,
						C:    c,
						Dir:  "right",
						Type: relationTypeName(solution[r][c], solution[r][c+1]),
					},
				})
			}
			if r+1 < g.size {
				pool = append(pool, clue{
					kind: clueRelation,
					relation: Relation{
						R:    r,
						C:    c,
						Dir:  "down",
						Type: relationTypeName(solution[r][c], solution[r+1][c]),
					},
				})
			}
		}
	}
	return pool
}

func (g *Generator) buildPuzzle(id string, solution [][]CellValue, pool []clue, active []bool) Puzzle {
	givens := make([]Given, 0)
	relations := make([]Relation, 0)
	for i, enabled := range active {
		if !enabled {
			continue
		}
		switch pool[i].kind {
		case clueGiven:
			givens = append(givens, pool[i].given)
		case clueRelation:
			relations = append(relations, pool[i].relation)
		}
	}
	return Puzzle{
		Version:   1,
		ID:        id,
		Size:      g.size,
		Rules:     DefaultRules(),
		Givens:    givens,
		Relations: relations,
		Solution:  EncodeBoard(solution),
	}
}

func classifyDifficulty(report SolveReport, seed int64) Difficulty {
	techniques := make([]string, 0, len(report.Techniques))
	score := 0
	for name, count := range report.Techniques {
		switch name {
		case "relation":
			score += count
		case "triple":
			score += count * 2
		case "balance":
			score += count * 2
		case "line-candidate":
			score += count * 5
		}
		techniques = append(techniques, name)
	}
	slices.Sort(techniques)

	label := "easy"
	switch {
	case report.SearchNodes > 0:
		label = "hard"
		score += 100 + min(report.SearchNodes, 50)
	case report.Techniques["line-candidate"] > 0 || score > 25:
		label = "medium"
	}

	return Difficulty{
		Label:       label,
		Score:       score,
		Techniques:  techniques,
		SearchNodes: report.SearchNodes,
		Seed:        seed,
	}
}

func relationTypeName(a, b CellValue) string {
	if a == b {
		return "same"
	}
	return "different"
}
