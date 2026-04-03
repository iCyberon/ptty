package detect

import (
	"os"
	"path/filepath"
)

var projectMarkers = []string{
	"package.json",
	"Cargo.toml",
	"go.mod",
	"pyproject.toml",
	"requirements.txt",
	"Gemfile",
	"pom.xml",
	"build.gradle",
	"build.gradle.kts",
	"composer.json",
	"mix.exs",
	"Makefile",
}

// FindProjectRoot walks up from dir looking for project marker files.
func FindProjectRoot(dir string) string {
	if dir == "" {
		return ""
	}

	current := dir
	for i := 0; i < 15; i++ {
		for _, marker := range projectMarkers {
			path := filepath.Join(current, marker)
			if _, err := os.Stat(path); err == nil {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}

func ProjectName(root string) string {
	if root == "" {
		return ""
	}
	return filepath.Base(root)
}
