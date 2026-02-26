package modals

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/git"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type BranchModal struct {
	styles   *styles.Styles
	visible  bool
	width    int
	height   int
	branches []*git.Branch
	cursor   int
}

func NewBranchModal(s *styles.Styles) BranchModal {
	return BranchModal{
		styles:  s,
		visible: false,
		width:   80,
		height:  24,
	}
}

// Height returns the number of terminal rows this component occupies when visible.
func (m BranchModal) Height() int {
	if !m.visible {
		return 0
	}
	// 2 border rows + 1 title row + branch rows (capped).
	rows := len(m.branches)
	if rows > 10 {
		rows = 10
	}
	if rows < 1 {
		rows = 1
	}
	return rows + 3 // border(2) + title(1) + branch rows
}

// View renders the inline branch picker panel.
func (m BranchModal) View() string {
	if !m.visible {
		return ""
	}

	theme := m.styles.Theme
	panelBg := theme.BackgroundPanel

	bgStyle := lipgloss.NewStyle().Background(panelBg)
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(panelBg).
		Bold(true)
	hintStyle := lipgloss.NewStyle().
		Foreground(theme.DiffContext).
		Background(panelBg).
		Italic(true)

	innerWidth := m.width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Adaptive hint text for the title row.
	titleText := " Branches"
	hintText := "Enter to checkout | Esc to close"
	titleRendered := titleStyle.Render(titleText)
	hintRendered := hintStyle.Render(hintText)
	titleGap := innerWidth - lipgloss.Width(titleText) - lipgloss.Width(hintText)
	if titleGap < 1 {
		// Try shorter hint.
		hintText = "Enter | Esc"
		hintRendered = hintStyle.Render(hintText)
		titleGap = innerWidth - lipgloss.Width(titleText) - lipgloss.Width(hintText)
		if titleGap < 1 {
			// Drop hint entirely.
			hintRendered = ""
			titleGap = innerWidth - lipgloss.Width(titleText)
			if titleGap < 0 {
				titleGap = 0
			}
		}
	}
	titleRow := titleRendered + bgStyle.Width(titleGap).Render("") + hintRendered

	var rows []string
	rows = append(rows, titleRow)

	maxVisible := 10
	if len(m.branches) < maxVisible {
		maxVisible = len(m.branches)
	}

	// Determine scroll window so the cursor is always visible.
	scrollStart := 0
	if m.cursor >= maxVisible {
		scrollStart = m.cursor - maxVisible + 1
	}
	scrollEnd := scrollStart + maxVisible
	if scrollEnd > len(m.branches) {
		scrollEnd = len(m.branches)
		scrollStart = scrollEnd - maxVisible
		if scrollStart < 0 {
			scrollStart = 0
		}
	}

	for i := scrollStart; i < scrollEnd; i++ {
		b := m.branches[i]
		isSelected := i == m.cursor

		var bg lipgloss.Color
		if isSelected {
			bg = theme.Selection
		} else {
			bg = panelBg
		}

		rowBg := lipgloss.NewStyle().Background(bg)
		nameStyle := lipgloss.NewStyle().Foreground(theme.BranchMain).Background(bg).Bold(true)
		currentStyle := lipgloss.NewStyle().Foreground(theme.Head).Background(bg)
		hashStyle := lipgloss.NewStyle().Foreground(theme.CommitHash).Background(bg)

		prefix := "  "
		if b.IsCurrent {
			prefix = currentStyle.Render("* ")
		} else {
			prefix = rowBg.Render("  ")
		}

		// Truncate branch name to fit. Reserve: prefix(2) + hash(8) + space(1) = 11
		nameAvail := innerWidth - 11
		if nameAvail < 6 {
			nameAvail = 6
		}
		displayName := b.Name
		nameRunes := []rune(displayName)
		if len(nameRunes) > nameAvail {
			displayName = string(nameRunes[:nameAvail-1]) + "…"
		}

		name := nameStyle.Render(displayName)
		hash := hashStyle.Render(" " + b.Hash[:7])
		row := prefix + name + hash

		visWidth := lipgloss.Width(row)
		if visWidth < innerWidth {
			row = row + rowBg.Width(innerWidth-visWidth).Render("")
		}

		row = lipgloss.NewStyle().Background(bg).Width(innerWidth).Render(row)
		rows = append(rows, row)
	}

	if len(m.branches) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(theme.Subtext).Background(panelBg).Italic(true)
		rows = append(rows, emptyStyle.Render("  No branches found"))
	}

	content := ""
	for i, r := range rows {
		if i > 0 {
			content += "\n"
		}
		content += r
	}

	bar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.BranchMain).
		BorderBackground(theme.Background).
		Background(panelBg).
		Width(m.width - 2).
		Render(content)

	return bar
}

func (m *BranchModal) Show(branches []*git.Branch) {
	m.visible = true
	m.branches = branches
	// Place cursor on the current branch.
	m.cursor = 0
	for i, b := range branches {
		if b.IsCurrent {
			m.cursor = i
			break
		}
	}
}

func (m *BranchModal) Hide() {
	m.visible = false
	m.branches = nil
	m.cursor = 0
}

func (m *BranchModal) IsVisible() bool {
	return m.visible
}

// MoveUp moves the branch cursor up.
func (m *BranchModal) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// MoveDown moves the branch cursor down.
func (m *BranchModal) MoveDown() {
	if m.cursor < len(m.branches)-1 {
		m.cursor++
	}
}

// SelectedBranch returns the currently highlighted branch, or nil.
func (m *BranchModal) SelectedBranch() *git.Branch {
	if m.cursor >= 0 && m.cursor < len(m.branches) {
		return m.branches[m.cursor]
	}
	return nil
}

func (m *BranchModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}
