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

// Custom message to handle the async API response
type aiResponseMsg string

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

	return model{
		chatLog:   "OmniKernel UI active. Type your command below.\n",
		textInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickEverySecond())
}

// getSocketClient configures HTTP to dial the local .sock file instead of a network port
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

// askPythonAPI sends the prompt over the Unix Socket
func askPythonAPI(query string) tea.Cmd {
	return func() tea.Msg {
		client := getSocketClient()

		reqBody, _ := json.Marshal(map[string]string{"query": query})

		// Note: "http://unix" is a dummy host required by Go's HTTP package,
		// the DialContext above forces it to use the socket file.
		resp, err := client.Post("http://unix/ask", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return aiResponseMsg(fmt.Sprintf("❌ Failed to contact Python API: %v", err))
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var result map[string]string
		json.Unmarshal(body, &result)

		return aiResponseMsg(result["response"])
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

			// Dispatch the HTTP request over the socket asynchronously
			return m, askPythonAPI(query)
		}

	case aiResponseMsg:
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
	chatWidget := chatStyle.Render(chatBox)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cpuWidget, ramWidget)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, chatWidget)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
