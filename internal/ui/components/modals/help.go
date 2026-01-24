package modals

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type HelpModal struct {
	styles  *styles.Styles
	visible bool
}

func NewHelpModal(styles *styles.Styles) HelpModal {
	return HelpModal{
		styles:  styles,
		visible: false,
	}
}

func (m HelpModal) View() string {
	if !m.visible {
		return ""
	}

	title := m.styles.Title.Render("Keybindings")

	helpText := `
Navigation:
  j/↓       - Move down
  k/↑       - Move up
  h/←       - Focus left panel
  l/→       - Focus right panel
  g/Home    - Go to top
  G/End     - Go to bottom
  Ctrl+D    - Page down
  Ctrl+U    - Page up

Actions:
  c         - Commit
  p         - Push
  P         - Pull
  f         - Fetch
  b         - Branch picker
  Enter     - View commit details

Clipboard:
  y         - Copy commit hash
  Y         - Copy commit message
  Ctrl+Y    - Copy diff
  
General:
  ?         - Toggle help
  q/Ctrl+C  - Quit

Note: Native terminal text selection works with mouse drag.
`

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.styles.Help.Render(helpText),
	)

	modal := m.styles.PanelFocused.Render(content)

	return lipgloss.Place(
		80, 24,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

func (m *HelpModal) Toggle() {
	m.visible = !m.visible
}

func (m *HelpModal) IsVisible() bool {
	return m.visible
}
