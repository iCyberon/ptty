package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot(t *testing.T) {
	// Create a temp directory structure
	root := t.TempDir()
	nested := filepath.Join(root, "src", "components")
	os.MkdirAll(nested, 0755)

	// Create a marker file at root
	os.WriteFile(filepath.Join(root, "package.json"), []byte(`{}`), 0644)

	// Should find root from nested dir
	got := FindProjectRoot(nested)
	if got != root {
		t.Errorf("FindProjectRoot(%q) = %q, want %q", nested, got, root)
	}

	// Should find root from root itself
	got = FindProjectRoot(root)
	if got != root {
		t.Errorf("FindProjectRoot(%q) = %q, want %q", root, got, root)
	}

	// Should return empty for unrelated dir
	other := t.TempDir()
	got = FindProjectRoot(other)
	if got != "" {
		t.Errorf("FindProjectRoot(%q) = %q, want empty", other, got)
	}

	// Empty input
	got = FindProjectRoot("")
	if got != "" {
		t.Errorf("FindProjectRoot(\"\") = %q, want empty", got)
	}
}

func TestProjectName(t *testing.T) {
	tests := []struct {
		root     string
		expected string
	}{
		{"/Users/dev/my-app", "my-app"},
		{"/home/user/projects/api-server", "api-server"},
		{"", ""},
	}

	for _, tt := range tests {
		got := ProjectName(tt.root)
		if got != tt.expected {
			t.Errorf("ProjectName(%q) = %q, want %q", tt.root, got, tt.expected)
		}
	}
}

func TestFindProjectRootGoMod(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "internal", "scanner")
	os.MkdirAll(nested, 0755)
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test"), 0644)

	got := FindProjectRoot(nested)
	if got != root {
		t.Errorf("FindProjectRoot(%q) = %q, want %q", nested, got, root)
	}
}
