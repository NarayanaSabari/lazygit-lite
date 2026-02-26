package modals

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type HelpModal struct {
	styles  *styles.Styles
	visible bool
	width   int
	height  int
}

func NewHelpModal(styles *styles.Styles) HelpModal {
	return HelpModal{
		styles:  styles,
		visible: false,
		width:   80,
		height:  24,
	}
}

// singleColumn returns true when the terminal is too narrow for a two-column layout.
func (m HelpModal) singleColumn() bool {
	return m.width < 60
}

// contentRowCount returns the number of content rows (title + key rows) in the
// help panel. This is used by both Height() and View() to stay consistent.
func (m HelpModal) contentRowCount() int {
	if m.singleColumn() {
		// Single-column: all entries stacked vertically.
		// Nav(1) + 6 + blank + Expand(1) + 3 + blank + Actions(1) + 5 + blank + Clipboard(1) + 3 + blank + General(1) + 2 = 26
		return 26 + 1 // +1 title
	}

	// Two-column layout.
	leftCount := 11
	rightCount := 15

	rows := leftCount
	if rightCount > rows {
		rows = rightCount
	}
	return rows + 1 // +1 for title row
}

// Height returns the number of terminal rows this component occupies when visible.
func (m HelpModal) Height() int {
	if !m.visible {
		return 0
	}
	h := m.contentRowCount() + 2 // +2 for RoundedBorder (top + bottom)
	// Cap height to available terminal height minus some margin (action bar + borders).
	maxH := m.height - 4
	if maxH < 6 {
		maxH = 6
	}
	if h > maxH {
		h = maxH
	}
	return h
}

// View renders the inline help panel (meant to sit above the action bar).
func (m HelpModal) View() string {
	if !m.visible {
		return ""
	}

	theme := m.styles.Theme

	panelBg := theme.BackgroundPanel

	innerWidth := m.width - 4 // border (2) + a bit of padding
	if innerWidth < 20 {
		innerWidth = 20
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(panelBg).
		Bold(true)

	// Adaptive key column width based on available space.
	keyColWidth := 12
	if innerWidth < 50 {
		keyColWidth = 10
	}
	if innerWidth < 36 {
		keyColWidth = 8
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(theme.BranchFeature).
		Background(panelBg).
		Bold(true).
		Width(keyColWidth)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.Subtext).
		Background(panelBg)

	sectionStyle := lipgloss.NewStyle().
		Foreground(theme.Head).
		Background(panelBg).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.DiffContext).
		Background(panelBg).
		Italic(true)

	bgStyle := lipgloss.NewStyle().Background(panelBg)

	makeRow := func(key, desc string) string {
		return bgStyle.Render(" ") + keyStyle.Render(key) + descStyle.Render(desc)
	}

	var leftLines []string
	leftLines = append(leftLines, sectionStyle.Render("Navigation"))
	leftLines = append(leftLines, makeRow("j / Down", "Move down"))
	leftLines = append(leftLines, makeRow("k / Up", "Move up"))
	leftLines = append(leftLines, makeRow("g / Home", "Go to top"))
	leftLines = append(leftLines, makeRow("G / End", "Go to bottom"))
	leftLines = append(leftLines, makeRow("Ctrl+D", "Page down"))
	leftLines = append(leftLines, makeRow("Ctrl+U", "Page up"))
	leftLines = append(leftLines, bgStyle.Render(""))
	leftLines = append(leftLines, sectionStyle.Render("Expand / Collapse"))
	leftLines = append(leftLines, makeRow("Enter", "Expand / toggle diff"))
	leftLines = append(leftLines, makeRow("Esc", "Collapse"))
	leftLines = append(leftLines, makeRow("j / k", "Navigate files"))

	var rightLines []string
	rightLines = append(rightLines, sectionStyle.Render("Actions"))
	rightLines = append(rightLines, makeRow("c", "Commit"))
	rightLines = append(rightLines, makeRow("p", "Push"))
	rightLines = append(rightLines, makeRow("P", "Pull"))
	rightLines = append(rightLines, makeRow("f", "Fetch"))
	rightLines = append(rightLines, makeRow("b", "Switch branch"))
	rightLines = append(rightLines, bgStyle.Render(""))
	rightLines = append(rightLines, sectionStyle.Render("Clipboard"))
	rightLines = append(rightLines, makeRow("y", "Copy hash"))
	rightLines = append(rightLines, makeRow("Y", "Copy message"))
	rightLines = append(rightLines, makeRow("Ctrl+Y", "Copy diff"))
	rightLines = append(rightLines, bgStyle.Render(""))
	rightLines = append(rightLines, sectionStyle.Render("General"))
	rightLines = append(rightLines, makeRow("?", "Toggle help"))
	rightLines = append(rightLines, makeRow("q", "Quit"))

	// Title row: adapt hint text for narrow widths.
	hintText := "? to close"
	titleText := " Keybindings"
	titleRendered := titleStyle.Render(titleText)
	hintRendered := hintStyle.Render(hintText)
	titleGap := innerWidth - lipgloss.Width(titleText) - lipgloss.Width(hintText)
	if titleGap < 1 {
		// Drop hint at very narrow widths.
		hintRendered = ""
		titleGap = innerWidth - lipgloss.Width(titleText)
		if titleGap < 0 {
			titleGap = 0
		}
	}
	titleRow := titleRendered + bgStyle.Width(titleGap).Render("") + hintRendered

	var contentRows []string

	if m.singleColumn() {
		// Single-column layout: stack left then right.
		allLines := append(leftLines, bgStyle.Render(""))
		allLines = append(allLines, rightLines...)
		for _, line := range allLines {
			contentRows = append(contentRows, line)
		}
	} else {
		// Two-column layout.
		halfWidth := innerWidth / 2

		// Equalize column heights.
		for len(leftLines) < len(rightLines) {
			leftLines = append(leftLines, bgStyle.Render(""))
		}
		for len(rightLines) < len(leftLines) {
			rightLines = append(rightLines, bgStyle.Render(""))
		}

		colStyle := lipgloss.NewStyle().Width(halfWidth).Background(panelBg)

		for i := 0; i < len(leftLines); i++ {
			left := colStyle.Render(leftLines[i])
			right := colStyle.Render(rightLines[i])
			contentRows = append(contentRows, left+right)
		}
	}

	// Cap visible rows if the modal would exceed available height.
	maxContentRows := m.height - 6 // borders(2) + title(1) + action bar(1) + margin(2)
	if maxContentRows < 4 {
		maxContentRows = 4
	}
	if len(contentRows) > maxContentRows {
		contentRows = contentRows[:maxContentRows]
	}

	content := titleRow + "\n" + strings.Join(contentRows, "\n")

	bar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.BranchFeature).
		BorderBackground(theme.Background).
		Background(theme.BackgroundPanel).
		Width(m.width - 2).
		Render(content)

	return bar
}

func (m *HelpModal) Toggle() {
	m.visible = !m.visible
}

func (m *HelpModal) IsVisible() bool {
	return m.visible
}

func (m *HelpModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}
