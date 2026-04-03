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
	data   []scanner.PortInfo
	cursor int
	width  int
	height int
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

func (m *ProcessesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *ProcessesModel) Update(msg tea.Msg) tea.Cmd {
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
		}
	}
	return nil
}

func (m *ProcessesModel) View() string {
	var b strings.Builder

	if len(m.data) == 0 {
		b.WriteString("\n  No dev processes found.\n")
		return b.String()
	}

	const (
		colPID       = 8
		colProcess   = 22
		colCPU       = 8
		colMemory    = 10
		colProject   = 16
		colFramework = 14
	)

	header := "  " +
		tableHeaderStyle.Render(pad("PID", colPID)) +
		tableHeaderStyle.Render(pad("PROCESS", colProcess)) +
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

		cpuPlain := fmt.Sprintf("%.1f%%", p.CPU)
		uptimePlain := formatDuration(p.Uptime)

		row := "  " +
			dimStyle.Render(pad(fmt.Sprintf("%d", p.PID), colPID)) +
			pad(truncate(p.ProcessName, colProcess-2), colProcess) +
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
