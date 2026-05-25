# Local AI File Organizer Named Golem: Project Plan

## Overview and Goal
The goal of this project is to build a lightweight, fast, and secure local file organizer CLI. Instead of relying on a slow, heavyweight AI agent framework to search and modify your filesystem, this system uses a hybrid architecture. A custom Go binary handles all high-speed, secure filesystem execution, while a micro-LLM running locally in Docker strictly acts as a natural-language-to-JSON parser. This ensures absolute security, instant execution, and zero risk of the AI hallucinating destructive system commands.

## Architecture and Orchestration
The system is divided into two strict components:

1. **LLM (Dockerized Ollama):** Runs a micro-model (e.g., Osmosis-Structure 0.6B or Qwen 2.5 Coder 1.5B). It receives natural language requests from the CLI and outputs a strict JSON schema defining the intended action, source, destination, and file patterns.
2. **CLI (Go CLI Host Binary):** Runs natively on the host OS. It captures terminal input, communicates with the Ollama API, parses the JSON, and enforces zero-trust path scoping before executing any filesystem commands. 

## Required Tools
- **CLI:** Compiles the custom binary (`flag` or `cobra` for CLI, `net/http` for API requests, `os` and `filepath` for system operations) -already installed on system
- **Dockerized Ollama:** Exposes the `/api/generate` endpoint on `localhost:11434`.-already pulled. running. model osmossis-structure has been pulled. Qwen2.5-coder:1.5b & 3b and Llama3.2:1b & 3b pulled as backup.
- **Micro-LLM:** A structured-output optimized model under 3B parameters.-done-osmosis-structure. 

## Phase 1: Go Execution CLI and Security Sandbox
Build the core application and security boundaries in Go without any AI integration.
- Define a Go struct to mirror the expected JSON payload (action, source, destination, file pattern).
- Implement directory constraint logic using `filepath.Clean` to strictly sandbox operations to a predefined work directory. 
- Hardcode rejections for destructive actions like `DELETE` or paths escaping to root `/`.
- Build the `filepath.WalkDir` search logic for recursive directory scanning.
- Implement the copy and move functions, including timestamp/hash appending for collision detection.
- Add a `--dry-run` flag to safely preview all intended filesystem operations.

## Phase 2: LLM Bridge and JSON Parsing
Connect the CLI to the local AI parser.
- Write the HTTP client logic in Go to communicate with the local Ollama API.
- Draft a rigid system prompt enforcing the LLM to act only as a JSON translator.
- Force JSON structured output using Ollama's API parameters.
- Unmarshal the LLM's JSON response directly into the Phase 1 Go struct.
- Validate that the data types and paths returned by the LLM pass the Phase 1 security sandbox.

## Phase 3: CLI Wrapper and Polish
Finalize the tool for daily terminal use.
- Wire up the terminal input capture to pass user strings to the LLM bridge.
- Add terminal coloring and formatting to highlight `--dry-run` output clearly.
- Compile the Go project into a single static binary.
- Move the compiled binary to your host's local bin directory for global access.

## Phase 4: Advanced Interactive REPL & Slash Commands
- Upgraded the CLI from a one-shot execution model to a continuous interactive REPL using `c-bata/go-prompt`.
- Enforced a strictly stateless architecture to guarantee absolute command isolation and prevent hallucination drift between requests.
- Built interactive slash commands (e.g., `/config model`, `/clear`, `/debug`, `/help`, `/exit`) to allow dynamic runtime configuration.
- Added dynamic model fetching, polling Ollama for available local models under 3B parameters.
- Implemented rich, colorful terminal UI formatting using `fatih/color`.

## Phase 5: Advanced Safeties, Fuzzy Matching & Expanded Operations
- Upgraded the sandbox constraints to allow user-confirmed sandbox escapes rather than hard rejections.
- Replaced the strict hard-rejection of `DELETE` operations with a secure, user-configurable whitelist constraint.
- Implemented intelligent fuzzy-matching and typo auto-correction with interactive user prompts.
- Handled advanced TTY state restoration (`stty sane`) to cleanly recover from raw-mode terminal hijacking.

## Phase 6: Testing, Documentation & Security Audit
- Built a comprehensive deterministic bash integration testing suite (`test/integration.sh`).
- Implemented an advanced LLM Reliability Benchmark suite (`internal/llm/eval_test.go`), cleanly separating Core Usability cases from Edge Cases to maintain an >80% real-world reliability baseline.
- Implemented a suite of Charm VHS (`.tape`) scripts to automatically generate high-quality `.gif` demonstrations.
- Created a deterministic dummy file fixture generator for sandbox testing.
- Conducted a strict security audit confirming zero RCE vulnerabilities and robust path-traversal boundaries.
- Completed a full code-quality pass, ensuring the project complies with `go vet` and `staticcheck` standards.

## Phase 7: UI Modernization and UX Polish
- Stripped unnecessary debug text from the default output, showing clean, truncated execution lists (max 5 items) for large operations.
- Built an interactive `[y/N/e to expand]` prompt flow that dynamically expands large file lists cleanly inside the shell.
- Echoes user input seamlessly in the REPL using a custom bold, white theme for better visibility.
- Implemented a global `/debug` mode to expose raw JSON plans and deep-search logic logs for advanced developers.