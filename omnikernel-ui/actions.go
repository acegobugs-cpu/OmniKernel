package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// RunShellCommand executes an arbitrary shell command safely with a 10-second timeout.
func RunShellCommand(command string) string {
	blacklist := []string{"rm -rf /", "mkfs", "dd if="}
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
	cmd := exec.Command("sudo", "kill", "-9", strconv.Itoa(pid))
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("❌ Failed to kill PID %d: %s", pid, strings.TrimSpace(string(output)))
	}
	return fmt.Sprintf("✅ Successfully executed SIGKILL on PID %d.", pid)
}

// WriteSystemFile writes or overwrites a file with new content.
func WriteSystemFile(filepath string, content string) string {
	if strings.HasPrefix(filepath, "/etc") || strings.HasPrefix(filepath, "/bin") {
		return "❌ Security Block: Writing to system directories is forbidden."
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
		cmd = exec.Command("docker", "stop", containerID)
	case "restart":
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
