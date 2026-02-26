package modals

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type CommitModal struct {
	input   textinput.Model
	styles  *styles.Styles
	visible bool
	width   int
	height  int
}

func NewCommitModal(s *styles.Styles) CommitModal {
	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.CharLimit = 500
	ti.Width = 60

	// Style the text input with themed backgrounds.
	panelBg := s.Theme.BackgroundPanel
	ti.PromptStyle = lipgloss.NewStyle().
		Foreground(s.Theme.BranchFeature).
		Background(panelBg).
		Bold(true)
	ti.TextStyle = lipgloss.NewStyle().
		Foreground(s.Theme.Foreground).
		Background(panelBg)
	ti.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(s.Theme.DiffContext).
		Background(panelBg)
	ti.Cursor.Style = lipgloss.NewStyle().
		Background(s.Theme.Foreground)
	ti.Prompt = "  "

	return CommitModal{
		input:   ti,
		styles:  s,
		visible: false,
		width:   80,
		height:  24,
	}
}

func (m CommitModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m CommitModal) Update(msg tea.Msg) (CommitModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// Height returns the number of terminal rows this component occupies when visible.
func (m CommitModal) Height() int {
	if !m.visible {
		return 0
	}
	return 3 // border top + input line + border bottom
}

// View renders the inline commit input bar (meant to sit above the action bar).
func (m CommitModal) View() string {
	if !m.visible {
		return ""
	}

	theme := m.styles.Theme
	panelBg := theme.BackgroundPanel
	bgStyle := lipgloss.NewStyle().Background(panelBg)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.BranchFeature).
		Background(panelBg).
		Bold(true)
	hintStyle := lipgloss.NewStyle().
		Foreground(theme.DiffContext).
		Background(panelBg).
		Italic(true)

	label := labelStyle.Render(" Commit:")
	tiView := m.input.View()

	// Adaptive hint: drop or shorten based on available width.
	labelWidth := lipgloss.Width(label)
	tiWidth := lipgloss.Width(tiView)
	innerAvail := m.width - 4 // border left/right + small padding

	hintText := "  Enter to commit | Esc to cancel"
	hintWidth := lipgloss.Width(hintText)
	used := labelWidth + 1 + tiWidth + hintWidth
	if used > innerAvail {
		hintText = "  Enter | Esc"
		hintWidth = lipgloss.Width(hintText)
		used = labelWidth + 1 + tiWidth + hintWidth
		if used > innerAvail {
			hintText = ""
		}
	}

	var hint string
	if hintText != "" {
		hint = hintStyle.Render(hintText)
	}

	innerContent := label + bgStyle.Render(" ") + tiView + hint

	// Pad to full width with themed background.
	visWidth := lipgloss.Width(innerContent)
	if visWidth < m.width-2 {
		innerContent = innerContent + bgStyle.Width(m.width-2-visWidth).Render("")
	}

	bar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.BranchFeature).
		BorderBackground(theme.Background).
		Background(panelBg).
		Width(m.width - 2).
		Render(innerContent)

	return bar
}

func (m *CommitModal) Show() {
	m.visible = true
	m.input.Focus()
	m.input.SetValue("")
}

func (m *CommitModal) Hide() {
	m.visible = false
	m.input.Blur()
}

func (m *CommitModal) IsVisible() bool {
	return m.visible
}

func (m *CommitModal) Value() string {
	return m.input.Value()
}

func (m *CommitModal) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Text input gets remaining space after label("Commit:" ~10) + padding.
	// At very narrow widths, the hint will be dropped in View(), so we
	// only need to account for the label.
	tiWidth := width - 16 // label + borders + small pad
	if tiWidth < 10 {
		tiWidth = 10
	}
	if tiWidth > 80 {
		tiWidth = 80
	}
	m.input.Width = tiWidth
}
