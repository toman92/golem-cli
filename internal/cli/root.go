package cli

import (
	"bufio"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/c-bata/go-prompt"
	"github.com/toman92/golem-cli/internal/fileops"
	"github.com/toman92/golem-cli/internal/llm"
	"github.com/toman92/golem-cli/internal/logger"
	"github.com/toman92/golem-cli/internal/models"
	"github.com/toman92/golem-cli/internal/ui"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var (
	cfgFile          string
	sandboxRoot      string
	dryRun           bool
	modelName        string
	endpoint         string
	includeHidden    bool
	autoConfirm      bool
	maxRetries       int
	debugMode        bool
	excludeDirs      []string
	deletableFolders []string
)

func getHistoryFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".golem_history")
}

func loadHistory() []string {
	var history []string
	f, err := os.Open(getHistoryFile())
	if err != nil {
		return history
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			history = append(history, text)
		}
	}
	return history
}

func appendHistory(req string) {
	f, err := os.OpenFile(getHistoryFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(req + "\n")
		f.Close()
	}
}

func clearHistoryFile() {
	os.Remove(getHistoryFile())
}

func deepSearchDir(targetName string, searchRoot string) string {
	var found string
	err := filepath.WalkDir(searchRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "build" || name == "dist" {
			if path != searchRoot {
				return filepath.SkipDir
			}
		}

		for _, excluded := range excludeDirs {
			if name == excluded {
				return filepath.SkipDir
			}
		}

		if strings.EqualFold(name, targetName) {
			found = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return ""
	}
	return found
}

func completer(d prompt.Document) []prompt.Suggest {
	text := d.TextBeforeCursor()
	if !strings.HasPrefix(text, "/") {
		return []prompt.Suggest{}
	}
	s := []prompt.Suggest{
		{Text: "/config model", Description: "Select the LLM model to use"},
		{Text: "/config auto-confirm", Description: "Toggle auto-confirm for prompts"},
		{Text: "/config include-hidden", Description: "Toggle including hidden files in searches"},
		{Text: "/config exclude-dirs", Description: "Manage directories to exclude from searches"},
		{Text: "/config deletable-folders", Description: "Manage directories allowed for deletion"},
		{Text: "/debug", Description: "Toggle debug mode on/off"},
		{Text: "/clear", Description: "Clear command history"},
		{Text: "/help", Description: "Show available commands"},
		{Text: "/license", Description: "Show GNU GPL v3 license details"},
		{Text: "/exit", Description: "Exit golem"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func initLogger() {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".golem.log")
	if err := logger.Init(logPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize logger: %v\n", err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "golem [request]",
	Short: "A fast, secure local AI file organizer",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogger()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		defer restoreTerminal()

		// Handle OS interrupts gracefully to restore terminal state
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			restoreTerminal()
			os.Exit(0)
		}()

		llmClient := llm.NewClient(endpoint, modelName)
		ensureModelSelected(llmClient)

		if len(args) > 0 {
			request := strings.Join(args, " ")
			err := processRequest(llmClient, request)
			if err != nil {
				if err.Error() == "operation cancelled by user" {
					ui.PrintWarning("Operation cancelled.\n")
					return nil
				}
				return err
			}
			return nil
		}

		// Ping Ollama in the background to load the model into memory
		// so the user's first prompt executes instantly.
		go llmClient.PreloadModel()

		ui.PrintLLM("Golem - Local AI File Organizer\n")
		ui.PrintLLM("Type your request, or use '/' for commands. Type '/exit' to quit.\n")
		ui.PrintLLM("Using model: %s at %s\n", modelName, endpoint)

		// Setup loop for go-prompt
		for {
			req := prompt.Input(
				"golem ❯ ",
				completer,
				prompt.OptionHistory(loadHistory()),
				prompt.OptionPrefixTextColor(prompt.Cyan),
				prompt.OptionCompletionOnDown(),
			)

			// Force cooked mode so bufio.Reader doesn't hang due to go-prompt's background goroutines
			restoreTerminal()

			req = strings.TrimSpace(req)
			if req == "" {
				continue
			}

			if req == "/exit" || req == "/quit" || req == "exit" {
				ui.PrintPlan("Goodbye!\n")
				return nil
			}

			if strings.HasPrefix(req, "/") {
				handleSlashCommand(req, llmClient)
				continue
			}

			if len(req) < 3 {
				ui.PrintError("Request too short. Please provide a more descriptive command.\n")
				continue
			}

			appendHistory(req)
			ui.PrintUser("\n👤 User: %s\n", req)

			err := processRequest(llmClient, req)
			if err != nil {
				if err.Error() == "operation cancelled by user" {
					ui.PrintWarning("\nOperation cancelled.\n")
				} else {
					slog.Error("Request processing failed", "error", err)
					ui.PrintError("Error: %v\n", err)
				}
			}
		}
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage golem configuration",
}

var configModelCmd = &cobra.Command{
	Use:   "model",
	Short: "Select the Ollama model to use",
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogger()
		llmClient := llm.NewClient(endpoint, modelName)
		return promptModelSelection(llmClient, true)
	},
}

func Execute() {
	args := os.Args[1:]
	isCommand := len(args) == 0

	for _, arg := range args {
		if arg == "config" || arg == "help" || arg == "completion" || arg == "-h" || arg == "--help" {
			isCommand = true
			break
		}
	}

	if isCommand {
		rootCmd.AddCommand(configCmd)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.golem.yaml)")
	rootCmd.PersistentFlags().StringVarP(&sandboxRoot, "dir", "d", ".", "Sandbox directory root")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview operations without modifying files")
	rootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "qwen2.5-coder:1.5b", "Ollama model name")
	rootCmd.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "http://localhost:11434", "Ollama API endpoint")
	rootCmd.PersistentFlags().BoolVar(&includeHidden, "include-hidden", false, "Process hidden files and directories (starting with '.')")
	rootCmd.PersistentFlags().BoolVar(&autoConfirm, "auto-confirm", false, "Execute operations without prompting for confirmation")
	rootCmd.PersistentFlags().IntVar(&maxRetries, "max-retries", 3, "Number of times to retry LLM generation on failure")

	viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag("endpoint", rootCmd.PersistentFlags().Lookup("endpoint"))
	viper.BindPFlag("include-hidden", rootCmd.PersistentFlags().Lookup("include-hidden"))
	viper.BindPFlag("auto-confirm", rootCmd.PersistentFlags().Lookup("auto-confirm"))
	viper.BindPFlag("max-retries", rootCmd.PersistentFlags().Lookup("max-retries"))

	configCmd.AddCommand(configModelCmd)
	// We dynamically add configCmd in Execute() so it doesn't intercept natural language requests
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".golem")
	}

	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	if viper.IsSet("dir") {
		sandboxRoot = viper.GetString("dir")
	}
	if viper.IsSet("model") {
		modelName = viper.GetString("model")
	}
	if viper.IsSet("endpoint") {
		endpoint = viper.GetString("endpoint")
	}
	if viper.IsSet("include-hidden") {
		includeHidden = viper.GetBool("include-hidden")
	}
	if viper.IsSet("auto-confirm") {
		autoConfirm = viper.GetBool("auto-confirm")
	}
	if viper.IsSet("max-retries") {
		maxRetries = viper.GetInt("max-retries")
	}

	if viper.IsSet("exclude-dirs") {
		excludeDirs = viper.GetStringSlice("exclude-dirs")
	} else {
		excludeDirs = []string{"node_modules", "build", "dist", ".git", ".venv"}
	}

	if viper.IsSet("deletable-folders") {
		deletableFolders = viper.GetStringSlice("deletable-folders")
	} else {
		deletableFolders = []string{"node_modules", "build", "dist"}
	}

	sandboxRoot = expandHome(sandboxRoot)
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

func sanitizeSource(path string, patterns []string) (string, []string) {
	path = expandHome(path)

	// Strip leading ./ if it exists to help with LLM hallucinations
	if strings.HasPrefix(path, "./") {
		path = strings.TrimPrefix(path, "./")
	} else if strings.HasPrefix(path, ".\\") {
		path = strings.TrimPrefix(path, ".\\")
	}

	if idx := strings.Index(path, "*"); idx != -1 {
		slashIdx := strings.LastIndex(path[:idx], string(filepath.Separator))
		if slashIdx == -1 && filepath.Separator != '/' {
			slashIdx = strings.LastIndex(path[:idx], "/")
		}

		basePath := "."
		if slashIdx != -1 {
			basePath = path[:slashIdx]
			if basePath == "" {
				basePath = "/"
			}
		}

		base := filepath.Base(path)
		if strings.Contains(base, "*") && base != "*" {
			addPattern := true
			for _, p := range patterns {
				if p == base {
					addPattern = false
					break
				}
			}
			if addPattern {
				patterns = append(patterns, base)
			}
		}
		path = basePath
	}

	return path, patterns
}

func saveConfig() {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".golem.yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		slog.Error("Failed to save config", "error", err)
		ui.PrintError("Warning: failed to save config to %s\n", configPath)
	}
}

func handleSlashCommand(req string, c *llm.Client) {
	parts := strings.SplitN(req, " ", 3)
	cmd := parts[0]

	switch cmd {
	case "/help":
		ui.PrintLLM("\nAvailable Commands:\n")
		ui.PrintPlan("  /config model          - Select a different Ollama model\n")
		ui.PrintPlan("  /config auto-confirm   - Toggle auto-confirmation of file operations\n")
		ui.PrintPlan("  /config include-hidden - Toggle processing of hidden files\n")
		ui.PrintPlan("  /config deletable-folders - Set deletable folders whitelist\n")
		ui.PrintPlan("  /debug                 - Toggle debug mode on/off\n")
		ui.PrintPlan("  /clear                 - Clear command history\n")
		ui.PrintPlan("  /license               - Show license information\n")
		ui.PrintPlan("  /help                  - Show this help message\n")
		ui.PrintPlan("  /exit                  - Quit the application\n\n")
	case "/license":
		ui.PrintLLM("\nGolem is licensed under the GNU General Public License v3.0.\n")
		ui.PrintPlan("Copyright (C) 2026 Toman92\n\n")
		ui.PrintPlan("This program is free software: you can redistribute it and/or modify\n")
		ui.PrintPlan("it under the terms of the GNU General Public License as published by\n")
		ui.PrintPlan("the Free Software Foundation, either version 3 of the License, or\n")
		ui.PrintPlan("(at your option) any later version.\n\n")
		ui.PrintPlan("This program is distributed in the hope that it will be useful,\n")
		ui.PrintPlan("but WITHOUT ANY WARRANTY; without even the implied warranty of\n")
		ui.PrintPlan("MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the\n")
		ui.PrintPlan("GNU General Public License for more details.\n\n")
		ui.PrintPlan("You should have received a copy of the GNU General Public License\n")
		ui.PrintPlan("along with this program. If not, see <https://www.gnu.org/licenses/>.\n\n")
	case "/debug":
		debugMode = !debugMode
		if debugMode {
			ui.PrintSuccess("Debug mode enabled. Detailed plans and logs will be shown.\n")
		} else {
			ui.PrintSuccess("Debug mode disabled. Standard clean UI restored.\n")
		}
	case "/clear":

		clearHistoryFile()
		ui.PrintSuccess("Command history cleared.\n")
	case "/config":
		if len(parts) < 2 {
			ui.PrintError("Usage: /config [model|auto-confirm|include-hidden]\n")
			return
		}
		sub := parts[1]
		switch sub {
		case "model":
			promptModelSelection(c, true)
		case "auto-confirm":
			autoConfirm = !autoConfirm
			viper.Set("auto-confirm", autoConfirm)
			saveConfig()
			ui.PrintSuccess("Auto-confirm set to: %v\n", autoConfirm)
		case "include-hidden":
			includeHidden = !includeHidden
			viper.Set("include-hidden", includeHidden)
			saveConfig()
			ui.PrintSuccess("Include hidden files set to: %v\n", includeHidden)
		case "exclude-dirs":
			if len(parts) > 2 {
				val := strings.Join(parts[2:], " ")
				excludeDirs = strings.Split(strings.ReplaceAll(val, " ", ""), ",")
				viper.Set("exclude-dirs", excludeDirs)
				saveConfig()
				ui.PrintSuccess("Exclude dirs set to: %v\n", excludeDirs)
			} else {
				ui.PrintSuccess("Current exclude-dirs: %v\n", excludeDirs)
				ui.PrintLLM("Use /config exclude-dirs dir1,dir2 to change\n")
			}
		case "deletable-folders":
			if len(parts) > 2 {
				val := strings.Join(parts[2:], " ")
				deletableFolders = strings.Split(strings.ReplaceAll(val, " ", ""), ",")
				viper.Set("deletable-folders", deletableFolders)
				saveConfig()
				ui.PrintSuccess("Deletable folders set to: %v\n", deletableFolders)
			} else {
				ui.PrintSuccess("Current deletable-folders: %v\n", deletableFolders)
				ui.PrintLLM("Use /config deletable-folders dir1,dir2 to change\n")
			}
		default:
			ui.PrintError("Unknown config option: %s\n", sub)
		}
	default:
		ui.PrintError("Unknown command: %s. Type /help for options.\n", cmd)
	}
}

func ensureModelSelected(c *llm.Client) {
	models, err := c.ListSmallModels()
	if err != nil {
		slog.Error("Failed to list models during check", "error", err)
		return
	}

	found := false
	for _, m := range models {
		if m == modelName {
			found = true
			break
		}
	}

	if !found {
		ui.PrintWarning("\nConfigured model '%s' was not found or is >3B parameters.\n", modelName)
		_ = promptModelSelection(c, false)
	}
}

func promptModelSelection(c *llm.Client, force bool) error {
	models, err := c.ListSmallModels()
	if err != nil {
		return fmt.Errorf("failed to fetch models from Ollama: %w", err)
	}

	if len(models) == 0 {
		return fmt.Errorf("no models under 3B parameters found in Ollama")
	}

	ui.PrintLLM("Available lightweight models (< 3B parameters):\n")
	for i, m := range models {
		ui.PrintPlan("  [%d] %s\n", i+1, m)
	}

	restoreTerminal()
	reader := bufio.NewScanner(os.Stdin)
	for {
		ui.CurrentTheme.Question.Print("Select a model by number: ")
		if !reader.Scan() {
			return nil
		}

		input := strings.TrimSpace(reader.Text())
		idx, err := strconv.Atoi(input)
		if err == nil && idx >= 1 && idx <= len(models) {
			selected := models[idx-1]
			viper.Set("model", selected)
			saveConfig()

			ui.PrintSuccess("Saved '%s' as default model.\n", selected)
			ui.PrintLLM("\nUsing model: %s at %s\n", selected, endpoint)

			modelName = selected
			c.Model = selected
			return nil
		}
		ui.PrintError("Invalid selection. Try again.\n")
	}
}

func restoreTerminal() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}
	cmd := exec.Command("stty", "sane")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func confirmAction(promptMsg string) bool {
	// Force cooked mode so bufio.Reader doesn't hang due to go-prompt's background goroutines
	// and so we don't get weird newline echo behavior.
	restoreTerminal()

	fmt.Printf("%s [y/N]: ", promptMsg)
	reader := bufio.NewReader(os.Stdin)
	ans, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	ans = strings.TrimSpace(strings.ToLower(ans))
	return ans == "y" || ans == "yes"
}

func confirmExpandAction(promptMsg string, expandFn func()) bool {
	restoreTerminal()
	for {
		fmt.Printf("%s [y/N/e to expand]: ", promptMsg)
		reader := bufio.NewReader(os.Stdin)
		ans, err := reader.ReadString('\n')
		if err != nil {
			return false
		}

		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans == "e" || ans == "expand" {
			expandFn()
			return confirmAction(promptMsg)
		}
		if ans == "y" || ans == "yes" {
			return true
		}
		if ans == "n" || ans == "no" || ans == "" {
			return false
		}
	}
}

func levenshtein(s, t string) int {
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				min := d[i-1][j]
				if d[i][j-1] < min {
					min = d[i][j-1]
				}
				if d[i-1][j-1] < min {
					min = d[i-1][j-1]
				}
				d[i][j] = min + 1
			}
		}
	}
	return d[len(s)][len(t)]
}

func resolveSmartPath(rawPath string, label string, isDest bool, srcInHome bool) (string, bool, error) {
	if !filepath.IsAbs(rawPath) {
		checkLocal := filepath.Join(sandboxRoot, rawPath)
		if _, err := os.Stat(checkLocal); os.IsNotExist(err) {
			home, _ := os.UserHomeDir()
			checkHome := filepath.Join(home, rawPath)
			if stat, err := os.Stat(checkHome); err == nil && stat.IsDir() {
				rawPath = checkHome
			} else if strings.HasPrefix(rawPath, ".") && !strings.HasPrefix(rawPath, "./") && !strings.HasPrefix(rawPath, "..") {
				stripped := rawPath[1:]
				if stat, err := os.Stat(filepath.Join(sandboxRoot, stripped)); err == nil && stat.IsDir() {
					rawPath = stripped
				} else if stat, err := os.Stat(filepath.Join(home, stripped)); err == nil && stat.IsDir() {
					rawPath = filepath.Join(home, stripped)
				} else {
					rawPath = stripped
				}
			} else if isDest {
				parentHome := filepath.Dir(checkHome)
				parentLocal := filepath.Dir(checkLocal)

				localParentExists := false
				if stat, err := os.Stat(parentLocal); err == nil && stat.IsDir() {
					localParentExists = true
				}

				homeParentExists := false
				if stat, err := os.Stat(parentHome); err == nil && stat.IsDir() {
					homeParentExists = true
				}

				if srcInHome {
					rawPath = checkHome
				} else if homeParentExists && !localParentExists {
					rawPath = checkHome
				}
			}
		}
	}

	finalPath, escapes, err := fileops.ResolvePath(sandboxRoot, rawPath)
	if err != nil {
		return "", escapes, err
	}

	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		current := filepath.Clean(finalPath)
		var missingParts []string

		for {
			if stat, err := os.Stat(current); err == nil && stat.IsDir() {
				break
			}
			parent := filepath.Dir(current)
			if current == parent {
				break
			}
			missingParts = append([]string{filepath.Base(current)}, missingParts...)
			current = parent
		}

		if len(missingParts) > 0 {
			targetName := missingParts[0]

			bestMatch := ""
			bestDist := 3
			if entries, err := os.ReadDir(current); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						dist := levenshtein(strings.ToLower(targetName), strings.ToLower(entry.Name()))
						if dist > 0 && dist < bestDist {
							bestMatch = entry.Name()
							bestDist = dist
						}
					}
				}
			}

			userAborted := false
			if bestMatch != "" {
				ui.PrintWarning("\n💡 %s folder '%s' not found in '%s'.\n", label, targetName, current)
				ui.PrintWarning("   Did you mean '%s'?\n", bestMatch)
				promptSuffix := fmt.Sprintf("create '%s' instead", targetName)
				if !isDest || len(missingParts) > 1 {
					promptSuffix = "continue"
				}
				if confirmAction(fmt.Sprintf("   Press Y to auto-correct, or N to %s", promptSuffix)) {
					reconstructed := filepath.Join(current, bestMatch)
					for i := 1; i < len(missingParts); i++ {
						reconstructed = filepath.Join(reconstructed, missingParts[i])
					}
					finalPath = reconstructed
					finalPath, escapes, _ = fileops.ResolvePath(sandboxRoot, finalPath)
				} else {
					userAborted = true
				}
			}

			if _, err := os.Stat(finalPath); os.IsNotExist(err) {
				if !isDest || len(missingParts) > 1 {
					if debugMode {
						ui.PrintLLM("\n🔍 Deep searching for '%s'...\n", targetName)
					}
					home, _ := os.UserHomeDir()
					if deepFound := deepSearchDir(targetName, home); deepFound != "" {
						if debugMode {
							ui.PrintSuccess("   Found at: %s\n", deepFound)
						}
						reconstructed := deepFound
						for i := 1; i < len(missingParts); i++ {
							reconstructed = filepath.Join(reconstructed, missingParts[i])
						}
						finalPath = reconstructed
						finalPath, escapes, _ = fileops.ResolvePath(sandboxRoot, finalPath)
					} else {
						if !isDest {
							ui.PrintError("\n❌ %s path component '%s' does not exist.\n", label, targetName)
							return "", escapes, fmt.Errorf("path does not exist")
						}
					}
				} else if userAborted && !isDest {
					ui.PrintError("\n❌ %s path '%s' does not exist.\n", label, rawPath)
					return "", escapes, fmt.Errorf("operation cancelled by user")
				} else if !isDest {
					ui.PrintError("\n❌ %s path '%s' does not exist.\n", label, rawPath)
					return "", escapes, fmt.Errorf("path does not exist")
				}
			}
		}
	}

	if escapes {
		if !confirmAction(fmt.Sprintf("\n%s path '%s' is outside your current folder. Allow?", label, finalPath)) {
			slog.Info("User cancelled due to path escape", "path", finalPath)
			return "", escapes, fmt.Errorf("operation cancelled by user")
		}
	}

	return finalPath, escapes, nil
}

func processDeleteAction(op *models.Operation, dirs []string) error {
	var matchedDirs []string
	for _, d := range dirs {
		base := filepath.Base(d)
		allowed := false
		for _, allowedDir := range deletableFolders {
			if base == allowedDir {
				allowed = true
				break
			}
		}
		if allowed {
			matchedDirs = append(matchedDirs, d)
		} else {
			slog.Warn("Skipped deletion of non-whitelisted directory", "dir", d)
		}
	}

	if len(matchedDirs) == 0 {
		ui.PrintWarning("No explicitly whitelisted directories found matching the criteria.\n")
		return nil
	}

	ui.PrintPlan("Found %d directories to DELETE.\n", len(matchedDirs))
	
	limit := len(matchedDirs)
	if !debugMode && limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		ui.PrintPlan("  [PLAN] DELETE: %s\n", matchedDirs[i])
	}
	if !debugMode && len(matchedDirs) > 5 {
		ui.PrintPlan("  ... and %d more directories.\n", len(matchedDirs)-5)
	}

	if dryRun {
		ui.PrintLLM("\n🔍 Dry run completed. No directories were deleted.\n")
		return nil
	}

	if !autoConfirm {
		prompt := fmt.Sprintf("\nDANGER: Proceed with PERMANENTLY deleting %d directories?", len(matchedDirs))
		if !debugMode && len(matchedDirs) > 5 {
			if !confirmExpandAction(prompt, func() {
				for i := 5; i < len(matchedDirs); i++ {
					ui.PrintPlan("  [PLAN] DELETE: %s\n", matchedDirs[i])
				}
			}) {
				slog.Info("User cancelled DELETE operation")
				return fmt.Errorf("operation cancelled by user")
			}
		} else {
			if !confirmAction(prompt) {
				slog.Info("User cancelled DELETE operation")
				return fmt.Errorf("operation cancelled by user")
			}
		}
	}

	for _, d := range matchedDirs {
		if err := os.RemoveAll(d); err != nil {
			ui.PrintError("Failed to delete %s: %v\n", d, err)
		} else {
			ui.PrintSuccess("Deleted: %s\n", d)
		}
	}
	return nil
}

func processFileAction(src, dst string, op *models.Operation, files []string) error {
	ui.PrintPlan("Found %d files to %s.\n", len(files), op.Action)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		ui.PrintPlan("  [PLAN] CREATE: %s\n", dst)
	}

	displayLimit := len(files)
	if !debugMode && displayLimit > 5 {
		displayLimit = 5
	}
	for i := 0; i < displayLimit; i++ {
		f := files[i]
		rel, err := filepath.Rel(src, f)
		if err != nil || rel == "." {
			rel = filepath.Base(f)
		}
		flattened := strings.ReplaceAll(rel, string(os.PathSeparator), "_")
		targetPath := filepath.Join(dst, flattened)
		ui.PrintPlan("  [PLAN] %s: %s -> %s\n", op.Action, f, targetPath)
	}
	if !debugMode && len(files) > 5 {
		ui.PrintPlan("  ... and %d more files.\n", len(files)-5)
	}

	if dryRun {
		ui.PrintLLM("\n🔍 Dry run completed. No files were modified.\n")
		return nil
	}

	collisionStrategy := "c" // copy (default handleCollision)
	collisionsFound := 0
	for _, f := range files {
		rel, err := filepath.Rel(src, f)
		if err != nil || rel == "." {
			rel = filepath.Base(f)
		}
		flattened := strings.ReplaceAll(rel, string(os.PathSeparator), "_")
		targetPath := filepath.Join(dst, flattened)

		if _, err := os.Stat(targetPath); err == nil {
			collisionsFound++
		}
	}

	if collisionsFound > 0 {
		ui.PrintWarning("\n⚠️  Found %d file(s) that already exist at the destination.\n", collisionsFound)
		for {
			restoreTerminal()
			fmt.Printf("Do you want to (r)eplace all, (c)reate copies, or (s)kip all? [r/c/s]: ")
			reader := bufio.NewReader(os.Stdin)
			ans, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read collision strategy: %w", err)
			}
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans == "r" || ans == "replace" {
				collisionStrategy = "r"
				break
			} else if ans == "c" || ans == "copy" {
				collisionStrategy = "c"
				break
			} else if ans == "s" || ans == "skip" {
				collisionStrategy = "s"
				break
			}
		}
	} else if !autoConfirm {
		prompt := fmt.Sprintf("\nProceed with processing %d files?", len(files))
		if !debugMode && len(files) > 5 {
			if !confirmExpandAction(prompt, func() {
				for i := 5; i < len(files); i++ {
					f := files[i]
					rel, err := filepath.Rel(src, f)
					if err != nil || rel == "." {
						rel = filepath.Base(f)
					}
					flattened := strings.ReplaceAll(rel, string(os.PathSeparator), "_")
					targetPath := filepath.Join(dst, flattened)
					ui.PrintPlan("  [PLAN] %s: %s -> %s\n", op.Action, f, targetPath)
				}
			}) {
				slog.Info("User cancelled operation before processing files")
				return fmt.Errorf("operation cancelled by user")
			}
		} else {
			if !confirmAction(prompt) {
				slog.Info("User cancelled operation before processing files")
				return fmt.Errorf("operation cancelled by user")
			}
		}
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		slog.Error("Failed to create destination directory", "error", err)
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	var successCount int32
	var uiMutex sync.Mutex

	g := new(errgroup.Group)
	limit := runtime.NumCPU() * 2
	if limit < 4 {
		limit = 4
	}
	g.SetLimit(limit)

	for _, f := range files {
		f := f // capture loop variable
		g.Go(func() error {
			rel, err := filepath.Rel(src, f)
			if err != nil || rel == "." {
				rel = filepath.Base(f)
			}
			flattened := strings.ReplaceAll(rel, string(os.PathSeparator), "_")
			targetPath := filepath.Join(dst, flattened)

			var finalDst string
			var opErr error
			var skipped bool
			if op.Action == models.ActionCopy {
				finalDst, skipped, opErr = fileops.CopyFile(f, targetPath, collisionStrategy)
			} else {
				finalDst, skipped, opErr = fileops.MoveFile(f, targetPath, collisionStrategy)
			}

			uiMutex.Lock()
			defer uiMutex.Unlock()

			if opErr != nil {
				slog.Error("File operation failed", "action", op.Action, "file", f, "error", opErr)
				ui.PrintError("  ❌ Error %s file %s: %v\n", op.Action, flattened, opErr)
			} else if skipped {
				slog.Info("File operation skipped", "action", op.Action, "file", finalDst)
				ui.PrintWarning("  ⏭️  SKIPPED: %s -> %s\n", flattened, finalDst)
			} else {
				slog.Info("File operation successful", "action", op.Action, "file", finalDst)
				ui.PrintSuccess("  ✅ %s: %s -> %s\n", op.Action, flattened, finalDst)
				atomic.AddInt32(&successCount, 1)
			}
			return nil
		})
	}

	_ = g.Wait()

	finalCount := atomic.LoadInt32(&successCount)
	slog.Info("Request processing completed", "successCount", finalCount, "total", len(files))
	ui.PrintSuccess("\n🎉 Completed! Successfully processed %d/%d files.\n", finalCount, len(files))

	return nil
}

func processRequest(client *llm.Client, request string) error {
	var op *models.Operation
	var err error

	fmt.Println() // Add a leading newline before spinner
	ui.ShowSpinner("🤔 Thinking...", func() {
		op, err = client.ParseRequest(request, maxRetries)
	})

	if err != nil {
		return err
	}

	if debugMode {
		ui.PrintPlan("\n📝 Plan:\n")
		ui.PrintPlan("  Action: %s\n", op.Action)
		ui.PrintPlan("  Source: %s\n", op.Source)
		ui.PrintPlan("  Destination: %s\n", op.Destination)
		ui.PrintPlan("  Patterns: %v\n\n", op.Patterns)
	}

	op.Source = strings.ReplaceAll(op.Source, "\\", "/")
	op.Destination = strings.ReplaceAll(op.Destination, "\\", "/")

	srcPath, opPatterns := sanitizeSource(op.Source, op.Patterns)
	op.Patterns = opPatterns
	dstPath, _ := sanitizeSource(op.Destination, nil)

	// LLM File Extraction Heuristic: If the LLM placed a specific file name inside the Source
	// string instead of Patterns, and it's not a valid directory, extract it to Patterns.
	baseName := filepath.Base(srcPath)
	ext := filepath.Ext(srcPath)
	if ext != "" && ext != baseName {
		isDir := false
		if stat, err := os.Stat(filepath.Join(sandboxRoot, srcPath)); err == nil && stat.IsDir() {
			isDir = true
		} else {
			home, _ := os.UserHomeDir()
			if stat, err := os.Stat(filepath.Join(home, srcPath)); err == nil && stat.IsDir() {
				isDir = true
			}
		}

		if !isDir {
			op.Patterns = append(op.Patterns, baseName)
			srcPath = filepath.Dir(srcPath)
		}
	}

	src, _, err := resolveSmartPath(srcPath, "Source", false, false)
	if err != nil {
		return err
	}

	var files []string
	var dirs []string

	if op.Action == models.ActionDelete {
		dirs, err = fileops.FindDirectories(src, op.Patterns, op.ExcludePatterns, includeHidden)
		if err != nil {
			return fmt.Errorf("failed to find directories: %w", err)
		}
	} else {
		files, err = fileops.FindFiles(src, op.Patterns, op.ExcludePatterns, includeHidden, excludeDirs)
		if err != nil {
			return fmt.Errorf("failed to find files: %w", err)
		}
		if len(files) == 0 {
			ui.PrintWarning("\nNo files found in source matching the criteria.\n")
			return nil
		}
	}

	var dst string
	if op.Action != "DELETE" {
		home, _ := os.UserHomeDir()
		srcInHome := filepath.IsAbs(src) && strings.HasPrefix(src, home) && !strings.HasPrefix(src, sandboxRoot)
		var dstErr error
		dst, _, dstErr = resolveSmartPath(dstPath, "Destination", true, srcInHome)
		if dstErr != nil {
			return dstErr
		}
	}

	if op.Action == "DELETE" {
		return processDeleteAction(op, dirs)
	}

	return processFileAction(src, dst, op, files)
}
