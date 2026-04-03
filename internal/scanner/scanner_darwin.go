//go:build darwin

package scanner

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/iCyberon/ptty/internal/detect"
)

type darwinScanner struct{}

func New() Scanner {
	return &darwinScanner{}
}

func (s *darwinScanner) ListPorts(devOnly bool) ([]PortInfo, error) {
	raw, err := execCommand("lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n")
	if err != nil {
		return nil, fmt.Errorf("lsof: %w", err)
	}

	entries := parseLsofOutput(raw)
	if len(entries) == 0 {
		return nil, nil
	}

	pidSet := make(map[int]bool)
	for _, e := range entries {
		pidSet[e.pid] = true
	}
	pids := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		pids = append(pids, pid)
	}

	psInfo := batchPsInfo(pids)
	cwds := batchCwd(pids)
	dockerInfo := batchDockerInfo()

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
			Command:     e.command,
			Status:      StatusHealthy,
		}

		if ps, ok := psInfo[e.pid]; ok {
			info.Command = ps.command
			info.Memory = ps.rss
			info.StartTime = ps.startTime
			if !ps.startTime.IsZero() {
				info.Uptime = time.Since(ps.startTime)
			}
			if ps.stat != "" && strings.Contains(ps.stat, "Z") {
				info.Status = StatusZombie
			} else if ps.ppid == 1 {
				info.Status = StatusOrphaned
			}
		}

		if cwd, ok := cwds[e.pid]; ok {
			info.CWD = cwd
		}

		var dockerImage string
		if di, ok := dockerInfo[e.port]; ok {
			dockerImage = di.image
			info.ProcessName = di.image
			info.Command = fmt.Sprintf("docker:%s", di.containerID)
		}

		projectRoot := detect.FindProjectRoot(info.CWD)
		info.ProjectName = detect.ProjectName(projectRoot)
		info.Framework = detect.DetectFramework(projectRoot, info.CWD, info.Command, info.ProcessName, dockerImage)

		if devOnly && !detect.IsDevProcess(info.ProcessName, info.Command) {
			continue
		}

		ports = append(ports, info)
	}

	return ports, nil
}

func (s *darwinScanner) GetPortDetail(port int) (*PortInfo, error) {
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
			if ps := getSinglePsCPU(info.PID); ps != nil {
				info.CPU = ps.cpu
			}
			return info, nil
		}
	}

	return nil, fmt.Errorf("no process found listening on port %d", port)
}

func (s *darwinScanner) GetAllProcesses(devOnly bool) ([]PortInfo, error) {
	raw, err := execCommand("ps", "-eo", "pid=,pcpu=,pmem=,rss=,lstart=,command=")
	if err != nil {
		return nil, fmt.Errorf("ps: %w", err)
	}

	procs := parseAllProcesses(raw)
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

func (s *darwinScanner) FindOrphans() ([]PortInfo, error) {
	raw, err := execCommand("ps", "-eo", "pid=,ppid=,stat=,rss=,lstart=,command=")
	if err != nil {
		return nil, fmt.Errorf("ps: %w", err)
	}

	return parseOrphans(raw), nil
}

func (s *darwinScanner) KillProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

type rawPortEntry struct {
	pid         int
	port        int
	processName string
	command     string
}

type psInfo struct {
	pid       int
	ppid      int
	stat      string
	rss       uint64
	startTime time.Time
	command   string
	cpu       float64
}

type dockerEntry struct {
	containerID string
	image       string
	ports       string
}

func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func parseLsofOutput(raw string) []rawPortEntry {
	var entries []rawPortEntry
	lines := strings.Split(raw, "\n")

	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		processName := fields[0]
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		var name string
		for _, f := range fields[8:] {
			if strings.Contains(f, ":") && f != "(LISTEN)" {
				name = f
				break
			}
		}
		if name == "" {
			continue
		}
		port := parsePortFromLsofName(name)
		if port <= 0 {
			continue
		}

		entries = append(entries, rawPortEntry{
			pid:         pid,
			port:        port,
			processName: processName,
			command:     processName,
		})
	}

	return entries
}

func parsePortFromLsofName(name string) int {
	idx := strings.LastIndex(name, ":")
	if idx < 0 {
		return 0
	}
	port, err := strconv.Atoi(name[idx+1:])
	if err != nil {
		return 0
	}
	return port
}

func batchPsInfo(pids []int) map[int]psInfo {
	if len(pids) == 0 {
		return nil
	}

	pidStrs := make([]string, len(pids))
	for i, pid := range pids {
		pidStrs[i] = strconv.Itoa(pid)
	}

	args := []string{"-p", strings.Join(pidStrs, ","), "-o", "pid=,ppid=,stat=,rss=,lstart=,command="}
	raw, err := execCommand("ps", args...)
	if err != nil {
		return nil
	}

	result := make(map[int]psInfo)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		info := parsePsLine(line)
		if info.pid > 0 {
			result[info.pid] = info
		}
	}

	return result
}

func parsePsLine(line string) psInfo {
	var info psInfo
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return info
	}

	info.pid, _ = strconv.Atoi(fields[0])
	info.ppid, _ = strconv.Atoi(fields[1])
	info.stat = fields[2]
	rss, _ := strconv.ParseUint(fields[3], 10, 64)
	info.rss = rss * 1024

	if len(fields) >= 9 {
		lstart := strings.Join(fields[4:9], " ")
		info.startTime = parseLstart(lstart)
		info.command = strings.Join(fields[9:], " ")
	}

	return info
}

func parseLstart(lstart string) time.Time {
	layouts := []string{
		"Mon Jan  2 15:04:05 2006",
		"Mon Jan 2 15:04:05 2006",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, lstart)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func batchCwd(pids []int) map[int]string {
	if len(pids) == 0 {
		return nil
	}

	pidStrs := make([]string, len(pids))
	for i, pid := range pids {
		pidStrs[i] = strconv.Itoa(pid)
	}

	args := []string{"-a", "-d", "cwd", "-p", strings.Join(pidStrs, ",")}
	raw, err := execCommand("lsof", args...)
	if err != nil {
		return nil
	}

	result := make(map[int]string)
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		result[pid] = fields[len(fields)-1]
	}

	return result
}

func batchDockerInfo() map[int]dockerEntry {
	raw, err := execCommand("docker", "ps", "--format", "{{.ID}}\t{{.Image}}\t{{.Ports}}")
	if err != nil {
		return nil // Docker not running, skip
	}

	result := make(map[int]dockerEntry)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}

		entry := dockerEntry{
			containerID: parts[0],
			image:       parts[1],
			ports:       parts[2],
		}

		for _, mapping := range strings.Split(entry.ports, ", ") {
			port := parseDockerPort(mapping)
			if port > 0 {
				result[port] = entry
			}
		}
	}

	return result
}

func parseDockerPort(mapping string) int {
	arrow := strings.Index(mapping, "->")
	if arrow < 0 {
		return 0
	}
	hostPart := mapping[:arrow]
	colonIdx := strings.LastIndex(hostPart, ":")
	if colonIdx < 0 {
		return 0
	}
	port, err := strconv.Atoi(hostPart[colonIdx+1:])
	if err != nil {
		return 0
	}
	return port
}

func getGitBranch(dir string) string {
	raw, err := execCommand("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(raw)
}

func getProcessTree(pid int) []ProcessNode {
	raw, err := execCommand("ps", "-eo", "pid=,ppid=,comm=")
	if err != nil {
		return nil
	}

	parentMap := make(map[int]int)
	nameMap := make(map[int]string)

	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		p, _ := strconv.Atoi(fields[0])
		pp, _ := strconv.Atoi(fields[1])
		if p > 0 {
			parentMap[p] = pp
			nameMap[p] = strings.Join(fields[2:], " ")
		}
	}

	var tree []ProcessNode
	current := pid
	for i := 0; i < 8; i++ {
		ppid, ok := parentMap[current]
		if !ok || current <= 1 {
			break
		}
		tree = append(tree, ProcessNode{
			PID:     current,
			Name:    nameMap[current],
			Command: nameMap[current],
		})
		current = ppid
	}

	for i, j := 0, len(tree)-1; i < j; i, j = i+1, j-1 {
		tree[i], tree[j] = tree[j], tree[i]
	}

	return tree
}

func getSinglePsCPU(pid int) *psInfo {
	raw, err := execCommand("ps", "-p", strconv.Itoa(pid), "-o", "pid=,pcpu=")
	if err != nil {
		return nil
	}
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) < 2 {
		return nil
	}
	cpu, _ := strconv.ParseFloat(fields[1], 64)
	return &psInfo{pid: pid, cpu: cpu}
}

func parseAllProcesses(raw string) []PortInfo {
	var procs []PortInfo
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 {
			continue
		}

		cpu, _ := strconv.ParseFloat(fields[1], 64)
		rss, _ := strconv.ParseUint(fields[3], 10, 64)
		lstart := strings.Join(fields[4:9], " ")
		command := strings.Join(fields[9:], " ")

		processName := extractProcessName(command)

		procs = append(procs, PortInfo{
			PID:         pid,
			ProcessName: processName,
			Command:     command,
			CPU:         cpu,
			Memory:      rss * 1024,
			StartTime:   parseLstart(lstart),
			Uptime:      time.Since(parseLstart(lstart)),
			Status:      StatusHealthy,
		})
	}
	return procs
}

func parseOrphans(raw string) []PortInfo {
	var orphans []PortInfo
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		info := parsePsLine(line)
		if info.pid <= 1 {
			continue
		}

		status := StatusHealthy
		if strings.Contains(info.stat, "Z") {
			status = StatusZombie
		} else if info.ppid == 1 {
			status = StatusOrphaned
		} else {
			continue // Not an orphan or zombie
		}

		processName := extractProcessName(info.command)

		orphans = append(orphans, PortInfo{
			PID:         info.pid,
			ProcessName: processName,
			Command:     info.command,
			Memory:      info.rss,
			StartTime:   info.startTime,
			Uptime:      time.Since(info.startTime),
			Status:      status,
		})
	}
	return orphans
}

func extractProcessName(command string) string {
	if command == "" {
		return ""
	}
	parts := strings.Fields(command)
	name := parts[0]
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}
