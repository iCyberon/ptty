package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFromDockerImage(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"postgres:15", "PostgreSQL"},
		{"redis:7-alpine", "Redis"},
		{"mongo:latest", "MongoDB"},
		{"mysql:8.0", "MySQL"},
		{"nginx:latest", "nginx"},
		{"localstack/localstack", "LocalStack"},
		{"confluentinc/cp-kafka:7.0", "Kafka"},
		{"minio/minio", "MinIO"},
		{"unknown-image", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := detectFromDockerImage(tt.image)
		if got != tt.expected {
			t.Errorf("detectFromDockerImage(%q) = %q, want %q", tt.image, got, tt.expected)
		}
	}
}

func TestDetectFromPackageJSON(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			"Next.js project",
			`{"dependencies": {"next": "14.0.0", "react": "18.0.0"}}`,
			"Next.js",
		},
		{
			"Vite project",
			`{"devDependencies": {"vite": "5.0.0"}, "dependencies": {"react": "18.0.0"}}`,
			"React",
		},
		{
			"Express API",
			`{"dependencies": {"express": "4.18.0"}}`,
			"Express",
		},
		{
			"Angular project",
			`{"dependencies": {"@angular/core": "17.0.0"}}`,
			"Angular",
		},
		{
			"NestJS project",
			`{"dependencies": {"@nestjs/core": "10.0.0"}}`,
			"NestJS",
		},
		{
			"Empty package.json",
			`{}`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "package.json")
			os.WriteFile(path, []byte(tt.content), 0644)

			got := detectFromPackageJSON(path)
			if got != tt.expected {
				t.Errorf("detectFromPackageJSON() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectFromCommand(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"next dev", "Next.js"},
		{"/usr/local/bin/vite build", "Vite"},
		{"python -m flask run", "Flask"},
		{"python manage.py runserver", "Django"},
		{"rails server", "Rails"},
		{"cargo run", "Rust"},
		{"", ""},
		{"some random command", ""},
	}

	for _, tt := range tests {
		got := detectFromCommand(tt.command)
		if got != tt.expected {
			t.Errorf("detectFromCommand(%q) = %q, want %q", tt.command, got, tt.expected)
		}
	}
}

func TestDetectFromProcessName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"node", "Node.js"},
		{"python3", "Python"},
		{"postgres", "PostgreSQL"},
		{"redis-server", "Redis"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		got := detectFromProcessName(tt.name)
		if got != tt.expected {
			t.Errorf("detectFromProcessName(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestDetectFrameworkIntegration(t *testing.T) {
	// Create a Next.js project
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"),
		[]byte(`{"dependencies": {"next": "14.0.0", "react": "18.0.0"}}`), 0644)

	got := DetectFramework("", dir, "node .next/server.js", "node", "")
	if got != "Next.js" {
		t.Errorf("DetectFramework for Next.js project = %q, want %q", got, "Next.js")
	}

	// Docker takes priority
	got = DetectFramework("", dir, "", "node", "postgres:15")
	if got != "PostgreSQL" {
		t.Errorf("DetectFramework with docker image = %q, want %q", got, "PostgreSQL")
	}
}

func TestDetectPythonFramework(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "requirements.txt"),
		[]byte("flask==3.0.0\nrequests==2.31.0"), 0644)

	got := detectPythonFramework(dir)
	if got != "Flask" {
		t.Errorf("detectPythonFramework = %q, want %q", got, "Flask")
	}
}
