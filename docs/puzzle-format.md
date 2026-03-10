# Puzzle Format

## Storage goals

The format needs to satisfy four consumers at once:

1. The browser needs a compact, render-friendly format.
2. The Go generator needs a lossless representation of every clue.
3. Difficulty analysis needs access to the full solution.
4. Future save-data should be able to reference a stable puzzle id without mutating the source puzzle.

## Chosen structure

Use a single JSON object per puzzle.

- `version`: schema version for forward compatibility.
- `id`: stable external identifier.
- `size`: board width and height. The current game assumes a square board.
- `rules`: explicit rule block so variants can reuse the format later.
- `givens`: fixed cell values.
- `relations`: edge clues between neighboring cells.
- `solution`: canonical solved board as row strings using `S` and `M`.
- `difficulty`: generator output, not player state.

## Why relations are stored as edges

Each clue only relates two adjacent cells. Storing a relation as:

```json
{ "r": 2, "c": 3, "dir": "right", "type": "same" }
```

means:

- The clue starts at `(2, 3)`.
- It points to the cell on the right.
- The two cells must match.

This is better than storing two full coordinate pairs because:

- It is smaller on disk.
- It avoids duplicate edge definitions.
- It maps directly to the browser layout because clues sit between two cells.

## Why the solution is stored as strings

`solution` is stored as strings like `MSMSMS` instead of nested objects because:

- It is easy to diff.
- It is compact.
- JavaScript can split a row into characters immediately.
- Go can validate and marshal it with minimal overhead.

## Separate save data later

Puzzle data should stay immutable. If save/load is added later, store it separately:

```json
{
  "puzzleId": "tango-6x6-001",
  "state": ["..SM..", "M...S.", "..."],
  "updatedAt": "2026-03-10T12:00:00Z"
}
```

That keeps authored/generated puzzle data clean while allowing player progress to evolve independently.
