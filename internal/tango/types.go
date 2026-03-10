package tango

import (
	"fmt"
	"strings"
)

type CellValue uint8

const (
	Unknown CellValue = iota
	Sun
	Moon
)

type Rules struct {
	Balance     bool `json:"balance"`
	MaxAdjacent int  `json:"maxAdjacent"`
}

type Given struct {
	R     int    `json:"r"`
	C     int    `json:"c"`
	Value string `json:"value"`
}

type Relation struct {
	R    int    `json:"r"`
	C    int    `json:"c"`
	Dir  string `json:"dir"`
	Type string `json:"type"`
}

type Difficulty struct {
	Label       string   `json:"label"`
	Score       int      `json:"score"`
	Techniques  []string `json:"techniques,omitempty"`
	SearchNodes int      `json:"searchNodes,omitempty"`
	Seed        int64    `json:"seed,omitempty"`
}

type Puzzle struct {
	Version    int        `json:"version"`
	ID         string     `json:"id"`
	Size       int        `json:"size"`
	Rules      Rules      `json:"rules"`
	Givens     []Given    `json:"givens"`
	Relations  []Relation `json:"relations"`
	Solution   []string   `json:"solution,omitempty"`
	Difficulty Difficulty `json:"difficulty"`
}

type SolveReport struct {
	Solved      bool
	Techniques  map[string]int
	SearchNodes int
}

func DefaultRules() Rules {
	return Rules{
		Balance:     true,
		MaxAdjacent: 2,
	}
}

func Opposite(v CellValue) CellValue {
	switch v {
	case Sun:
		return Moon
	case Moon:
		return Sun
	default:
		return Unknown
	}
}

func (v CellValue) Symbol() string {
	switch v {
	case Sun:
		return "S"
	case Moon:
		return "M"
	default:
		return "."
	}
}

func (v CellValue) Name() string {
	switch v {
	case Sun:
		return "sun"
	case Moon:
		return "moon"
	default:
		return "unknown"
	}
}

func ParseCellValue(raw string) (CellValue, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "s", "sun":
		return Sun, nil
	case "m", "moon":
		return Moon, nil
	case ".", "", "unknown":
		return Unknown, nil
	default:
		return Unknown, fmt.Errorf("unsupported cell value %q", raw)
	}
}

func EncodeBoard(board [][]CellValue) []string {
	lines := make([]string, len(board))
	for r := range board {
		var b strings.Builder
		b.Grow(len(board[r]))
		for _, cell := range board[r] {
			b.WriteString(cell.Symbol())
		}
		lines[r] = b.String()
	}
	return lines
}

func DecodeBoard(lines []string) ([][]CellValue, error) {
	board := make([][]CellValue, len(lines))
	for r, line := range lines {
		board[r] = make([]CellValue, len(line))
		for c, ch := range line {
			value, err := ParseCellValue(string(ch))
			if err != nil {
				return nil, err
			}
			board[r][c] = value
		}
	}
	return board, nil
}
