package tango

import (
	"errors"
	"sort"
)

type relationKind uint8

const (
	relationSame relationKind = iota + 1
	relationDifferent
)

type indexedRelation struct {
	R    int
	C    int
	Dir  string
	Kind relationKind
	A    int
	B    int
}

type solver struct {
	size         int
	rules        Rules
	relations    []indexedRelation
	byCell       map[int][]indexedRelation
	linePatterns [][]CellValue
	searchNodes  int
}

func AnalyzeDifficulty(p Puzzle) (SolveReport, error) {
	s, board, err := newSolver(p)
	if err != nil {
		return SolveReport{}, err
	}

	report := SolveReport{
		Techniques: map[string]int{},
	}
	ok, changed := s.applyDeterministic(board, report.Techniques)
	if !ok {
		return report, errors.New("puzzle is contradictory")
	}
	for changed {
		ok, changed = s.applyDeterministic(board, report.Techniques)
		if !ok {
			return report, errors.New("puzzle is contradictory")
		}
	}
	if s.isSolved(board) {
		report.Solved = true
		return report, nil
	}

	count, _ := s.countSolutions(cloneCells(board), 1)
	report.Solved = count == 1
	report.SearchNodes = s.searchNodes
	return report, nil
}

func CountSolutions(p Puzzle, limit int) (int, error) {
	s, board, err := newSolver(p)
	if err != nil {
		return 0, err
	}
	count, _ := s.countSolutions(board, limit)
	return count, nil
}

func newSolver(p Puzzle) (*solver, []CellValue, error) {
	if p.Size%2 != 0 || p.Size < 4 {
		return nil, nil, errors.New("size must be an even number >= 4")
	}

	board := make([]CellValue, p.Size*p.Size)
	for _, given := range p.Givens {
		if given.R < 0 || given.R >= p.Size || given.C < 0 || given.C >= p.Size {
			return nil, nil, errors.New("given is out of bounds")
		}
		value, err := ParseCellValue(given.Value)
		if err != nil {
			return nil, nil, err
		}
		board[given.R*p.Size+given.C] = value
	}

	indexed := make([]indexedRelation, 0, len(p.Relations))
	byCell := map[int][]indexedRelation{}
	for _, relation := range p.Relations {
		a := relation.R*p.Size + relation.C
		var b int
		switch relation.Dir {
		case "right":
			if relation.C+1 >= p.Size {
				return nil, nil, errors.New("right relation is out of bounds")
			}
			b = relation.R*p.Size + relation.C + 1
		case "down":
			if relation.R+1 >= p.Size {
				return nil, nil, errors.New("down relation is out of bounds")
			}
			b = (relation.R+1)*p.Size + relation.C
		default:
			return nil, nil, errors.New("unsupported relation direction")
		}

		var kind relationKind
		switch relation.Type {
		case "same":
			kind = relationSame
		case "different":
			kind = relationDifferent
		default:
			return nil, nil, errors.New("unsupported relation type")
		}

		indexedRelation := indexedRelation{
			R:    relation.R,
			C:    relation.C,
			Dir:  relation.Dir,
			Kind: kind,
			A:    a,
			B:    b,
		}
		indexed = append(indexed, indexedRelation)
		byCell[a] = append(byCell[a], indexedRelation)
		byCell[b] = append(byCell[b], indexedRelation)
	}

	return &solver{
		size:         p.Size,
		rules:        p.Rules,
		relations:    indexed,
		byCell:       byCell,
		linePatterns: generateLinePatterns(p.Size),
	}, board, nil
}

func (s *solver) countSolutions(board []CellValue, limit int) (int, []CellValue) {
	ok, changed := s.applyDeterministic(board, nil)
	if !ok {
		return 0, nil
	}
	for changed {
		ok, changed = s.applyDeterministic(board, nil)
		if !ok {
			return 0, nil
		}
	}
	if s.isSolved(board) {
		return 1, cloneCells(board)
	}

	idx, candidates := s.pickCell(board)
	if idx == -1 || len(candidates) == 0 {
		return 0, nil
	}

	count := 0
	var firstSolution []CellValue
	for _, value := range candidates {
		next := cloneCells(board)
		next[idx] = value
		s.searchNodes++
		found, solution := s.countSolutions(next, limit-count)
		if found > 0 && firstSolution == nil {
			firstSolution = solution
		}
		count += found
		if count >= limit {
			break
		}
	}
	return count, firstSolution
}

func (s *solver) pickCell(board []CellValue) (int, []CellValue) {
	bestIdx := -1
	var bestCandidates []CellValue
	for idx, value := range board {
		if value != Unknown {
			continue
		}
		candidates := s.candidates(board, idx)
		if len(candidates) == 0 {
			return idx, nil
		}
		if bestIdx == -1 || len(candidates) < len(bestCandidates) {
			bestIdx = idx
			bestCandidates = candidates
			if len(bestCandidates) == 1 {
				return bestIdx, bestCandidates
			}
		}
	}
	return bestIdx, bestCandidates
}

func (s *solver) candidates(board []CellValue, idx int) []CellValue {
	if board[idx] != Unknown {
		return []CellValue{board[idx]}
	}
	options := make([]CellValue, 0, 2)
	for _, value := range []CellValue{Sun, Moon} {
		board[idx] = value
		if s.isPlacementValid(board, idx) {
			options = append(options, value)
		}
		board[idx] = Unknown
	}
	return options
}

func (s *solver) applyDeterministic(board []CellValue, counters map[string]int) (bool, bool) {
	changed := false
	for {
		progress := false

		ok, applied := s.applyRelations(board, counters)
		if !ok {
			return false, false
		}
		progress = progress || applied

		ok, applied = s.applyTriples(board, counters)
		if !ok {
			return false, false
		}
		progress = progress || applied

		ok, applied = s.applyBalance(board, counters)
		if !ok {
			return false, false
		}
		progress = progress || applied

		ok, applied = s.applyLineCandidates(board, counters)
		if !ok {
			return false, false
		}
		progress = progress || applied

		if !progress {
			return true, changed
		}
		changed = true
	}
}

func (s *solver) applyRelations(board []CellValue, counters map[string]int) (bool, bool) {
	changes := 0
	for _, relation := range s.relations {
		a := board[relation.A]
		b := board[relation.B]
		switch {
		case a != Unknown && b == Unknown:
			value := a
			if relation.Kind == relationDifferent {
				value = Opposite(a)
			}
			ok, changed := s.setCell(board, relation.B, value)
			if !ok {
				return false, false
			}
			if changed {
				changes++
			}
		case a == Unknown && b != Unknown:
			value := b
			if relation.Kind == relationDifferent {
				value = Opposite(b)
			}
			ok, changed := s.setCell(board, relation.A, value)
			if !ok {
				return false, false
			}
			if changed {
				changes++
			}
		case a != Unknown && b != Unknown:
			if !s.relationSatisfied(relation, a, b) {
				return false, false
			}
		}
	}
	if changes > 0 && counters != nil {
		counters["relation"] += changes
	}
	return true, changes > 0
}

func (s *solver) applyTriples(board []CellValue, counters map[string]int) (bool, bool) {
	changes := 0
	for r := 0; r < s.size; r++ {
		for c := 0; c <= s.size-3; c++ {
			positions := []int{
				r*s.size + c,
				r*s.size + c + 1,
				r*s.size + c + 2,
			}
			ok, applied := s.resolveTriple(board, positions)
			if !ok {
				return false, false
			}
			if applied {
				changes++
			}
		}
	}
	for c := 0; c < s.size; c++ {
		for r := 0; r <= s.size-3; r++ {
			positions := []int{
				r*s.size + c,
				(r+1)*s.size + c,
				(r+2)*s.size + c,
			}
			ok, applied := s.resolveTriple(board, positions)
			if !ok {
				return false, false
			}
			if applied {
				changes++
			}
		}
	}
	if changes > 0 && counters != nil {
		counters["triple"] += changes
	}
	return true, changes > 0
}

func (s *solver) resolveTriple(board []CellValue, positions []int) (bool, bool) {
	values := []CellValue{
		board[positions[0]],
		board[positions[1]],
		board[positions[2]],
	}
	unknowns := 0
	unknownIndex := -1
	for i, value := range values {
		if value == Unknown {
			unknowns++
			unknownIndex = i
		}
	}
	if unknowns == 0 {
		if values[0] == values[1] && values[1] == values[2] {
			return false, false
		}
		return true, false
	}
	if unknowns > 1 {
		return true, false
	}

	var forced CellValue
	switch unknownIndex {
	case 0:
		if values[1] != Unknown && values[1] == values[2] {
			forced = Opposite(values[1])
		}
	case 1:
		if values[0] != Unknown && values[0] == values[2] {
			forced = Opposite(values[0])
		}
	case 2:
		if values[0] != Unknown && values[0] == values[1] {
			forced = Opposite(values[0])
		}
	}
	if forced == Unknown {
		return true, false
	}
	return s.setCell(board, positions[unknownIndex], forced)
}

func (s *solver) applyBalance(board []CellValue, counters map[string]int) (bool, bool) {
	changes := 0
	target := s.size / 2
	for r := 0; r < s.size; r++ {
		suns, moons, unknowns := s.lineCounts(board, true, r)
		if suns > target || moons > target {
			return false, false
		}
		if unknowns == 0 {
			if suns != target || moons != target {
				return false, false
			}
			continue
		}
		switch {
		case suns == target:
			for c := 0; c < s.size; c++ {
				ok, changed := s.setCell(board, r*s.size+c, Moon)
				if !ok {
					return false, false
				}
				if changed {
					changes++
				}
			}
		case moons == target:
			for c := 0; c < s.size; c++ {
				ok, changed := s.setCell(board, r*s.size+c, Sun)
				if !ok {
					return false, false
				}
				if changed {
					changes++
				}
			}
		}
	}
	for c := 0; c < s.size; c++ {
		suns, moons, unknowns := s.lineCounts(board, false, c)
		if suns > target || moons > target {
			return false, false
		}
		if unknowns == 0 {
			if suns != target || moons != target {
				return false, false
			}
			continue
		}
		switch {
		case suns == target:
			for r := 0; r < s.size; r++ {
				ok, changed := s.setCell(board, r*s.size+c, Moon)
				if !ok {
					return false, false
				}
				if changed {
					changes++
				}
			}
		case moons == target:
			for r := 0; r < s.size; r++ {
				ok, changed := s.setCell(board, r*s.size+c, Sun)
				if !ok {
					return false, false
				}
				if changed {
					changes++
				}
			}
		}
	}
	if changes > 0 && counters != nil {
		counters["balance"] += changes
	}
	return true, changes > 0
}

func (s *solver) applyLineCandidates(board []CellValue, counters map[string]int) (bool, bool) {
	changes := 0
	for r := 0; r < s.size; r++ {
		candidates := s.rowCandidates(board, r)
		if len(candidates) == 0 {
			return false, false
		}
		for c := 0; c < s.size; c++ {
			idx := r*s.size + c
			if board[idx] != Unknown {
				continue
			}
			value := candidates[0][c]
			consistent := true
			for _, candidate := range candidates[1:] {
				if candidate[c] != value {
					consistent = false
					break
				}
			}
			if consistent {
				ok, changed := s.setCell(board, idx, value)
				if !ok {
					return false, false
				}
				if changed {
					changes++
				}
			}
		}
	}
	for c := 0; c < s.size; c++ {
		candidates := s.colCandidates(board, c)
		if len(candidates) == 0 {
			return false, false
		}
		for r := 0; r < s.size; r++ {
			idx := r*s.size + c
			if board[idx] != Unknown {
				continue
			}
			value := candidates[0][r]
			consistent := true
			for _, candidate := range candidates[1:] {
				if candidate[r] != value {
					consistent = false
					break
				}
			}
			if consistent {
				ok, changed := s.setCell(board, idx, value)
				if !ok {
					return false, false
				}
				if changed {
					changes++
				}
			}
		}
	}
	if changes > 0 && counters != nil {
		counters["line-candidate"] += changes
	}
	return true, changes > 0
}

func (s *solver) rowCandidates(board []CellValue, row int) [][]CellValue {
	candidates := make([][]CellValue, 0, len(s.linePatterns))
	for _, pattern := range s.linePatterns {
		trial := cloneCells(board)
		ok := true
		for c, value := range pattern {
			idx := row*s.size + c
			if trial[idx] != Unknown && trial[idx] != value {
				ok = false
				break
			}
			trial[idx] = value
		}
		if !ok {
			continue
		}
		for c := 0; c < s.size; c++ {
			if !s.isPlacementValid(trial, row*s.size+c) {
				ok = false
				break
			}
		}
		if ok {
			candidates = append(candidates, cloneCells(pattern))
		}
	}
	return candidates
}

func (s *solver) colCandidates(board []CellValue, col int) [][]CellValue {
	candidates := make([][]CellValue, 0, len(s.linePatterns))
	for _, pattern := range s.linePatterns {
		trial := cloneCells(board)
		ok := true
		for r, value := range pattern {
			idx := r*s.size + col
			if trial[idx] != Unknown && trial[idx] != value {
				ok = false
				break
			}
			trial[idx] = value
		}
		if !ok {
			continue
		}
		for r := 0; r < s.size; r++ {
			if !s.isPlacementValid(trial, r*s.size+col) {
				ok = false
				break
			}
		}
		if ok {
			candidates = append(candidates, cloneCells(pattern))
		}
	}
	return candidates
}

func (s *solver) setCell(board []CellValue, idx int, value CellValue) (bool, bool) {
	current := board[idx]
	if current == value {
		return true, false
	}
	if current != Unknown && current != value {
		return false, false
	}
	board[idx] = value
	if !s.isPlacementValid(board, idx) {
		board[idx] = current
		return false, false
	}
	return true, current == Unknown
}

func (s *solver) isSolved(board []CellValue) bool {
	for idx, value := range board {
		if value == Unknown || !s.isPlacementValid(board, idx) {
			return false
		}
	}
	return true
}

func (s *solver) isPlacementValid(board []CellValue, idx int) bool {
	value := board[idx]
	if value == Unknown {
		return true
	}

	r := idx / s.size
	c := idx % s.size
	target := s.size / 2

	rowSuns, rowMoons, rowUnknowns := s.lineCounts(board, true, r)
	colSuns, colMoons, colUnknowns := s.lineCounts(board, false, c)
	if rowSuns > target || rowMoons > target || colSuns > target || colMoons > target {
		return false
	}
	if rowUnknowns == 0 && (rowSuns != target || rowMoons != target) {
		return false
	}
	if colUnknowns == 0 && (colSuns != target || colMoons != target) {
		return false
	}

	if s.hasTriple(board, r, c, true) || s.hasTriple(board, r, c, false) {
		return false
	}

	for _, relation := range s.byCell[idx] {
		other := relation.A
		if other == idx {
			other = relation.B
		}
		if board[other] == Unknown {
			continue
		}
		a := board[relation.A]
		b := board[relation.B]
		if !s.relationSatisfied(relation, a, b) {
			return false
		}
	}

	return true
}

func (s *solver) hasTriple(board []CellValue, r, c int, horizontal bool) bool {
	if horizontal {
		start := max(0, c-2)
		end := min(s.size-3, c)
		for x := start; x <= end; x++ {
			a := board[r*s.size+x]
			b := board[r*s.size+x+1]
			candidate := board[r*s.size+x+2]
			if a != Unknown && a == b && b == candidate {
				return true
			}
		}
		return false
	}

	start := max(0, r-2)
	end := min(s.size-3, r)
	for y := start; y <= end; y++ {
		a := board[y*s.size+c]
		b := board[(y+1)*s.size+c]
		candidate := board[(y+2)*s.size+c]
		if a != Unknown && a == b && b == candidate {
			return true
		}
	}
	return false
}

func (s *solver) relationSatisfied(relation indexedRelation, a, b CellValue) bool {
	switch relation.Kind {
	case relationSame:
		return a == b
	case relationDifferent:
		return a != b
	default:
		return false
	}
}

func (s *solver) lineCounts(board []CellValue, rowMode bool, index int) (int, int, int) {
	suns := 0
	moons := 0
	unknowns := 0
	for i := 0; i < s.size; i++ {
		cellIndex := i*s.size + index
		if rowMode {
			cellIndex = index*s.size + i
		}
		switch board[cellIndex] {
		case Sun:
			suns++
		case Moon:
			moons++
		default:
			unknowns++
		}
	}
	return suns, moons, unknowns
}

func generateLinePatterns(size int) [][]CellValue {
	target := size / 2
	current := make([]CellValue, size)
	patterns := make([][]CellValue, 0)

	var build func(pos, suns, moons int)
	build = func(pos, suns, moons int) {
		if suns > target || moons > target {
			return
		}
		if pos >= 3 && current[pos-1] == current[pos-2] && current[pos-2] == current[pos-3] {
			return
		}
		if pos == size {
			if suns == target && moons == target {
				patterns = append(patterns, cloneCells(current))
			}
			return
		}
		current[pos] = Sun
		build(pos+1, suns+1, moons)
		current[pos] = Moon
		build(pos+1, suns, moons+1)
	}

	build(0, 0, 0)
	sort.Slice(patterns, func(i, j int) bool {
		for x := 0; x < len(patterns[i]); x++ {
			if patterns[i][x] == patterns[j][x] {
				continue
			}
			return patterns[i][x] < patterns[j][x]
		}
		return false
	})
	return patterns
}

func cloneCells(values []CellValue) []CellValue {
	out := make([]CellValue, len(values))
	copy(out, values)
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
