package operation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetFileName returns the base name of the file from the provided file path.
func GetFileName(path string) string {
	return filepath.Base(path)
}

// GetHostDir returns the directory part of the given path as an absolute path if possible, falling back to the relative directory.
func GetHostDir(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Dir(path)
	}
	return filepath.Dir(absPath)
}

// ValidateFilePath checks if the provided file path is valid and does not contain illegal or unsafe patterns.
func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed in '%s'", path)
	}
	return nil
}

// GetFileSize returns the size of a file
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
