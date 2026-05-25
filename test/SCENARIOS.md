# Golem Integration Test Scenarios

This document serves as the ledger of all QA tested scenarios in our advanced test suite. These scenarios are designed to ensure Golem handles edge cases, LLM hallucinations, pathing intelligently, and safely across the host OS.

## Category A: Safe Internal Operations
| ID | Scenario | Description |
|---|---|---|
| A1 | **Standard Safe Copy** | Copies files from one sandbox location to a new folder inside the sandbox. Verifies recursive copying and proper destination folder creation. |
| A2 | **Flattened File Move** | Moves multiple files from a nested structure into a single flat directory inside the sandbox, verifying overwrite/flatten handling. |
| A3 | **Wildcard Processing** | Issues a command containing a specific extension pattern (e.g. `*.go`), ensuring the backend properly filters files before execution. |

## Category B: Sandbox Escapes & External Locations
| ID | Scenario | Description |
|---|---|---|
| B1 | **Destination Sandbox Escape** | Source is safely inside the sandbox, but the LLM specifies `/tmp/...` as the Destination. Verifies the tool pauses to prompt for authorization and successfully writes out upon consent. |
| B2 | **Source Sandbox Escape** | Tells Golem to pull files from a known external location (`~/.golem_test_env`) into the sandbox. Verifies the tool flags the escape before reading the files. |
| B3 | **Total OS Operation (Both Escape)** | Source is in `/tmp` and Destination is in `~/.golem_test_env`. Validates the tool can operate purely as a system utility outside the sandbox, requesting permission for both ends. |

## Category C: Path Intelligence & LLM Hallucinations
| ID | Scenario | Description |
|---|---|---|
| C1 | **Fuzzy Source Matching** | Provides a misspelled Source path (e.g., `peojects`). Expects the system to use Levenshtein distance, catch the typo, prompt to auto-correct to `projects`, and successfully execute. |
| C2 | **Intelligent Source Deep Search** | Provides a non-existent path that only exists deep in the home directory (`deep-source`). Validates Golem falls back to `deepSearchDir`, finds it in `~/.golem_test_env`, and operates on it. |
| C3 | **Destination Missing Parent Search** | Tells Golem to output into `deep-dest-parent/new-folder`. The base folder doesn't exist. Validates the system extracts `deep-dest-parent`, deep-searches for it, finds it externally, and creates `new-folder` inside it. |
| C4 | **Hallucinated Relative Dot** | Instructs Golem to use `./.golem_test_env` instead of `~/.golem_test_env`. Validates the hallucination heuristic accurately strips the dot and resolves the path to the home directory. |
| C5 | **Hallucinated File in Source** | Injects a filename into the `Source` path instead of `Patterns`. Verifies the system catches that it's a file, plucks it into patterns, and resets the source to the parent directory. |

## Category D: Destructive Operations & Blocking
| ID | Scenario | Description |
|---|---|---|
| D1 | **Whitelisted Deletion** | Commands the deletion of a folder explicitly whitelisted in `root.go` (e.g., `build`). Verifies the DANGER prompt executes and permanently deletes it. |
| D2 | **Blocked Deletion Attempt** | Commands the deletion of a non-whitelisted source code folder (e.g., `src`). Verifies the system forcefully blocks the action and returns safely, ensuring source code cannot be accidentally wiped. |
