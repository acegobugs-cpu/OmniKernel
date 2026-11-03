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
	barHeight := r1Height - 7
	barHeight = max(barHeight, 2)

	// 1. CPU CHART
	cpuGroup := makeChartGroup("CPU", []barData{
		{Label: "Usage", Value: m.cpuUsage, Max: 100, Color: "42", Text: fmt.Sprintf("%.0f%%", m.cpuUsage)},
	}, barHeight)

	// 2. RAM CHART (Max height is Total RAM)
	ramGroup := makeChartGroup("Memory", []barData{
		{Label: "Total", Value: m.ramTotalGB, Max: m.ramTotalGB, Color: "240", Text: fmt.Sprintf("%.1fG", m.ramTotalGB)},
		{Label: "Used", Value: m.ramUsedGB, Max: m.ramTotalGB, Color: "205", Text: fmt.Sprintf("%.1fG", m.ramUsedGB)},
		{Label: "Avail", Value: m.ramAvailGB, Max: m.ramTotalGB, Color: "42", Text: fmt.Sprintf("%.1fG", m.ramAvailGB)},
		{Label: "Cache", Value: m.ramCachedGB, Max: m.ramTotalGB, Color: "220", Text: fmt.Sprintf("%.1fG", m.ramCachedGB)},
		{Label: "Free", Value: m.ramFreeGB, Max: m.ramTotalGB, Color: "39", Text: fmt.Sprintf("%.1fG", m.ramFreeGB)},
	}, barHeight)

	// 3. SWAP CHART
	swapGroup := makeChartGroup("Swap", []barData{
		{Label: "Total", Value: m.swapTotalGB, Max: m.swapTotalGB, Color: "240", Text: fmt.Sprintf("%.1fG", m.swapTotalGB)},
		{Label: "Used", Value: m.swapUsedGB, Max: m.swapTotalGB, Color: "203", Text: fmt.Sprintf("%.1fG", m.swapUsedGB)},
		{Label: "Free", Value: m.swapFreeGB, Max: m.swapTotalGB, Color: "39", Text: fmt.Sprintf("%.1fG", m.swapFreeGB)},
	}, barHeight)

	// 4. Disk CHART
	var diskGroups []string
	for _, disk := range m.disks {
		dg := makeChartGroup(disk.MountPoint, []barData{
			{Label: "Total", Value: disk.TotalGB, Max: disk.TotalGB, Color: "240", Text: fmt.Sprintf("%.0fG", disk.TotalGB)},
			{Label: "Used", Value: disk.UsedGB, Max: disk.TotalGB, Color: "205", Text: fmt.Sprintf("%.0fG", disk.UsedGB)},
			{Label: "Free", Value: disk.FreeGB, Max: disk.TotalGB, Color: "42", Text: fmt.Sprintf("%.0fG", disk.FreeGB)},
		}, barHeight)
		diskGroups = append(diskGroups, dg)
	}

	// Join all disk charts dynamically
	diskMetrics := ""
	if len(diskGroups) > 0 {
		// Only one JoinHorizontal is needed
		diskMetrics = lipgloss.JoinHorizontal(lipgloss.Top, diskGroups...)
	} else {
		diskMetrics = "No disk data" // Fallback to prove the UI block is rendering
	}

	// Join CPU, RAM, and Swap side-by-side with spacing
	sysMetrics := lipgloss.JoinHorizontal(lipgloss.Top,
		cpuGroup,
		lipgloss.NewStyle().Width(4).Render(""), // Spacer
		ramGroup,
		lipgloss.NewStyle().Width(4).Render(""), // Spacer
		swapGroup,
		lipgloss.NewStyle().Width(4).Render(""), // Spacer
		diskMetrics,
	)
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

// Braille characters from bottom to top (0 to 4 dots tall)
var brailleLevels = []string{" ", "⣀", "⣤", "⣶", "⣿"}

// barData represents a single vertical column in the chart
type barData struct {
	Label string  // e.g., "Used", "Free"
	Value float64 // The actual number
	Max   float64 // 100% height of the bar
	Color string  // ANSI color code
	Text  string  // Human-readable string below the bar (e.g., "16GB")
}

// drawBrailleBar generates a single vertical bar with labels beneath it
func drawBrailleBar(data barData, height int) string {
	var sb strings.Builder

	// Calculate percentage (0.0 to 1.0)
	percent := 0.0
	if data.Max > 0 {
		percent = data.Value / data.Max
	}
	if percent > 1.0 {
		percent = 1.0
	}
	if percent < 0.0 {
		percent = 0.0
	}

	// Total vertical resolution is 4 dots per terminal line
	totalDots := float64(height * 4)
	activeDots := int(math.Round(percent * totalDots))

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(data.Color))

	// Draw from the top of the chart down to the bottom
	for row := height - 1; row >= 0; row-- {
		dotsInRow := activeDots - (row * 4)
		if dotsInRow < 0 {
			dotsInRow = 0
		}
		if dotsInRow > 4 {
			dotsInRow = 4
		}

		sb.WriteString(style.Render(brailleLevels[dotsInRow]) + "\n")
	}

	// Wrap the bar and labels in a centered Lipgloss column
	// The horizontal padding creates the gap between adjacent bars
	colStyle := lipgloss.NewStyle().Align(lipgloss.Center).Padding(0, 1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Grey text
	valStyle := lipgloss.NewStyle().Bold(true)

	content := fmt.Sprintf("%s%s\n%s", sb.String(), labelStyle.Render(data.Label), valStyle.Render(data.Text))
	return colStyle.Render(content)
}

// makeChartGroup joins multiple bars side-by-side with a shared title
func makeChartGroup(title string, bars []barData, barHeight int) string {
	var renderedBars []string
	for _, b := range bars {
		renderedBars = append(renderedBars, drawBrailleBar(b, barHeight))
	}

	// Join all the vertical bar columns horizontally, aligning their bottoms
	chartContent := lipgloss.JoinHorizontal(lipgloss.Bottom, renderedBars...)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)

	// Stack the title on top of the bars
	return lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(title), chartContent)
}
