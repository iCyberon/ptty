package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iCyberon/ptty/internal/scanner"
)

type PortsModel struct {
	data      []scanner.PortInfo
	filtered  []scanner.PortInfo
	cursor    int
	width     int
	height    int
	detail    *scanner.PortInfo
	filtering bool
	filter    textinput.Model
}

func NewPortsModel() *PortsModel {
	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = 50
	return &PortsModel{
		filter: fi,
	}
}

func (m *PortsModel) SetData(ports []scanner.PortInfo) {
	m.data = ports
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m *PortsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *PortsModel) applyFilter() {
	filterStr := strings.ToLower(m.filter.Value())
	if filterStr == "" {
		m.filtered = m.data
		return
	}
	m.filtered = nil
	for _, p := range m.data {
		text := strings.ToLower(fmt.Sprintf(":%d %s %s %s %s",
			p.Port, p.ProcessName, p.ProjectName, p.Framework, p.Command))
		if strings.Contains(text, filterStr) {
			m.filtered = append(m.filtered, p)
		}
	}
}

func (m *PortsModel) Update(msg tea.Msg, s scanner.Scanner) tea.Cmd {
	if msg, ok := msg.(detailFetchedMsg); ok {
		if msg.err == nil && msg.info != nil {
			m.detail = msg.info
		}
		return nil
	}

	if m.detail != nil {
		return m.updateDetail(msg, s)
	}

	if m.filtering {
		return m.updateFilter(msg, s)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if m.cursor < len(m.filtered) {
				info := m.filtered[m.cursor]
				return m.fetchDetail(info.Port, s)
			}
		case key.Matches(msg, keys.Kill):
			if m.cursor < len(m.filtered) {
				pid := m.filtered[m.cursor].PID
				return killCmd(s, pid)
			}
		case key.Matches(msg, keys.Filter):
			m.filtering = true
			m.filter.Focus()
			return textinput.Blink
		}
	}

	return nil
}

func (m *PortsModel) updateFilter(msg tea.Msg, s scanner.Scanner) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Esc):
			m.filtering = false
			m.filter.SetValue("")
			m.filter.Blur()
			m.applyFilter()
			return nil
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return nil
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return nil
		case key.Matches(msg, keys.Enter):
			if m.cursor < len(m.filtered) {
				info := m.filtered[m.cursor]
				return m.fetchDetail(info.Port, s)
			}
			return nil
		}
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return cmd
}

type detailFetchedMsg struct {
	info *scanner.PortInfo
	err  error
}

func (m *PortsModel) fetchDetail(port int, s scanner.Scanner) tea.Cmd {
	return func() tea.Msg {
		info, err := s.GetPortDetail(port)
		return detailFetchedMsg{info: info, err: err}
	}
}

func (m *PortsModel) updateDetail(msg tea.Msg, s scanner.Scanner) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Esc):
			m.detail = nil
			if m.filtering {
				m.filter.Focus()
				return textinput.Blink
			}
			return nil
		case key.Matches(msg, keys.Kill):
			pid := m.detail.PID
			m.detail = nil
			return killCmd(s, pid)
		case key.Matches(msg, keys.Refresh):
			port := m.detail.Port
			return m.fetchDetail(port, s)
		}
	}
	return nil
}

func (m *PortsModel) View() string {
	if m.detail != nil {
		return m.viewDetail()
	}
	return m.viewTable()
}

func (m *PortsModel) viewTable() string {
	var b strings.Builder

	if m.filtering {
		b.WriteString("  " + m.filter.View() + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString("\n  No listening ports found.\n")
		b.WriteString(dimStyle.Render("  Start a dev server to see it here.\n"))
		return b.String()
	}

	const (
		colPort      = 8
		colPID       = 8
		colProject   = 16
		colFramework = 14
		colUptime    = 10
		colStatus    = 12
		indent       = 2
		minProcess   = 16
		maxProcess   = 40
	)
	fixedCols := colPort + colPID + colProject + colFramework + colUptime + colStatus + indent
	colProcess := m.width - fixedCols
	if colProcess < minProcess {
		colProcess = minProcess
	} else if colProcess > maxProcess {
		colProcess = maxProcess
	}

	header := "  " +
		tableHeaderStyle.Render(pad("PORT", colPort)) +
		tableHeaderStyle.Render(pad("PROCESS", colProcess)) +
		tableHeaderStyle.Render(pad("PID", colPID)) +
		tableHeaderStyle.Render(pad("PROJECT", colProject)) +
		tableHeaderStyle.Render(pad("FRAMEWORK", colFramework)) +
		tableHeaderStyle.Render(pad("UPTIME", colUptime)) +
		tableHeaderStyle.Render("STATUS")
	b.WriteString(header + "\n")

	visibleRows := m.height - 3
	if m.filtering {
		visibleRows--
	}
	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}

	for i := startIdx; i < len(m.filtered) && i < startIdx+visibleRows; i++ {
		p := m.filtered[i]
		project := p.ProjectName
		if project == "" {
			project = "—"
		}
		framework := p.Framework
		fwPlain := framework
		if framework == "" {
			framework = "—"
			fwPlain = "—"
		} else {
			framework = frameworkStyle(fwPlain).Render(fwPlain)
		}

		statusPlain := statusText(p.Status)
		status := renderStatus(p.Status)
		portPlain := fmt.Sprintf(":%d", p.Port)
		uptimePlain := formatDuration(p.Uptime)

		row := "  " +
			portStyle.Render(pad(portPlain, colPort)) +
			pad(truncate(p.ProcessName, colProcess-2), colProcess) +
			dimStyle.Render(pad(fmt.Sprintf("%d", p.PID), colPID)) +
			pad(truncate(project, colProject-2), colProject) +
			styledPad(framework, fwPlain, colFramework) +
			dimStyle.Render(pad(uptimePlain, colUptime)) +
			styledPad(status, statusPlain, 0)

		if i == m.cursor {
			row = selectedRowStyle.Render(row)
		}

		b.WriteString(row + "\n")
	}

	return b.String()
}

func (m *PortsModel) viewDetail() string {
	info := m.detail
	var b strings.Builder

	title := headerStyle.Render(fmt.Sprintf("  Port :%d", info.Port))
	b.WriteString(title + "\n\n")

	grid := []struct{ label, value string }{
		{"Process", fmt.Sprintf("%s (PID %d)", info.ProcessName, info.PID)},
		{"Status", renderStatus(info.Status)},
		{"Framework", frameworkStyle(info.Framework).Render(info.Framework)},
		{"Memory", formatMemory(info.Memory)},
		{"CPU", formatCPU(info.CPU)},
		{"Uptime", formatDuration(info.Uptime)},
	}

	if info.CWD != "" {
		grid = append(grid, struct{ label, value string }{"Directory", info.CWD})
	}
	if info.ProjectName != "" {
		grid = append(grid, struct{ label, value string }{"Project", info.ProjectName})
	}
	if info.GitBranch != "" {
		grid = append(grid, struct{ label, value string }{"Branch", branchStyle.Render(info.GitBranch)})
	}

	for _, item := range grid {
		label := detailLabelStyle.Render(item.label)
		b.WriteString(fmt.Sprintf("  %s  %s\n", label, item.value))
	}

	if len(info.ProcessTree) > 0 {
		b.WriteString("\n  " + tableHeaderStyle.Render("Process Tree") + "\n")
		for i, node := range info.ProcessTree {
			indent := strings.Repeat("   ", i)
			marker := "└─ "
			suffix := ""
			if i == len(info.ProcessTree)-1 {
				suffix = dimStyle.Render(" ← this")
			}
			b.WriteString(fmt.Sprintf("  %s%s%s (PID %d)%s\n",
				indent, marker, node.Name, node.PID, suffix))
		}
	}

	return b.String()
}

func killCmd(s scanner.Scanner, pid int) tea.Cmd {
	return func() tea.Msg {
		err := s.KillProcess(pid)
		return killResultMsg{pid: pid, err: err}
	}
}

func pad(s string, width int) string {
	if width <= 0 {
		return s
	}
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func styledPad(styled, plain string, width int) string {
	if width <= 0 {
		return styled
	}
	visible := lipgloss.Width(plain)
	if visible >= width {
		return styled
	}
	return styled + strings.Repeat(" ", width-visible)
}

func statusText(s scanner.Status) string {
	switch s {
	case scanner.StatusHealthy:
		return "● healthy"
	case scanner.StatusOrphaned:
		return "● orphaned"
	case scanner.StatusZombie:
		return "● zombie"
	default:
		return "● unknown"
	}
}

func renderStatus(s scanner.Status) string {
	switch s {
	case scanner.StatusHealthy:
		return statusHealthy.Render("● healthy")
	case scanner.StatusOrphaned:
		return statusOrphaned.Render("● orphaned")
	case scanner.StatusZombie:
		return statusZombie.Render("● zombie")
	default:
		return dimStyle.Render("● unknown")
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

func formatMemory(bytes uint64) string {
	if bytes == 0 {
		return "—"
	}
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatCPU(cpu float64) string {
	s := fmt.Sprintf("%.1f%%", cpu)
	switch {
	case cpu > 25:
		return cpuHigh.Render(s)
	case cpu > 5:
		return cpuMedium.Render(s)
	default:
		return cpuLow.Render(s)
	}
}

func truncate(s string, maxLen int) string {
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) > maxLen-1 {
		return string(runes[:maxLen-1]) + "…"
	}
	return s
}
