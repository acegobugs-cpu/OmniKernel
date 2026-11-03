package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	netstat "github.com/shirou/gopsutil/v3/net"
)

type tickMsg time.Time

func tickEverySecond() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type aiDoneMsg string
type aiToolCallMsg struct {
	ToolCallID string
	Name       string
	Arguments  string
}

type apiResponse struct {
	Status     string `json:"status"`
	Response   string `json:"response,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Arguments  string `json:"arguments,omitempty"`
}

type DiskInfo struct {
	MountPoint string
	TotalGB    float64
	UsedGB     float64
	FreeGB     float64
}

type model struct {
	width       int
	height      int
	cpuUsage    float64
	coreUsages  []float64
	ramUsage    float64
	ramUsedGB   float64
	ramAvailGB  float64
	ramCachedGB float64
	ramFreeGB   float64
	ramTotalGB  float64
	disks       []DiskInfo
	diskUsages  []string
	swapUsedGB  float64
	swapFreeGB  float64
	swapTotalGB float64
	cpuTemp     float64
	netSentMB   float64
	netRecvMB   float64
	hostOS      string
	uptimeSec   uint64
	load1       float64
	load5       float64
	load15      float64
	chatLog     string
	textInput   textinput.Model
	isThinking  bool
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Ask the agent a question (e.g. check logs, top procs, disk usage)..."
	ti.Focus()
	ti.CharLimit = 256
	return model{
		chatLog:   "OmniKernel UI active. Ready for commands.\n",
		textInput: ti,
		hostOS:    "Linux",
	}
}

func (m model) Init() tea.Cmd { return tea.Batch(textinput.Blink, tickEverySecond()) }

func getSocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/omni.sock"
	}
	return filepath.Join(home, ".omnikernel", "omni.sock")
}

func getSocketClient() *http.Client {
	socketPath := getSocketPath()
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 60 * time.Second,
	}
}

func findPythonBackendDir() string {
	if custom := os.Getenv("OMNIKERNEL_AI_DIR"); custom != "" {
		return custom
	}

	cwd, err := os.Getwd()
	if err == nil {
		if _, err := os.Stat(filepath.Join(cwd, "ai", "api.py")); err == nil {
			return filepath.Join(cwd, "ai")
		}
		if _, err := os.Stat(filepath.Join(cwd, "api.py")); err == nil {
			return cwd
		}
	}

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		if _, err := os.Stat(filepath.Join(exeDir, "ai", "api.py")); err == nil {
			return filepath.Join(exeDir, "ai")
		}
	}

	return "/home/ace/Documents/Avvi/OmniKernel/ai"
}

func startPythonAPI() *exec.Cmd {
	aiDir := findPythonBackendDir()

	// Ensure ~/.omnikernel directory exists
	home, _ := os.UserHomeDir()
	omniDir := filepath.Join(home, ".omnikernel")
	_ = os.MkdirAll(omniDir, 0755)

	socketPath := getSocketPath()
	_ = os.Remove(socketPath)

	// Determine runner: uv or python3
	var cmd *exec.Cmd
	if _, err := exec.LookPath("uv"); err == nil {
		cmd = exec.Command("uv", "run", "python", "api.py")
	} else {
		cmd = exec.Command("python3", "api.py")
	}

	cmd.Dir = aiDir
	// Ensure PYTHONPATH has both the ai directory and its parent for package imports
	parentDir := filepath.Dir(aiDir)
	currentPythonPath := os.Getenv("PYTHONPATH")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PYTHONPATH=%s:%s:%s", aiDir, parentDir, currentPythonPath),
	)

	err := cmd.Start()
	if err != nil {
		fmt.Printf("❌ Failed to start Python AI backend: %v\n", err)
		os.Exit(1)
	}

	// Wait up to 5 seconds for the socket to appear
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return cmd
}

func stepPythonAPI(payload map[string]string) tea.Cmd {
	return func() tea.Msg {
		client := getSocketClient()
		reqBody, err := json.Marshal(payload)
		if err != nil {
			return aiDoneMsg(fmt.Sprintf("❌ Failed to encode request: %v", err))
		}

		resp, err := client.Post("http://unix/step", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return aiDoneMsg(fmt.Sprintf("❌ Failed to contact Python AI brain: %v", err))
		}
		defer resp.Body.Close()

		var result apiResponse
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return aiDoneMsg(fmt.Sprintf("❌ Error reading API response: %v", err))
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return aiDoneMsg(fmt.Sprintf("❌ Failed to parse API response: %v", err))
		}

		if result.Status == "tool_call" {
			return aiToolCallMsg{
				ToolCallID: result.ToolCallID,
				Name:       result.Name,
				Arguments:  result.Arguments,
			}
		}
		return aiDoneMsg(result.Response)
	}
}

func runToolAndStep(msg aiToolCallMsg) tea.Cmd {
	return func() tea.Msg {
		toolResult := ExecuteTool(msg.Name, msg.Arguments)
		return stepPythonAPI(map[string]string{
			"tool_call_id": msg.ToolCallID,
			"tool_result":  toolResult,
		})()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = (msg.Width * 50 / 100) - 6
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() == "" || m.isThinking {
				return m, nil
			}
			query := m.textInput.Value()
			m.chatLog += fmt.Sprintf("\nUser 💻: %s\n[Agent is thinking...]\n", query)
			m.textInput.SetValue("")
			m.isThinking = true
			return m, stepPythonAPI(map[string]string{"query": query})
		}

	case aiToolCallMsg:
		m.chatLog += fmt.Sprintf("⚙️ Agent executed Go tool: %s\n", msg.Name)
		return m, runToolAndStep(msg)

	case aiDoneMsg:
		answer := string(msg)
		// Cleanly replace thinking indicator if present
		if strings.Contains(m.chatLog, "[Agent is thinking...]\n") {
			m.chatLog = strings.Replace(m.chatLog, "[Agent is thinking...]\n", fmt.Sprintf("OmniKernel 🧠:\n%s\n", answer), 1)
		} else {
			m.chatLog += fmt.Sprintf("OmniKernel 🧠:\n%s\n", answer)
		}
		m.isThinking = false
		return m, nil

	case tickMsg:
		// 1. CPU Overall
		if percentages, err := cpu.Percent(0, false); err == nil && len(percentages) > 0 {
			m.cpuUsage = percentages[0]
		}
		// 2. CPU Cores
		if cores, err := cpu.Percent(0, true); err == nil {
			m.coreUsages = cores
		}
		// 3. RAM
		if v, err := mem.VirtualMemory(); err == nil {
			m.ramUsage = v.UsedPercent
			m.ramUsedGB = float64(v.Used) / (1024 * 1024 * 1024)
			m.ramTotalGB = float64(v.Total) / (1024 * 1024 * 1024)
		}
		// 4. Disks
		if parts, err := disk.Partitions(false); err == nil {
			var activeDisks []DiskInfo
			const gb = 1024.0 * 1024.0 * 1024.0 // Byte to Gigabyte conversion factor

			for _, p := range parts {
				// Filter out system mounts that clutter the UI
				if strings.HasPrefix(p.Mountpoint, "/snap") || strings.HasPrefix(p.Mountpoint, "/boot") {
					continue
				}

				if u, err := disk.Usage(p.Mountpoint); err == nil {
					// Prevent adding empty partitions or CD-ROM drives
					if u.Total == 0 {
						continue
					}

					activeDisks = append(activeDisks, DiskInfo{
						MountPoint: p.Mountpoint,
						TotalGB:    float64(u.Total) / gb,
						UsedGB:     float64(u.Used) / gb,
						FreeGB:     float64(u.Free) / gb,
					})
				}
			}
			m.disks = activeDisks
		}
		// 5. Temperature
		if temps, err := host.SensorsTemperatures(); err == nil && len(temps) > 0 {
			m.cpuTemp = temps[0].Temperature
		}
		// 6. Network I/O
		if n, err := netstat.IOCounters(false); err == nil && len(n) > 0 {
			m.netSentMB = float64(n[0].BytesSent) / (1024 * 1024)
			m.netRecvMB = float64(n[0].BytesRecv) / (1024 * 1024)
		}
		// 7. Host Info & Uptime
		if hInfo, err := host.Info(); err == nil {
			m.hostOS = fmt.Sprintf("%s (%s)", hInfo.Platform, hInfo.KernelVersion)
			m.uptimeSec = hInfo.Uptime
		}
		// 8. Load Average
		if l, err := load.Avg(); err == nil {
			m.load1 = l.Load1
			m.load5 = l.Load5
			m.load15 = l.Load15
		}

		return m, tickEverySecond()
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func main() {
	apiCmd := startPythonAPI()
	defer func() {
		if apiCmd != nil && apiCmd.Process != nil {
			_ = apiCmd.Process.Kill()
		}
		_ = os.Remove(getSocketPath())
	}()

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running OmniKernel: %v\n", err)
		os.Exit(1)
	}
}
