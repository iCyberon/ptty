//go:build linux

package scanner

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/iCyberon/ptty/internal/detect"
)

type linuxScanner struct{}

func New() Scanner {
	return &linuxScanner{}
}

func (s *linuxScanner) ListPorts(devOnly bool) ([]PortInfo, error) {
	entries, err := parseProcNetTCP()
	if err != nil {
		entries, err = parseSS()
		if err != nil {
			return nil, fmt.Errorf("failed to scan ports: %w", err)
		}
	}

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

		enrichFromProc(&info)

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

func (s *linuxScanner) GetPortDetail(port int) (*PortInfo, error) {
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
			info.CPU = readProcCPU(info.PID)
			return info, nil
		}
	}

	return nil, fmt.Errorf("no process found listening on port %d", port)
}

func (s *linuxScanner) GetAllProcesses(devOnly bool) ([]PortInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("reading /proc: %w", err)
	}

	var procs []PortInfo
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}

		info := PortInfo{PID: pid, Status: StatusHealthy}
		enrichFromProc(&info)

		if info.ProcessName == "" {
			continue
		}

		if devOnly && !detect.IsDevProcess(info.ProcessName, info.Command) {
			continue
		}

		procs = append(procs, info)
	}

	return procs, nil
}

func (s *linuxScanner) FindOrphans() ([]PortInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("reading /proc: %w", err)
	}

	var orphans []PortInfo
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 1 {
			continue
		}

		stat := readProcStat(pid)
		if stat.state == "Z" {
			info := PortInfo{
				PID:         pid,
				ProcessName: stat.comm,
				Status:      StatusZombie,
			}
			orphans = append(orphans, info)
		} else if stat.ppid == 1 && detect.IsDevProcess(stat.comm, stat.comm) {
			info := PortInfo{
				PID:         pid,
				ProcessName: stat.comm,
				Status:      StatusOrphaned,
			}
			enrichFromProc(&info)
			orphans = append(orphans, info)
		}
	}

	return orphans, nil
}

func (s *linuxScanner) KillProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

type rawPortEntry struct {
	pid         int
	port        int
	processName string
	inode       uint64
}

func parseProcNetTCP() ([]rawPortEntry, error) {
	var all []rawPortEntry

	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines[1:] { // skip header
			fields := strings.Fields(line)
			if len(fields) < 10 {
				continue
			}

			if fields[3] != "0A" {
				continue
			}

			port := parseHexPort(fields[1])
			if port <= 0 {
				continue
			}

			inode, _ := strconv.ParseUint(fields[9], 10, 64)
			all = append(all, rawPortEntry{
				port:  port,
				inode: inode,
			})
		}
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no listening ports found in /proc/net/tcp")
	}

	inodeToPID := buildInodePIDMap()
	for i := range all {
		if pid, ok := inodeToPID[all[i].inode]; ok {
			all[i].pid = pid
			all[i].processName = readProcComm(pid)
		}
	}

	return all, nil
}

func parseHexPort(localAddr string) int {
	parts := strings.SplitN(localAddr, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	portBytes, err := hex.DecodeString(parts[1])
	if err != nil || len(portBytes) < 2 {
		return 0
	}
	return int(portBytes[0])<<8 | int(portBytes[1])
}

func buildInodePIDMap() map[uint64]int {
	result := make(map[uint64]int)

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return result
	}

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}

		fdPath := filepath.Join("/proc", entry.Name(), "fd")
		fds, err := os.ReadDir(fdPath)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdPath, fd.Name()))
			if err != nil {
				continue
			}
			if strings.HasPrefix(link, "socket:[") {
				inodeStr := link[8 : len(link)-1]
				inode, _ := strconv.ParseUint(inodeStr, 10, 64)
				if inode > 0 {
					result[inode] = pid
				}
			}
		}
	}

	return result
}

func parseSS() ([]rawPortEntry, error) {
	raw, err := execCommand("ss", "-tlnp")
	if err != nil {
		return nil, err
	}

	var entries []rawPortEntry
	for _, line := range strings.Split(raw, "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		port := parseSSPort(fields[3])
		if port <= 0 {
			continue
		}

		pid := 0
		processName := ""
		for _, f := range fields {
			if strings.HasPrefix(f, "users:") {
				pid, processName = parseSSUsers(f)
			}
		}

		entries = append(entries, rawPortEntry{
			pid:         pid,
			port:        port,
			processName: processName,
		})
	}

	return entries, nil
}

func parseSSPort(addr string) int {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return 0
	}
	port, _ := strconv.Atoi(addr[idx+1:])
	return port
}

func parseSSUsers(field string) (int, string) {
	pidIdx := strings.Index(field, "pid=")
	if pidIdx < 0 {
		return 0, ""
	}
	rest := field[pidIdx+4:]
	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		commaIdx = strings.Index(rest, ")")
	}
	if commaIdx < 0 {
		return 0, ""
	}
	pid, _ := strconv.Atoi(rest[:commaIdx])

	nameStart := strings.Index(field, "((\"")
	nameEnd := strings.Index(field, "\",")
	name := ""
	if nameStart >= 0 && nameEnd > nameStart {
		name = field[nameStart+3 : nameEnd]
	}

	return pid, name
}

type procStat struct {
	pid   int
	comm  string
	state string
	ppid  int
}

func readProcStat(pid int) procStat {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return procStat{}
	}

	content := string(data)
	// comm is in parens and may itself contain parens, so use last ")"
	start := strings.Index(content, "(")
	end := strings.LastIndex(content, ")")
	if start < 0 || end < 0 || end <= start {
		return procStat{}
	}

	comm := content[start+1 : end]
	rest := strings.Fields(content[end+2:])
	if len(rest) < 2 {
		return procStat{pid: pid, comm: comm}
	}

	ppid, _ := strconv.Atoi(rest[1])
	return procStat{
		pid:   pid,
		comm:  comm,
		state: rest[0],
		ppid:  ppid,
	}
}

func readProcComm(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readProcCmdline(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(data), "\x00", " ")
}

func readProcCwd(pid int) string {
	link, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		return ""
	}
	return link
}

func readProcMemory(pid int) uint64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseUint(fields[1], 10, 64)
				return kb * 1024
			}
		}
	}
	return 0
}

func readProcStartTime(pid int) time.Time {
	info, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func readProcCPU(pid int) float64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	content := string(data)
	end := strings.LastIndex(content, ")")
	if end < 0 {
		return 0
	}
	fields := strings.Fields(content[end+2:])
	if len(fields) < 20 {
		return 0
	}
	utime, _ := strconv.ParseFloat(fields[11], 64)
	stime, _ := strconv.ParseFloat(fields[12], 64)
	startTick, _ := strconv.ParseFloat(fields[19], 64)

	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	uptimeFields := strings.Fields(string(uptimeData))
	if len(uptimeFields) < 1 {
		return 0
	}
	systemUptime, _ := strconv.ParseFloat(uptimeFields[0], 64)

	const clockTick = 100.0
	processSeconds := (utime + stime) / clockTick
	processWallTime := systemUptime - (startTick / clockTick)
	if processWallTime <= 0 {
		return 0
	}

	return (processSeconds / processWallTime) * 100.0
}

func enrichFromProc(info *PortInfo) {
	if info.PID <= 0 {
		return
	}
	if info.ProcessName == "" {
		info.ProcessName = readProcComm(info.PID)
	}
	info.Command = readProcCmdline(info.PID)
	info.CWD = readProcCwd(info.PID)
	info.Memory = readProcMemory(info.PID)
	info.StartTime = readProcStartTime(info.PID)
	if !info.StartTime.IsZero() {
		info.Uptime = time.Since(info.StartTime)
	}

	stat := readProcStat(info.PID)
	if stat.state == "Z" {
		info.Status = StatusZombie
	} else if stat.ppid == 1 {
		info.Status = StatusOrphaned
	}
}

func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
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
		stat := readProcStat(current)
		if stat.pid <= 1 {
			break
		}
		tree = append(tree, ProcessNode{
			PID:  current,
			Name: stat.comm,
		})
		current = stat.ppid
	}

	for i, j := 0, len(tree)-1; i < j; i, j = i+1, j-1 {
		tree[i], tree[j] = tree[j], tree[i]
	}

	return tree
}
