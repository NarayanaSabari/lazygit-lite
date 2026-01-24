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
	LineBranchR    = "├"
	LineBranchL    = "┤"
	LineCornerTR   = "╭"
	LineCornerBR   = "╰"
	LineCornerTL   = "╮"
	LineCornerBL   = "╯"
	LineMergeDown  = "┬"
	LineMergeUp    = "┴"
)

type GraphRenderer struct {
	theme  styles.Theme
	colors []lipgloss.Color
	graph  *GraphBuilder
}

type Vertex struct {
	id       int
	hash     string
	parents  []int
	children []int
	x        int
	color    int
}

type LaneState struct {
	lanes    []int
	maxLanes int
}

type GraphBuilder struct {
	commits       []*git.Commit
	vertices      []*Vertex
	commitIndex   map[string]int
	laneSnapshots []LaneState
	maxLanes      int
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
	gb := &GraphBuilder{
		commits:       commits,
		vertices:      make([]*Vertex, len(commits)),
		commitIndex:   make(map[string]int),
		laneSnapshots: make([]LaneState, len(commits)),
		maxLanes:      0,
	}

	for i, c := range commits {
		gb.commitIndex[c.Hash] = i
		gb.vertices[i] = &Vertex{
			id:       i,
			hash:     c.Hash,
			parents:  make([]int, 0),
			children: make([]int, 0),
			x:        -1,
			color:    -1,
		}
	}

	for i, c := range commits {
		for _, parentHash := range c.Parents {
			if parentIdx, exists := gb.commitIndex[parentHash]; exists {
				gb.vertices[i].parents = append(gb.vertices[i].parents, parentIdx)
				gb.vertices[parentIdx].children = append(gb.vertices[parentIdx].children, i)
			}
		}
	}

	gb.computeLayout()
	g.graph = gb
}

func (gb *GraphBuilder) computeLayout() {
	if len(gb.vertices) == 0 {
		return
	}

	lanes := make([]int, 0)

	for i := 0; i < len(gb.vertices); i++ {
		v := gb.vertices[i]

		assignedLane := -1

		for _, childIdx := range v.children {
			child := gb.vertices[childIdx]
			if child.x >= 0 {
				isFirstParent := len(child.parents) > 0 && child.parents[0] == i
				if isFirstParent {
					if assignedLane == -1 || child.x < assignedLane {
						assignedLane = child.x
					}
				}
			}
		}

		if assignedLane == -1 {
			assignedLane = findAvailableLane(lanes)
		}

		for len(lanes) <= assignedLane {
			lanes = append(lanes, -1)
		}

		v.x = assignedLane
		v.color = assignedLane % 5
		lanes[assignedLane] = i

		for _, childIdx := range v.children {
			child := gb.vertices[childIdx]
			childIsFirstParent := len(child.parents) > 0 && child.parents[0] == i
			if !childIsFirstParent && child.x != assignedLane {
				for laneIdx := range lanes {
					if lanes[laneIdx] == childIdx {
						lanes[laneIdx] = -1
					}
				}
			}
		}

		lanesCopy := make([]int, len(lanes))
		copy(lanesCopy, lanes)
		gb.laneSnapshots[i] = LaneState{lanes: lanesCopy, maxLanes: len(lanes)}

		if len(v.parents) > 0 {
			firstParent := v.parents[0]
			lanes[assignedLane] = firstParent
		} else {
			lanes[assignedLane] = -1
		}

		for j := 1; j < len(v.parents); j++ {
			parentIdx := v.parents[j]
			parentLane := findAvailableLane(lanes)
			for len(lanes) <= parentLane {
				lanes = append(lanes, -1)
			}
			lanes[parentLane] = parentIdx
		}

		lanes = trimEmptyTrailingLanes(lanes)

		if len(lanes) > gb.maxLanes {
			gb.maxLanes = len(lanes)
		}
	}
}

func findAvailableLane(lanes []int) int {
	for i, occupant := range lanes {
		if occupant == -1 {
			return i
		}
	}
	return len(lanes)
}

func trimEmptyTrailingLanes(lanes []int) []int {
	lastActive := -1
	for i := len(lanes) - 1; i >= 0; i-- {
		if lanes[i] != -1 {
			lastActive = i
			break
		}
	}
	if lastActive == -1 {
		return lanes[:0]
	}
	return lanes[:lastActive+1]
}

func (g *GraphRenderer) RenderCommitLine(commit *git.Commit, index int) string {
	if g.graph == nil || index >= len(g.graph.vertices) {
		return g.renderSimple(commit, index)
	}

	v := g.graph.vertices[index]
	snapshot := g.graph.laneSnapshots[index]

	numLanes := snapshot.maxLanes
	if numLanes == 0 {
		numLanes = 1
	}
	if numLanes > 10 {
		numLanes = 10
	}

	graphParts := make([]string, numLanes)

	for lane := 0; lane < numLanes; lane++ {
		if lane == v.x {
			color := g.colors[v.color%len(g.colors)]
			graphParts[lane] = lipgloss.NewStyle().Foreground(color).Render(CommitSymbol)
		} else if lane < len(snapshot.lanes) && snapshot.lanes[lane] != -1 {
			laneColor := g.colors[lane%len(g.colors)]
			graphParts[lane] = lipgloss.NewStyle().Foreground(laneColor).Render(LineVertical)
		} else {
			graphParts[lane] = " "
		}
	}

	graphStr := strings.Join(graphParts, "")

	hashStyle := lipgloss.NewStyle().Foreground(g.theme.CommitHash)
	subjectStyle := lipgloss.NewStyle().Foreground(g.theme.Foreground)
	relTime := formatRelativeTime(commit.Date)

	line := fmt.Sprintf("%s %s %s %s",
		graphStr,
		hashStyle.Render(commit.ShortHash),
		subjectStyle.Render(truncate(commit.Subject, 50)),
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
