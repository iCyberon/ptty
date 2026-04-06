package detect

import "strings"

var blocklist = []string{
	"Slack", "Chrome", "Google Chrome", "Brave", "Firefox", "Safari",
	"Spotify", "Figma", "Code Helper", "Code Helper (Renderer)",
	"iTerm2", "Terminal", "Finder", "Dock", "WindowServer",
	"SystemUIServer", "loginwindow", "launchd", "kernel_task",
	"mds_stores", "mds", "mdworker", "mdworker_shared",
	"Warp", "Arc", "Microsoft Edge", "Opera",
	"Notion", "Discord", "Telegram", "WhatsApp", "Messages",
	"Mail", "Calendar", "Reminders", "Notes",
	"Activity Monitor", "System Preferences", "System Settings",
	"ControlCenter", "ControlCe", "rapportd", "sharingd",
	"SetappAge", "IPNExtens", "Raycast",
}

var allowlist = []string{
	// JavaScript runtimes & tools
	"node", "deno", "bun", "npm", "yarn", "pnpm",
	"webpack", "vite", "next", "nuxt",
	// Python
	"python", "python3",
	"uvicorn", "gunicorn", "flask", "django", "celery",
	// Ruby
	"ruby", "rails", "puma", "unicorn", "sidekiq",
	// JVM
	"java", "scala", "kotlin", "gradle", "mvn", "sbt", "lein",
	// Go
	"go",
	// Rust
	"cargo",
	// Dart/Flutter
	"dart", "flutter",
	// Swift
	"swift",
	// PHP
	"php",
	// .NET
	"dotnet", "dcp", "dcpctrl", "iisexpress", "w3wp",
	"mono", "mono-sgen", "tye", "daprd", "msbuild",
	// Elixir/Erlang
	"elixir", "erlang", "beam.smp", "mix",
	// Other languages
	"zig", "ghc", "crystal", "gleam", "perl", "lua", "luajit", "ocaml",
	// Databases
	"postgres", "redis-server", "mongod", "mysqld",
	"clickhouse-server", "neo4j", "cockroach", "nats-server", "meilisearch",
	// Web servers & proxies
	"nginx", "caddy", "httpd", "apache2", "openresty",
	"traefik", "haproxy", "envoy",
	// Observability
	"prometheus", "grafana", "grafana-server",
	// Infrastructure
	"consul", "vault",
	// Containers & mobile
	"docker", "com.docke", "containerd",
	"adb",
}

func IsDevProcess(processName, command string) bool {
	lower := strings.ToLower(processName)

	for _, blocked := range blocklist {
		if strings.EqualFold(processName, blocked) {
			return false
		}
	}

	for _, allowed := range allowlist {
		if strings.EqualFold(processName, allowed) {
			return true
		}
	}

	if strings.HasPrefix(lower, "com.docke") || lower == "docker" {
		return true
	}

	cmdLower := strings.ToLower(command)
	devKeywords := []string{
		"node", "python", "ruby", "java", "go run", "go build",
		"cargo run", "deno", "bun run", "npm ", "yarn ", "pnpm ",
		"flask", "django", "uvicorn", "rails", "spring",
		"webpack", "vite", "next", "nuxt", "angular",
		"dotnet", "aspire",
		"flutter", "dart", "swift",
		"scala", "sbt", "kotlin",
		"crystal", "gleam", "zig", "perl", "lua",
		"celery", "streamlit", "gradio",
		"artisan", "symfony",
		"docker", "postgres", "redis", "mongo", "mysql",
	}
	for _, kw := range devKeywords {
		if strings.Contains(cmdLower, kw) {
			return true
		}
	}

	return false
}
