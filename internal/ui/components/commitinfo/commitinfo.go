package commitinfo

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/git"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type Model struct {
	viewport viewport.Model
	commit   *git.Commit
	styles   *styles.Styles
	width    int
	height   int
}

func New(styles *styles.Styles, width, height int) Model {
	vp := viewport.New(width, height)
	return Model{
		viewport: vp,
		styles:   styles,
		width:    width,
		height:   height,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.commit == nil {
		return m.styles.Panel.Render("Select a commit to view details")
	}
	return m.viewport.View()
}

func (m *Model) SetCommit(commit *git.Commit) {
	m.commit = commit

	content := m.renderCommitInfo()
	m.viewport.SetContent(content)
}

func (m *Model) renderCommitInfo() string {
	if m.commit == nil {
		return ""
	}

	hashStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.CommitHash).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Subtext).Bold(true)

	details := fmt.Sprintf("%s %s\n", labelStyle.Render("Commit:"), hashStyle.Render(m.commit.Hash))
	details += fmt.Sprintf("%s %s <%s>\n", labelStyle.Render("Author:"), m.commit.Author, m.commit.Email)
	details += fmt.Sprintf("%s %s\n", labelStyle.Render("Date:"), m.commit.Date.Format("Mon Jan 2 15:04:05 2006"))
	details += fmt.Sprintf("\n%s", m.commit.Message)

	return details
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}
