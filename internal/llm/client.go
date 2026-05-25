package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/toman92/golem-cli/internal/models"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Format   string        `json:"format"`
	Stream   bool          `json:"stream"`
}

type OllamaChatResponse struct {
	Message ChatMessage `json:"message"`
}

type TagResponse struct {
	Models []struct {
		Name    string `json:"name"`
		Details struct {
			ParameterSize string `json:"parameter_size"`
		} `json:"details"`
	} `json:"models"`
}

type Client struct {
	Endpoint string
	Model    string
}

// NewClient creates a new HTTP client wrapper to interact with the local Ollama instance.
func NewClient(endpoint, model string) *Client {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5-coder:1.5b"
	}
	return &Client{
		Endpoint: endpoint,
		Model:    model,
	}
}


// PreloadModel sends an empty request to Ollama to load the model into memory.
// This prevents the user from experiencing a 10-15 second delay on their first request.
func (c *Client) PreloadModel() {
	slog.Info("Preloading model in background", "model", c.Model)

	reqBody := map[string]string{
		"model": c.Model,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return
	}

	url := fmt.Sprintf("%s/api/generate", c.Endpoint)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err == nil {
		resp.Body.Close()
		slog.Info("Model preloaded successfully", "model", c.Model)
	}
}

func (c *Client) sendChatRequest(reqBody OllamaChatRequest) (*OllamaChatResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("Failed to marshal LLM request", "error", err)
		return nil, err
	}

	url := fmt.Sprintf("%s/api/chat", c.Endpoint)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("HTTP request to Ollama failed", "url", url, "error", err)
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "connect: ") {
			//lint:ignore ST1005 Intentionally formatted for UI display
			return nil, fmt.Errorf("\n🚨 Connection Refused: Golem cannot connect to the Ollama service at %s.\n\nPlease ensure that your Ollama service or Docker container is running and accessible before using Golem.", c.Endpoint)
		}
		return nil, fmt.Errorf("failed to contact Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Error("Ollama API returned non-200 status", "status", resp.StatusCode, "body", string(bodyBytes))
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp OllamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		slog.Error("Failed to decode Ollama response JSON", "error", err)
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return &ollamaResp, nil
}

// ParseRequest sends the natural language string to Ollama with a strict JSON format request.
// It parses the structured output into our internal models.Operation struct.
// We use 'format: "json"' in the API payload to guarantee structured output from the LLM.
// Every request is stateless to prevent the LLM from hallucinating patterns based on previous requests.
func (c *Client) ParseRequest(userRequest string, maxRetries int) (*models.Operation, error) {
	if maxRetries < 1 {
		maxRetries = 1
	}

	var lastErr error
	var lastResp string

	userMsg := ChatMessage{Role: "user", Content: userRequest}

	for i := 0; i < maxRetries; i++ {
		slog.Info("Sending request to LLM", "model", c.Model, "endpoint", c.Endpoint, "attempt", i+1)

		messages := []ChatMessage{
			{Role: "system", Content: GetSystemPrompt()},
			userMsg,
		}

		reqBody := OllamaChatRequest{
			Model:    c.Model,
			Messages: messages,
			Format:   "json",
			Stream:   false,
		}

		ollamaResp, err := c.sendChatRequest(reqBody)
		if err != nil {
			return nil, err
		}

		var op models.Operation
		if err := json.Unmarshal([]byte(ollamaResp.Message.Content), &op); err != nil {
			lastErr = err
			lastResp = ollamaResp.Message.Content
			slog.Warn("Failed to unmarshal LLM string output into Operation struct", "attempt", i+1, "error", err)
			continue
		}

		postProcessOperation(&op)

		if op.Action != models.ActionCopy && op.Action != models.ActionMove && op.Action != models.ActionDelete {
			lastErr = fmt.Errorf("invalid action: %s", op.Action)
			lastResp = ollamaResp.Message.Content
			slog.Warn("Invalid Action returned by LLM", "attempt", i+1, "action", op.Action)
			continue
		}

		slog.Info("LLM successfully parsed request", "action", op.Action, "source", op.Source, "destination", op.Destination)

		return &op, nil
	}

	slog.Error("LLM exhausted retries", "maxRetries", maxRetries, "model", c.Model)
	//lint:ignore ST1005 Intentionally formatted for UI display
	friendlyErr := fmt.Errorf("The LLM failed to return a valid JSON plan after %d attempts.\n\nError: %v\n\nThe model '%s' might be hallucinating unsupported formats (like JSON Schemas). Try using the '/config model' command in the shell to switch to a more capable lightweight model like 'qwen2.5-coder:1.5b'.\n\nRaw Model Output:\n%s", maxRetries, lastErr, c.Model, lastResp)
	return nil, friendlyErr
}

// isUnder3B parses the parameter size string (e.g. "8.2B", "400M") and returns true if it's <= 3B.
// This constraint exists to ensure the local LLM executes instantaneously and doesn't drain
// system resources for simple file organization tasks.
func isUnder3B(size string) bool {
	size = strings.ToUpper(strings.TrimSpace(size))
	if size == "" {
		return false // ignore unknown sizes
	}

	if strings.HasSuffix(size, "B") {
		numStr := strings.TrimSuffix(size, "B")
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return false
		}
		return val <= 3.0
	}

	// If it's in Millions (M) or Thousands (K), it's definitely under 3B
	if strings.HasSuffix(size, "M") || strings.HasSuffix(size, "K") {
		return true
	}

	return false
}

// ListSmallModels queries Ollama for available models and filters the list
// down to models with <= 3B parameters.
func (c *Client) ListSmallModels() ([]string, error) {
	slog.Info("Fetching available models from Ollama")

	url := fmt.Sprintf("%s/api/tags", c.Endpoint)
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("Failed to fetch tags from Ollama", "error", err)
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "connect: ") {
			//lint:ignore ST1005 Intentionally formatted for UI display
			return nil, fmt.Errorf("\n🚨 Connection Refused: Golem cannot connect to the Ollama service at %s.\n\nPlease ensure that your Ollama service or Docker container is running and accessible before using Golem.", c.Endpoint)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models, status code: %d", resp.StatusCode)
	}

	var tags TagResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	var smallModels []string
	for _, m := range tags.Models {
		if isUnder3B(m.Details.ParameterSize) {
			smallModels = append(smallModels, m.Name)
		}
	}

	slog.Info("Found qualifying small models", "count", len(smallModels))
	return smallModels, nil
}

func postProcessOperation(op *models.Operation) {
	// 1. Action Normalization
	actionStr := strings.ToUpper(string(op.Action))
	switch actionStr {
	case "DUPLICATE", "CLONE", "GRAB", "EXTRACT":
		op.Action = models.ActionCopy
	case "TRANSFER", "RELOCATE", "TAKE":
		op.Action = models.ActionMove
	case "TRASH", "WIPE", "REMOVE":
		op.Action = models.ActionDelete
	default:
		op.Action = models.Action(actionStr)
	}

	// 2. Wildcard Expansion
	for i, pat := range op.Patterns {
		pat = strings.TrimSpace(pat)
		if strings.HasPrefix(pat, ".") && !strings.Contains(pat, "/") {
			op.Patterns[i] = "*" + pat
		}
	}

	for i, pat := range op.ExcludePatterns {
		pat = strings.TrimSpace(pat)
		if strings.HasPrefix(pat, ".") && !strings.Contains(pat, "/") {
			op.ExcludePatterns[i] = "*" + pat
		}
	}
}
