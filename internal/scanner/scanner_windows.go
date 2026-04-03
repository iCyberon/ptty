//go:build windows

package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iCyberon/ptty/internal/detect"
)

type windowsScanner struct{}

func New() Scanner {
	return &windowsScanner{}
}

func (s *windowsScanner) ListPorts(devOnly bool) ([]PortInfo, error) {
	raw, err := execCommand("netstat", "-ano", "-p", "TCP")
	if err != nil {
		return nil, fmt.Errorf("netstat: %w", err)
	}

	entries := parseNetstatOutput(raw)

	seen := make(map[int]bool)
	var ports []PortInfo

	for _, e := range entries {
		if seen[e.port] {
			continue
		}
		seen[e.port] = true

		info := PortInfo{
			Port:        e.port,
			PID:         e.pid,
			ProcessName: e.processName,
			Status:      StatusHealthy,
		}

		enrichFromTasklist(&info)

		projectRoot := detect.FindProjectRoot(info.CWD)
		info.ProjectName = detect.ProjectName(projectRoot)
		info.Framework = detect.DetectFramework(projectRoot, info.CWD, info.Command, info.ProcessName, "")

		if devOnly && !detect.IsDevProcess(info.ProcessName, info.Command) {
			continue
		}

		ports = append(ports, info)
	}

	return ports, nil
}

func (s *windowsScanner) GetPortDetail(port int) (*PortInfo, error) {
	ports, err := s.ListPorts(false)
	if err != nil {
		return nil, err
	}

	for i := range ports {
		if ports[i].Port == port {
			info := &ports[i]
			if info.CWD != "" {
				info.GitBranch = getGitBranch(info.CWD)
			}
			info.ProcessTree = getProcessTree(info.PID)
			return info, nil
		}
	}

	return nil, fmt.Errorf("no process found listening on port %d", port)
}

func (s *windowsScanner) GetAllProcesses(devOnly bool) ([]PortInfo, error) {
	raw, err := execCommand("tasklist", "/FO", "CSV", "/V")
	if err != nil {
		return nil, fmt.Errorf("tasklist: %w", err)
	}

	procs := parseTasklistOutput(raw)

	if !devOnly {
		return procs, nil
	}

	var filtered []PortInfo
	for _, p := range procs {
		if detect.IsDevProcess(p.ProcessName, p.Command) {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

func (s *windowsScanner) FindOrphans() ([]PortInfo, error) {
	// Windows lacks PPID=1 orphan detection; use "Not Responding" status instead.
	raw, err := execCommand("tasklist", "/FO", "CSV", "/V")
	if err != nil {
		return nil, fmt.Errorf("tasklist: %w", err)
	}

	var orphans []PortInfo
	for _, line := range strings.Split(raw, "\n")[1:] {
		fields := parseCSVLine(line)
		if len(fields) < 9 {
			continue
		}

		pid, _ := strconv.Atoi(strings.Trim(fields[1], "\""))
		if pid <= 4 {
			continue
		}

		name := strings.Trim(fields[0], "\"")
		status := strings.Trim(fields[6], "\"")

		if strings.Contains(strings.ToLower(status), "not responding") {
			orphans = append(orphans, PortInfo{
				PID:         pid,
				ProcessName: name,
				Status:      StatusOrphaned,
			})
		}
	}

	return orphans, nil
}

func (s *windowsScanner) KillProcess(pid int) error {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F")
	return cmd.Run()
}

type rawPortEntry struct {
	pid         int
	port        int
	processName string
}

func parseNetstatOutput(raw string) []rawPortEntry {
	var entries []rawPortEntry

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		if fields[0] != "TCP" {
			continue
		}
		if fields[3] != "LISTENING" {
			continue
		}

		port := parseNetstatPort(fields[1])
		if port <= 0 {
			continue
		}

		pid, _ := strconv.Atoi(fields[4])
		if pid <= 0 {
			continue
		}

		entries = append(entries, rawPortEntry{
			pid:  pid,
			port: port,
		})
	}

	return entries
}

func parseNetstatPort(addr string) int {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return 0
	}
	port, _ := strconv.Atoi(addr[idx+1:])
	return port
}

func parseTasklistOutput(raw string) []PortInfo {
	var procs []PortInfo

	for _, line := range strings.Split(raw, "\n")[1:] {
		fields := parseCSVLine(line)
		if len(fields) < 6 {
			continue
		}

		name := strings.Trim(fields[0], "\"")
		pid, _ := strconv.Atoi(strings.Trim(fields[1], "\""))
		if pid <= 4 {
			continue
		}

		memStr := strings.Trim(fields[4], "\"")
		memStr = strings.ReplaceAll(memStr, ",", "")
		memStr = strings.ReplaceAll(memStr, " K", "")
		memStr = strings.TrimSpace(memStr)
		memKB, _ := strconv.ParseUint(memStr, 10, 64)

		procs = append(procs, PortInfo{
			PID:         pid,
			ProcessName: strings.TrimSuffix(name, ".exe"),
			Command:     name,
			Memory:      memKB * 1024,
			Status:      StatusHealthy,
		})
	}

	return procs
}

func parseCSVLine(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range line {
		switch {
		case r == '"':
			inQuotes = !inQuotes
			current.WriteRune(r)
		case r == ',' && !inQuotes:
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

func enrichFromTasklist(info *PortInfo) {
	if info.PID <= 0 {
		return
	}

	raw, err := execCommand("tasklist", "/FI", fmt.Sprintf("PID eq %d", info.PID), "/FO", "CSV", "/V")
	if err != nil {
		return
	}

	for _, line := range strings.Split(raw, "\n")[1:] {
		fields := parseCSVLine(line)
		if len(fields) < 6 {
			continue
		}

		name := strings.Trim(fields[0], "\"")
		info.ProcessName = strings.TrimSuffix(name, ".exe")

		memStr := strings.Trim(fields[4], "\"")
		memStr = strings.ReplaceAll(memStr, ",", "")
		memStr = strings.ReplaceAll(memStr, " K", "")
		memStr = strings.TrimSpace(memStr)
		memKB, _ := strconv.ParseUint(memStr, 10, 64)
		info.Memory = memKB * 1024
	}

	raw, err = execCommand("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", info.PID),
		"get", "CommandLine,ExecutablePath", "/format:list")
	if err != nil {
		return
	}

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CommandLine=") {
			info.Command = strings.TrimPrefix(line, "CommandLine=")
		}
		if strings.HasPrefix(line, "ExecutablePath=") {
			exePath := strings.TrimPrefix(line, "ExecutablePath=")
			if exePath != "" {
				info.CWD = filepath.Dir(exePath)
			}
		}
	}
}

func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "CHCP=65001")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func getGitBranch(dir string) string {
	raw, err := execCommand("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(raw)
}

func getProcessTree(pid int) []ProcessNode {
	var tree []ProcessNode
	current := pid

	for i := 0; i < 8; i++ {
		raw, err := execCommand("wmic", "process", "where",
			fmt.Sprintf("ProcessId=%d", current), "get", "Name,ParentProcessId", "/format:list")
		if err != nil || current <= 4 {
			break
		}

		var name string
		var ppid int
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Name=") {
				name = strings.TrimPrefix(line, "Name=")
			}
			if strings.HasPrefix(line, "ParentProcessId=") {
				ppid, _ = strconv.Atoi(strings.TrimPrefix(line, "ParentProcessId="))
			}
		}

		if name == "" {
			break
		}

		tree = append(tree, ProcessNode{
			PID:  current,
			Name: strings.TrimSuffix(name, ".exe"),
		})

		if ppid <= 4 || ppid == current {
			break
		}
		current = ppid
	}

	for i, j := 0, len(tree)-1; i < j; i, j = i+1, j-1 {
		tree[i], tree[j] = tree[j], tree[i]
	}

	return tree
}
