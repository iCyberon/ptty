package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/iCyberon/ptty/internal/scanner"
)

type ProcessesModel struct {
	data       []scanner.PortInfo
	cursor     int
	width      int
	height     int
	detail     *scanner.PortInfo
	portsByPID map[int][]int // PID → listening ports
}

func NewProcessesModel() *ProcessesModel {
	return &ProcessesModel{}
}

func (m *ProcessesModel) SetData(procs []scanner.PortInfo) {
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPU > procs[j].CPU
	})
	m.data = procs
	if m.cursor >= len(m.data) {
		m.cursor = max(0, len(m.data)-1)
	}
}

func (m *ProcessesModel) SetPortData(ports []scanner.PortInfo) {
	m.portsByPID = make(map[int][]int)
	for _, p := range ports {
		m.portsByPID[p.PID] = append(m.portsByPID[p.PID], p.Port)
	}
}

func (m *ProcessesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *ProcessesModel) Update(msg tea.Msg, s scanner.Scanner) tea.Cmd {
	if msg, ok := msg.(detailFetchedMsg); ok {
		if msg.err == nil && msg.info != nil {
			m.detail = msg.info
		}
		return nil
	}

	if m.detail != nil {
		return m.updateDetail(msg, s)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.data)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if m.cursor < len(m.data) {
				p := m.data[m.cursor]
				if ports, ok := m.portsByPID[p.PID]; ok && len(ports) > 0 {
					return m.fetchDetail(ports[0], s)
				}
			}
		case key.Matches(msg, keys.Kill):
			if m.cursor < len(m.data) {
				pid := m.data[m.cursor].PID
				return killCmd(s, pid)
			}
		}
	}
	return nil
}

func (m *ProcessesModel) updateDetail(msg tea.Msg, s scanner.Scanner) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Esc):
			m.detail = nil
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

func (m *ProcessesModel) fetchDetail(port int, s scanner.Scanner) tea.Cmd {
	return func() tea.Msg {
		info, err := s.GetPortDetail(port)
		return detailFetchedMsg{info: info, err: err}
	}
}

func (m *ProcessesModel) View() string {
	if m.detail != nil {
		return m.viewDetail()
	}
	return m.viewTable()
}

func (m *ProcessesModel) viewTable() string {
	var b strings.Builder

	if len(m.data) == 0 {
		b.WriteString("\n  No dev processes found.\n")
		return b.String()
	}

	const (
		colPID       = 8
		colPort      = 8
		colCPU       = 8
		colMemory    = 10
		colProject   = 16
		colFramework = 14
		colUptime    = 10
		indent       = 2
		minProcess   = 16
		maxProcess   = 40
	)
	fixedCols := colPID + colPort + colCPU + colMemory + colProject + colFramework + colUptime + indent
	colProcess := m.width - fixedCols
	if colProcess < minProcess {
		colProcess = minProcess
	} else if colProcess > maxProcess {
		colProcess = maxProcess
	}

	header := "  " +
		tableHeaderStyle.Render(pad("PID", colPID)) +
		tableHeaderStyle.Render(pad("PROCESS", colProcess)) +
		tableHeaderStyle.Render(pad("PORT", colPort)) +
		tableHeaderStyle.Render(pad("CPU%", colCPU)) +
		tableHeaderStyle.Render(pad("MEMORY", colMemory)) +
		tableHeaderStyle.Render(pad("PROJECT", colProject)) +
		tableHeaderStyle.Render(pad("FRAMEWORK", colFramework)) +
		tableHeaderStyle.Render("UPTIME")
	b.WriteString(header + "\n")

	visibleRows := m.height - 2
	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}

	for i := startIdx; i < len(m.data) && i < startIdx+visibleRows; i++ {
		p := m.data[i]
		project := p.ProjectName
		if project == "" {
			project = "—"
		}
		fwPlain := p.Framework
		if fwPlain == "" {
			fwPlain = "—"
		}
		framework := frameworkStyle(p.Framework).Render(fwPlain)

		portPlain := "—"
		if ports, ok := m.portsByPID[p.PID]; ok && len(ports) > 0 {
			portPlain = fmt.Sprintf(":%d", ports[0])
		}

		cpuPlain := fmt.Sprintf("%.1f%%", p.CPU)
		uptimePlain := formatDuration(p.Uptime)

		row := "  " +
			dimStyle.Render(pad(fmt.Sprintf("%d", p.PID), colPID)) +
			pad(truncate(p.ProcessName, colProcess-2), colProcess) +
			portStyle.Render(pad(portPlain, colPort)) +
			styledPad(formatCPU(p.CPU), cpuPlain, colCPU) +
			pad(formatMemory(p.Memory), colMemory) +
			pad(truncate(project, colProject-2), colProject) +
			styledPad(framework, fwPlain, colFramework) +
			dimStyle.Render(uptimePlain)

		if i == m.cursor {
			row = selectedRowStyle.Render(row)
		}

		b.WriteString(row + "\n")
	}

	return b.String()
}

func (m *ProcessesModel) viewDetail() string {
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
