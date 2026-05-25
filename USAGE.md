# Golem Usage Guide

Golem brings the power of local AI to your file organization tasks, turning natural language into safe, precise filesystem operations.

## 🚀 Getting Started

There are two primary ways to interact with Golem: Interactive Mode (REPL) and Single-Command Mode.

### Interactive Shell (REPL)
To launch the vibrant, persistent shell, run Golem without any arguments:

```bash
golem
```

The interactive shell features:
- **Command History**: Press `Up` or `Down` arrows to cycle through past requests.
- **Auto-complete**: Type `/` to bring up a dropdown of available slash commands.
- **Live Configuration**: Change settings on the fly without restarting.

### Single-Command Mode
If you prefer a quick, one-off execution, pass your request directly as arguments:

```bash
golem "copy all my jpg files from downloads to the photos directory"
```
*Note: This mode is great for scripting or alias integration in your `~/.bashrc` or `~/.zshrc`.*

## ⚙️ Slash Commands (Interactive Mode)

While in the REPL, type `/` to access commands:
- `/config model` - Opens an interactive model picker to select your Ollama model.
- `/config auto-confirm` - Toggles the execution confirmation prompt (be careful!).
- `/config include-hidden` - Toggles processing of hidden files (e.g., `.git`, `.env`).
- `/debug` - Toggles debug mode to show raw AI JSON plans and detailed path search logs.
- `/clear` - Clears the command history.
- `/help` - Displays the command list.
- `/exit` - Quits the shell cleanly.

## 🛡️ Safety First

Golem was designed with a Zero-Trust architecture to ensure your files are safe from AI hallucinations.

1. **Dry-Run by Default**: Golem will explicitly outline the source and destination of all intended changes (truncating to 5 lines for clean reading). It pauses and asks `Proceed with processing X files? [y/N/e to expand]` before modifying your system. Typing `e` will cleanly print out the remaining files for full visibility.
2. **Hidden File Protection**: Directories and files starting with a `.` are ignored by default.
3. **Sandbox Enforcement**: By default, Golem isolates operations to your current working directory. Moving or copying files outside of this path requires an explicit confirmation step.

*You can bypass confirmation by typing `/config auto-confirm` or passing the `--auto-confirm` flag, though this is only recommended for advanced users.*

## 🤖 LLM Configuration

Golem connects to a local Ollama instance (default: `http://localhost:11434`). To ensure lightning-fast responses, Golem filters the model list to lightweight models **3B parameters or fewer**. 

**Resilience & Retries:**
If the model hallucinates an invalid JSON response, Golem will automatically retry up to 3 times in the background. If it continually fails, it will gracefully suggest trying a more capable model, such as `qwen2.5-coder:1.5b`.

## 🛠️ Configuration & Flags

### Command-Line Flags
- `-d, --dir`: Sets the root sandbox directory (default is your current working directory).
- `--dry-run`: Previews the operations without asking to proceed, and exits immediately.
- `--auto-confirm`: Bypasses the confirmation prompt and executes immediately.
- `--include-hidden`: Allows Golem to process hidden files and directories.
- `--max-retries`: Sets the number of times to retry a failed LLM generation (default: 3).
- `-m, --model`: Override the model used for the specific execution.
- `-e, --endpoint`: Override the Ollama endpoint.
- `--config`: Manually specify the location of the `.golem.yaml` configuration file.

### Configuration File (`.golem.yaml`)
Golem saves its state in a `~/.golem.yaml` file. You can manually edit this file to configure default behavior, exclude specific directories, or designate certain folders as deletable.

Example `~/.golem.yaml`:
```yaml
auto-confirm: false
deletable-folders:
    - temp
    - cache
dir: ""
endpoint: http://localhost:11434
exclude-dirs:
    - node_modules
    - .git
    - build
include-hidden: false
max-retries: 3
model: qwen2.5-coder:1.5b
```
*Note: Environment variables (e.g., `MODEL`, `ENDPOINT`) are also supported. See `.env.example` in the project root.*

## 🔍 Checking Logs

Golem keeps your terminal output pristine by writing detailed, structured logs to a background file. If an operation fails or behaves unexpectedly, you can view the execution traces:

```bash
cat ~/.golem.log
```
*(Or use `tail -f ~/.golem.log` in a separate terminal to watch events in real-time.)*
