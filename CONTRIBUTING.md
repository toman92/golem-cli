# Contributing to Golem

First off, thank you for considering contributing to Golem! It's people like you that make Golem such a great tool.

## Code of Conduct

By participating in this project, you are expected to uphold our standards of conduct. Please be welcoming, inclusive, and respectful to all contributors.

## How Can I Contribute?

### Reporting Bugs
If you find a bug, please open an issue describing the bug. Please include:
- Your operating system and version.
- The version of Go you are using.
- A detailed description of the bug and how to reproduce it.
- Any relevant logs (usually found in `~/.golem.log`).

### Suggesting Enhancements
Enhancements and feature requests are welcome! When opening a feature request, please provide:
- A clear and descriptive title.
- A detailed description of the proposed feature.
- Why this feature would be useful.

### Pull Requests
1. Fork the repository and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs or features, update the documentation (`README.md`, `USAGE.md`, etc.).
4. Ensure the test suite passes (if available).
5. Make sure your code is formatted correctly (`go fmt ./...`).
6. Issue that pull request!

## Development Setup

### Prerequisites
- [Go](https://golang.org/doc/install) 1.26.3 or higher.
- [Ollama](https://ollama.ai/) installed locally and running (for testing the LLM bridge).
- A local Ollama model (e.g., `qwen2.5-coder:1.5b`).

### Setting Up the Environment
1. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/golem-cli.git
   cd golem-cli
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the binary:
   ```bash
   go build -o golem ./cmd/golem
   ```

### Running the Application Locally
You can run the application directly from the source code:
```bash
go run ./cmd/golem
```

### Testing & Benchmarks
We maintain a strict >80% reliability baseline for Core Usability in the LLM parser. If you modify the core prompts or engine logic, you MUST run the reliability benchmarks locally.

To run the full suite (which tests complex natural language prompts including core and edge cases over 10 iterations):
```bash
go test -v ./internal/llm -run TestLLMReliability -count=1
```
*Note: You must have Ollama running locally with `qwen2.5-coder:1.5b` (or your configured model) to run the benchmarks.*

If you add a new feature that the LLM must parse (like a new Action or edge case), you must add a new `EvalCase` to the `benchmarkCases` slice in `internal/llm/eval_test.go`.

For standard integration tests testing sandbox constraints:
```bash
./test/integration.sh
```

### Generating Demo Scripts
If you make UI changes, please regenerate the demo GIFs. You will need [Charm VHS](https://github.com/charmbracelet/vhs) installed.
```bash
./demo/run_all.sh
```

## Architecture Overview
- **The CLI (`internal/CLI`)**: Contains the core filesystem logic (`fs.go`) and the security constraints (`sandbox.go`).
- **The LLM Bridge (`internal/llm`)**: Manages the API client to Ollama (`client.go`) and handles the JSON parsing and prompts (`prompt.go`).
- **The UI (`internal/cli` & `internal/ui`)**: Uses `go-prompt` for the REPL and contains the styling and terminal handling logic. *(Note: `go-prompt` provides the best interactive experience on macOS/Linux but may have rendering bugs on native Windows terminals).*

If you have any questions or need help, feel free to open an issue to discuss!
