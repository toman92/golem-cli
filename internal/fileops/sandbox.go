package fileops

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

// ResolvePath securely computes the absolute target path relative to the sandbox root.
// It returns the clean absolute path and a boolean flag indicating if the target
// escapes the designated sandbox root. This logic exists to prevent arbitrary file
// access (e.g. Directory Traversal via "../") outside the allowed workspace, enforcing our Security-First standard.
func ResolvePath(sandboxRoot, targetPath string) (string, bool, error) {
	slog.Info("Resolving path", "sandboxRoot", sandboxRoot, "targetPath", targetPath)

	absRoot, err := filepath.Abs(sandboxRoot)
	if err != nil {
		slog.Error("Failed to get absolute path for sandbox root", "error", err)
		return "", false, fmt.Errorf("failed to get absolute sandbox root: %w", err)
	}

	var absTarget string
	if filepath.IsAbs(targetPath) {
		absTarget = filepath.Clean(targetPath)
	} else {
		absTarget = filepath.Clean(filepath.Join(absRoot, targetPath))
	}

	rootWithSep := absRoot
	if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
		rootWithSep += string(filepath.Separator)
	}

	if absTarget == absRoot {
		slog.Info("Target is exactly sandbox root", "absTarget", absTarget)
		return absTarget, false, nil
	}

	targetWithSep := absTarget
	if !strings.HasSuffix(targetWithSep, string(filepath.Separator)) {
		targetWithSep += string(filepath.Separator)
	}

	escapes := !strings.HasPrefix(targetWithSep, rootWithSep)
	if escapes {
		slog.Info("Path escape detected", "absTarget", absTarget, "absRoot", absRoot)
	} else {
		slog.Info("Path resolved securely", "absTarget", absTarget)
	}

	return absTarget, escapes, nil
}
