package graph

import (
	"fmt"
	"strings"
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
	graph  *GraphBuilder
}

type GraphBuilder struct {
	commits     []*git.Commit
	commitMap   map[string]int
	commitLanes map[string]int
	activeLanes []string
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

func (g *GraphRenderer) InitGraph(commits []*git.Commit) {
	g.graph = &GraphBuilder{
		commits:     commits,
		commitMap:   make(map[string]int),
		commitLanes: make(map[string]int),
		activeLanes: make([]string, 0),
	}

	for i, c := range commits {
		g.graph.commitMap[c.Hash] = i
	}
}

func (gb *GraphBuilder) getLaneForCommit(commit *git.Commit) int {
	if lane, exists := gb.commitLanes[commit.Hash]; exists {
		return lane
	}

	if len(commit.Parents) > 0 {
		parentHash := commit.Parents[0]
		if parentLane, exists := gb.commitLanes[parentHash]; exists {
			gb.commitLanes[commit.Hash] = parentLane
			return parentLane
		}
	}

	for i, hash := range gb.activeLanes {
		if hash == "" {
			gb.activeLanes[i] = commit.Hash
			gb.commitLanes[commit.Hash] = i
			return i
		}
	}

	lane := len(gb.activeLanes)
	gb.activeLanes = append(gb.activeLanes, commit.Hash)
	gb.commitLanes[commit.Hash] = lane
	return lane
}

func (gb *GraphBuilder) updateActiveLanes(commit *git.Commit, lane int) {
	gb.activeLanes[lane] = ""

	if len(commit.Parents) > 0 {
		parentHash := commit.Parents[0]
		gb.activeLanes[lane] = parentHash
	}
}

func (g *GraphRenderer) RenderCommitLine(commit *git.Commit, index int) string {
	if g.graph == nil {
		return g.renderSimple(commit, index)
	}

	lane := g.graph.getLaneForCommit(commit)

	numLanes := len(g.graph.activeLanes)
	if numLanes == 0 {
		numLanes = 1
	}

	graphParts := make([]string, numLanes)

	for i := 0; i < numLanes; i++ {
		if i == lane {
			color := g.colors[i%len(g.colors)]
			graphParts[i] = lipgloss.NewStyle().Foreground(color).Render(CommitSymbol)
		} else if g.graph.activeLanes[i] != "" {
			laneColor := g.colors[i%len(g.colors)]
			graphParts[i] = lipgloss.NewStyle().Foreground(laneColor).Render(LineVertical)
		} else {
			graphParts[i] = " "
		}
	}

	g.graph.updateActiveLanes(commit, lane)

	graphStr := strings.Join(graphParts, " ")

	hashStyle := lipgloss.NewStyle().Foreground(g.theme.CommitHash)
	subjectStyle := lipgloss.NewStyle().Foreground(g.theme.Foreground)
	relTime := formatRelativeTime(commit.Date)

	line := fmt.Sprintf("%s  %s  %s  %s",
		graphStr,
		hashStyle.Render(commit.ShortHash),
		subjectStyle.Render(truncate(commit.Subject, 45)),
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
