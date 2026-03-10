# Tango

Pure HTML/CSS/JavaScript web version of the Sun and Moon puzzle, plus a Go puzzle generator.

## Puzzle format

The project uses one JSON format across the browser and the generator:

```json
{
  "version": 1,
  "id": "tango-6x6-001",
  "size": 6,
  "rules": {
    "balance": true,
    "maxAdjacent": 2
  },
  "givens": [
    { "r": 0, "c": 1, "value": "sun" }
  ],
  "relations": [
    { "r": 0, "c": 0, "dir": "right", "type": "different" },
    { "r": 1, "c": 3, "dir": "down", "type": "same" }
  ],
  "solution": [
    "MSMSMS",
    "SMMSSM",
    "MSSMMS",
    "SMMSMS",
    "MSMSSM",
    "SMMSMS"
  ],
  "difficulty": {
    "label": "medium",
    "score": 18,
    "techniques": ["relation", "triple", "balance"]
  }
}
```

`givens` store locked cells. `relations` store edge clues between adjacent cells only, using `dir: "right"` or `dir: "down"` so the same clue is never duplicated. `solution` is stored as row strings using `S` and `M` for compactness and easy parsing in both Go and JavaScript.

More detail is in [docs/puzzle-format.md](/Users/babo/Projects/tango/docs/puzzle-format.md).

## Run the web app

Serve the repo root with any static server and open [web/index.html](/Users/babo/Projects/tango/web/index.html).

```bash
python3 -m http.server
```

Then visit [http://localhost:8000/web/](http://localhost:8000/web/).

The browser client reads the bundled puzzle pack from [web/generated-check](/Users/babo/Projects/tango/web/generated-check) through [web/puzzle-manifest.json](/Users/babo/Projects/tango/web/puzzle-manifest.json), and it can still load standalone generator output through the `Load JSON` button.

## Generate puzzles

```bash
go run ./cmd/puzzlegen -count 5 -size 6 -out ./generated
```

Each generated file includes:

- A unique solution.
- A difficulty label based on which solver techniques were required.
- The RNG seed used to create the puzzle.
