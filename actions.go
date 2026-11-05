package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// getIntArg safely extracts an integer parameter from a generic map
func getIntArg(args map[string]any, key string, defaultVal int) int {
	if val, ok := args[key]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
	}
	return defaultVal
}

// getStringArg safely extracts a string parameter from a generic map
func getStringArg(args map[string]any, key string, defaultVal string) string {
	if val, ok := args[key]; ok && val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return defaultVal
}

// ExecuteTool routes an AI tool call to the corresponding native Go implementation.
func ExecuteTool(name string, arguments string) string {
	var args map[string]any
	if arguments != "" {
		_ = json.Unmarshal([]byte(arguments), &args)
	}
	if args == nil {
		args = make(map[string]any)
	}

	switch name {
	case "check_hardware":
		return GetSystemMetricsText()

	case "check_processes", "check_process":
		limit := getIntArg(args, "limit", 5)
		if limit <= 0 {
			limit = 5
		}
		return GetTopProcessesText(limit)

	case "find_process_by_name":
		term := getStringArg(args, "search_term", "")
		if term == "" {
			return "❌ Missing required parameter 'search_term'."
		}
		return FindProcessByName(term)

	case "kill_process":
		pid := getIntArg(args, "pid", 0)
		if pid <= 0 {
			return "❌ Invalid or missing Process ID (PID)."
		}
		return KillProcess(pid)

	case "manage_docker":
		action := getStringArg(args, "action", "")
		containerID := getStringArg(args, "container_id", "")
		if action == "" {
			return "❌ Missing required parameter 'action'."
		}
		return ManageDocker(action, containerID)

	case "read_system_file":
		path := getStringArg(args, "filepath", "")
		if path == "" {
			return "❌ Missing required parameter 'filepath'."
		}
		return ReadSystemFile(path)

	case "write_system_file":
		path := getStringArg(args, "filepath", "")
		content := getStringArg(args, "content", "")
		if path == "" {
			return "❌ Missing required parameter 'filepath'."
		}
		return WriteSystemFile(path, content)

	case "check_system_logs":
		lines := getIntArg(args, "lines", 50)
		service := getStringArg(args, "service", "")
		return GetSystemLogs(lines, service)

	case "run_shell_command":
		cmdStr := getStringArg(args, "command", "")
		if cmdStr == "" {
			return "❌ Missing required parameter 'command'."
		}
		return RunShellCommand(cmdStr)

	default:
		return fmt.Sprintf("❌ Tool '%s' not implemented in Go backend.", name)
	}
}

// RunShellCommand executes an arbitrary shell command safely with a 10-second timeout.
func RunShellCommand(command string) string {
	blacklist := []string{"rm -rf /", "rm -rf /*", "mkfs", "dd if=", ":(){ :|:& };:"}
	for _, badCmd := range blacklist {
		if strings.Contains(command, badCmd) {
			return "❌ Security Block: That command is highly destructive and blacklisted."
		}
	}

	// Create a strict 10-second timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("❌ Error: Command '%s' timed out after 10 seconds.", command)
	}

	outStr := strings.TrimSpace(string(output))
	if err != nil {
		if outStr == "" {
			return fmt.Sprintf("❌ Error executing command: %v", err)
		}
		return fmt.Sprintf("❌ Error: %s", outStr)
	}

	if outStr == "" {
		return fmt.Sprintf("✅ Command '%s' executed successfully with no output.", command)
	}
	return fmt.Sprintf("--- Output of '%s' ---\n%s", command, outStr)
}

// KillProcess forcefully kills a process by its PID using SIGKILL.
func KillProcess(pid int) string {
	if pid <= 1 {
		return "❌ Security Block: Killing system init process (PID <= 1) is forbidden."
	}

	// Try killing with standard user privileges first
	cmd := exec.Command("kill", "-9", strconv.Itoa(pid))
	output, err := cmd.CombinedOutput()
	if err == nil {
		return fmt.Sprintf("✅ Successfully executed SIGKILL on PID %d.", pid)
	}

	// If failed, report the error safely without hanging the TUI on a sudo password prompt
	outStr := strings.TrimSpace(string(output))
	if outStr != "" {
		return fmt.Sprintf("❌ Failed to kill PID %d: %s", pid, outStr)
	}
	return fmt.Sprintf("❌ Failed to kill PID %d: %v (requires elevated privileges)", pid, err)
}

// WriteSystemFile writes or overwrites a file with new content.
func WriteSystemFile(filepath string, content string) string {
	forbidden := []string{"/etc", "/bin", "/sbin", "/usr", "/boot", "/sys", "/proc", "/dev"}
	for _, prefix := range forbidden {
		if strings.HasPrefix(filepath, prefix) {
			return fmt.Sprintf("❌ Security Block: Writing to '%s' system directory is forbidden.", prefix)
		}
	}

	err := os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("❌ Failed to write file: %v", err)
	}
	return fmt.Sprintf("✅ Successfully wrote %d characters to %s.", len(content), filepath)
}

// ReadSystemFile reads the contents of a specific file.
func ReadSystemFile(filepath string) string {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Sprintf("❌ Failed to read file: %v", err)
	}
	return string(data)
}

// ManageDocker handles listing, stopping, or restarting containers.
func ManageDocker(action string, containerID string) string {
	var cmd *exec.Cmd

	switch action {
	case "list":
		cmd = exec.Command("docker", "ps", "-a")
	case "stop":
		if containerID == "" {
			return "❌ Missing required container ID to stop."
		}
		cmd = exec.Command("docker", "stop", containerID)
	case "restart":
		if containerID == "" {
			return "❌ Missing required container ID to restart."
		}
		cmd = exec.Command("docker", "restart", containerID)
	default:
		return "❌ Invalid docker action. Use list, stop, or restart."
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("❌ Docker command failed: %s", strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output))
}
