package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/iCyberon/ptty/internal/scanner"
)

type cleanState int

const (
	cleanNormal cleanState = iota
	cleanConfirming
	cleanKilling
	cleanResults
)

type cleanEntry struct {
	info     scanner.PortInfo
	selected bool
	killed   bool
	err      error
}

type CleanModel struct {
	entries []cleanEntry
	cursor  int
	state   cleanState
	width   int
	height  int
}

func NewCleanModel() *CleanModel {
	return &CleanModel{}
}

func (m *CleanModel) SetData(orphans []scanner.PortInfo) {
	m.entries = make([]cleanEntry, len(orphans))
	for i, o := range orphans {
		m.entries[i] = cleanEntry{info: o}
	}
	m.state = cleanNormal
	if m.cursor >= len(m.entries) {
		m.cursor = max(0, len(m.entries)-1)
	}
}

func (m *CleanModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *CleanModel) Update(msg tea.Msg, s scanner.Scanner) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case cleanNormal:
			return m.updateNormal(msg, s)
		case cleanConfirming:
			return m.updateConfirming(msg, s)
		case cleanResults:
			switch {
			case key.Matches(msg, keys.Refresh):
				return func() tea.Msg {
					orphans, err := s.FindOrphans()
					return orphansScannedMsg{orphans: orphans, err: err}
				}
			}
		}
	case cleanKillResultMsg:
		if msg.index >= 0 && msg.index < len(m.entries) {
			m.entries[msg.index].killed = true
			m.entries[msg.index].err = msg.err
		}
		return nil
	}
	return nil
}

func (m *CleanModel) updateNormal(msg tea.KeyMsg, s scanner.Scanner) tea.Cmd {
	switch {
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case msg.String() == " ":
		if m.cursor < len(m.entries) {
			m.entries[m.cursor].selected = !m.entries[m.cursor].selected
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		}
	case key.Matches(msg, keys.SelectAll):
		allSelected := true
		for _, e := range m.entries {
			if !e.selected {
				allSelected = false
				break
			}
		}
		for i := range m.entries {
			m.entries[i].selected = !allSelected
		}
	case key.Matches(msg, keys.Kill):
		hasSelected := false
		for _, e := range m.entries {
			if e.selected {
				hasSelected = true
				break
			}
		}
		if hasSelected {
			m.state = cleanConfirming
		}
	case key.Matches(msg, keys.Refresh):
		return func() tea.Msg {
			orphans, err := s.FindOrphans()
			return orphansScannedMsg{orphans: orphans, err: err}
		}
	}
	return nil
}

func (m *CleanModel) updateConfirming(msg tea.KeyMsg, s scanner.Scanner) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		m.state = cleanKilling
		var cmds []tea.Cmd
		for i, e := range m.entries {
			if e.selected {
				idx := i
				pid := e.info.PID
				cmds = append(cmds, func() tea.Msg {
					err := s.KillProcess(pid)
					return cleanKillResultMsg{index: idx, err: err}
				})
			}
		}
		m.state = cleanResults
		return tea.Batch(cmds...)
	case "n", "N", "esc":
		m.state = cleanNormal
	}
	return nil
}

type cleanKillResultMsg struct {
	index int
	err   error
}

func (m *CleanModel) View() string {
	var b strings.Builder

	if len(m.entries) == 0 {
		b.WriteString("\n  No orphaned or zombie processes found.\n")
		b.WriteString(dimStyle.Render("  Everything looks clean!\n"))
		return b.String()
	}

	const (
		colCheck   = 5
		colPID     = 8
		colProcess = 22
		colMemory  = 10
	)

	header := "  " +
		pad(" ", colCheck) +
		tableHeaderStyle.Render(pad("PID", colPID)) +
		tableHeaderStyle.Render(pad("PROCESS", colProcess)) +
		tableHeaderStyle.Render(pad("MEMORY", colMemory)) +
		tableHeaderStyle.Render("STATUS")
	b.WriteString(header + "\n")

	visibleRows := m.height - 4
	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}

	for i := startIdx; i < len(m.entries) && i < startIdx+visibleRows; i++ {
		e := m.entries[i]

		checkbox := "[ ]"
		if e.selected {
			checkbox = "[✓]"
		}
		if m.state == cleanResults && (e.killed || e.selected) {
			if e.err != nil {
				checkbox = statusZombie.Render("[✕]")
			} else {
				checkbox = statusHealthy.Render("[✓]")
			}
		}

		statusPlain := statusText(e.info.Status)
		status := renderStatus(e.info.Status)

		checkPlain := "[ ]"
		if e.selected {
			checkPlain = "[✓]"
		}
		if m.state == cleanResults && (e.killed || e.selected) {
			checkPlain = "[✓]"
			if e.err != nil {
				checkPlain = "[✕]"
			}
		}

		row := "  " +
			styledPad(checkbox, checkPlain, colCheck) +
			dimStyle.Render(pad(fmt.Sprintf("%d", e.info.PID), colPID)) +
			pad(truncate(e.info.ProcessName, colProcess-2), colProcess) +
			pad(formatMemory(e.info.Memory), colMemory) +
			styledPad(status, statusPlain, 0)

		if i == m.cursor && m.state == cleanNormal {
			row = selectedRowStyle.Render(row)
		}

		b.WriteString(row + "\n")
	}

	switch m.state {
	case cleanConfirming:
		selected := 0
		for _, e := range m.entries {
			if e.selected {
				selected++
			}
		}
		b.WriteString(fmt.Sprintf("\n  "+statusOrphaned.Render("Kill %d processes? [y/N]"), selected))
	case cleanResults:
		b.WriteString("\n  " + dimStyle.Render("Press r to refresh"))
	}

	return b.String()
}
