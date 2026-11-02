package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRunShellCommand(t *testing.T) {
	// Normal command
	out := RunShellCommand("echo 'omnikernel-test'")
	if !strings.Contains(out, "omnikernel-test") {
		t.Fatalf("expected output to contain 'omnikernel-test', got: %s", out)
	}

	// Blacklisted command
	outBlacklist := RunShellCommand("rm -rf /")
	if !strings.Contains(outBlacklist, "Security Block") {
		t.Fatalf("expected security block for blacklisted command, got: %s", outBlacklist)
	}
}

func TestWriteAndReadSystemFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello from OmniKernel!"

	writeRes := WriteSystemFile(testFile, content)
	if !strings.Contains(writeRes, "Successfully wrote") {
		t.Fatalf("expected write success, got: %s", writeRes)
	}

	readRes := ReadSystemFile(testFile)
	if readRes != content {
		t.Fatalf("expected read content %q, got %q", content, readRes)
	}

	// Forbidden path
	secRes := WriteSystemFile("/etc/omnikernel_test", "fail")
	if !strings.Contains(secRes, "Security Block") {
		t.Fatalf("expected security block on /etc, got: %s", secRes)
	}
}

func TestExecuteTool(t *testing.T) {
	// 1. check_hardware
	res := ExecuteTool("check_hardware", "{}")
	if !strings.Contains(res, "Live System Metrics") {
		t.Fatalf("expected Live System Metrics, got: %s", res)
	}

	// 2. run_shell_command
	resCmd := ExecuteTool("run_shell_command", `{"command": "echo 'ok'"}`)
	if !strings.Contains(resCmd, "ok") {
		t.Fatalf("expected shell command output 'ok', got: %s", resCmd)
	}

	// 3. Unknown tool
	resUnknown := ExecuteTool("nonexistent_tool", "{}")
	if !strings.Contains(resUnknown, "not implemented") {
		t.Fatalf("expected not implemented error, got: %s", resUnknown)
	}
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		secs     uint64
		expected string
	}{
		{30, "0m 30s"},
		{120, "2m 0s"},
		{3665, "1h 1m"},
		{90000, "1d 1h 0m"},
	}

	for _, c := range cases {
		got := formatUptime(c.secs)
		if got != c.expected {
			t.Errorf("formatUptime(%d) = %q; want %q", c.secs, got, c.expected)
		}
	}
}
