package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iCyberon/ptty/internal/scanner"
	"github.com/iCyberon/ptty/internal/updater"
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

type updateCheckMsg struct {
	result *updater.CheckResult
	err    error
}

type updateAppliedMsg struct {
	version string
	err     error
}

type postUpdateMsg struct {
	version string
}

type hideUpdateBarMsg struct{}

type updateBarKind int

const (
	updateBarInfo updateBarKind = iota
	updateBarSuccess
	updateBarError
)

type AppModel struct {
	scanner   scanner.Scanner
	activeTab int
	width     int
	height    int
	showAll   bool

	ports *PortsModel
	procs *ProcessesModel
	watch *WatchModel
	clean *CleanModel

	up              *updater.Updater
	version         string
	updateNotice    string
	updateBarStyle  updateBarKind
	showUpdateBar   bool
	updateAvailable bool
	updateVersion   string
	updateRelease   *selfupdate.Release

	err error
}

func NewApp(s scanner.Scanner, initialTab int, version string) AppModel {
	up, _ := updater.New(version)
	m := AppModel{
		scanner:   s,
		activeTab: initialTab,
		showAll:   false,
		up:        up,
		version:   version,
	}
	m.ports = NewPortsModel()
	m.procs = NewProcessesModel()
	m.watch = NewWatchModel()
	m.clean = NewCleanModel()
	return m
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.scanPorts(),
		m.scanProcesses(),
		m.scanOrphans(),
		m.tickCmd(),
		m.watch.Init(),
		m.checkPostUpdate(),
	}
	if m.up != nil {
		cmds = append(cmds, m.checkForUpdate())
	}
	return tea.Batch(cmds...)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.ports.filtering || !m.ports.filter.Focused() {
			switch {
			case key.Matches(msg, keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, keys.Esc):
				if m.ports.detail != nil || m.procs.detail != nil || m.clean.state == cleanConfirming || m.ports.filtering {
					break
				}
				return m, tea.Quit
			case key.Matches(msg, keys.Update):
				if m.updateAvailable && m.updateRelease != nil {
					_, canWrite := updater.CanWrite()
					if !canWrite {
						m.updateNotice = "Cannot update: permission denied. Run: sudo ptty update"
						m.updateBarStyle = updateBarError
						m.showUpdateBar = true
						return m, nil
					}
					m.updateNotice = fmt.Sprintf("Updating to v%s...", m.updateVersion)
					m.updateBarStyle = updateBarInfo
					return m, m.applyUpdate()
				}
				return m, nil
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

	case postUpdateMsg:
		if msg.version != "" {
			m.updateNotice = fmt.Sprintf("Updated to v%s!", msg.version)
			m.updateBarStyle = updateBarSuccess
			m.showUpdateBar = true
			return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return hideUpdateBarMsg{}
			})
		}
		return m, nil

	case updateCheckMsg:
		if msg.err != nil || msg.result == nil {
			return m, nil
		}
		m.updateAvailable = true
		m.updateVersion = msg.result.Version
		m.updateRelease = msg.result.Release
		m.showUpdateBar = true
		m.updateBarStyle = updateBarInfo
		_, canWrite := updater.CanWrite()
		if canWrite {
			m.updateNotice = fmt.Sprintf("v%s available — press U to update", msg.result.Version)
		} else {
			m.updateNotice = fmt.Sprintf("v%s available — run: sudo ptty update", msg.result.Version)
		}
		return m, nil

	case updateAppliedMsg:
		if msg.err != nil {
			m.updateNotice = fmt.Sprintf("Update failed: %s", msg.err)
			m.updateBarStyle = updateBarError
			m.showUpdateBar = true
			return m, nil
		}
		return m, nil

	case hideUpdateBarMsg:
		m.showUpdateBar = false
		m.updateNotice = ""
		return m, nil
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
	left := headerStyle.Render("⚡ ptty") + subtitleStyle.Render(fmt.Sprintf(" — %d listening ports", portCount))

	var right string
	if m.showUpdateBar && m.updateNotice != "" {
		var style lipgloss.Style
		switch m.updateBarStyle {
		case updateBarSuccess:
			style = updateBarSuccessStyle
		case updateBarError:
			style = updateBarErrorStyle
		default:
			style = updateBarStyle
		}
		right = style.Render(m.updateNotice)
	} else {
		right = dimStyle.Render("v" + m.version)
	}

	headerGap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if headerGap < 1 {
		headerGap = 1
	}
	b.WriteString(left + strings.Repeat(" ", headerGap) + right + "\n\n")

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
		if m.ports.filtering && m.ports.filter.Focused() {
			parts = append(parts, helpKeyStyle.Render("esc")+" "+helpDescStyle.Render("cancel"))
			parts = append(parts, helpKeyStyle.Render("↓")+" "+helpDescStyle.Render("navigate"))
		} else if m.ports.filtering {
			parts = append(parts, helpKeyStyle.Render("x")+" "+helpDescStyle.Render("kill"))
			parts = append(parts, helpKeyStyle.Render("esc")+" "+helpDescStyle.Render("clear"))
			parts = append(parts, helpKeyStyle.Render("enter")+" "+helpDescStyle.Render("detail"))
			parts = append(parts, helpKeyStyle.Render("/")+" "+helpDescStyle.Render("edit filter"))
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

	if m.updateAvailable {
		parts = append(parts, helpKeyStyle.Render("U")+" "+helpDescStyle.Render("update"))
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

func (m AppModel) checkPostUpdate() tea.Cmd {
	return func() tea.Msg {
		version := updater.ReadAndClearUpdatedVersion()
		return postUpdateMsg{version: version}
	}
}

func (m AppModel) checkForUpdate() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result, err := m.up.CheckLatest(ctx)
		return updateCheckMsg{result: result, err: err}
	}
}

func (m AppModel) applyUpdate() tea.Cmd {
	release := m.updateRelease
	version := m.updateVersion
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		if err := m.up.Apply(ctx, release); err != nil {
			return updateAppliedMsg{err: err}
		}

		if err := updater.WriteUpdatedVersion(version); err != nil {
			return updateAppliedMsg{err: err}
		}

		if err := updater.Restart(); err != nil {
			return updateAppliedMsg{version: version, err: err}
		}

		return updateAppliedMsg{version: version}
	}
}
