package scanner

import "time"

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
