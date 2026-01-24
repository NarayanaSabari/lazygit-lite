package styles

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	Theme        Theme
	Panel        lipgloss.Style
	PanelFocused lipgloss.Style
	Title        lipgloss.Style
	StatusBar    lipgloss.Style
	CommitHash   lipgloss.Style
	BranchName   lipgloss.Style
	Selected     lipgloss.Style
	Help         lipgloss.Style
}

func NewStyles(theme Theme) *Styles {
	return &Styles{
		Theme: theme,
		Panel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(0, 1),
		PanelFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.Head).
			Padding(0, 1),
		Title: lipgloss.NewStyle().
			Foreground(theme.Foreground).
			Bold(true).
			Padding(0, 1),
		StatusBar: lipgloss.NewStyle().
			Foreground(theme.Subtext).
			Background(theme.Selection).
			Padding(0, 1),
		CommitHash: lipgloss.NewStyle().
			Foreground(theme.CommitHash),
		BranchName: lipgloss.NewStyle().
			Foreground(theme.BranchFeature).
			Bold(true),
		Selected: lipgloss.NewStyle().
			Foreground(theme.Foreground).
			Background(theme.Selection),
		Help: lipgloss.NewStyle().
			Foreground(theme.Subtext),
	}
}
