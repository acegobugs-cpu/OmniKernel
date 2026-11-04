package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("21"))

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
	r1Height = max(r1Height, 6)
	bottomHeight := max(m.height-r1Height, 8)

	r1LeftWidth := m.width * 70 / 100
	r1RightWidth := m.width - r1LeftWidth

	bSmallWidth := m.width * 25 / 100
	bLargeWidth := m.width - (bSmallWidth * 2)

	bSmallHeightRow2 := bottomHeight / 2
	bSmallHeightRow3 := bottomHeight - bSmallHeightRow2

	// --- 2. BOX BUILDER HELPER ---
	makeBox := func(w, h int, title, content, borderColor string) string {
		innerW := max(w-2, 1)
		innerH := max(h-2, 1)

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
	barHeight := r1Height - 8
	barHeight = max(barHeight, 2)

	var allBars []string

	// 1. DISKS (Dynamically added first)
	for _, disk := range m.disks {
		b := drawStackedBar(stackedBarData{
			Title: disk.MountPoint,
			Used:  disk.UsedGB,
			Free:  disk.FreeGB,
			Total: disk.TotalGB,
			Labels: []string{
				fmt.Sprintf("T:%.0fG", disk.TotalGB),
				fmt.Sprintf("U:%.0fG", disk.UsedGB),
				fmt.Sprintf("F:%.0fG", disk.FreeGB),
			},
		}, barHeight)
		allBars = append(allBars, b)
	}

	// 2. SWAP
	swapBar := drawStackedBar(stackedBarData{
		Title: "Swap",
		Used:  m.swapUsedGB,
		Free:  m.swapFreeGB,
		Total: m.swapTotalGB,
		Labels: []string{
			fmt.Sprintf("T:%.1fG", m.swapTotalGB),
			fmt.Sprintf("U:%.1fG", m.swapUsedGB),
			fmt.Sprintf("F:%.1fG", m.swapFreeGB),
		},
	}, barHeight)
	allBars = append(allBars, swapBar)

	// 3. MEMORY
	memBar := drawStackedBar(stackedBarData{
		Title: "Mem",
		Used:  m.ramUsedGB,
		Free:  m.ramFreeGB,
		Total: m.ramTotalGB,
		Labels: []string{
			fmt.Sprintf("T:%.1fG", m.ramTotalGB),
			fmt.Sprintf("U:%.1fG", m.ramUsedGB),
			fmt.Sprintf("F:%.1fG", m.ramFreeGB),
		},
	}, barHeight)
	allBars = append(allBars, memBar)

	// 4. CPU (Total is always 100%)
	cpuFree := 100.0 - m.cpuUsage
	cpuBar := drawStackedBar(stackedBarData{
		Title: "CPU",
		Used:  m.cpuUsage,
		Free:  cpuFree,
		Total: 100.0,
		Labels: []string{
			"T:100%",
			fmt.Sprintf("U:%.0f%%", m.cpuUsage),
			fmt.Sprintf("F:%.0f%%", cpuFree),
		},
	}, barHeight)
	allBars = append(allBars, cpuBar)

	// Join all the bars side-by-side.
	// Lipgloss automatically handles the spacing because we added `.Padding(0, 1)` in drawStackedBar.
	sysMetrics := lipgloss.JoinHorizontal(lipgloss.Top, allBars...)

	r1LeftBox := makeBox(r1LeftWidth, r1Height, "💻 System Overview", sysMetrics, "63")

	var coresStr strings.Builder
	for i, c := range m.coreUsages {
		if i >= 8 {
			fmt.Fprintf(&coresStr, "...and %d more cores\n", len(m.coreUsages)-8)
			break
		}
		fmt.Fprintf(&coresStr, "Core %d: %.1f%%\n", i, c)
	}
	if m.cpuTemp > 0 {
		fmt.Fprintf(&coresStr, "\nTemp: %.1f°C", m.cpuTemp)
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
	maxVisibleLines = max(maxVisibleLines, 2)

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

type stackedBarData struct {
	Title  string
	Used   float64
	Free   float64
	Total  float64
	Labels []string
}

// drawStackedBar draws a single vertical bar with colored segments.
// Bottom = Red (Used), Top = Green (Free), Middle = Gray (Total/Other).
func drawStackedBar(data stackedBarData, height int) string {
	var sb strings.Builder

	// Safely handle 0 totals
	if data.Total <= 0 {
		data.Total = 1
	}

	// Calculate how many "dots" of height each segment gets
	totalDots := float64(height * 4)
	usedDots := int(math.Round((data.Used / data.Total) * totalDots))
	freeDots := int(math.Round((data.Free / data.Total) * totalDots))

	// Draw the bar from the top (highest row) to bottom (row 0)
	for row := height - 1; row >= 0; row-- {
		// Find the vertical midpoint of the current character block
		dotMid := (row * 4) + 2

		color := "240" // Gray by default (Cache/System/Unused space)

		if dotMid < usedDots {
			color = "203" // Red for Used (Bottom of the bar)
		} else if dotMid >= (int(totalDots) - freeDots) {
			color = "42" // Green for Free (Top of the bar)
		}

		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		sb.WriteString(style.Render("⣿⣿") + "\n")
	}

	// Compile the text labels below the bar
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	sb.WriteString(titleStyle.Render(data.Title) + "\n")
	for _, lbl := range data.Labels {
		sb.WriteString(textStyle.Render(lbl) + "\n")
	}

	// Wrap in a centered Lipgloss column with horizontal padding
	colStyle := lipgloss.NewStyle().Align(lipgloss.Center).Padding(0, 1)
	return colStyle.Render(sb.String())
}
