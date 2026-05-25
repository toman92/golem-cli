package ui

import "github.com/fatih/color"

// Theme holds the configured colors for different UI elements
type Theme struct {
	LLM    *color.Color
	Plan     *color.Color
	Success  *color.Color
	Error    *color.Color
	Warning  *color.Color
	Question *color.Color
	UserMsg  *color.Color
	Prefix   string
}

// CurrentTheme is the active UI theme
var CurrentTheme *Theme

func init() {
	// Initialize default vibrant theme
	CurrentTheme = &Theme{
		LLM:    color.New(color.FgCyan, color.Bold),
		Plan:     color.New(color.FgMagenta),
		Success:  color.New(color.FgGreen, color.Bold),
		Error:    color.New(color.FgRed, color.Bold),
		Warning:  color.New(color.FgYellow),
		Question: color.New(color.FgHiBlue),
		UserMsg:  color.New(color.FgWhite, color.Bold),
		Prefix:   "golem ❯ ",
	}
}

// PrintLLM prints messages from the LLM or engine (like "Thinking...")
func PrintLLM(format string, a ...interface{}) {
	CurrentTheme.LLM.Printf(format, a...)
}

// PrintPlan prints proposed actions
func PrintPlan(format string, a ...interface{}) {
	CurrentTheme.Plan.Printf(format, a...)
}

// PrintSuccess prints successful actions
func PrintSuccess(format string, a ...interface{}) {
	CurrentTheme.Success.Printf(format, a...)
}

// PrintError prints errors
func PrintError(format string, a ...interface{}) {
	CurrentTheme.Error.Printf(format, a...)
}

// PrintWarning prints warnings
func PrintWarning(format string, a ...interface{}) {
	CurrentTheme.Warning.Printf(format, a...)
}

// PrintUser prints the user's echoed message
func PrintUser(format string, a ...interface{}) {
	CurrentTheme.UserMsg.Printf(format, a...)
}
