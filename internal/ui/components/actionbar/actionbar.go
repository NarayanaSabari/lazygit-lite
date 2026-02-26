package actionbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type Model struct {
	styles  *styles.Styles
	status  string
	branch  string
	width   int
	message string
}

func New(styles *styles.Styles, width int) Model {
	return Model{
		styles: styles,
		width:  width,
		branch: "main",
	}
}

func (m Model) View() string {
	theme := m.styles.Theme
	bg := theme.BackgroundElement

	// Key hints with highlighted keys.
	keyStyle := lipgloss.NewStyle().
		Foreground(theme.BranchFeature).
		Background(bg).
		Bold(true)
	sepStyle := lipgloss.NewStyle().
		Foreground(theme.DiffContext).
		Background(bg)
	descStyle := lipgloss.NewStyle().
		Foreground(theme.Subtext).
		Background(bg)
	spacerStyle := lipgloss.NewStyle().Background(bg)

	sep := sepStyle.Render(" | ")

	keys := []struct{ key, desc string }{
		{"Enter", "expand"},
		{"Esc", "collapse"},
		{"c", "commit"},
		{"p", "push"},
		{"P", "pull"},
		{"f", "fetch"},
		{"b", "branch"},
		{"?", "help"},
	}

	// Branch indicator on the right.
	branchDisplay := m.branch
	branchStyle := lipgloss.NewStyle().Foreground(theme.BranchMain).Background(bg).Bold(true)
	branchIcon := branchStyle.Render("⎇ ")
	// Truncate branch name if it would consume more than 1/3 of the width.
	maxBranchLen := m.width / 3
	if maxBranchLen < 8 {
		maxBranchLen = 8
	}
	branchRunes := []rune(branchDisplay)
	if len(branchRunes) > maxBranchLen {
		branchDisplay = string(branchRunes[:maxBranchLen-1]) + "…"
	}
	branchName := branchStyle.Render(branchDisplay)
	rightPart := branchIcon + branchName
	rightWidth := lipgloss.Width(rightPart)

	var leftPart string
	// Status message if present.
	if m.message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(theme.Tag).Background(bg)
		msg := m.message
		// Truncate message if it would overflow.
		maxMsgWidth := m.width - rightWidth - 2
		if maxMsgWidth < 4 {
			maxMsgWidth = 4
		}
		msgRunes := []rune(msg)
		if len(msgRunes) > maxMsgWidth {
			msg = string(msgRunes[:maxMsgWidth-1]) + "…"
		}
		leftPart = msgStyle.Render(msg)
	} else {
		// Progressively drop key hints from the right until they fit.
		availWidth := m.width - rightWidth - 2 // 2 = minimum spacer
		for numKeys := len(keys); numKeys > 0; numKeys-- {
			var parts []string
			for _, k := range keys[:numKeys] {
				parts = append(parts, keyStyle.Render(k.key)+descStyle.Render(" "+k.desc))
			}
			candidate := strings.Join(parts, sep)
			if lipgloss.Width(candidate) <= availWidth || numKeys == 1 {
				leftPart = candidate
				break
			}
		}
		// If even one key doesn't fit, just show "? help".
		if leftPart == "" {
			leftPart = keyStyle.Render("?") + descStyle.Render(" help")
		}
	}

	leftWidth := lipgloss.Width(leftPart)
	padding := m.width - leftWidth - rightWidth
	if padding < 1 {
		padding = 1
	}

	spacer := spacerStyle.Width(padding).Render("")

	bar := lipgloss.NewStyle().
		Background(bg).
		Width(m.width).
		Render(leftPart + spacer + rightPart)

	return bar
}

func (m *Model) SetBranch(branch string) {
	m.branch = branch
}

func (m *Model) SetWidth(width int) {
	m.width = width
}

func (m *Model) SetMessage(msg string) {
	m.message = msg
}

func (m *Model) ClearMessage() {
	m.message = ""
}
