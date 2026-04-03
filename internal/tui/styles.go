package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorBlue     = lipgloss.Color("#7aa2f7")
	colorGreen    = lipgloss.Color("#9ece6a")
	colorYellow   = lipgloss.Color("#e0af68")
	colorRed      = lipgloss.Color("#f7768e")
	colorPurple   = lipgloss.Color("#bb9af7")
	colorCyan     = lipgloss.Color("#7dcfff")
	colorSubtle   = lipgloss.Color("#565f89")
	colorText     = lipgloss.Color("#c0caf5")
	colorDarkBg   = lipgloss.Color("#24283b")
	colorBorder   = lipgloss.Color("#3b4261")

	activeTabStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Padding(0, 2)

	tabBarStyle = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottomForeground(colorBorder)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTopForeground(colorBorder)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Background(colorDarkBg).
			Padding(0, 1)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
				Background(colorDarkBg)

	statusHealthy  = lipgloss.NewStyle().Foreground(colorGreen)
	statusOrphaned = lipgloss.NewStyle().Foreground(colorYellow)
	statusZombie   = lipgloss.NewStyle().Foreground(colorRed)

	cpuLow    = lipgloss.NewStyle().Foreground(colorGreen)
	cpuMedium = lipgloss.NewStyle().Foreground(colorYellow)
	cpuHigh   = lipgloss.NewStyle().Foreground(colorRed)

	portStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Width(10)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	watchNewStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	watchClosedStyle = lipgloss.NewStyle().
				Foreground(colorRed).
				Bold(true)

	watchTimeStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	branchStyle = lipgloss.NewStyle().Foreground(colorPurple)
	dimStyle    = lipgloss.NewStyle().Foreground(colorSubtle)

	updateBarStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	updateBarSuccessStyle = lipgloss.NewStyle().
				Foreground(colorBlue).
				Bold(true)

	updateBarErrorStyle = lipgloss.NewStyle().
				Foreground(colorRed)
)

var frameworkColors = map[string]lipgloss.Color{
	"Next.js":    lipgloss.Color("#c0caf5"),
	"Vite":       lipgloss.Color("#e0af68"),
	"React":      lipgloss.Color("#7dcfff"),
	"Vue":        lipgloss.Color("#9ece6a"),
	"Angular":    lipgloss.Color("#f7768e"),
	"Svelte":     lipgloss.Color("#ff3e00"),
	"Express":    lipgloss.Color("#c0caf5"),
	"Go":         lipgloss.Color("#73daca"),
	"Rust":       lipgloss.Color("#e0af68"),
	"Python":     lipgloss.Color("#e0af68"),
	"Django":     lipgloss.Color("#9ece6a"),
	"Flask":      lipgloss.Color("#c0caf5"),
	"Rails":      lipgloss.Color("#f7768e"),
	"PostgreSQL": lipgloss.Color("#bb9af7"),
	"Redis":      lipgloss.Color("#f7768e"),
	"MongoDB":    lipgloss.Color("#9ece6a"),
	"MySQL":      lipgloss.Color("#7dcfff"),
	"nginx":      lipgloss.Color("#9ece6a"),
	"Node.js":    lipgloss.Color("#9ece6a"),
	"Deno":       lipgloss.Color("#c0caf5"),
	"Bun":        lipgloss.Color("#e0af68"),
}

func frameworkStyle(name string) lipgloss.Style {
	if c, ok := frameworkColors[name]; ok {
		return lipgloss.NewStyle().Foreground(c)
	}
	return lipgloss.NewStyle().Foreground(colorText)
}
