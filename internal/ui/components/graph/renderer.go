package graph

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/git"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

const (
	CommitSymbol   = "●"
	LineVertical   = "│"
	LineHorizontal = "─"
	LineCross      = "┼"
	LineBranchR    = "├"
	LineMerge      = "┬"
)

type GraphRenderer struct {
	theme  styles.Theme
	colors []lipgloss.Color
}

func NewGraphRenderer(theme styles.Theme) *GraphRenderer {
	return &GraphRenderer{
		theme: theme,
		colors: []lipgloss.Color{
			theme.Graph1,
			theme.Graph2,
			theme.Graph3,
			theme.Graph4,
			theme.Graph5,
		},
	}
}

func (g *GraphRenderer) RenderCommitLine(commit *git.Commit, index int) string {
	colorIndex := index % len(g.colors)
	color := g.colors[colorIndex]

	commitStyle := lipgloss.NewStyle().Foreground(color)
	hashStyle := lipgloss.NewStyle().Foreground(g.theme.CommitHash)

	graphSymbol := commitStyle.Render(CommitSymbol)

	relTime := formatRelativeTime(commit.Date)

	line := fmt.Sprintf("%s %s %s (%s)",
		graphSymbol,
		hashStyle.Render(commit.ShortHash),
		truncate(commit.Subject, 60),
		relTime,
	)

	return line
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatRelativeTime(t interface{}) string {
	return "2h ago"
}
