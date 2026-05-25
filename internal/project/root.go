// Package project provides project-level utilities for git-vine.
package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot traverses upward from currentDir until it finds a directory
// containing a .wtf subdirectory (the git-vine project root).
func FindRoot(currentDir string) (string, error) {
	dir := currentDir
	for {
		wtfPath := filepath.Join(dir, ".wtf")
		if info, err := os.Stat(wtfPath); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside a git-vine project (no .wtf directory found)")
		}
		dir = parent
	}
}
