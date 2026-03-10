package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"tango/internal/tango"
)

func main() {
	count := flag.Int("count", 1, "number of puzzles to generate")
	size := flag.Int("size", 6, "board size; must be even")
	outDir := flag.String("out", "generated", "output directory")
	seed := flag.Int64("seed", time.Now().UnixNano(), "base RNG seed")
	flag.Parse()

	if *count < 1 {
		panic("count must be >= 1")
	}
	if *size < 4 || *size%2 != 0 {
		panic("size must be an even number >= 4")
	}

	for i := 0; i < *count; i++ {
		puzzleSeed := *seed + int64(i*7919)
		id := fmt.Sprintf("tango-%dx%d-%03d", *size, *size, i+1)
		generator := tango.NewGenerator(*size, puzzleSeed)
		puzzle, err := generator.Generate(id)
		if err != nil {
			panic(err)
		}

		path := filepath.Join(*outDir, id+".json")
		if err := tango.WritePuzzle(path, puzzle); err != nil {
			panic(err)
		}

		fmt.Printf("%s difficulty=%s score=%d relations=%d givens=%d\n",
			path,
			puzzle.Difficulty.Label,
			puzzle.Difficulty.Score,
			len(puzzle.Relations),
			len(puzzle.Givens),
		)
	}
}
