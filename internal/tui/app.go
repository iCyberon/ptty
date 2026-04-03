package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iCyberon/ptty/internal/scanner"
)

const (
	tabPorts     = 0
	tabProcesses = 1
	tabWatch     = 2
	tabClean     = 3
)

var tabNames = []string{"Ports", "Processes", "Watch", "Clean"}

type portsScannedMsg struct {
	ports []scanner.PortInfo
	err   error
}

type processesScannedMsg struct {
	procs []scanner.PortInfo
	err   error
}

type orphansScannedMsg struct {
	orphans []scanner.PortInfo
	err     error
}

type tickMsg time.Time

type killResultMsg struct {
	pid int
	err error
}

type AppModel struct {
	scanner   scanner.Scanner
	activeTab int
	width     int
	height    int
	showAll   bool

	ports   *PortsModel
	procs   *ProcessesModel
	watch   *WatchModel
	clean   *CleanModel

	err error
}

func NewApp(s scanner.Scanner, initialTab int) AppModel {
	m := AppModel{
		scanner:   s,
		activeTab: initialTab,
		showAll:   false,
	}
	m.ports = NewPortsModel()
	m.procs = NewProcessesModel()
	m.watch = NewWatchModel()
	m.clean = NewCleanModel()
	return m
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.scanPorts(),
		m.scanProcesses(),
		m.scanOrphans(),
		m.tickCmd(),
		m.watch.Init(),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.ports.filtering {
			switch {
			case key.Matches(msg, keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, keys.Esc):
				if m.ports.detail != nil || m.procs.detail != nil || m.clean.state == cleanConfirming {
					break
				}
				return m, tea.Quit
			case key.Matches(msg, keys.Tab1):
				m.activeTab = tabPorts
				return m, nil
			case key.Matches(msg, keys.Tab2):
				m.activeTab = tabProcesses
				return m, nil
			case key.Matches(msg, keys.Tab3):
				m.activeTab = tabWatch
				return m, nil
			case key.Matches(msg, keys.Tab4):
				m.activeTab = tabClean
				cmds = append(cmds, m.scanOrphans())
				return m, tea.Batch(cmds...)
			case key.Matches(msg, keys.NextTab):
				m.activeTab = (m.activeTab + 1) % len(tabNames)
				if m.activeTab == tabClean {
					cmds = append(cmds, m.scanOrphans())
					return m, tea.Batch(cmds...)
				}
				return m, nil
			case key.Matches(msg, keys.All):
				m.showAll = !m.showAll
				cmds = append(cmds, m.scanPorts(), m.scanProcesses())
				return m, tea.Batch(cmds...)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentHeight := m.height - 5
		m.ports.SetSize(m.width, contentHeight)
		m.procs.SetSize(m.width, contentHeight)
		m.watch.SetSize(m.width, contentHeight)
		m.clean.SetSize(m.width, contentHeight)
		return m, nil

	case portsScannedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.ports.SetData(msg.ports)
			m.procs.SetPortData(msg.ports)
			m.watch.UpdatePorts(msg.ports)
		}
		return m, nil

	case processesScannedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.procs.SetData(msg.procs)
		}
		return m, nil

	case orphansScannedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.clean.SetData(msg.orphans)
		}
		return m, nil

	case tickMsg:
		cmds = append(cmds, m.scanPorts(), m.scanProcesses(), m.scanOrphans(), m.tickCmd())
		return m, tea.Batch(cmds...)

	case killResultMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		cmds = append(cmds, m.scanPorts(), m.scanProcesses(), m.scanOrphans())
		return m, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	switch m.activeTab {
	case tabPorts:
		cmd = m.ports.Update(msg, m.scanner)
	case tabProcesses:
		cmd = m.procs.Update(msg, m.scanner)
	case tabWatch:
		cmd = m.watch.Update(msg, m.scanner)
	case tabClean:
		cmd = m.clean.Update(msg, m.scanner)
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	portCount := len(m.ports.data)
	header := headerStyle.Render("⚡ ptty")
	subtitle := subtitleStyle.Render(fmt.Sprintf(" — %d listening ports", portCount))
	b.WriteString(header + subtitle + "\n\n")

	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf("%s [%d]", name, i+1)
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	b.WriteString(tabBarStyle.Render(tabBar) + "\n")

	switch m.activeTab {
	case tabPorts:
		b.WriteString(m.ports.View())
	case tabProcesses:
		b.WriteString(m.procs.View())
	case tabWatch:
		b.WriteString(m.watch.View())
	case tabClean:
		b.WriteString(m.clean.View())
	}

	b.WriteString("\n")
	footer := m.renderFooter()
	b.WriteString(footerStyle.Render(footer))

	return b.String()
}

func (m AppModel) renderFooter() string {
	var parts []string

	switch m.activeTab {
	case tabPorts:
		if m.ports.filtering {
			parts = append(parts, helpKeyStyle.Render("esc")+" "+helpDescStyle.Render("cancel"))
			parts = append(parts, helpKeyStyle.Render("enter")+" "+helpDescStyle.Render("apply"))
		} else if m.ports.detail != nil {
			parts = append(parts, helpKeyStyle.Render("x")+" "+helpDescStyle.Render("kill"))
			parts = append(parts, helpKeyStyle.Render("esc")+" "+helpDescStyle.Render("back"))
			parts = append(parts, helpKeyStyle.Render("r")+" "+helpDescStyle.Render("refresh"))
		} else {
			parts = append(parts, helpKeyStyle.Render("↑↓")+" "+helpDescStyle.Render("navigate"))
			parts = append(parts, helpKeyStyle.Render("enter")+" "+helpDescStyle.Render("detail"))
			parts = append(parts, helpKeyStyle.Render("x")+" "+helpDescStyle.Render("kill"))
			parts = append(parts, helpKeyStyle.Render("/")+" "+helpDescStyle.Render("filter"))
			parts = append(parts, helpKeyStyle.Render("a")+" "+helpDescStyle.Render("all"))
		}
	case tabProcesses:
		if m.procs.detail != nil {
			parts = append(parts, helpKeyStyle.Render("x")+" "+helpDescStyle.Render("kill"))
			parts = append(parts, helpKeyStyle.Render("esc")+" "+helpDescStyle.Render("back"))
			parts = append(parts, helpKeyStyle.Render("r")+" "+helpDescStyle.Render("refresh"))
		} else {
			parts = append(parts, helpKeyStyle.Render("↑↓")+" "+helpDescStyle.Render("navigate"))
			parts = append(parts, helpKeyStyle.Render("enter")+" "+helpDescStyle.Render("detail"))
			parts = append(parts, helpKeyStyle.Render("x")+" "+helpDescStyle.Render("kill"))
			parts = append(parts, helpKeyStyle.Render("a")+" "+helpDescStyle.Render("all"))
		}
	case tabWatch:
		parts = append(parts, helpKeyStyle.Render("c")+" "+helpDescStyle.Render("clear"))
		parts = append(parts, helpKeyStyle.Render("p")+" "+helpDescStyle.Render("pause"))
	case tabClean:
		parts = append(parts, helpKeyStyle.Render("space")+" "+helpDescStyle.Render("select"))
		parts = append(parts, helpKeyStyle.Render("A")+" "+helpDescStyle.Render("select all"))
		parts = append(parts, helpKeyStyle.Render("x")+" "+helpDescStyle.Render("kill"))
	}

	parts = append(parts, helpKeyStyle.Render("q")+" "+helpDescStyle.Render("quit"))

	allStr := "dev only"
	if m.showAll {
		allStr = "all"
	}
	right := dimStyle.Render(fmt.Sprintf("%d ports · %s", len(m.ports.data), allStr))

	left := strings.Join(parts, "  ")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func (m AppModel) scanPorts() tea.Cmd {
	return func() tea.Msg {
		devOnly := !m.showAll
		ports, err := m.scanner.ListPorts(devOnly)
		return portsScannedMsg{ports: ports, err: err}
	}
}

func (m AppModel) scanProcesses() tea.Cmd {
	return func() tea.Msg {
		devOnly := !m.showAll
		procs, err := m.scanner.GetAllProcesses(devOnly)
		return processesScannedMsg{procs: procs, err: err}
	}
}

func (m AppModel) scanOrphans() tea.Cmd {
	return func() tea.Msg {
		orphans, err := m.scanner.FindOrphans()
		return orphansScannedMsg{orphans: orphans, err: err}
	}
}

func (m AppModel) tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
