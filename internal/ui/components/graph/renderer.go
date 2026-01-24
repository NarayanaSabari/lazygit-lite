package graph

import (
	"fmt"
	"time"

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
	LineBranchL    = "┤"
	LineMerge      = "┬"
	LineSplit      = "┴"
	LineCornerTL   = "╮"
	LineCornerTR   = "╭"
	LineCornerBL   = "╯"
	LineCornerBR   = "╰"
)

type GraphRenderer struct {
	theme  styles.Theme
	colors []lipgloss.Color
	lanes  map[string]int
	graph  *GraphBuilder
}

type GraphBuilder struct {
	commits []*git.Commit
	lanes   []string
	laneMap map[string]int
	parents map[string][]string
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
		lanes: make(map[string]int),
	}
}

func (g *GraphRenderer) InitGraph(commits []*git.Commit) {
	g.graph = &GraphBuilder{
		commits: commits,
		lanes:   make([]string, 0),
		laneMap: make(map[string]int),
		parents: make(map[string][]string),
	}

	for _, c := range commits {
		g.graph.parents[c.Hash] = c.Parents
	}

	for _, c := range commits {
		g.graph.assignLane(c)
	}
}

func (gb *GraphBuilder) assignLane(commit *git.Commit) int {
	if lane, ok := gb.laneMap[commit.Hash]; ok {
		return lane
	}

	targetLane := -1

	if len(commit.Parents) > 0 {
		parentHash := commit.Parents[0]
		if parentLane, ok := gb.laneMap[parentHash]; ok {
			targetLane = parentLane
		}
	}

	if targetLane == -1 {
		for i, hash := range gb.lanes {
			if hash == "" {
				targetLane = i
				break
			}
		}
	}

	if targetLane == -1 {
		targetLane = len(gb.lanes)
		gb.lanes = append(gb.lanes, "")
	}

	gb.lanes[targetLane] = commit.Hash
	gb.laneMap[commit.Hash] = targetLane

	return targetLane
}

func (g *GraphRenderer) RenderCommitLine(commit *git.Commit, index int) string {
	if g.graph == nil {
		return g.renderSimple(commit, index)
	}

	lane := g.graph.laneMap[commit.Hash]
	color := g.colors[lane%len(g.colors)]

	graphParts := make([]string, len(g.graph.lanes))
	for i := range graphParts {
		if i == lane {
			graphParts[i] = lipgloss.NewStyle().Foreground(color).Render(CommitSymbol)
		} else if g.graph.lanes[i] != "" {
			laneColor := g.colors[i%len(g.colors)]
			graphParts[i] = lipgloss.NewStyle().Foreground(laneColor).Render(LineVertical)
		} else {
			graphParts[i] = " "
		}
	}

	graphStr := ""
	for _, part := range graphParts {
		graphStr += part + " "
	}

	hashStyle := lipgloss.NewStyle().Foreground(g.theme.CommitHash)
	authorStyle := lipgloss.NewStyle().Foreground(g.theme.Subtext)
	relTime := formatRelativeTime(commit.Date)

	line := fmt.Sprintf("%s %s  %s  %s  %s",
		graphStr,
		hashStyle.Render(commit.ShortHash),
		truncate(commit.Subject, 50),
		authorStyle.Render(truncate(commit.Author, 20)),
		relTime,
	)

	return line
}

func (g *GraphRenderer) renderSimple(commit *git.Commit, index int) string {
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

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
