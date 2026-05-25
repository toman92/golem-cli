package logger

import (
	"log/slog"
	"os"
	"path/filepath"
)

// Init initializes a global slog logger that writes to a file in append mode.
// We use a file-based structured logger to ensure comprehensive observability
// (e.g., tracking path resolutions and LLM request failures) without cluttering
// the CLI's standard output, which should remain clean for interactive use.
func Init(logFilePath string) error {
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Logger initialized")
	return nil
}
