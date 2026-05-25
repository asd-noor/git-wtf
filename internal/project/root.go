// Package project provides project-level utilities for git-wtf.
package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot traverses upward from currentDir until it finds a directory
// containing a .bare subdirectory (the git-wtf project root).
func FindRoot(currentDir string) (string, error) {
	for {
		barePath := filepath.Join(currentDir, ".bare")
		if info, err := os.Stat(barePath); err == nil && info.IsDir() {
			return currentDir, nil
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			return "", fmt.Errorf("not inside a managed git-wtf project (no .bare directory found)")
		}
		currentDir = parent
	}
}
