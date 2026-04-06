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
	// Core databases
	"postgres":               "PostgreSQL",
	"redis":                  "Redis",
	"mongo":                  "MongoDB",
	"mysql":                  "MySQL",
	"mariadb":                "MariaDB",
	"clickhouse":             "ClickHouse",
	"neo4j":                  "Neo4j",
	"cockroachdb":            "CockroachDB",
	"couchdb":                "CouchDB",
	"influxdb":               "InfluxDB",
	"scylladb":               "ScyllaDB",
	"arangodb":               "ArangoDB",
	"cassandra":              "Cassandra",
	"valkey":                 "Valkey",
	"dragonfly":              "Dragonfly",
	"surrealdb":              "SurrealDB",
	// Vector databases
	"qdrant":                 "Qdrant",
	"chromadb":               "ChromaDB",
	"weaviate":               "Weaviate",
	"milvusdb":               "Milvus",
	// Search
	"elasticsearch":          "Elasticsearch",
	"opensearch":             "OpenSearch",
	"typesense":              "Typesense",
	"meilisearch":            "Meilisearch",
	// Messaging
	"rabbitmq":               "RabbitMQ",
	"confluentinc/cp-kafka":  "Kafka",
	"nats":                   "NATS",
	"pulsar":                 "Apache Pulsar",
	"zookeeper":              "ZooKeeper",
	// Object storage & caching
	"minio":                  "MinIO",
	"memcached":              "Memcached",
	"localstack":             "LocalStack",
	// Proxies & web servers
	"nginx":                  "nginx",
	"traefik":                "Traefik",
	"haproxy":                "HAProxy",
	"envoyproxy":             "Envoy",
	// Observability
	"grafana":                "Grafana",
	"prom/prometheus":        "Prometheus",
	"jaegertracing":          "Jaeger",
	// Infrastructure
	"consul":                 "Consul",
	"vault":                  "Vault",
	"keycloak":               "Keycloak",
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
	"solid-js":           "Solid",
	"@builder.io/qwik":   "Qwik",
	"express":            "Express",
	"fastify":            "Fastify",
	"hono":               "Hono",
	"koa":                "Koa",
	"@nestjs/core":       "NestJS",
	"@adonisjs/core":     "Adonis",
	"gatsby":             "Gatsby",
	"@11ty/eleventy":     "Eleventy",
	"strapi":             "Strapi",
	"payload":            "Payload",
	"electron":           "Electron",
	"expo":               "Expo",
	"@tauri-apps/cli":    "Tauri",
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
		{"Package.swift", "Swift"},
		{"build.sbt", "Scala"},
		{"shard.yml", "Crystal"},
		{"gleam.toml", "Gleam"},
		{"dune-project", "OCaml"},
		{"cpanfile", "Perl"},
		{"deps.edn", "Clojure"},
		{"project.clj", "Clojure"},
		{"build.zig", "Zig"},
		{"stack.yaml", "Haskell"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(projectRoot, c.file)); err == nil {
			return c.framework
		}
	}

	// Haskell via .cabal glob
	if matches, _ := filepath.Glob(filepath.Join(projectRoot, "*.cabal")); len(matches) > 0 {
		return "Haskell"
	}

	if fw := detectDartFramework(projectRoot); fw != "" {
		return fw
	}

	if fw := detectPHPFramework(projectRoot); fw != "" {
		return fw
	}

	if fw := detectPythonFramework(projectRoot); fw != "" {
		return fw
	}

	if _, err := os.Stat(filepath.Join(projectRoot, "Gemfile")); err == nil {
		return detectRubyFramework(projectRoot)
	}

	if fw := detectJavaFramework(projectRoot); fw != "" {
		return fw
	}

	if fw := detectDotnetFramework(projectRoot); fw != "" {
		return fw
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
		"@angular/core", "@nestjs/core", "@redwoodjs/core", "@adonisjs/core",
		"@builder.io/qwik", "@11ty/eleventy", "@tauri-apps/cli",
		"electron", "expo", "strapi", "payload",
		"svelte", "solid-js", "vue", "react",
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
		case strings.Contains(content, "tornado"):
			return "Tornado"
		case strings.Contains(content, "sanic"):
			return "Sanic"
		case strings.Contains(content, "streamlit"):
			return "Streamlit"
		case strings.Contains(content, "gradio"):
			return "Gradio"
		case strings.Contains(content, "celery"):
			return "Celery"
		case strings.Contains(content, "aiohttp"):
			return "aiohttp"
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
	if strings.Contains(content, "hanami") {
		return "Hanami"
	}
	return "Ruby"
}

func detectDartFramework(root string) string {
	path := filepath.Join(root, "pubspec.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := strings.ToLower(string(data))
	if strings.Contains(content, "flutter") {
		return "Flutter"
	}
	return "Dart"
}

func detectPHPFramework(root string) string {
	path := filepath.Join(root, "composer.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := strings.ToLower(string(data))
	switch {
	case strings.Contains(content, "laravel/framework"):
		return "Laravel"
	case strings.Contains(content, "symfony/framework-bundle"):
		return "Symfony"
	case strings.Contains(content, "slim/slim"):
		return "Slim"
	case strings.Contains(content, "cakephp/cakephp"):
		return "CakePHP"
	}
	// Check for WordPress marker file
	if _, err := os.Stat(filepath.Join(root, "wp-config.php")); err == nil {
		return "WordPress"
	}
	return "PHP"
}

func detectJavaFramework(root string) string {
	for _, f := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
		path := filepath.Join(root, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		switch {
		case strings.Contains(content, "spring-boot") || strings.Contains(content, "spring.boot"):
			return "Spring Boot"
		case strings.Contains(content, "quarkus"):
			return "Quarkus"
		case strings.Contains(content, "micronaut"):
			return "Micronaut"
		case strings.Contains(content, "vertx") || strings.Contains(content, "vert.x"):
			return "Vert.x"
		case strings.Contains(content, "kotlin"):
			return "Kotlin"
		default:
			return "Java"
		}
	}
	return ""
}

func detectDotnetFramework(root string) string {
	// Check for .csproj or .fsproj files first (more specific)
	for _, pattern := range []string{"*.csproj", "*.fsproj"} {
		matches, _ := filepath.Glob(filepath.Join(root, pattern))
		if len(matches) > 0 {
			return detectDotnetFromCsproj(matches[0])
		}
	}
	// Solution file means .NET but we can't determine specific framework
	matches, _ := filepath.Glob(filepath.Join(root, "*.sln"))
	if len(matches) > 0 {
		return ".NET"
	}
	return ""
}

func detectDotnetFromCsproj(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ".NET"
	}
	content := string(data)

	switch {
	case strings.Contains(content, "Aspire.Hosting") || strings.Contains(content, "IsAspireHost"):
		return ".NET Aspire"
	case strings.Contains(content, "Microsoft.NET.Sdk.BlazorWebAssembly") || strings.Contains(content, "Microsoft.AspNetCore.Components"):
		return "Blazor"
	case strings.Contains(content, "UseMaui") || strings.Contains(content, "Microsoft.Maui"):
		return ".NET MAUI"
	case strings.Contains(content, "Microsoft.NET.Sdk.Web") || strings.Contains(content, "Microsoft.AspNetCore"):
		return "ASP.NET Core"
	case strings.Contains(content, "UseWPF") || strings.Contains(content, "Microsoft.NET.Sdk.WindowsDesktop"):
		return "WPF"
	case strings.Contains(content, "UseWindowsForms"):
		return "WinForms"
	}
	return ".NET"
}

var commandKeywordMap = map[string]string{
	// JavaScript
	"next":       "Next.js",
	"vite":       "Vite",
	"nuxt":       "Nuxt",
	"angular":    "Angular",
	"webpack":    "Webpack",
	"remix":      "Remix",
	"astro":      "Astro",
	"gatsby":     "Gatsby",
	// Python
	"flask":      "Flask",
	"django":     "Django",
	"manage.py":  "Django",
	"uvicorn":    "Uvicorn",
	"gunicorn":   "Gunicorn",
	"celery":     "Celery",
	"streamlit":  "Streamlit",
	"gradio":     "Gradio",
	// Ruby
	"rails":      "Rails",
	// Rust
	"cargo":      "Rust",
	"rustc":      "Rust",
	// Java/JVM
	"spring":     "Spring",
	"gradlew":    "Java",
	"mvn":        "Java",
	"sbt":        "Scala",
	// .NET
	"dotnet":     ".NET",
	"aspire":     ".NET Aspire",
	"tye":        ".NET Tye",
	"daprd":      "Dapr",
	// PHP
	"artisan":    "Laravel",
	"symfony":    "Symfony",
	// Other languages
	"flutter":    "Flutter",
	"dart":       "Dart",
	"crystal":    "Crystal",
	"gleam":      "Gleam",
	"openresty":  "OpenResty",
}

// commandPriority ensures more specific keywords match before general ones.
var commandPriority = []string{
	"aspire", "manage.py",
}

func detectFromCommand(command string) string {
	if command == "" {
		return ""
	}
	lower := strings.ToLower(command)
	// Check high-priority keywords first
	for _, keyword := range commandPriority {
		if strings.Contains(lower, keyword) {
			return commandKeywordMap[keyword]
		}
	}
	for keyword, framework := range commandKeywordMap {
		if strings.Contains(lower, keyword) {
			return framework
		}
	}
	return ""
}

var processNameMap = map[string]string{
	// JavaScript runtimes
	"node":             "Node.js",
	"deno":             "Deno",
	"bun":              "Bun",
	// Python
	"python":           "Python",
	"python3":          "Python",
	// Ruby
	"ruby":             "Ruby",
	// JVM
	"java":             "Java",
	"scala":            "Scala",
	"kotlin":           "Kotlin",
	// Go
	"go":               "Go",
	// Rust
	"cargo":            "Rust",
	// Dart/Flutter
	"dart":             "Dart",
	"flutter":          "Flutter",
	// Swift
	"swift":            "Swift",
	// PHP
	"php":              "PHP",
	// .NET
	"dotnet":           ".NET",
	"dcp":              ".NET Aspire",
	"dcpctrl":          ".NET Aspire",
	"iisexpress":       "ASP.NET",
	"w3wp":             "ASP.NET",
	"mono":             "Mono",
	"mono-sgen":        "Mono",
	"tye":              ".NET Tye",
	"daprd":            "Dapr",
	// Elixir/Erlang
	"elixir":           "Elixir",
	"beam.smp":         "Erlang/Elixir",
	// Other languages
	"zig":              "Zig",
	"ghc":              "Haskell",
	"crystal":          "Crystal",
	"gleam":            "Gleam",
	"perl":             "Perl",
	"lua":              "Lua",
	"luajit":           "Lua",
	"ocaml":            "OCaml",
	"lein":             "Clojure",
	// Databases
	"postgres":         "PostgreSQL",
	"redis-server":     "Redis",
	"mongod":           "MongoDB",
	"mysqld":           "MySQL",
	"clickhouse-server": "ClickHouse",
	"neo4j":            "Neo4j",
	"cockroach":        "CockroachDB",
	"meilisearch":      "Meilisearch",
	// Messaging
	"nats-server":      "NATS",
	// Web servers & proxies
	"nginx":            "nginx",
	"caddy":            "Caddy",
	"httpd":            "Apache",
	"apache2":          "Apache",
	"openresty":        "OpenResty",
	"traefik":          "Traefik",
	"haproxy":          "HAProxy",
	"envoy":            "Envoy",
	// Observability
	"prometheus":       "Prometheus",
	"grafana":          "Grafana",
	"grafana-server":   "Grafana",
	// Infrastructure
	"consul":           "Consul",
	"vault":            "Vault",
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
