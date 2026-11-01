package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// GetSystemMetricsText gathers live system metrics using gopsutil.
func GetSystemMetricsText() string {
	var builder strings.Builder
	builder.WriteString("--- Live System Metrics ---\n")

	// CPU
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		builder.WriteString(fmt.Sprintf("CPU Usage: %.1f%%\n", cpuPercent[0]))
	}

	// RAM
	vMem, err := mem.VirtualMemory()
	if err == nil {
		builder.WriteString(fmt.Sprintf("RAM Usage: %.1f%% (%.2f GB / %.2f GB)\n",
			vMem.UsedPercent,
			float64(vMem.Used)/(1024*1024*1024),
			float64(vMem.Total)/(1024*1024*1024)))
	}

	// Disk
	d, err := disk.Usage("/")
	if err == nil {
		builder.WriteString(fmt.Sprintf("Disk Space (/): %.1f%% (%.2f GB / %.2f GB)\n",
			d.UsedPercent,
			float64(d.Used)/(1024*1024*1024),
			float64(d.Total)/(1024*1024*1024)))
	}

	// Network
	n, err := net.IOCounters(false)
	if err == nil && len(n) > 0 {
		builder.WriteString(fmt.Sprintf("Network: %.2f MB Sent | %.2f MB Received\n",
			float64(n[0].BytesSent)/(1024*1024),
			float64(n[0].BytesRecv)/(1024*1024)))
	}

	return builder.String()
}

// GetSystemLogs fetches recent system logs and strictly truncates them to protect API token limits.
func GetSystemLogs(lines int, service string) string {
	if lines > 100 {
		lines = 100 // Hard cap
	}

	args := []string{"-n", strconv.Itoa(lines), "--no-pager"}
	if service != "" {
		args = append(args, "-u", service)
	}

	cmd := exec.Command("journalctl", args...)
	output, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(output))

	if err != nil {
		return fmt.Sprintf("❌ Failed to fetch logs: %s", outStr)
	}

	if outStr == "" {
		return "No logs found."
	}

	// Truncate to 3000 characters to prevent API token explosion
	if len(outStr) > 3000 {
		outStr = "...[TRUNCATED]...\n" + outStr[len(outStr)-3000:]
	}

	return fmt.Sprintf("--- System Logs ---\n%s", outStr)
}

// GetTopProcessesText uses native Ubuntu 'ps' to grab the highest CPU consuming processes instantly.
func GetTopProcessesText(limit int) string {
	if limit <= 0 {
		limit = 5
	}

	// ps aux --sort=-%cpu | head -n (limit + 1)
	cmdStr := fmt.Sprintf("ps aux --sort=-%%cpu | head -n %d", limit+1)
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("❌ Failed to fetch processes: %v", err)
	}
	return fmt.Sprintf("--- Top Processes ---\n%s", strings.TrimSpace(string(output)))
}

// FindProcessByName searches for a running application.
func FindProcessByName(searchTerm string) string {
	cmdStr := fmt.Sprintf("ps aux | grep -i '%s' | grep -v grep", searchTerm)
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()

	if err != nil || len(output) == 0 {
		return fmt.Sprintf("❌ No process found matching: %s", searchTerm)
	}
	return fmt.Sprintf("--- Process Search Results ---\n%s", strings.TrimSpace(string(output)))
}
