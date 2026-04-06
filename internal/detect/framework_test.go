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

func TestDetectDotnetFramework(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{
			"ASP.NET Core project",
			"WebApp.csproj",
			`<Project Sdk="Microsoft.NET.Sdk.Web">
				<PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup>
			</Project>`,
			"ASP.NET Core",
		},
		{
			"Aspire AppHost",
			"AppHost.csproj",
			`<Project Sdk="Microsoft.NET.Sdk">
				<PropertyGroup><IsAspireHost>true</IsAspireHost></PropertyGroup>
				<ItemGroup><PackageReference Include="Aspire.Hosting" /></ItemGroup>
			</Project>`,
			".NET Aspire",
		},
		{
			"Blazor project",
			"BlazorApp.csproj",
			`<Project Sdk="Microsoft.NET.Sdk.BlazorWebAssembly">
				<PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup>
			</Project>`,
			"Blazor",
		},
		{
			"MAUI project",
			"MauiApp.csproj",
			`<Project Sdk="Microsoft.NET.Sdk">
				<PropertyGroup><UseMaui>true</UseMaui></PropertyGroup>
			</Project>`,
			".NET MAUI",
		},
		{
			"Console app",
			"ConsoleApp.csproj",
			`<Project Sdk="Microsoft.NET.Sdk">
				<PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup>
			</Project>`,
			".NET",
		},
		{
			"Solution file only",
			"MyApp.sln",
			"Microsoft Visual Studio Solution File",
			".NET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, tt.filename), []byte(tt.content), 0644)

			got := detectDotnetFramework(dir)
			if got != tt.expected {
				t.Errorf("detectDotnetFramework() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectDotnetFromCommand(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"dotnet run", ".NET"},
		{"dotnet watch run", ".NET"},
		{"/Users/user/.nuget/packages/aspire.hosting.orchestration.osx-arm64/13.0.0/tools/ext/dcpctrl", ".NET Aspire"},
	}

	for _, tt := range tests {
		got := detectFromCommand(tt.command)
		if got != tt.expected {
			t.Errorf("detectFromCommand(%q) = %q, want %q", tt.command, got, tt.expected)
		}
	}
}

func TestDetectDotnetProcessName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"dotnet", ".NET"},
		{"dcp", ".NET Aspire"},
		{"dcpctrl", ".NET Aspire"},
	}

	for _, tt := range tests {
		got := detectFromProcessName(tt.name)
		if got != tt.expected {
			t.Errorf("detectFromProcessName(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestDetectFrameworkAspireIntegration(t *testing.T) {
	// Simulate Aspire scenario: dcpctrl process with Aspire AppHost CWD
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "AppHost.csproj"),
		[]byte(`<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><PackageReference Include="Aspire.Hosting" /></ItemGroup></Project>`), 0644)

	got := DetectFramework("", dir,
		"/Users/user/.nuget/packages/aspire.hosting.orchestration.osx-arm64/13.0.0/tools/ext/dcpctrl",
		"dcpctrl", "")
	if got != ".NET Aspire" {
		t.Errorf("DetectFramework for Aspire project = %q, want %q", got, ".NET Aspire")
	}
}

func TestDetectPythonFramework(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"Flask", "flask==3.0.0\nrequests==2.31.0", "Flask"},
		{"Django", "django==4.2\ncelery==5.3", "Django"},
		{"FastAPI", "fastapi==0.104\nuvicorn==0.24", "FastAPI"},
		{"Tornado", "tornado==6.4", "Tornado"},
		{"Sanic", "sanic==23.12", "Sanic"},
		{"Streamlit", "streamlit==1.30", "Streamlit"},
		{"Gradio", "gradio==4.0", "Gradio"},
		{"Celery", "celery==5.3\nredis==5.0", "Celery"},
		{"aiohttp", "aiohttp==3.9", "aiohttp"},
		{"Plain Python", "requests==2.31.0\nnumpy==1.26", "Python"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(tt.content), 0644)
			got := detectPythonFramework(dir)
			if got != tt.expected {
				t.Errorf("detectPythonFramework = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectDartFramework(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			"Flutter project",
			"name: my_app\ndependencies:\n  flutter:\n    sdk: flutter",
			"Flutter",
		},
		{
			"Dart project",
			"name: my_app\ndependencies:\n  http: ^1.0.0",
			"Dart",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "pubspec.yaml"), []byte(tt.content), 0644)
			got := detectDartFramework(dir)
			if got != tt.expected {
				t.Errorf("detectDartFramework() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectPHPFramework(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			"Laravel",
			`{"require": {"laravel/framework": "^10.0"}}`,
			"Laravel",
		},
		{
			"Symfony",
			`{"require": {"symfony/framework-bundle": "^6.0"}}`,
			"Symfony",
		},
		{
			"Slim",
			`{"require": {"slim/slim": "^4.0"}}`,
			"Slim",
		},
		{
			"CakePHP",
			`{"require": {"cakephp/cakephp": "^4.0"}}`,
			"CakePHP",
		},
		{
			"Plain PHP",
			`{"require": {"monolog/monolog": "^3.0"}}`,
			"PHP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "composer.json"), []byte(tt.content), 0644)
			got := detectPHPFramework(dir)
			if got != tt.expected {
				t.Errorf("detectPHPFramework() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectPHPWordPress(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"require": {}}`), 0644)
	os.WriteFile(filepath.Join(dir, "wp-config.php"), []byte("<?php"), 0644)

	got := detectPHPFramework(dir)
	if got != "WordPress" {
		t.Errorf("detectPHPFramework with wp-config.php = %q, want %q", got, "WordPress")
	}
}

func TestDetectJavaFramework(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{
			"Spring Boot (pom.xml)",
			"pom.xml",
			`<dependency><groupId>org.springframework.boot</groupId><artifactId>spring-boot-starter</artifactId></dependency>`,
			"Spring Boot",
		},
		{
			"Quarkus",
			"pom.xml",
			`<dependency><groupId>io.quarkus</groupId><artifactId>quarkus-core</artifactId></dependency>`,
			"Quarkus",
		},
		{
			"Micronaut",
			"build.gradle",
			`implementation("io.micronaut:micronaut-runtime")`,
			"Micronaut",
		},
		{
			"Vert.x",
			"pom.xml",
			`<dependency><groupId>io.vertx</groupId><artifactId>vertx-core</artifactId></dependency>`,
			"Vert.x",
		},
		{
			"Kotlin (build.gradle.kts)",
			"build.gradle.kts",
			`plugins { kotlin("jvm") version "1.9.0" }`,
			"Kotlin",
		},
		{
			"Plain Java",
			"pom.xml",
			`<dependency><groupId>junit</groupId><artifactId>junit</artifactId></dependency>`,
			"Java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, tt.filename), []byte(tt.content), 0644)
			got := detectJavaFramework(dir)
			if got != tt.expected {
				t.Errorf("detectJavaFramework() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectRubyFramework(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"Rails", "gem 'rails', '~> 7.0'", "Rails"},
		{"Sinatra", "gem 'sinatra'", "Sinatra"},
		{"Hanami", "gem 'hanami', '~> 2.0'", "Hanami"},
		{"Plain Ruby", "gem 'httparty'", "Ruby"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "Gemfile"), []byte(tt.content), 0644)
			got := detectRubyFramework(dir)
			if got != tt.expected {
				t.Errorf("detectRubyFramework() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectNewLanguagesProjectFiles(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"Swift", "Package.swift", "Swift"},
		{"Scala", "build.sbt", "Scala"},
		{"Crystal", "shard.yml", "Crystal"},
		{"Gleam", "gleam.toml", "Gleam"},
		{"OCaml", "dune-project", "OCaml"},
		{"Perl", "cpanfile", "Perl"},
		{"Clojure (deps.edn)", "deps.edn", "Clojure"},
		{"Clojure (project.clj)", "project.clj", "Clojure"},
		{"Zig", "build.zig", "Zig"},
		{"Haskell (stack)", "stack.yaml", "Haskell"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, tt.filename), []byte(""), 0644)
			got := detectFromProjectFiles(dir)
			if got != tt.expected {
				t.Errorf("detectFromProjectFiles() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectHaskellCabal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "myproject.cabal"), []byte(""), 0644)

	got := detectFromProjectFiles(dir)
	if got != "Haskell" {
		t.Errorf("detectFromProjectFiles with .cabal = %q, want %q", got, "Haskell")
	}
}

func TestDetectNewDockerImages(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"clickhouse/clickhouse-server:latest", "ClickHouse"},
		{"neo4j:5", "Neo4j"},
		{"cockroachdb/cockroach:latest", "CockroachDB"},
		{"qdrant/qdrant:latest", "Qdrant"},
		{"chromadb/chroma:latest", "ChromaDB"},
		{"opensearchproject/opensearch:2", "OpenSearch"},
		{"typesense/typesense:0.25", "Typesense"},
		{"meilisearch/meilisearch:latest", "Meilisearch"},
		{"nats:latest", "NATS"},
		{"traefik:v2", "Traefik"},
		{"prom/prometheus:latest", "Prometheus"},
		{"grafana/grafana:latest", "Grafana"},
		{"jaegertracing/all-in-one:latest", "Jaeger"},
		{"keycloak/keycloak:latest", "Keycloak"},
		{"weaviate/weaviate:latest", "Weaviate"},
		{"milvusdb/milvus:latest", "Milvus"},
		{"surrealdb/surrealdb:latest", "SurrealDB"},
		{"valkey/valkey:latest", "Valkey"},
		{"docker.dragonflydb.io/dragonfly:latest", "Dragonfly"},
	}

	for _, tt := range tests {
		got := detectFromDockerImage(tt.image)
		if got != tt.expected {
			t.Errorf("detectFromDockerImage(%q) = %q, want %q", tt.image, got, tt.expected)
		}
	}
}

func TestDetectNewProcessNames(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"dart", "Dart"},
		{"flutter", "Flutter"},
		{"swift", "Swift"},
		{"scala", "Scala"},
		{"kotlin", "Kotlin"},
		{"zig", "Zig"},
		{"ghc", "Haskell"},
		{"crystal", "Crystal"},
		{"gleam", "Gleam"},
		{"perl", "Perl"},
		{"lua", "Lua"},
		{"luajit", "Lua"},
		{"ocaml", "OCaml"},
		{"lein", "Clojure"},
		{"openresty", "OpenResty"},
		{"clickhouse-server", "ClickHouse"},
		{"nats-server", "NATS"},
		{"cockroach", "CockroachDB"},
		{"traefik", "Traefik"},
		{"haproxy", "HAProxy"},
		{"envoy", "Envoy"},
		{"prometheus", "Prometheus"},
		{"grafana", "Grafana"},
		{"grafana-server", "Grafana"},
		{"consul", "Consul"},
		{"vault", "Vault"},
		{"meilisearch", "Meilisearch"},
		{"neo4j", "Neo4j"},
	}

	for _, tt := range tests {
		got := detectFromProcessName(tt.name)
		if got != tt.expected {
			t.Errorf("detectFromProcessName(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestDetectNewPackageJSONFrameworks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"Solid.js", `{"dependencies": {"solid-js": "1.8.0"}}`, "Solid"},
		{"Qwik", `{"dependencies": {"@builder.io/qwik": "1.3.0"}}`, "Qwik"},
		{"Electron", `{"dependencies": {"electron": "28.0.0"}}`, "Electron"},
		{"Expo", `{"dependencies": {"expo": "50.0.0"}}`, "Expo"},
		{"Adonis", `{"dependencies": {"@adonisjs/core": "6.0.0"}}`, "Adonis"},
		{"Eleventy", `{"devDependencies": {"@11ty/eleventy": "2.0.0"}}`, "Eleventy"},
		{"Strapi", `{"dependencies": {"strapi": "4.0.0"}}`, "Strapi"},
		{"Payload", `{"dependencies": {"payload": "2.0.0"}}`, "Payload"},
		{"Tauri", `{"devDependencies": {"@tauri-apps/cli": "1.5.0"}}`, "Tauri"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "package.json"), []byte(tt.content), 0644)
			got := detectFromPackageJSON(filepath.Join(dir, "package.json"))
			if got != tt.expected {
				t.Errorf("detectFromPackageJSON() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectNewCommandKeywords(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"flutter run", "Flutter"},
		{"dart run", "Dart"},
		{"crystal build src/app.cr", "Crystal"},
		{"gleam run", "Gleam"},
		{"sbt compile", "Scala"},
		{"celery worker", "Celery"},
		{"streamlit run app.py", "Streamlit"},
		{"php artisan serve", "Laravel"},
		{"symfony serve", "Symfony"},
		{"openresty -p .", "OpenResty"},
	}

	for _, tt := range tests {
		got := detectFromCommand(tt.command)
		if got != tt.expected {
			t.Errorf("detectFromCommand(%q) = %q, want %q", tt.command, got, tt.expected)
		}
	}
}

func TestFindProjectRootNewMarkers(t *testing.T) {
	tests := []struct {
		name   string
		marker string
	}{
		{"Dart", "pubspec.yaml"},
		{"Swift", "Package.swift"},
		{"Scala", "build.sbt"},
		{"Crystal", "shard.yml"},
		{"Gleam", "gleam.toml"},
		{"OCaml", "dune-project"},
		{"Perl", "cpanfile"},
		{"Clojure", "deps.edn"},
		{"Zig", "build.zig"},
		{"Haskell", "stack.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			sub := filepath.Join(dir, "src")
			os.MkdirAll(sub, 0755)
			os.WriteFile(filepath.Join(dir, tt.marker), []byte(""), 0644)

			got := FindProjectRoot(sub)
			if got != dir {
				t.Errorf("FindProjectRoot from src/ with %s = %q, want %q", tt.marker, got, dir)
			}
		})
	}
}

func TestFindProjectRootCabalGlob(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "src")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(dir, "myproject.cabal"), []byte(""), 0644)

	got := FindProjectRoot(sub)
	if got != dir {
		t.Errorf("FindProjectRoot with .cabal = %q, want %q", got, dir)
	}
}
