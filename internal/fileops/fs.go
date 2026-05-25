package fileops

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// fileHash computes a SHA-256 hash for the given file.
// Used exclusively during copy/move operations to detect content-level collisions,
// preventing duplicate operations while safely handling name conflicts.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// handleCollision generates a new unique file path by appending a timestamp.
// This ensures that when a file with the same name but different content is moved,
// we don't accidentally overwrite existing data.
func handleCollision(destPath string) string {
	ext := filepath.Ext(destPath)
	base := strings.TrimSuffix(destPath, ext)
	counter := 2
	for {
		newPath := fmt.Sprintf("%s_%d%s", base, counter, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

// CopyFile duplicates a file from src to dst.
// It automatically detects collisions; if a file at the destination has identical content,
// it skips the copy. If the content differs, it renames the destination file to preserve both.
// Returns the absolute final destination path used.
func CopyFile(src, dst string, strategy string) (string, bool, error) {
	slog.Info("Copying file", "src", src, "dst", dst)

	srcStat, err := os.Stat(src)
	if err != nil {
		slog.Error("Failed to stat source file", "src", src, "error", err)
		return "", false, err
	}
	if !srcStat.Mode().IsRegular() {
		err := fmt.Errorf("%s is not a regular file", src)
		slog.Error("Source is not a regular file", "src", src)
		return "", false, err
	}

	actualDst := dst
	skipped := false
	if dstStat, err := os.Stat(actualDst); err == nil {
		if strategy == "s" {
			slog.Info("Skipping existing file per user strategy", "dst", actualDst)
			return actualDst, true, nil
		} else if strategy == "r" {
			slog.Info("Replacing existing file per user strategy", "dst", actualDst)
			os.Remove(actualDst)
		} else {
			// strategy "c" or default. Always create copy for "c" unless identical.
			if srcStat.Size() == dstStat.Size() {
				srcHash, err1 := fileHash(src)
				dstHash, err2 := fileHash(actualDst)
				if err1 == nil && err2 == nil && srcHash == dstHash {
					slog.Info("Collision detected but contents are identical. Skipping copy.", "dst", actualDst)
					return actualDst, true, nil
				}
			}
			actualDst = handleCollision(actualDst)
			slog.Info("Collision detected, renamed destination to avoid overwrite", "newDst", actualDst)
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		slog.Error("Failed to open source file", "src", src, "error", err)
		return "", false, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(actualDst)
	if err != nil {
		slog.Error("Failed to create destination file", "dst", actualDst, "error", err)
		return "", false, err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		slog.Error("Failed to write to destination", "dst", actualDst, "error", err)
		return "", false, err
	}
	
	slog.Info("Copy successful", "finalDst", actualDst)
	return actualDst, skipped, nil
}

// MoveFile relocates a file from src to dst.
// Like CopyFile, it resolves naming conflicts by renaming the target if content differs,
// or silently drops the operation if identical content already exists at the destination.
func MoveFile(src, dst string, strategy string) (string, bool, error) {
	slog.Info("Moving file", "src", src, "dst", dst)

	srcStat, statErr := os.Stat(src)
	if statErr != nil {
		slog.Error("Failed to stat source file", "src", src, "error", statErr)
		return "", false, statErr
	}

	actualDst := dst
	skipped := false
	if dstStat, err := os.Stat(actualDst); err == nil {
		if strategy == "s" {
			slog.Info("Skipping existing file per user strategy. Preserving source.", "dst", actualDst)
			return actualDst, true, nil
		} else if strategy == "r" {
			slog.Info("Replacing existing file per user strategy", "dst", actualDst)
			os.Remove(actualDst)
		} else {
			// strategy "c" or default. Always create copy for "c" unless identical.
			if srcStat.Size() == dstStat.Size() {
				srcHash, err1 := fileHash(src)
				dstHash, err2 := fileHash(actualDst)
				if err1 == nil && err2 == nil && srcHash == dstHash {
					slog.Info("Collision detected but contents are identical. Dropping move to preserve identical destination.", "dst", actualDst)
					// File already exists with exact content. We can safely remove the source to complete the "move"
					os.Remove(src)
					return actualDst, true, nil
				}
			}
			actualDst = handleCollision(actualDst)
			slog.Info("Collision detected, renamed destination to avoid overwrite", "newDst", actualDst)
		}
	}

	err := os.Rename(src, actualDst)
	if err != nil {
		slog.Info("Rename failed (likely cross-device), falling back to Copy+Delete", "error", err)
		actualDst, skipped, err = CopyFile(src, actualDst, strategy)
		if err != nil {
			return "", false, err
		}
		// ONLY remove source if the copy actually occurred and wasn't skipped
		if !skipped {
			os.Remove(src)
		}
	}
	slog.Info("Move successful", "finalDst", actualDst)
	return actualDst, skipped, nil
}

// MatchPattern verifies if a filename matches any provided glob pattern or keyword.
// We fall back to simple string containment if glob matching fails, making the CLI
// much more robust for natural language inputs (e.g. "move invoices" matches "jan_invoice.pdf").
func MatchPattern(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		// The LLM sometimes hallucinates directory paths or globstars inside the pattern list.
		// Since we are already recursively walking the source directory and matching on the
		// base file name, we only care about the base pattern.
		cleanP := filepath.Base(p)
		cleanP = strings.ReplaceAll(cleanP, "**", "*")

		matched, err := filepath.Match(cleanP, name)
		if err == nil && matched {
			return true
		}
		if strings.Contains(strings.ToLower(name), strings.ToLower(cleanP)) {
			return true
		}
	}
	return false
}

// FindFiles recursively searches a directory for files matching the given patterns.
// It skips hidden files and directories if includeHidden is false.
// It explicitly skips any directory whose name is in excludeDirs.
// It ignores any file that matches excludePatterns.
// Returns a slice of absolute or relative paths to files matching the criteria.
func FindFiles(dir string, patterns []string, excludePatterns []string, includeHidden bool, excludeDirs []string) ([]string, error) {
	slog.Info("Starting file search", "dir", dir, "patterns", patterns, "excludePatterns", excludePatterns, "includeHidden", includeHidden)
	
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			slog.Error("WalkDir encountered error", "path", path, "error", err)
			return err
		}

		// Hidden file protection: skip files and directories starting with '.'
		if !includeHidden {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != ".." {
				if d.IsDir() {
					slog.Info("Skipping hidden directory", "dir", path)
					return filepath.SkipDir
				}
				slog.Info("Skipping hidden file", "file", path)
				return nil
			}
		}

		if d.IsDir() && d.Name() != "." && d.Name() != ".." {
			for _, ex := range excludeDirs {
				if d.Name() == ex {
					slog.Info("Skipping excluded directory", "dir", path)
					return filepath.SkipDir
				}
			}
		}

		if !d.IsDir() {
			if len(excludePatterns) > 0 && MatchPattern(d.Name(), excludePatterns) {
				return nil // Skip this file because it matches an exclude pattern
			}
			if MatchPattern(d.Name(), patterns) {
				files = append(files, path)
			}
		}
		return nil
	})
	
	if err != nil {
		slog.Error("File search failed", "error", err)
	} else {
		slog.Info("File search completed", "foundCount", len(files))
	}
	
	return files, err
}

// FindDirectories recursively searches for directories matching the given patterns.
// It is specifically used for the DELETE action. It skips recursing INTO matched directories
// since they will be deleted entirely. It ignores directories matching excludePatterns.
func FindDirectories(dir string, patterns []string, excludePatterns []string, includeHidden bool) ([]string, error) {
	slog.Info("Starting directory search", "dir", dir, "patterns", patterns, "excludePatterns", excludePatterns)
	
	var dirs []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !includeHidden {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != ".." {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() && d.Name() != "." && d.Name() != ".." {
			if len(excludePatterns) > 0 && MatchPattern(d.Name(), excludePatterns) {
				return filepath.SkipDir // Skip this directory because it matches an exclude pattern
			}
			if MatchPattern(d.Name(), patterns) {
				dirs = append(dirs, path)
				return filepath.SkipDir // Don't recurse into a dir we are going to delete
			}
		}
		return nil
	})
	
	return dirs, err
}
