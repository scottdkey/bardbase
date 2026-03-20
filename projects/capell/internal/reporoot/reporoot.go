// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Package reporoot provides a helper to locate the monorepo root directory.
package reporoot

import (
	"os"
	"path/filepath"
)

// Find walks up from the working directory to find the monorepo root
// (the first parent containing a .git directory). Falls back to the
// working directory if no .git is found.
func Find() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	wd, _ := os.Getwd()
	return wd
}
