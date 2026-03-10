const SYMBOLS = {
  0: "",
  1: "S",
  2: "M",
};

const VALUE_FROM_NAME = {
  sun: 1,
  moon: 2,
  S: 1,
  M: 2,
};

let puzzle = null;
let state = [];
let givenLookup = new Map();

const boardEl = document.getElementById("board");
const statusTextEl = document.getElementById("status-text");
const difficultyEl = document.getElementById("difficulty");
const puzzleIdEl = document.getElementById("puzzle-id");
const messageEl = document.getElementById("message");
const issuesEl = document.getElementById("issues");
const puzzlePickerEl = document.getElementById("puzzle-picker");

puzzlePickerEl.addEventListener("change", async (event) => {
  const fileName = event.target.value;
  if (!fileName) {
    return;
  }
  await loadPackedPuzzle(fileName);
});

document.getElementById("reset-btn").addEventListener("click", () => {
  resetState();
  renderBoard();
  showMessage("Puzzle reset.");
});

document.getElementById("check-btn").addEventListener("click", () => {
  renderBoard();
});

document.getElementById("reveal-btn").addEventListener("click", () => {
  state = decodeSolution(puzzle.solution);
  renderBoard();
  showMessage("Solution revealed.", "ok");
});

document.getElementById("file-input").addEventListener("change", async (event) => {
  const file = event.target.files?.[0];
  if (!file) {
    return;
  }
  const raw = await file.text();
  loadPuzzle(JSON.parse(raw));
  puzzlePickerEl.value = "";
  event.target.value = "";
});

function loadPuzzle(nextPuzzle) {
  puzzle = normalizePuzzle(nextPuzzle);
  givenLookup = new Map(
    puzzle.givens.map((given) => [keyFor(given.r, given.c), given.value]),
  );
  resetState();
  puzzleIdEl.textContent = puzzle.id;
  difficultyEl.textContent = puzzle.difficulty?.label ?? "-";
  showMessage("Puzzle loaded.");
  renderBoard();
}

async function loadPackedPuzzle(fileName) {
  const nextPuzzle = window.TANGO_GENERATED_PACK?.puzzles?.[fileName];
  if (!nextPuzzle) {
    showMessage(`Failed to load ${fileName}.`, "error");
    return;
  }
  showMessage("Loading puzzle...");
  loadPuzzle(nextPuzzle);
}

async function loadPuzzleManifest() {
  const packedPuzzleFiles = window.TANGO_GENERATED_PACK?.files ?? [];
  if (packedPuzzleFiles.length > 0) {
    populatePuzzlePicker(packedPuzzleFiles);
    puzzlePickerEl.value = packedPuzzleFiles[0];
    await loadPackedPuzzle(packedPuzzleFiles[0]);
    return;
  }
  puzzlePickerEl.innerHTML = '<option value="">No bundled puzzles</option>';
  showMessage("Using embedded sample puzzle.", "error");
  loadPuzzle(window.TANGO_SAMPLE_PUZZLE);
}

function populatePuzzlePicker(files) {
  puzzlePickerEl.innerHTML = "";

  const placeholder = document.createElement("option");
  placeholder.value = "";
  placeholder.textContent = "Select a puzzle";
  puzzlePickerEl.appendChild(placeholder);

  for (const fileName of files) {
    const option = document.createElement("option");
    option.value = fileName;
    option.textContent = fileName.replace(".json", "");
    puzzlePickerEl.appendChild(option);
  }
}

function normalizePuzzle(raw) {
  return {
    ...raw,
    givens: raw.givens.map((given) => ({
      ...given,
      value: typeof given.value === "string" ? given.value.toLowerCase() : given.value,
    })),
  };
}

function resetState() {
  state = new Array(puzzle.size * puzzle.size).fill(0);
  for (const given of puzzle.givens) {
    state[indexFor(given.r, given.c)] = VALUE_FROM_NAME[given.value];
  }
}

function renderBoard() {
  boardEl.innerHTML = "";
  boardEl.style.gridTemplateColumns = Array.from({ length: puzzle.size * 2 - 1 }, (_, index) =>
    index % 2 === 0 ? "var(--cell-size)" : "var(--clue-size)",
  ).join(" ");
  boardEl.style.gridTemplateRows = Array.from({ length: puzzle.size * 2 - 1 }, (_, index) =>
    index % 2 === 0 ? "var(--cell-size)" : "var(--clue-size)",
  ).join(" ");

  const validation = validateBoard(state, false);
  const errorKeys = new Set(validation.errorCells.map(([r, c]) => keyFor(r, c)));

  for (let r = 0; r < puzzle.size; r += 1) {
    for (let c = 0; c < puzzle.size; c += 1) {
      const cell = document.createElement("button");
      const value = state[indexFor(r, c)];
      const given = givenLookup.has(keyFor(r, c));
      cell.type = "button";
      cell.className = "cell";
      if (value === 1) {
        cell.classList.add("sun");
      }
      if (value === 2) {
        cell.classList.add("moon");
      }
      if (given) {
        cell.classList.add("given");
      }
      if (errorKeys.has(keyFor(r, c))) {
        cell.classList.add("error");
      }
      cell.textContent = SYMBOLS[value];
      cell.style.gridColumn = String(c * 2 + 1);
      cell.style.gridRow = String(r * 2 + 1);
      cell.setAttribute("aria-label", `Row ${r + 1} column ${c + 1}`);
      cell.addEventListener("click", () => {
        if (given) {
          return;
        }
        state[indexFor(r, c)] = (state[indexFor(r, c)] + 1) % 3;
        renderBoard();
      });
      boardEl.appendChild(cell);
    }
  }

  for (const relation of puzzle.relations) {
    const clue = document.createElement("div");
    clue.className = "relation";
    clue.textContent = relation.type === "same" ? "=" : "x";
    if (relation.dir === "right") {
      clue.style.gridColumn = String(relation.c * 2 + 2);
      clue.style.gridRow = String(relation.r * 2 + 1);
    } else {
      clue.classList.add("vertical");
      clue.style.gridColumn = String(relation.c * 2 + 1);
      clue.style.gridRow = String(relation.r * 2 + 2);
    }
    boardEl.appendChild(clue);
  }

  updateStatus(validation);
}

function validateBoard(boardState, includeIncomplete) {
  const errors = [];
  const errorCells = [];
  const size = puzzle.size;
  const maxCount = size / 2;

  for (let r = 0; r < size; r += 1) {
    const cells = [];
    for (let c = 0; c < size; c += 1) {
      cells.push(boardState[indexFor(r, c)]);
    }
    const suns = cells.filter((value) => value === 1).length;
    const moons = cells.filter((value) => value === 2).length;
    const blanks = cells.filter((value) => value === 0).length;
    if (suns > maxCount || moons > maxCount) {
      errors.push(`Row ${r + 1} has too many of one symbol.`);
      for (let c = 0; c < size; c += 1) {
        errorCells.push([r, c]);
      }
    }
    if (includeIncomplete && blanks === 0 && (suns !== maxCount || moons !== maxCount)) {
      errors.push(`Row ${r + 1} is not balanced.`);
    }
    for (let c = 0; c <= size - 3; c += 1) {
      if (cells[c] !== 0 && cells[c] === cells[c + 1] && cells[c + 1] === cells[c + 2]) {
        errors.push(`Row ${r + 1} contains three matching symbols in a row.`);
        errorCells.push([r, c], [r, c + 1], [r, c + 2]);
      }
    }
  }

  for (let c = 0; c < size; c += 1) {
    const cells = [];
    for (let r = 0; r < size; r += 1) {
      cells.push(boardState[indexFor(r, c)]);
    }
    const suns = cells.filter((value) => value === 1).length;
    const moons = cells.filter((value) => value === 2).length;
    const blanks = cells.filter((value) => value === 0).length;
    if (suns > maxCount || moons > maxCount) {
      errors.push(`Column ${c + 1} has too many of one symbol.`);
      for (let r = 0; r < size; r += 1) {
        errorCells.push([r, c]);
      }
    }
    if (includeIncomplete && blanks === 0 && (suns !== maxCount || moons !== maxCount)) {
      errors.push(`Column ${c + 1} is not balanced.`);
    }
    for (let r = 0; r <= size - 3; r += 1) {
      if (cells[r] !== 0 && cells[r] === cells[r + 1] && cells[r + 1] === cells[r + 2]) {
        errors.push(`Column ${c + 1} contains three matching symbols in a row.`);
        errorCells.push([r, c], [r + 1, c], [r + 2, c]);
      }
    }
  }

  for (const relation of puzzle.relations) {
    const a = boardState[indexFor(relation.r, relation.c)];
    const bIndex = relation.dir === "right"
      ? indexFor(relation.r, relation.c + 1)
      : indexFor(relation.r + 1, relation.c);
    const b = boardState[bIndex];
    if (a === 0 || b === 0) {
      continue;
    }
    const valid = relation.type === "same" ? a === b : a !== b;
    if (!valid) {
      errors.push(
        `The ${relation.type === "same" ? "=" : "x"} clue at row ${relation.r + 1}, column ${relation.c + 1} is broken.`,
      );
      errorCells.push([relation.r, relation.c]);
      errorCells.push(
        relation.dir === "right"
          ? [relation.r, relation.c + 1]
          : [relation.r + 1, relation.c],
      );
    }
  }

  const complete = boardState.every((value) => value !== 0);
  const solved = complete && (
    includeIncomplete
      ? errors.length === 0
      : validateBoard(boardState, true).errors.length === 0
  );

  return {
    errors: dedupe(errors),
    errorCells: dedupePairs(errorCells),
    complete,
    solved,
  };
}

function updateStatus(validation) {
  issuesEl.innerHTML = "";
  if (validation.solved) {
    statusTextEl.textContent = "Solved";
    showMessage("Puzzle solved.", "ok");
    return;
  }
  statusTextEl.textContent = validation.errors.length > 0 ? "Has conflicts" : "In progress";
  if (validation.errors.length === 0) {
    showMessage("No rule conflicts detected.");
    return;
  }
  showMessage("There are rule conflicts on the board.", "error");
  for (const error of validation.errors) {
    const item = document.createElement("li");
    item.textContent = error;
    issuesEl.appendChild(item);
  }
}

function showMessage(text, tone = "") {
  messageEl.textContent = text;
  messageEl.className = tone;
}

function indexFor(r, c) {
  return r * puzzle.size + c;
}

function keyFor(r, c) {
  return `${r}:${c}`;
}

function decodeSolution(lines) {
  return lines.flatMap((line) => [...line].map((value) => VALUE_FROM_NAME[value]));
}

function dedupe(values) {
  return [...new Set(values)];
}

function dedupePairs(values) {
  const seen = new Set();
  return values.filter(([r, c]) => {
    const key = keyFor(r, c);
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}

loadPuzzleManifest();
