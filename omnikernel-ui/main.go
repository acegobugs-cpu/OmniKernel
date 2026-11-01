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
	"os/exec"       // NEW
	"path/filepath" // NEW
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	widgetStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(1, 2).Width(30).Height(5)
	chatStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("39")).Padding(1, 2).Width(62).Height(12)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
)

type tickMsg time.Time

func tickEverySecond() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Messages for our asynchronous AI bridge
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

type model struct {
	cpuUsage   float64
	ramUsage   float64
	chatLog    string
	textInput  textinput.Model
	isThinking bool
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Ask the agent a question..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 58
	return model{chatLog: "OmniKernel UI active. Type your command below.\n", textInput: ti}
}

func (m model) Init() tea.Cmd { return tea.Batch(textinput.Blink, tickEverySecond()) }

func getSocketClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				home, _ := os.UserHomeDir()
				return net.Dial("unix", home+"/.omnikernel/omni.sock")
			},
		},
	}
}

// startPythonAPI silently boots the Python backend and waits for it to be ready
func startPythonAPI() *exec.Cmd {
	// The absolute path to your Python project
	pythonProjectDir := "/home/ace/Documents/Avvi/OmniKernel"

	// Start the Uvicorn server using uv
	cmd := exec.Command("uv", "run", "python", "api.py")
	cmd.Dir = pythonProjectDir

	// Start the command in the background
	err := cmd.Start()
	if err != nil {
		fmt.Printf("❌ Failed to start Python AI backend: %v\n", err)
		os.Exit(1)
	}

	// Wait for the Python server to create the socket file
	home, _ := os.UserHomeDir()
	socketPath := filepath.Join(home, ".omnikernel", "omni.sock")

	for i := 0; i < 50; i++ { // Timeout after 5 seconds
		if _, err := os.Stat(socketPath); err == nil {
			break // Socket is created, Python is ready!
		}
		time.Sleep(100 * time.Millisecond)
	}

	return cmd
}

// stepPythonAPI sends a payload (either a new query or a tool result) to Python
func stepPythonAPI(payload map[string]string) tea.Cmd {
	return func() tea.Msg {
		client := getSocketClient()
		reqBody, _ := json.Marshal(payload)

		resp, err := client.Post("http://unix/step", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return aiDoneMsg(fmt.Sprintf("❌ Failed to contact Python API: %v", err))
		}
		defer resp.Body.Close()

		var result apiResponse
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &result)

		// Route the response based on what Python decided to do
		if result.Status == "tool_call" {
			return aiToolCallMsg{ToolCallID: result.ToolCallID, Name: result.Name, Arguments: result.Arguments}
		}
		return aiDoneMsg(result.Response)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() == "" || m.isThinking {
				return m, nil
			}
			query := m.textInput.Value()
			m.chatLog = fmt.Sprintf("User 💻: %s\n\n[Agent is thinking...]", query)
			m.textInput.SetValue("")
			m.isThinking = true

			// Kick off the conversation
			return m, stepPythonAPI(map[string]string{"query": query})
		}

	case aiToolCallMsg:
		// PYTHON WANTS GO TO EXECUTE A TOOL!
		m.chatLog += fmt.Sprintf("\n⚙️ Agent triggered Go tool: %s", msg.Name)

		var args map[string]interface{}
		json.Unmarshal([]byte(msg.Arguments), &args)
		var toolResult string

		// Route to your native Go functions from actions.go and monitor.go
		switch msg.Name {
		case "check_hardware":
			toolResult = GetSystemMetricsText()
		case "check_system_logs":
			lines, _ := args["lines"].(int)
			service, _ := args["service"].(string)
			toolResult = GetSystemLogs(lines, service)
		case "check_process":
			lines, _ := args["lines"].(int)
			toolResult = GetTopProcessesText(lines)
		case "find_process_by_name":
			searchTerm, _ := args["search_term"].(string)
			toolResult = FindProcessByName(searchTerm)
		case "run_shell_command":
			cmdStr, _ := args["command"].(string)
			toolResult = RunShellCommand(cmdStr)
		case "kill_process":
			pid, _ := args["pid"].(int)
			toolResult = KillProcess(pid)
		case "write_system_file":
			filePath, _ := args["filepath"].(string)
			content, _ := args["content"].(string)
			toolResult = WriteSystemFile(filePath, content)
		case "read_system_file":
			filepath, _ := args["filepath"].(string)
			toolResult = ReadSystemFile(filepath)
		case "manage_docker":
			action, _ := args["action"].(string)
			id, _ := args["contaider_id"].(string)
			toolResult = ManageDocker(action, id)
		default:
			toolResult = fmt.Sprintf("❌ Tool '%s' not implemented in Go frontend.", msg.Name)
		}

		// Feed the result back into Python to continue the thought loop
		return m, stepPythonAPI(map[string]string{
			"tool_call_id": msg.ToolCallID,
			"tool_result":  toolResult,
		})

	case aiDoneMsg:
		// PYTHON FINISHED THINKING
		m.chatLog = fmt.Sprintf("OmniKernel 🧠:\n%s", string(msg))
		m.isThinking = false
		return m, nil

	case tickMsg:
		percentages, _ := cpu.Percent(0, false)
		if len(percentages) > 0 {
			m.cpuUsage = percentages[0]
		}
		v, _ := mem.VirtualMemory()
		if v != nil {
			m.ramUsage = v.UsedPercent
		}
		return m, tickEverySecond()
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	cpuWidget := widgetStyle.Render(fmt.Sprintf("%s\n\nLoad: %.1f%%", titleStyle.Render("💻 CPU"), m.cpuUsage))
	ramWidget := widgetStyle.Render(fmt.Sprintf("%s\n\nUsed: %.1f%%", titleStyle.Render("🧠 RAM"), m.ramUsage))
	chatBox := fmt.Sprintf("%s\n\n%s\n\n%s", titleStyle.Render("🤖 Agent Output"), m.chatLog, m.textInput.View())

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cpuWidget, ramWidget)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, chatStyle.Render(chatBox))
}

func main() {
	// 1. Boot the Python Brain in the background
	apiCmd := startPythonAPI()

	// 2. Ensure we assassinate the Python process when the Go UI closes
	defer func() {
		if apiCmd != nil && apiCmd.Process != nil {
			apiCmd.Process.Kill()
		}
	}()

	// 3. Start the Go TUI
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
