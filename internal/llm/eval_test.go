package llm_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/toman92/golem-cli/internal/llm"
	"github.com/toman92/golem-cli/internal/models"
)

// To run this benchmark manually:
// go test -v ./internal/llm -run TestLLMReliability -count=1

type EvalCase struct {
	Prompt          string
	ExpectedAction  models.Action
	ExpectedSource  string
	ExpectedDest    string
	ExpectedPattern string // We will check if this pattern exists in Patterns
	ExpectedExclude string // We will check if this pattern exists in ExcludePatterns
}

var benchmarkCases = []EvalCase{
	// 1. Basic Copy with natural language variations
	{
		Prompt:          "Copy all readme files from my personal folder and save them into a folder called project-overviews in my dev folder",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "personal",
		ExpectedDest:    "dev/project-overviews",
		ExpectedPattern: "README.md",
	},
	{
		Prompt:          "copy all .md files from robocode-ai to a folder called robocode-files-dump in my dev folder",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "robocode-ai",
		ExpectedDest:    "dev/robocode-files-dump",
		ExpectedPattern: "*.md",
	},
	{
		Prompt:          "copy all markdown files from this folder to a folder called golem-dump in my home folder",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  ".",
		ExpectedDest:    "golem-dump",
		ExpectedPattern: "*.md",
	},
	// 2. Complex Copy with Excludes
	{
		Prompt:          "copy all .md files from robocode-ai, except README.md files, to a folder called robocode-dump in my dev folder",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "robocode-ai",
		ExpectedDest:    "dev/robocode-dump",
		ExpectedPattern: "*.md",
		ExpectedExclude: "README.md",
	},
	{
		Prompt:          "copy all .md files, exlucding README.md files, from my dev/personal folder into a new folder called project-dump/docs in my dev folder",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "dev/personal",
		ExpectedDest:    "dev/project-dump/docs",
		ExpectedPattern: "*.md",
		ExpectedExclude: "README.md",
	},
	// 3. Single Files and Single Folders
	{
		Prompt:          "Move the budget.xlsx file from downloads to my documents folder",
		ExpectedAction:  models.ActionMove,
		ExpectedSource:  "downloads",
		ExpectedDest:    "documents",
		ExpectedPattern: "budget.xlsx",
	},
	{
		Prompt:          "Copy the entire projects folder from my desktop into my backups folder",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "desktop/projects",
		ExpectedDest:    "backups",
		ExpectedPattern: "*",
	},
	{
		Prompt:          "copy all files, except readme files and text files, from my documents folder to a doc-dump folder in my home directory",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "documents",
		ExpectedDest:    "doc-dump",
		ExpectedPattern: "",
		ExpectedExclude: "",
	},
	{
		Prompt:          "grab all the images from my downloads and put them in a folder called photos",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "downloads",
		ExpectedDest:    "photos",
		ExpectedPattern: "",
	},
	// 3. Deletion Operations
	{
		Prompt:          "delete the build directory in my web-app project",
		ExpectedAction:  models.ActionDelete,
		ExpectedSource:  "web-app",
		ExpectedDest:    "",
		ExpectedPattern: "build",
	},
	{
		Prompt:          "wipe out all node_modules folders inside the frontend folder",
		ExpectedAction:  models.ActionDelete,
		ExpectedSource:  "frontend",
		ExpectedDest:    "",
		ExpectedPattern: "node_modules",
	},
	{
		Prompt:          "get rid of the dist folder entirely",
		ExpectedAction:  models.ActionDelete,
		ExpectedSource:  "",
		ExpectedDest:    "",
		ExpectedPattern: "dist",
	},
	{
		Prompt:          "trash all .git folders in the subprojects dir",
		ExpectedAction:  models.ActionDelete,
		ExpectedSource:  "subprojects",
		ExpectedDest:    "",
		ExpectedPattern: ".git",
	},
	// 4. Moving Operations
	{
		Prompt:          "move all .go files from backend/src to backend/legacy",
		ExpectedAction:  models.ActionMove,
		ExpectedSource:  "backend/src",
		ExpectedDest:    "backend/legacy",
		ExpectedPattern: "*.go",
	},
	{
		Prompt:          "transfer the config.json file from settings to the backup directory",
		ExpectedAction:  models.ActionMove,
		ExpectedSource:  "settings",
		ExpectedDest:    "backup",
		ExpectedPattern: "config.json",
	},
	{
		Prompt:          "relocate all mp4 videos from my Downloads to the Movies/Archive folder",
		ExpectedAction:  models.ActionMove,
		ExpectedSource:  "Downloads",
		ExpectedDest:    "Movies/Archive",
		ExpectedPattern: "*.mp4",
	},
}

var edgeCases = []EvalCase{
	// 5. Short / Ambiguous Phrasing
	{
		Prompt:          "copy everything from here to desktop",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  ".",
		ExpectedDest:    "desktop",
		ExpectedPattern: "",
	},
	{
		Prompt:          "duplicate the src folder into src-backup",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "src",
		ExpectedDest:    "src-backup",
		ExpectedPattern: "",
	},
	{
		Prompt:          "take all python scripts out of old_repo and put them in new_repo, but leave out tests.py",
		ExpectedAction:  models.ActionMove,
		ExpectedSource:  "old_repo",
		ExpectedDest:    "new_repo",
		ExpectedPattern: "*.py",
		ExpectedExclude: "tests.py",
	},
	{
		Prompt:          "grab the logo.png file from assets/images and copy it to public/static",
		ExpectedAction:  models.ActionCopy,
		ExpectedSource:  "assets/images",
		ExpectedDest:    "public/static",
		ExpectedPattern: "logo.png",
	},
}

func contains(slice []string, val string) bool {
	cleanVal := strings.TrimLeft(val, "*")
	for _, item := range slice {
		cleanItem := strings.TrimLeft(item, "*")
		if strings.TrimRight(strings.ToLower(cleanItem), "/") == strings.ToLower(cleanVal) {
			return true
		}
	}
	return false
}

func matchPath(expected, actual string) bool {
	if expected == "" {
		return actual == "" || actual == "." || actual == "./"
	}
	// Normalise paths for basic comparison (e.g., removing trailing slashes)
	e := strings.TrimRight(strings.ReplaceAll(strings.ToLower(expected), "\\", "/"), "/")
	a := strings.TrimRight(strings.ReplaceAll(strings.ToLower(actual), "\\", "/"), "/")

	// Also strip leading "./" from actual for comparison
	a = strings.TrimPrefix(a, "./")

	// Check if expected is a substring or exactly matches the actual path
	return strings.Contains(a, e)
}

func TestLLMReliability(t *testing.T) {
	// Skip in CI, this requires a running Ollama server.
	if testing.Short() {
		t.Skip("Skipping LLM benchmark tests in short mode.")
	}

	endpoint := "http://localhost:11434"
	modelName := "qwen2.5-coder:1.5b" // Assuming this is the standard model used locally
	iterations := 10

	client := llm.NewClient(endpoint, modelName)
	client.PreloadModel()
	time.Sleep(2 * time.Second) // Give Ollama a moment to load into VRAM

	fmt.Printf("===================================================\n")
	fmt.Printf("Starting LLM Reliability Benchmark\n")
	fmt.Printf("Model: %s\n", modelName)
	fmt.Printf("Iterations per prompt: %d\n", iterations)
	fmt.Printf("===================================================\n\n")

	totalPasses := 0
	totalRuns := len(benchmarkCases) * iterations

	runBenchmark := func(cases []EvalCase, isEdge bool) float64 {
		localPasses := 0
		localRuns := len(cases) * iterations
		for i, c := range cases {
			fmt.Printf("[Case %d] %s\n", i+1, c.Prompt)
			passes := 0
			for iter := 1; iter <= iterations; iter++ {
				op, err := client.ParseRequest(c.Prompt, 1) // 1 retry to strictly evaluate raw failure rate

				if err != nil {
					fmt.Printf("  ❌ Iteration %d: Failed to parse request: %v\n", iter, err)
					continue
				}

				// Validate Action
				if op.Action != c.ExpectedAction {
					fmt.Printf("  ❌ Iteration %d: Action mismatch (Got: %s, Expected: %s)\n", iter, op.Action, c.ExpectedAction)
					continue
				}

				// Validate Source
				if !matchPath(c.ExpectedSource, op.Source) {
					fmt.Printf("  ❌ Iteration %d: Source mismatch (Got: %s, Expected to contain: %s)\n", iter, op.Source, c.ExpectedSource)
					continue
				}

				// Validate Destination (Only if not DELETE)
				if c.ExpectedAction != models.ActionDelete && !matchPath(c.ExpectedDest, op.Destination) {
					fmt.Printf("  ❌ Iteration %d: Destination mismatch (Got: %s, Expected to contain: %s)\n", iter, op.Destination, c.ExpectedDest)
					continue
				}

				// Validate Pattern
				if c.ExpectedPattern != "" {
					// The engine handles either the pattern in the patterns array,
					// or the pattern directly appended to the source path.
					patternInArray := contains(op.Patterns, c.ExpectedPattern)

					// Check if the expected pattern (e.g. "logo.png") is at the end of the source string
					// Strip the '*' from the expected pattern if we are checking the source string
					cleanPattern := strings.ReplaceAll(c.ExpectedPattern, "*", "")
					patternInSource := cleanPattern != "" && strings.HasSuffix(strings.TrimRight(op.Source, "/"), cleanPattern)

					if !patternInArray && !patternInSource {
						fmt.Printf("  ❌ Iteration %d: Missing Pattern (Got: %v, Source: %s, Expected: %s)\n", iter, op.Patterns, op.Source, c.ExpectedPattern)
						continue
					}
				}

				// Validate Exclude
				if c.ExpectedExclude != "" && !contains(op.ExcludePatterns, c.ExpectedExclude) {
					fmt.Printf("  ❌ Iteration %d: Missing Exclude (Got: %v, Expected: %s)\n", iter, op.ExcludePatterns, c.ExpectedExclude)
					continue
				}

				passes++
			}

			localPasses += passes
			totalPasses += passes
			successRate := float64(passes) / float64(iterations) * 100

			if passes == iterations {
				fmt.Printf("  ✅ 100%% Reliability (%d/%d)\n\n", passes, iterations)
			} else {
				fmt.Printf("  ⚠️  %.0f%% Reliability (%d/%d)\n\n", successRate, passes, iterations)
			}
		}
		return float64(localPasses) / float64(localRuns) * 100
	}

	coreRate := runBenchmark(benchmarkCases, false)
	fmt.Printf("\n--- Running Edge Cases ---\n\n")
	runBenchmark(edgeCases, true)

	overallRate := float64(totalPasses) / float64(totalRuns+len(edgeCases)*iterations) * 100
	fmt.Printf("===================================================\n")
	fmt.Printf("Benchmark Complete.\n")
	fmt.Printf("Overall Reliability (including edge cases): %.1f%%\n", overallRate)
	fmt.Printf("Core Usability Reliability: %.1f%%\n", coreRate)
	fmt.Printf("===================================================\n")

	if coreRate < 80.0 {
		t.Errorf("Core usability reliability is below 80%% (%.1f%%)", coreRate)
	}
}
