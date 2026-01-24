package actionbar

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type Model struct {
	styles *styles.Styles
	status string
	branch string
	width  int
}

func New(styles *styles.Styles, width int) Model {
	return Model{
		styles: styles,
		width:  width,
		branch: "main",
	}
}

func (m Model) View() string {
	helpText := "[c]ommit  [p]ush  [P]ull  [f]etch  [b]ranch  [?]help"
	statusText := m.branch + " ✓"

	leftPart := m.styles.Help.Render(helpText)
	rightPart := m.styles.BranchName.Render(statusText)

	padding := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if padding < 0 {
		padding = 0
	}

	spacer := lipgloss.NewStyle().Width(padding).Render(" ")

	return m.styles.StatusBar.Render(leftPart + spacer + rightPart)
}

func (m *Model) SetBranch(branch string) {
	m.branch = branch
}

func (m *Model) SetWidth(width int) {
	m.width = width
}
