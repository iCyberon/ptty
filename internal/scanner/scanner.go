package scanner

import (
	"strconv"
	"strings"
	"time"
)

func unescapeString(s string) string {
	if !strings.Contains(s, `\x`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if i+3 < len(s) && s[i] == '\\' && s[i+1] == 'x' {
			v, err := strconv.ParseUint(s[i+2:i+4], 16, 8)
			if err == nil {
				b.WriteByte(byte(v))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusOrphaned Status = "orphaned"
	StatusZombie   Status = "zombie"
)

type ProcessNode struct {
	PID     int
	Name    string
	Command string
}

type PortInfo struct {
	Port        int           `json:"port"`
	PID         int           `json:"pid"`
	ProcessName string        `json:"processName"`
	Command     string        `json:"command"`
	CWD         string        `json:"cwd,omitempty"`
	ProjectName string        `json:"projectName,omitempty"`
	Framework   string        `json:"framework,omitempty"`
	Uptime      time.Duration `json:"uptime"`
	StartTime   time.Time     `json:"startTime"`
	Status      Status        `json:"status"`
	Memory      uint64        `json:"memory"`
	CPU         float64       `json:"cpu"`
	GitBranch   string        `json:"gitBranch,omitempty"`
	ProcessTree []ProcessNode `json:"processTree,omitempty"`
}

type Scanner interface {
	ListPorts(devOnly bool) ([]PortInfo, error)
	GetPortDetail(port int) (*PortInfo, error)
	GetAllProcesses(devOnly bool) ([]PortInfo, error)
	FindOrphans() ([]PortInfo, error)
	KillProcess(pid int) error
}
