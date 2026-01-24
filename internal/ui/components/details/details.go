package details

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type Model struct {
	viewport viewport.Model
	diff     string
	styles   *styles.Styles
	width    int
	height   int
}

func New(styles *styles.Styles, width, height int) Model {
	vp := viewport.New(width, height)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3
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
	if m.diff == "" {
		return m.styles.Panel.Render("Select a commit to view diff")
	}
	return m.viewport.View()
}

func (m *Model) SetDiff(diff string) {
	m.diff = diff
	m.viewport.SetContent(diff)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}
