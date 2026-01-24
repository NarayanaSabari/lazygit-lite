package graph

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/lazygit-lite/internal/git"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type Model struct {
	list     list.Model
	commits  []*git.Commit
	renderer *GraphRenderer
	width    int
	height   int
}

type commitItem struct {
	commit *git.Commit
	line   string
}

func (i commitItem) FilterValue() string { return i.commit.Subject }
func (i commitItem) Title() string       { return i.line }
func (i commitItem) Description() string { return "" }

func New(commits []*git.Commit, theme styles.Theme, width, height int) Model {
	renderer := NewGraphRenderer(theme)
	renderer.InitGraph(commits)

	items := make([]list.Item, len(commits))
	for i, commit := range commits {
		line := renderer.RenderCommitLine(commit, i)
		items[i] = commitItem{
			commit: commit,
			line:   line,
		}
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Commits"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return Model{
		list:     l,
		commits:  commits,
		renderer: renderer,
		width:    width,
		height:   height,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.list.View()
}

func (m Model) SelectedCommit() *git.Commit {
	if item, ok := m.list.SelectedItem().(commitItem); ok {
		return item.commit
	}
	return nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
}
