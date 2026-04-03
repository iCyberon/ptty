package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/iCyberon/ptty/internal/scanner"
)

type watchEventType int

const (
	watchNew watchEventType = iota
	watchClosed
)

type watchEvent struct {
	timestamp time.Time
	eventType watchEventType
	port      scanner.PortInfo
}

type watchTickMsg time.Time

type WatchModel struct {
	events    []watchEvent
	prevPorts map[int]scanner.PortInfo
	paused    bool
	width     int
	height    int
	scrollPos int
}

func NewWatchModel() *WatchModel {
	return &WatchModel{
		prevPorts: make(map[int]scanner.PortInfo),
	}
}

func (m *WatchModel) Init() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return watchTickMsg(t)
	})
}

func (m *WatchModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *WatchModel) UpdatePorts(ports []scanner.PortInfo) {
	if m.paused {
		return
	}

	current := make(map[int]scanner.PortInfo)
	for _, p := range ports {
		current[p.Port] = p
	}

	now := time.Now()

	for port, info := range current {
		if _, existed := m.prevPorts[port]; !existed && len(m.prevPorts) > 0 {
			m.events = append(m.events, watchEvent{
				timestamp: now,
				eventType: watchNew,
				port:      info,
			})
		}
	}

	for port, info := range m.prevPorts {
		if _, exists := current[port]; !exists {
			m.events = append(m.events, watchEvent{
				timestamp: now,
				eventType: watchClosed,
				port:      info,
			})
		}
	}

	m.prevPorts = current
}

func (m *WatchModel) Update(msg tea.Msg, s scanner.Scanner) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Clear):
			m.events = nil
		case key.Matches(msg, keys.Pause):
			m.paused = !m.paused
			if !m.paused {
				return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return watchTickMsg(t)
				})
			}
		case key.Matches(msg, keys.Up):
			if m.scrollPos > 0 {
				m.scrollPos--
			}
		case key.Matches(msg, keys.Down):
			maxScroll := len(m.events) - m.height + 3
			if maxScroll > 0 && m.scrollPos < maxScroll {
				m.scrollPos++
			}
		}
	case watchTickMsg:
		if !m.paused {
			return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return watchTickMsg(t)
			})
		}
	}
	return nil
}

func (m *WatchModel) View() string {
	var b strings.Builder

	if m.paused {
		b.WriteString("  " + statusOrphaned.Render("⏸ PAUSED") + "\n")
	}

	if len(m.events) == 0 {
		b.WriteString("\n  Watching for port changes...\n")
		b.WriteString(dimStyle.Render("  Events will appear here when ports open or close.\n"))
		return b.String()
	}

	const (
		colTime      = 10
		colEvent     = 10
		colPort      = 8
		colProcess   = 20
		colFramework = 14
	)

	header := "  " +
		tableHeaderStyle.Render(pad("TIME", colTime)) +
		tableHeaderStyle.Render(pad("EVENT", colEvent)) +
		tableHeaderStyle.Render(pad("PORT", colPort)) +
		tableHeaderStyle.Render(pad("PROCESS", colProcess)) +
		tableHeaderStyle.Render(pad("FRAMEWORK", colFramework)) +
		tableHeaderStyle.Render("PROJECT")
	b.WriteString(header + "\n")

	visibleRows := m.height - 3
	startIdx := m.scrollPos
	if startIdx > len(m.events) {
		startIdx = len(m.events)
	}

	for i := startIdx; i < len(m.events) && i < startIdx+visibleRows; i++ {
		e := m.events[i]
		tsPlain := e.timestamp.Format("15:04:05")
		ts := watchTimeStyle.Render(pad(tsPlain, colTime))

		var eventStyled, eventPlain string
		if e.eventType == watchNew {
			eventPlain = "▲ NEW"
			eventStyled = watchNewStyle.Render(eventPlain)
		} else {
			eventPlain = "▼ CLOSED"
			eventStyled = watchClosedStyle.Render(eventPlain)
		}

		portPlain := fmt.Sprintf(":%d", e.port.Port)
		fwPlain := e.port.Framework
		if fwPlain == "" {
			fwPlain = "—"
		}
		fwStyled := frameworkStyle(e.port.Framework).Render(fwPlain)
		project := e.port.ProjectName
		if project == "" {
			project = "—"
		}

		row := "  " +
			ts +
			styledPad(eventStyled, eventPlain, colEvent) +
			portStyle.Render(pad(portPlain, colPort)) +
			pad(truncate(e.port.ProcessName, colProcess-2), colProcess) +
			styledPad(fwStyled, fwPlain, colFramework) +
			project

		b.WriteString(row + "\n")
	}

	b.WriteString("\n  " + dimStyle.Render(fmt.Sprintf("%d events", len(m.events))) + "\n")

	return b.String()
}
