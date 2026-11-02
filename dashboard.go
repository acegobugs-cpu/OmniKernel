package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

func formatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm %ds", mins, seconds%60)
}

// View computes the UI frame based on the current window dimensions.
func (m model) View() string {
	if m.width == 0 {
		return "Initializing OmniKernel UI..."
	}

	// --- 1. RESPONSIVE DIMENSIONS ---
	r1Height := m.height * 38 / 100
	if r1Height < 6 {
		r1Height = 6
	}
	bottomHeight := m.height - r1Height
	if bottomHeight < 8 {
		bottomHeight = 8
	}

	r1LeftWidth := m.width * 70 / 100
	r1RightWidth := m.width - r1LeftWidth

	bSmallWidth := m.width * 25 / 100
	bLargeWidth := m.width - (bSmallWidth * 2)

	bSmallHeightRow2 := bottomHeight / 2
	bSmallHeightRow3 := bottomHeight - bSmallHeightRow2

	// --- 2. BOX BUILDER HELPER ---
	makeBox := func(w, h int, title, content, borderColor string) string {
		innerW := w - 2
		if innerW < 1 {
			innerW = 1
		}
		innerH := h - 2
		if innerH < 1 {
			innerH = 1
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderColor)).
			Padding(0, 1).
			Width(innerW).
			Height(innerH)

		titleStr := titleStyle.Render(title)
		return style.Render(fmt.Sprintf("%s\n\n%s", titleStr, content))
	}

	// --- 3. TOP ROW ---
	diskStr := strings.Join(m.diskUsages, " | ")
	if diskStr == "" {
		diskStr = "None"
	}
	topWideContent := fmt.Sprintf("CPU Load: %.1f%%\nRAM Used: %.1f%% (%.2f GB / %.2f GB)\nDisks: %s",
		m.cpuUsage, m.ramUsage, m.ramUsedGB, m.ramTotalGB, diskStr)
	r1LeftBox := makeBox(r1LeftWidth, r1Height, "💻 System Overview", topWideContent, "63")

	var coresStr strings.Builder
	for i, c := range m.coreUsages {
		if i >= 8 {
			coresStr.WriteString(fmt.Sprintf("...and %d more cores\n", len(m.coreUsages)-8))
			break
		}
		coresStr.WriteString(fmt.Sprintf("Core %d: %.1f%%\n", i, c))
	}
	if m.cpuTemp > 0 {
		coresStr.WriteString(fmt.Sprintf("\nTemp: %.1f°C", m.cpuTemp))
	}
	r1RightBox := makeBox(r1RightWidth, r1Height, "🔥 Cores & Temp", coresStr.String(), "63")

	// --- 4. BOTTOM LEFT GRID (Live Sensors) ---
	netContent := fmt.Sprintf("Sent: %.2f MB\nRecv: %.2f MB", m.netSentMB, m.netRecvMB)
	r2b1 := makeBox(bSmallWidth, bSmallHeightRow2, "🌐 Network I/O", netContent, "63")

	hostContent := fmt.Sprintf("OS: %s\nUp: %s", m.hostOS, formatUptime(m.uptimeSec))
	r2b2 := makeBox(bSmallWidth, bSmallHeightRow2, "🖥️ Host & Uptime", hostContent, "63")

	freeRAM := m.ramTotalGB - m.ramUsedGB
	if freeRAM < 0 {
		freeRAM = 0
	}
	ramContent := fmt.Sprintf("Used:  %.2f GB\nFree:  %.2f GB\nTotal: %.2f GB", m.ramUsedGB, freeRAM, m.ramTotalGB)
	r3b1 := makeBox(bSmallWidth, bSmallHeightRow3, "🧠 Memory", ramContent, "63")

	loadContent := fmt.Sprintf("1m:  %.2f\n5m:  %.2f\n15m: %.2f", m.load1, m.load5, m.load15)
	r3b2 := makeBox(bSmallWidth, bSmallHeightRow3, "⚖️ Load Average", loadContent, "63")

	// --- 5. BOTTOM RIGHT (Agent Chat Panel) ---
	chatLines := strings.Split(m.chatLog, "\n")
	maxVisibleLines := bottomHeight - 5
	if maxVisibleLines < 2 {
		maxVisibleLines = 2
	}
	if len(chatLines) > maxVisibleLines {
		chatLines = chatLines[len(chatLines)-maxVisibleLines:]
	}
	visibleChat := strings.Join(chatLines, "\n")
	chatContent := fmt.Sprintf("%s\n\n%s", visibleChat, m.textInput.View())
	bottomLargeBox := makeBox(bLargeWidth, bottomHeight, "🤖 Agent Chat", chatContent, "39")

	// --- 6. ASSEMBLE THE UI ---
	row1 := lipgloss.JoinHorizontal(lipgloss.Top, r1LeftBox, r1RightBox)

	bLeftRow2 := lipgloss.JoinHorizontal(lipgloss.Top, r2b1, r2b2)
	bLeftRow3 := lipgloss.JoinHorizontal(lipgloss.Top, r3b1, r3b2)
	bottomLeftGrid := lipgloss.JoinVertical(lipgloss.Left, bLeftRow2, bLeftRow3)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, bottomLeftGrid, bottomLargeBox)

	fullUI := lipgloss.JoinVertical(lipgloss.Left, row1, bottomRow)

	return fullUI
}
