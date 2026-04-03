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
	"node", "python", "python3", "ruby", "java", "go", "cargo",
	"deno", "bun", "npm", "yarn", "pnpm", "php", "dotnet",
	"elixir", "erlang", "beam.smp", "mix",
	"uvicorn", "gunicorn", "flask", "django",
	"rails", "puma", "unicorn", "sidekiq",
	"webpack", "vite", "next", "nuxt",
	"postgres", "redis-server", "mongod", "mysqld",
	"nginx", "caddy", "httpd", "apache2",
	"docker", "com.docke", "containerd",
	"adb", "gradle", "mvn",
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
		"docker", "postgres", "redis", "mongo", "mysql",
	}
	for _, kw := range devKeywords {
		if strings.Contains(cmdLower, kw) {
			return true
		}
	}

	return false
}
