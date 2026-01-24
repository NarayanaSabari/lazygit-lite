package modals

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type CommitModal struct {
	textarea textarea.Model
	styles   *styles.Styles
	visible  bool
}

func NewCommitModal(styles *styles.Styles) CommitModal {
	ta := textarea.New()
	ta.Placeholder = "Commit message..."
	ta.SetWidth(60)
	ta.SetHeight(10)
	ta.CharLimit = 500

	return CommitModal{
		textarea: ta,
		styles:   styles,
		visible:  false,
	}
}

func (m CommitModal) Init() tea.Cmd {
	return textarea.Blink
}

func (m CommitModal) Update(msg tea.Msg) (CommitModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m CommitModal) View() string {
	if !m.visible {
		return ""
	}

	title := m.styles.Title.Render("Commit Message")
	help := m.styles.Help.Render("Ctrl+Enter: Commit | Esc: Cancel")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.textarea.View(),
		"",
		help,
	)

	modal := m.styles.PanelFocused.Render(content)

	return lipgloss.Place(
		80, 24,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

func (m *CommitModal) Show() {
	m.visible = true
	m.textarea.Focus()
	m.textarea.SetValue("")
}

func (m *CommitModal) Hide() {
	m.visible = false
	m.textarea.Blur()
}

func (m *CommitModal) IsVisible() bool {
	return m.visible
}

func (m *CommitModal) Value() string {
	return m.textarea.Value()
}
