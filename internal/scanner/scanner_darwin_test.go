//go:build darwin

package scanner

import (
	"testing"
	"time"
)

func TestParseLsofOutput(t *testing.T) {
	raw := `COMMAND     PID   USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
node      12847 vahagn   23u  IPv4 0x1234567890      0t0  TCP *:3000 (LISTEN)
node      13201 vahagn   24u  IPv4 0x1234567891      0t0  TCP 127.0.0.1:5173 (LISTEN)
postgres    892 vahagn    5u  IPv4 0x1234567892      0t0  TCP *:5432 (LISTEN)
`

	entries := parseLsofOutput(raw)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	tests := []struct {
		idx         int
		port        int
		pid         int
		processName string
	}{
		{0, 3000, 12847, "node"},
		{1, 5173, 13201, "node"},
		{2, 5432, 892, "postgres"},
	}

	for _, tt := range tests {
		e := entries[tt.idx]
		if e.port != tt.port {
			t.Errorf("entry %d: expected port %d, got %d", tt.idx, tt.port, e.port)
		}
		if e.pid != tt.pid {
			t.Errorf("entry %d: expected pid %d, got %d", tt.idx, tt.pid, e.pid)
		}
		if e.processName != tt.processName {
			t.Errorf("entry %d: expected processName %q, got %q", tt.idx, tt.processName, e.processName)
		}
	}
}

func TestParsePortFromLsofName(t *testing.T) {
	tests := []struct {
		name     string
		expected int
	}{
		{"*:3000", 3000},
		{"127.0.0.1:5173", 5173},
		{"[::1]:8080", 8080},
		{"*:5432", 5432},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := parsePortFromLsofName(tt.name)
		if got != tt.expected {
			t.Errorf("parsePortFromLsofName(%q) = %d, want %d", tt.name, got, tt.expected)
		}
	}
}

func TestParsePsLine(t *testing.T) {
	line := "12847     1 Ss   65536 Thu Apr  3 10:30:00 2025 /usr/local/bin/node server.js"
	info := parsePsLine(line)

	if info.pid != 12847 {
		t.Errorf("expected pid 12847, got %d", info.pid)
	}
	if info.ppid != 1 {
		t.Errorf("expected ppid 1, got %d", info.ppid)
	}
	if info.stat != "Ss" {
		t.Errorf("expected stat Ss, got %q", info.stat)
	}
	if info.rss != 65536*1024 {
		t.Errorf("expected rss %d, got %d", 65536*1024, info.rss)
	}
	if info.command != "/usr/local/bin/node server.js" {
		t.Errorf("expected command %q, got %q", "/usr/local/bin/node server.js", info.command)
	}
}

func TestParseLstart(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"Thu Apr  3 10:30:00 2025", time.Date(2025, 4, 3, 10, 30, 0, 0, time.UTC)},
		{"Mon Jan 2 08:15:30 2023", time.Date(2023, 1, 2, 8, 15, 30, 0, time.UTC)},
		{"invalid", time.Time{}},
	}

	for _, tt := range tests {
		got := parseLstart(tt.input)
		if !got.Equal(tt.expected) {
			t.Errorf("parseLstart(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestParseDockerPort(t *testing.T) {
	tests := []struct {
		mapping  string
		expected int
	}{
		{"0.0.0.0:5432->5432/tcp", 5432},
		{":::6379->6379/tcp", 6379},
		{"0.0.0.0:8080->80/tcp", 8080},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := parseDockerPort(tt.mapping)
		if got != tt.expected {
			t.Errorf("parseDockerPort(%q) = %d, want %d", tt.mapping, got, tt.expected)
		}
	}
}

func TestExtractProcessName(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"/usr/local/bin/node server.js", "node"},
		{"python3 -m flask run", "python3"},
		{"postgres", "postgres"},
		{"/usr/bin/go run .", "go"},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractProcessName(tt.command)
		if got != tt.expected {
			t.Errorf("extractProcessName(%q) = %q, want %q", tt.command, got, tt.expected)
		}
	}
}

func TestParseOrphans(t *testing.T) {
	raw := `12847     1 Ss   65536 Thu Apr  3 10:30:00 2025 /usr/local/bin/node server.js
13201   100 S    32768 Thu Apr  3 11:00:00 2025 /usr/bin/python3 app.py
14000     1 Z        0 Thu Apr  3 12:00:00 2025 defunct
`
	orphans := parseOrphans(raw)

	if len(orphans) != 2 {
		t.Fatalf("expected 2 orphans, got %d", len(orphans))
	}

	if orphans[0].PID != 12847 || orphans[0].Status != StatusOrphaned {
		t.Errorf("expected PID 12847 orphaned, got PID %d status %s", orphans[0].PID, orphans[0].Status)
	}
	if orphans[1].PID != 14000 || orphans[1].Status != StatusZombie {
		t.Errorf("expected PID 14000 zombie, got PID %d status %s", orphans[1].PID, orphans[1].Status)
	}
}

func TestIntegrationListPorts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s := New()
	ports, err := s.ListPorts(false)
	if err != nil {
		t.Fatalf("ListPorts: %v", err)
	}
	// Just verify it doesn't crash and returns a valid slice
	t.Logf("Found %d listening ports", len(ports))
	for _, p := range ports {
		t.Logf("  :%d %s (PID %d)", p.Port, p.ProcessName, p.PID)
	}
}
