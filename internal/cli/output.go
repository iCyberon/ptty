package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/iCyberon/ptty/internal/scanner"
)

func PrintPortTable(ports []scanner.PortInfo, colorEnabled bool) {
	if !colorEnabled {
		color.NoColor = true
	}

	if len(ports) == 0 {
		fmt.Println("No listening ports found.")
		fmt.Println("Start a dev server to see it here.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	header := color.New(color.FgHiBlack).SprintFunc()
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		header("PORT"), header("PROCESS"), header("PID"),
		header("PROJECT"), header("FRAMEWORK"), header("UPTIME"), header("STATUS"))

	for _, p := range ports {
		portStr := color.New(color.FgHiBlue, color.Bold).Sprintf(":%d", p.Port)
		pidStr := color.New(color.FgHiBlack).Sprintf("%d", p.PID)
		project := p.ProjectName
		if project == "" {
			project = "—"
		}
		framework := p.Framework
		if framework == "" {
			framework = "—"
		}
		uptime := formatDuration(p.Uptime)
		status := formatStatus(p.Status)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			portStr, p.ProcessName, pidStr, project, framework, uptime, status)
	}

	w.Flush()
	fmt.Printf("\n%d ports\n", len(ports))
}

func PrintPortDetail(info *scanner.PortInfo, colorEnabled bool) {
	if !colorEnabled {
		color.NoColor = true
	}

	title := color.New(color.FgHiBlue, color.Bold)
	label := color.New(color.FgHiBlack)

	title.Printf("Port :%d\n\n", info.Port)

	fmt.Printf("  %s  %s (PID %d)\n", label.Sprint("Process"), info.ProcessName, info.PID)
	fmt.Printf("  %s  %s\n", label.Sprint("Status "), formatStatus(info.Status))

	if info.Framework != "" {
		fmt.Printf("  %s  %s\n", label.Sprint("Framewk"), info.Framework)
	}

	fmt.Printf("  %s  %s\n", label.Sprint("Memory "), formatMemory(info.Memory))
	fmt.Printf("  %s  %.1f%%\n", label.Sprint("CPU    "), info.CPU)
	fmt.Printf("  %s  %s\n", label.Sprint("Uptime "), formatDuration(info.Uptime))

	if info.CWD != "" {
		fmt.Printf("  %s  %s\n", label.Sprint("Dir    "), info.CWD)
	}
	if info.ProjectName != "" {
		fmt.Printf("  %s  %s\n", label.Sprint("Project"), info.ProjectName)
	}
	if info.GitBranch != "" {
		branchColor := color.New(color.FgMagenta)
		fmt.Printf("  %s  %s\n", label.Sprint("Branch "), branchColor.Sprint(info.GitBranch))
	}

	if len(info.ProcessTree) > 0 {
		fmt.Printf("\n  %s\n", label.Sprint("Process Tree"))
		for i, node := range info.ProcessTree {
			indent := ""
			prefix := "└─ "
			for j := 0; j < i; j++ {
				indent += "   "
			}
			marker := ""
			if i == len(info.ProcessTree)-1 {
				marker = " ← this"
			}
			fmt.Printf("  %s%s%s (PID %d)%s\n", indent, prefix, node.Name, node.PID, marker)
		}
	}
}

func PrintProcessTable(procs []scanner.PortInfo, colorEnabled bool) {
	if !colorEnabled {
		color.NoColor = true
	}

	if len(procs) == 0 {
		fmt.Println("No dev processes found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	header := color.New(color.FgHiBlack).SprintFunc()
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		header("PID"), header("PROCESS"), header("CPU%"),
		header("MEMORY"), header("PROJECT"), header("FRAMEWORK"), header("UPTIME"))

	for _, p := range procs {
		cpuStr := formatCPU(p.CPU)
		project := p.ProjectName
		if project == "" {
			project = "—"
		}
		framework := p.Framework
		if framework == "" {
			framework = "—"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			p.PID, p.ProcessName, cpuStr, formatMemory(p.Memory),
			project, framework, formatDuration(p.Uptime))
	}

	w.Flush()
	fmt.Printf("\n%d processes\n", len(procs))
}

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func RunCleanInteractive(s scanner.Scanner, orphans []scanner.PortInfo, colorEnabled bool) error {
	if !colorEnabled {
		color.NoColor = true
	}

	if len(orphans) == 0 {
		fmt.Println("No orphaned or zombie processes found. Everything looks clean!")
		return nil
	}

	fmt.Printf("Found %d orphaned/zombie processes:\n\n", len(orphans))

	for _, o := range orphans {
		status := formatStatus(o.Status)
		fmt.Printf("  PID %-8d  %-20s  %s  %s\n",
			o.PID, o.ProcessName, formatMemory(o.Memory), status)
	}

	fmt.Printf("\nKill all %d processes? [y/N] ", len(orphans))

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	success := 0
	for _, o := range orphans {
		err := s.KillProcess(o.PID)
		if err != nil {
			errColor := color.New(color.FgRed)
			errColor.Printf("  ✕ PID %d (%s): %v\n", o.PID, o.ProcessName, err)
		} else {
			okColor := color.New(color.FgGreen)
			okColor.Printf("  ✓ PID %d (%s) killed\n", o.PID, o.ProcessName)
			success++
		}
	}

	fmt.Printf("\nCleaned %d/%d processes.\n", success, len(orphans))
	return nil
}

func RunWatchJSON(s scanner.Scanner, devOnly bool) error {
	enc := json.NewEncoder(os.Stdout)
	prevPorts := make(map[int]scanner.PortInfo)

	ports, err := s.ListPorts(devOnly)
	if err != nil {
		return err
	}
	for _, p := range ports {
		prevPorts[p.Port] = p
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ports, err := s.ListPorts(devOnly)
		if err != nil {
			continue
		}

		current := make(map[int]scanner.PortInfo)
		for _, p := range ports {
			current[p.Port] = p
		}

		for port, info := range current {
			if _, existed := prevPorts[port]; !existed {
				enc.Encode(map[string]interface{}{
					"event":     "new",
					"timestamp": time.Now().Format(time.RFC3339),
					"port":      info,
				})
			}
		}

		for port, info := range prevPorts {
			if _, exists := current[port]; !exists {
				enc.Encode(map[string]interface{}{
					"event":     "closed",
					"timestamp": time.Now().Format(time.RFC3339),
					"port":      info,
				})
			}
		}

		prevPorts = current
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

func formatMemory(bytes uint64) string {
	if bytes == 0 {
		return "—"
	}
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatStatus(s scanner.Status) string {
	switch s {
	case scanner.StatusHealthy:
		return color.New(color.FgGreen).Sprint("● healthy")
	case scanner.StatusOrphaned:
		return color.New(color.FgYellow).Sprint("● orphaned")
	case scanner.StatusZombie:
		return color.New(color.FgRed).Sprint("● zombie")
	default:
		return color.New(color.FgHiBlack).Sprint("● unknown")
	}
}

func formatCPU(cpu float64) string {
	s := fmt.Sprintf("%.1f%%", cpu)
	switch {
	case cpu > 25:
		return color.New(color.FgRed).Sprint(s)
	case cpu > 5:
		return color.New(color.FgYellow).Sprint(s)
	default:
		return color.New(color.FgGreen).Sprint(s)
	}
}
