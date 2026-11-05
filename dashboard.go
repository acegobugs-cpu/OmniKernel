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

	barCellWidth := 14
	// Increased placeholder height from 5 to 7 rows to ensure the graph doesn't overlap the legends
	barCellHeight := 7
	barPixW := 100
	barPixH := 100

	var phBuilder strings.Builder
	for i := 0; i < barCellHeight; i++ {
		phBuilder.WriteString(strings.Repeat(" ", barCellWidth))
		if i < barCellHeight-1 {
			phBuilder.WriteString("\n")
		}
	}
	placeholder := phBuilder.String()

	startRow := 6 // Row 1-3 = Border/Title/Space. Row 4-5 = Chart Title. Row 6 = Image Start.
	startCol := 5
	colStep := barCellWidth + 2

	var paddedBars []string
	padStyle := lipgloss.NewStyle().Padding(0, 1)
	textStyle := lipgloss.NewStyle().Width(barCellWidth).Align(lipgloss.Center)

	// Pre-define styles for the legend matching the graph colors exactly
	styleU := lipgloss.NewStyle().Foreground(lipgloss.Color("#DC323C")).Width(barCellWidth).Align(lipgloss.Center)
	styleA := lipgloss.NewStyle().Foreground(lipgloss.Color("#A3D926")).Width(barCellWidth).Align(lipgloss.Center)
	styleC := lipgloss.NewStyle().Foreground(lipgloss.Color("#DCC828")).Width(barCellWidth).Align(lipgloss.Center)
	styleF := lipgloss.NewStyle().Foreground(lipgloss.Color("#28C850")).Width(barCellWidth).Align(lipgloss.Center)

	addChart := func(used, free, total float64, name string) {
		if startCol+barCellWidth > r1LeftWidth-2 {
			return
		}

		// Calculate missing metrics
		cache := total - used - free
		if cache < 0 {
			cache = 0
		}
		avail := free + cache

		pct := func(v float64) float64 {
			if total <= 0 {
				return 0
			}
			return (v / total) * 100
		}

		img := generateSixelChart(used, cache, free, total, barPixW, barPixH)

		imgCol := startCol + 2
		overlays = append(overlays, sixelOverlay{row: startRow, col: imgCol, data: img})

		titleStr := fmt.Sprintf("%s\n%.1fG", name, total)

		// Render legends with their dedicated colors
		legendStr := fmt.Sprintf("%s\n%s\n%s\n%s",
			styleU.Render(fmt.Sprintf("U:%.1fG %2.0f%%", used, pct(used))),
			styleA.Render(fmt.Sprintf("A:%.1fG %2.0f%%", avail, pct(avail))),
			styleC.Render(fmt.Sprintf("C:%.1fG %2.0f%%", cache, pct(cache))),
			styleF.Render(fmt.Sprintf("F:%.1fG %2.0f%%", free, pct(free))))

		block := fmt.Sprintf("%s\n%s\n%s",
			textStyle.Render(titleStr),
			placeholder,
			legendStr)

		paddedBars = append(paddedBars, padStyle.Render(block))
		startCol += colStep
	}

	// Disks
	for _, disk := range m.disks {
		mount := disk.MountPoint
		if len(mount) > 10 {
			mount = mount[:10]
		}
		addChart(disk.UsedGB, disk.FreeGB, disk.TotalGB, mount)
	}
	// Swap
	addChart(m.swapUsedGB, m.swapFreeGB, m.swapTotalGB, "Swap")
	// Mem
	addChart(m.ramUsedGB, m.ramFreeGB, m.ramTotalGB, "Memory")

	sysMetrics := lipgloss.JoinHorizontal(lipgloss.Top, paddedBars...)
	r1LeftBox := makeBox(r1LeftWidth, r1Height, "💻 System Overview", sysMetrics, "63")

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

	finalOutput.WriteString("\x1b[s")

	for _, o := range overlays {
		finalOutput.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", o.row, o.col, o.data))
	}

	finalOutput.WriteString("\x1b[u")

	return finalOutput.String()
}

func generateSixelChart(used, cache, free, total float64, pixelWidth, pixelHeight int) string {
	if total <= 0 {
		total = 1
	}
	dc := gg.NewContext(pixelWidth, pixelHeight)

	// Added a base dark-gray background so completely empty charts (like Swap at 0) don't render black/invisible
	dc.SetColor(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	dc.DrawRectangle(0, 0, float64(pixelWidth), float64(pixelHeight))
	dc.Fill()

	usedH := (used / total) * float64(pixelHeight)
	cacheH := (cache / total) * float64(pixelHeight)
	freeH := (free / total) * float64(pixelHeight)

	// Create a beautiful linear gradient for the Free -> Cache transition
	if freeH+cacheH > 0 {
		grad := gg.NewLinearGradient(0, 0, 0, freeH+cacheH)
		grad.AddColorStop(0, color.RGBA{R: 40, G: 200, B: 80, A: 255})  // Green at the top (Free)
		grad.AddColorStop(1, color.RGBA{R: 220, G: 200, B: 40, A: 255}) // Yellow at the bottom (Cache)

		dc.SetFillStyle(grad)
		dc.DrawRectangle(0, 0, float64(pixelWidth), freeH+cacheH)
		dc.Fill()
	}

	// Solid Red for the Used portion at the bottom
	dc.SetColor(color.RGBA{R: 220, G: 50, B: 60, A: 255})
	dc.DrawRectangle(0, freeH+cacheH, float64(pixelWidth), usedH)
	dc.Fill()

	// 1-pixel white border to frame the whole chart
	dc.SetColor(color.RGBA{R: 255, G: 255, B: 255, A: 100})
	dc.SetLineWidth(1)
	dc.DrawRectangle(0.5, 0.5, float64(pixelWidth)-1, float64(pixelHeight)-1)
	dc.Stroke()

	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	_ = enc.Encode(dc.Image())
	return buf.String()
}
