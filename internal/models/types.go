package models

type Action string

const (
	ActionCopy   Action = "COPY"
	ActionMove   Action = "MOVE"
	ActionDelete Action = "DELETE"
)

type Operation struct {
	Action          Action   `json:"action"`
	Source          string   `json:"source"`
	Destination     string   `json:"destination"`
	Patterns        []string `json:"patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
}
