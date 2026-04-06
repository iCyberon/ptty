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
	"pubspec.yaml",
	"Package.swift",
	"build.sbt",
	"stack.yaml",
	"shard.yml",
	"gleam.toml",
	"dune-project",
	"cpanfile",
	"deps.edn",
	"project.clj",
	"build.zig",
	"Makefile",
}

// projectMarkerGlobs are glob patterns for project files with variable names.
var projectMarkerGlobs = []string{
	"*.sln",
	"*.csproj",
	"*.fsproj",
	"*.cabal",
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
		for _, pattern := range projectMarkerGlobs {
			matches, _ := filepath.Glob(filepath.Join(current, pattern))
			if len(matches) > 0 {
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
