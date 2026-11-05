package main

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fogleman/gg"
	"github.com/mattn/go-sixel"
)

var titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

type sixelOverlay struct {
	row, col int
	data     string
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing OmniKernel UI..."
	}

	r1Height := m.height * 38 / 100
	r1Height = max(r1Height, 6)
	bottomHeight := max(m.height-r1Height, 8)
	r1LeftWidth := m.width * 70 / 100
	r1RightWidth := m.width - r1LeftWidth
	bSmallWidth := m.width * 25 / 100
	bLargeWidth := m.width - (bSmallWidth * 2)
	bSmallHeightRow2 := bottomHeight / 2
	bSmallHeightRow3 := bottomHeight - bSmallHeightRow2

	makeBox := func(w, h int, title, content, borderColor string) string {
		innerW := max(w-2, 1)
		innerH := max(h-2, 1)
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderColor)).
			Padding(0, 1).
			Width(innerW).
			Height(innerH)
		return style.Render(fmt.Sprintf("%s\n\n%s", titleStyle.Render(title), content))
	}

	// --- ROW 1 LEFT (Safe Absolute Overlays) ---
	var overlays []sixelOverlay

	// Increased width to 8 cells so the text labels fit without breaking the layout
	barCellWidth := 8
	barCellHeight := 5
	barPixW := barCellWidth * 10
	barPixH := barCellHeight * 20

	// 1. Create a pure text placeholder for Lipgloss to measure correctly
	var phBuilder strings.Builder
	for i := 0; i < barCellHeight; i++ {
		phBuilder.WriteString(strings.Repeat(" ", barCellWidth))
		if i < barCellHeight-1 {
			phBuilder.WriteString("\n")
		}
	}
	placeholder := phBuilder.String()

	// 2. Start coordinates for the first graph
	startRow := 4
	startCol := 5
	colStep := barCellWidth + 4 // spacing between charts (width + 2 padding left + 2 padding right)

	var paddedBars []string
	padStyle := lipgloss.NewStyle().Padding(0, 2)
	// Enforce strict width on the text so it perfectly matches the image width
	textStyle := lipgloss.NewStyle().Width(barCellWidth).Align(lipgloss.Center)

	addChart := func(used, free, total float64, label string) {
		if startCol+barCellWidth > r1LeftWidth-2 {
			return
		}

		img := generateSixelChart(used, free, total, barPixW, barPixH)
		overlays = append(overlays, sixelOverlay{row: startRow, col: startCol, data: img})

		formattedText := textStyle.Render(label)
		block := fmt.Sprintf("%s\n%s", placeholder, formattedText)
		paddedBars = append(paddedBars, padStyle.Render(block))

		startCol += colStep
	}

	// Disks
	for _, disk := range m.disks {
		mount := disk.MountPoint
		if len(mount) > 8 {
			mount = mount[:8] // truncate long mount points to fit
		}
		addChart(disk.UsedGB, disk.FreeGB, disk.TotalGB,
			fmt.Sprintf("%s\nT:%.0fG\nU:%.0fG", mount, disk.TotalGB, disk.UsedGB))
	}
	// Swap
	addChart(m.swapUsedGB, m.swapFreeGB, m.swapTotalGB,
		fmt.Sprintf("Swap\nT:%.1fG\nU:%.1fG", m.swapTotalGB, m.swapUsedGB))
	// Mem
	addChart(m.ramUsedGB, m.ramFreeGB, m.ramTotalGB,
		fmt.Sprintf("Mem\nT:%.1fG\nU:%.1fG", m.ramTotalGB, m.ramUsedGB))

	// CPU Graph removed per request

	sysMetrics := lipgloss.JoinHorizontal(lipgloss.Top, paddedBars...)
	r1LeftBox := makeBox(r1LeftWidth, r1Height, "💻 System Overview (🟩 Free | ⬜ Cache | 🟥 Used)", sysMetrics, "63")

	// --- ROW 1 RIGHT ---
	var coresStr strings.Builder
	for i, c := range m.coreUsages {
		if i >= 8 {
			break
		}
		fmt.Fprintf(&coresStr, "Core %d: %.1f%%\n", i, c)
	}
	r1RightBox := makeBox(r1RightWidth, r1Height, "🔥 Cores", coresStr.String(), "63")

	// --- BOTTOM ROW ---
	netContent := fmt.Sprintf("Sent: %.2f MB\nRecv: %.2f MB", m.netSentMB, m.netRecvMB)
	r2b1 := makeBox(bSmallWidth, bSmallHeightRow2, "🌐 Network I/O", netContent, "63")

	hostContent := fmt.Sprintf("OS: %s\nUp: %s", m.hostOS, formatUptime(m.uptimeSec))
	r2b2 := makeBox(bSmallWidth, bSmallHeightRow2, "🖥️ Host & Uptime", hostContent, "63")

	ramContent := fmt.Sprintf("Used:  %.2f GB\nFree:  %.2f GB", m.ramUsedGB, m.ramFreeGB)
	r3b1 := makeBox(bSmallWidth, bSmallHeightRow3, "🧠 Memory", ramContent, "63")

	loadContent := fmt.Sprintf("1m:  %.2f\n5m:  %.2f\n15m: %.2f", m.load1, m.load5, m.load15)
	r3b2 := makeBox(bSmallWidth, bSmallHeightRow3, "⚖️ Load Average", loadContent, "63")

	chatLines := strings.Split(m.chatLog, "\n")
	maxVisible := max(bottomHeight-5, 2)
	if len(chatLines) > maxVisible {
		chatLines = chatLines[len(chatLines)-maxVisible:]
	}
	chatContent := fmt.Sprintf("%s\n\n%s", strings.Join(chatLines, "\n"), m.textInput.View())
	bottomLargeBox := makeBox(bLargeWidth, bottomHeight, "🤖 Agent Chat", chatContent, "39")

	// --- ASSEMBLY ---
	row1 := lipgloss.JoinHorizontal(lipgloss.Top, r1LeftBox, r1RightBox)
	bLeftGrid := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, r2b1, r2b2),
		lipgloss.JoinHorizontal(lipgloss.Top, r3b1, r3b2),
	)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, bLeftGrid, bottomLargeBox)
	fullUI := lipgloss.JoinVertical(lipgloss.Left, row1, bottomRow)

	// --- DECOUPLED RENDERING ---
	var finalOutput strings.Builder

	finalOutput.WriteString(fullUI)

	for _, o := range overlays {
		finalOutput.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", o.row, o.col, o.data))
	}

	finalOutput.WriteString("\x1b[H")

	return finalOutput.String()
}

func generateSixelChart(used, free, total float64, pixelWidth, pixelHeight int) string {
	if total <= 0 {
		total = 1
	}
	dc := gg.NewContext(pixelWidth, pixelHeight)

	usedH := (used / total) * float64(pixelHeight)
	freeH := (free / total) * float64(pixelHeight)
	cacheH := float64(pixelHeight) - usedH - freeH

	// Green (Free)
	dc.SetColor(color.RGBA{R: 40, G: 200, B: 80, A: 255})
	dc.DrawRectangle(0, 0, float64(pixelWidth), freeH)
	dc.Fill()

	// Gray (Cache/Total)
	dc.SetColor(color.RGBA{R: 100, G: 100, B: 100, A: 255})
	dc.DrawRectangle(0, freeH, float64(pixelWidth), cacheH)
	dc.Fill()

	// Red (Used)
	dc.SetColor(color.RGBA{R: 220, G: 50, B: 60, A: 255})
	dc.DrawRectangle(0, freeH+cacheH, float64(pixelWidth), usedH)
	dc.Fill()

	// 1-pixel border
	dc.SetColor(color.RGBA{R: 255, G: 255, B: 255, A: 100})
	dc.SetLineWidth(1)
	dc.DrawRectangle(0.5, 0.5, float64(pixelWidth)-1, float64(pixelHeight)-1)
	dc.Stroke()

	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	_ = enc.Encode(dc.Image())
	return buf.String()
}
