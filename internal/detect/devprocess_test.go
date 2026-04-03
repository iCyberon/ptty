package detect

import "testing"

func TestIsDevProcess(t *testing.T) {
	tests := []struct {
		name     string
		process  string
		command  string
		expected bool
	}{
		{"node is dev", "node", "node server.js", true},
		{"python is dev", "python3", "python3 app.py", true},
		{"go is dev", "go", "go run .", true},
		{"postgres is dev", "postgres", "postgres", true},
		{"redis is dev", "redis-server", "redis-server", true},
		{"Spotify is not dev", "Spotify", "/Applications/Spotify.app", false},
		{"Chrome is not dev", "Chrome", "/Applications/Google Chrome.app", false},
		{"Slack is not dev", "Slack", "/Applications/Slack.app", false},
		{"ControlCe is not dev", "ControlCe", "ControlCenter", false},
		{"Docker is dev", "com.docker.backend", "com.docker.backend", true},
		{"unknown with node command is dev", "unknown", "node server.js", true},
		{"unknown without dev keywords", "unknown", "/usr/bin/something", false},
		{"case insensitive allowlist", "Node", "node server.js", true},
		{"case insensitive blocklist", "slack", "/Applications/Slack.app", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDevProcess(tt.process, tt.command)
			if got != tt.expected {
				t.Errorf("IsDevProcess(%q, %q) = %v, want %v", tt.process, tt.command, got, tt.expected)
			}
		})
	}
}
