package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DetectFramework identifies the framework using a tiered strategy:
// Docker image > project file deps > command keywords > process name.
func DetectFramework(projectRoot, cwd, command, processName, dockerImage string) string {
	if fw := detectFromDockerImage(dockerImage); fw != "" {
		return fw
	}

	if projectRoot == "" {
		projectRoot = FindProjectRoot(cwd)
	}
	if fw := detectFromProjectFiles(projectRoot); fw != "" {
		return fw
	}

	if fw := detectFromCommand(command); fw != "" {
		return fw
	}

	return detectFromProcessName(processName)
}

var dockerImageMap = map[string]string{
	"postgres":               "PostgreSQL",
	"redis":                  "Redis",
	"mongo":                  "MongoDB",
	"mysql":                  "MySQL",
	"mariadb":                "MariaDB",
	"nginx":                  "nginx",
	"localstack":             "LocalStack",
	"rabbitmq":               "RabbitMQ",
	"confluentinc/cp-kafka":  "Kafka",
	"elasticsearch":          "Elasticsearch",
	"minio":                  "MinIO",
	"memcached":              "Memcached",
	"cassandra":              "Cassandra",
	"consul":                 "Consul",
	"vault":                  "Vault",
}

func detectFromDockerImage(image string) string {
	if image == "" {
		return ""
	}
	lower := strings.ToLower(image)
	for pattern, name := range dockerImageMap {
		if strings.Contains(lower, pattern) {
			return name
		}
	}
	return ""
}

var packageJSONFrameworks = map[string]string{
	"next":               "Next.js",
	"nuxt":               "Nuxt",
	"@sveltejs/kit":      "SvelteKit",
	"svelte":             "Svelte",
	"@remix-run/react":   "Remix",
	"astro":              "Astro",
	"vite":               "Vite",
	"@angular/core":      "Angular",
	"vue":                "Vue",
	"react":              "React",
	"express":            "Express",
	"fastify":            "Fastify",
	"hono":               "Hono",
	"koa":                "Koa",
	"@nestjs/core":       "NestJS",
	"gatsby":             "Gatsby",
	"webpack":            "Webpack",
	"esbuild":            "esbuild",
	"parcel":             "Parcel",
	"@redwoodjs/core":    "RedwoodJS",
}

func detectFromProjectFiles(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}

	pkgPath := filepath.Join(projectRoot, "package.json")
	if fw := detectFromPackageJSON(pkgPath); fw != "" {
		return fw
	}

	checks := []struct {
		file      string
		framework string
	}{
		{"Cargo.toml", "Rust"},
		{"go.mod", "Go"},
		{"mix.exs", "Elixir"},
		{"composer.json", "PHP"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(projectRoot, c.file)); err == nil {
			return c.framework
		}
	}

	if fw := detectPythonFramework(projectRoot); fw != "" {
		return fw
	}

	if _, err := os.Stat(filepath.Join(projectRoot, "Gemfile")); err == nil {
		return detectRubyFramework(projectRoot)
	}

	for _, f := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(projectRoot, f)); err == nil {
			return "Java"
		}
	}

	return ""
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func detectFromPackageJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}

	// Ordered by specificity so more specific frameworks match first
	priority := []string{
		"next", "nuxt", "@sveltejs/kit", "@remix-run/react", "astro", "gatsby",
		"@angular/core", "@nestjs/core", "@redwoodjs/core",
		"svelte", "vue", "react",
		"express", "fastify", "hono", "koa",
		"vite", "webpack", "esbuild", "parcel",
	}

	allDeps := make(map[string]bool)
	for k := range pkg.Dependencies {
		allDeps[k] = true
	}
	for k := range pkg.DevDependencies {
		allDeps[k] = true
	}

	for _, dep := range priority {
		if allDeps[dep] {
			if fw, ok := packageJSONFrameworks[dep]; ok {
				return fw
			}
		}
	}

	return ""
}

func detectPythonFramework(root string) string {
	for _, f := range []string{"pyproject.toml", "requirements.txt", "setup.py"} {
		path := filepath.Join(root, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		switch {
		case strings.Contains(content, "django"):
			return "Django"
		case strings.Contains(content, "flask"):
			return "Flask"
		case strings.Contains(content, "fastapi"):
			return "FastAPI"
		case strings.Contains(content, "starlette"):
			return "Starlette"
		default:
			return "Python"
		}
	}
	return ""
}

func detectRubyFramework(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "Gemfile"))
	if err != nil {
		return "Ruby"
	}
	content := strings.ToLower(string(data))
	if strings.Contains(content, "rails") {
		return "Rails"
	}
	if strings.Contains(content, "sinatra") {
		return "Sinatra"
	}
	return "Ruby"
}

var commandKeywordMap = map[string]string{
	"next":      "Next.js",
	"vite":      "Vite",
	"nuxt":      "Nuxt",
	"angular":   "Angular",
	"webpack":   "Webpack",
	"remix":     "Remix",
	"astro":     "Astro",
	"gatsby":    "Gatsby",
	"flask":     "Flask",
	"django":    "Django",
	"manage.py": "Django",
	"uvicorn":   "Uvicorn",
	"gunicorn":  "Gunicorn",
	"rails":     "Rails",
	"cargo":     "Rust",
	"rustc":     "Rust",
	"spring":    "Spring",
	"gradlew":   "Java",
	"mvn":       "Java",
}

func detectFromCommand(command string) string {
	if command == "" {
		return ""
	}
	lower := strings.ToLower(command)
	for keyword, framework := range commandKeywordMap {
		if strings.Contains(lower, keyword) {
			return framework
		}
	}
	return ""
}

var processNameMap = map[string]string{
	"node":         "Node.js",
	"python":       "Python",
	"python3":      "Python",
	"ruby":         "Ruby",
	"java":         "Java",
	"go":           "Go",
	"cargo":        "Rust",
	"deno":         "Deno",
	"bun":          "Bun",
	"php":          "PHP",
	"dotnet":       "dotnet",
	"elixir":       "Elixir",
	"beam.smp":     "Erlang/Elixir",
	"postgres":     "PostgreSQL",
	"redis-server": "Redis",
	"mongod":       "MongoDB",
	"mysqld":       "MySQL",
	"nginx":        "nginx",
	"caddy":        "Caddy",
	"httpd":        "Apache",
	"apache2":      "Apache",
}

func detectFromProcessName(name string) string {
	if name == "" {
		return ""
	}
	if fw, ok := processNameMap[strings.ToLower(name)]; ok {
		return fw
	}
	return ""
}
