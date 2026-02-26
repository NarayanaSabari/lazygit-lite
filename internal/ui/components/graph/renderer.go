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
	commits           []*git.Commit
	vertices          []*Vertex
	commitIndex       map[string]int
	laneSnapshots     []LaneState // lane state AT each commit (before parent assignment)
	postLaneSnapshots []LaneState // lane state AFTER each commit (after parent assignment)
	maxLanes          int
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
		commits:           commits,
		vertices:          make([]*Vertex, len(commits)),
		commitIndex:       make(map[string]int),
		laneSnapshots:     make([]LaneState, len(commits)),
		postLaneSnapshots: make([]LaneState, len(commits)),
		maxLanes:          0,
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

		// Capture the post-snapshot: lane state after this commit's parents
		// have been assigned. This is used for connector lines.
		postCopy := make([]int, len(lanes))
		copy(postCopy, lanes)
		gb.postLaneSnapshots[i] = LaneState{lanes: postCopy, maxLanes: len(lanes)}

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

// RenderCommitLine renders a single commit line. maxWidth is the available
// character width so the line can be truncated to prevent wrapping.
// bg is the background color to use for all text in this line (allows the
// caller to pass Selection for highlighted rows, BackgroundPanel for expanded
// headers, etc.).
func (g *GraphRenderer) RenderCommitLine(commit *git.Commit, index int, maxWidth int, bg lipgloss.Color) string {
	if g.graph == nil || index >= len(g.graph.vertices) {
		return g.renderSimple(commit, index, bg)
	}

	isUncommitted := commit.Hash == git.UncommittedHash

	v := g.graph.vertices[index]
	snapshot := g.graph.laneSnapshots[index]

	// Use global maxLanes so every row has the same graph column width.
	numLanes := g.graph.maxLanes
	if numLanes == 0 {
		numLanes = 1
	}
	if numLanes > 10 {
		numLanes = 10
	}

	graphParts := make([]string, numLanes)
	isMerge := len(commit.Parents) > 1

	mergeTargetLanes := make(map[int]bool)
	if isMerge {
		for j := 1; j < len(v.parents); j++ {
			parentIdx := v.parents[j]
			if parentIdx < len(g.graph.vertices) {
				parentVertex := g.graph.vertices[parentIdx]
				if parentVertex.x >= 0 {
					mergeTargetLanes[parentVertex.x] = true
				}
			}
		}
	}

	for lane := 0; lane < numLanes; lane++ {
		if lane == v.x {
			if isUncommitted {
				// Distinct symbol for uncommitted changes.
				uncommittedColor := g.theme.CommitHash // Peach/orange from theme
				graphParts[lane] = lipgloss.NewStyle().Foreground(uncommittedColor).Background(bg).Bold(true).Render("◌")
			} else {
				color := g.colors[v.color%len(g.colors)]
				graphParts[lane] = lipgloss.NewStyle().Foreground(color).Background(bg).Render(CommitSymbol)
			}
		} else if mergeTargetLanes[lane] {
			laneColor := g.colors[lane%len(g.colors)]
			if lane > v.x {
				graphParts[lane] = lipgloss.NewStyle().Foreground(laneColor).Background(bg).Render(LineCornerBR)
			} else {
				graphParts[lane] = lipgloss.NewStyle().Foreground(laneColor).Background(bg).Render(LineCornerBL)
			}
		} else if lane < len(snapshot.lanes) && snapshot.lanes[lane] != -1 {
			laneColor := g.colors[lane%len(g.colors)]
			graphParts[lane] = lipgloss.NewStyle().Foreground(laneColor).Background(bg).Render(LineVertical)
		} else if isMerge && lane > v.x && lane < numLanes {
			foundMerge := false
			for targetLane := range mergeTargetLanes {
				if lane > v.x && lane < targetLane {
					laneColor := g.colors[targetLane%len(g.colors)]
					graphParts[lane] = lipgloss.NewStyle().Foreground(laneColor).Background(bg).Render(LineHorizontal)
					foundMerge = true
					break
				}
			}
			if !foundMerge {
				graphParts[lane] = lipgloss.NewStyle().Background(bg).Render(" ")
			}
		} else {
			graphParts[lane] = lipgloss.NewStyle().Background(bg).Render(" ")
		}
	}

	graphStr := strings.Join(graphParts, "")

	var refStr string
	if len(commit.Refs) > 0 {
		refStr = g.renderRefs(commit.Refs, bg)
	}

	hashStyle := lipgloss.NewStyle().Foreground(g.theme.CommitHash).Background(bg)
	dateStyle := lipgloss.NewStyle().Foreground(g.theme.Subtext).Background(bg)
	subjectStyle := lipgloss.NewStyle().Foreground(g.theme.Foreground).Background(bg)
	spacer := lipgloss.NewStyle().Background(bg).Render(" ")

	// Uncommitted changes get a distinct hash and subject color.
	if isUncommitted {
		uncommittedColor := g.theme.CommitHash // Peach/orange from theme
		hashStyle = lipgloss.NewStyle().Foreground(uncommittedColor).Background(bg).Bold(true)
		subjectStyle = lipgloss.NewStyle().Foreground(uncommittedColor).Background(bg).Italic(true)
	}

	// Build the line: graph | hash | (refs) | subject | relative-time
	relTime := formatRelativeTime(commit.Date)

	// Calculate how much space the prefix (graph + hash + refs) and time consume
	// so we can truncate the subject to fit within maxWidth.
	prefix := graphStr + spacer + hashStyle.Render(commit.ShortHash)
	if refStr != "" {
		prefix = prefix + spacer + refStr
	}
	prefixWidth := lipgloss.Width(prefix)

	timeStr := dateStyle.Render(relTime)
	timeWidth := lipgloss.Width(timeStr)

	// Available width for subject = maxWidth - prefix - time - gaps (2 spacers + 1 gap before time)
	subjectAvail := maxWidth - prefixWidth - timeWidth - 3 // 1 spacer before subject + min 2 for time gap
	if subjectAvail < 4 {
		subjectAvail = 4
	}

	subject := commit.Subject
	subjectRunes := []rune(subject)
	if len(subjectRunes) > subjectAvail {
		subject = string(subjectRunes[:subjectAvail-1]) + "…"
	}

	line := prefix + spacer + subjectStyle.Render(subject)

	// Append the relative timestamp right-aligned if there's room.
	lineWidth := lipgloss.Width(line)
	gap := maxWidth - lineWidth - timeWidth - 1
	if gap > 1 {
		line = line + lipgloss.NewStyle().Background(bg).Width(gap).Render("") + timeStr
	}

	return line
}

func (g *GraphRenderer) renderRefs(refs []git.Ref, bg lipgloss.Color) string {
	var parts []string

	decoBg := g.theme.BackgroundPanel

	for _, ref := range refs {
		var style lipgloss.Style
		var icon string

		switch ref.RefType {
		case git.RefTypeTag:
			style = lipgloss.NewStyle().
				Foreground(g.theme.Tag).
				Background(decoBg).
				Bold(true).
				Padding(0, 1)
			icon = "t:"
		case git.RefTypeBranch:
			if ref.IsHead {
				style = lipgloss.NewStyle().
					Foreground(g.theme.Head).
					Background(decoBg).
					Bold(true).
					Padding(0, 1)
				icon = "* "
			} else if ref.IsRemote {
				style = lipgloss.NewStyle().
					Foreground(g.theme.BranchFeature).
					Background(decoBg).
					Padding(0, 1)
				icon = ""
			} else {
				style = lipgloss.NewStyle().
					Foreground(g.theme.BranchMain).
					Background(decoBg).
					Bold(true).
					Padding(0, 1)
				icon = ""
			}
		}

		parts = append(parts, style.Render(icon+ref.Name))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, lipgloss.NewStyle().Background(bg).Render(" "))
}

func (g *GraphRenderer) renderSimple(commit *git.Commit, index int, bg lipgloss.Color) string {
	colorIndex := index % len(g.colors)
	color := g.colors[colorIndex]

	commitStyle := lipgloss.NewStyle().Foreground(color).Background(bg)
	hashStyle := lipgloss.NewStyle().Foreground(g.theme.CommitHash).Background(bg)
	subjectStyle := lipgloss.NewStyle().Foreground(g.theme.Foreground).Background(bg)
	spacer := lipgloss.NewStyle().Background(bg).Render(" ")

	graphSymbol := commitStyle.Render(CommitSymbol)

	return graphSymbol + spacer + hashStyle.Render(commit.ShortHash) + spacer + subjectStyle.Render(commit.Subject)
}

func (g *GraphRenderer) MaxLanes() int {
	if g.graph == nil {
		return 1
	}
	if g.graph.maxLanes == 0 {
		return 1
	}
	return g.graph.maxLanes
}

// RenderLaneGutter renders the lane gutter (vertical continuation lines)
// for display alongside expanded content rows. It uses the post-snapshot
// for the given commit index — the lane state AFTER that commit's parents
// have been assigned — since expanded content sits visually between the
// commit and the next row. The returned string is exactly the width of
// the lane columns (one character per lane).
func (g *GraphRenderer) RenderLaneGutter(index int, bg lipgloss.Color) string {
	if g.graph == nil || index >= len(g.graph.vertices) {
		return ""
	}

	postSnap := g.graph.postLaneSnapshots[index]
	// Use global maxLanes for consistent gutter width.
	numLanes := g.graph.maxLanes
	if numLanes == 0 {
		numLanes = 1
	}
	if numLanes > 10 {
		numLanes = 10
	}

	parts := make([]string, numLanes)
	for lane := 0; lane < numLanes; lane++ {
		if lane < len(postSnap.lanes) && postSnap.lanes[lane] != -1 {
			laneColor := g.colors[lane%len(g.colors)]
			parts[lane] = lipgloss.NewStyle().Foreground(laneColor).Background(bg).Render(LineVertical)
		} else {
			parts[lane] = lipgloss.NewStyle().Background(bg).Render(" ")
		}
	}
	return strings.Join(parts, "")
}

// NeedsConnectorLine returns true if commit at `index` requires a dedicated
// connector row below it — i.e., it has fork lanes that branch out horizontally
// and cannot be represented on the commit rows alone. When false, the connector
// row can be omitted to keep the graph compact.
func (g *GraphRenderer) NeedsConnectorLine(index int) bool {
	if g.graph == nil || index >= len(g.graph.vertices) {
		return false
	}

	v := g.graph.vertices[index]
	if len(v.parents) <= 1 {
		// Single parent (or root commit) — no fork, no connector needed.
		return false
	}

	// Multi-parent commit (merge). Check if any new lanes were created
	// for secondary parents that weren't already active.
	postSnap := g.graph.postLaneSnapshots[index]
	snap := g.graph.laneSnapshots[index]

	for j := 1; j < len(v.parents); j++ {
		parentIdx := v.parents[j]
		if parentIdx < len(g.graph.vertices) {
			for lane := 0; lane < len(postSnap.lanes); lane++ {
				if postSnap.lanes[lane] == parentIdx {
					wasActive := lane < len(snap.lanes) && snap.lanes[lane] != -1
					if !wasActive {
						// A new fork lane was created — need the connector row
						// to draw the horizontal bridging + corner.
						return true
					}
				}
			}
		}
	}
	return false
}

// RenderConnectorLine renders the vertical/branching connector line between
// commit row `index` and the next commit row. This creates the flowchart-like
// appearance by drawing │ for straight lanes and ╭/╰/─ for branch/merge paths.
// bg is the background color to use for this connector row.
func (g *GraphRenderer) RenderConnectorLine(index int, maxWidth int, bg lipgloss.Color) string {
	if g.graph == nil || index >= len(g.graph.vertices) {
		return ""
	}

	// The post-snapshot tells us which lanes are active AFTER commit `index`
	// has processed its parents (i.e., the state heading into the next commit).
	postSnap := g.graph.postLaneSnapshots[index]
	snap := g.graph.laneSnapshots[index]

	v := g.graph.vertices[index]

	// Use global maxLanes for consistent column width across all rows.
	numLanes := g.graph.maxLanes
	if numLanes == 0 {
		numLanes = 1
	}
	if numLanes > 10 {
		numLanes = 10
	}

	// Determine which lanes are "new" (forking off to a second parent) vs
	// "continuing" (same lane, straight │) vs "ending" (lane goes away).
	//
	// For each lane in the connector:
	//   - If the lane is active in postSnap → draw │ (continuing down)
	//   - If the lane was active in snap but not in postSnap → lane is ending
	//   - Special: if this commit forked (has >1 parents), draw branching lines
	//     from the commit's lane to the new parent lanes

	// Build a set of "new fork lanes" — lanes that were NOT in the pre-snapshot
	// but ARE in the post-snapshot. These are the merge/fork parent lanes.
	forkLanes := make(map[int]bool) // lane -> true if this is a new fork
	commitLane := v.x

	if len(v.parents) > 1 {
		for j := 1; j < len(v.parents); j++ {
			parentIdx := v.parents[j]
			if parentIdx < len(g.graph.vertices) {
				// Find which lane this parent ended up in within postSnap.
				for lane := 0; lane < len(postSnap.lanes); lane++ {
					if postSnap.lanes[lane] == parentIdx {
						// Check if this lane wasn't active in the pre-snapshot
						// (i.e., it's a newly forked lane).
						wasActive := lane < len(snap.lanes) && snap.lanes[lane] != -1
						if !wasActive {
							forkLanes[lane] = true
						}
					}
				}
			}
		}
	}

	// Also detect if the commit's own lane continues down to first parent
	// in the same lane (straight │) or if it moves to a different lane.
	firstParentContinues := false
	if len(v.parents) > 0 {
		firstParentIdx := v.parents[0]
		if commitLane < len(postSnap.lanes) && postSnap.lanes[commitLane] == firstParentIdx {
			firstParentContinues = true
		}
	}

	graphParts := make([]string, numLanes)

	// Determine the horizontal span that needs fork connection lines.
	// We need horizontal lines between the commit lane and each fork target.
	// forkMin/forkMax define the full range that needs horizontal bridging.
	forkMin, forkMax := numLanes, -1
	for lane := range forkLanes {
		if lane < forkMin {
			forkMin = lane
		}
		if lane > forkMax {
			forkMax = lane
		}
	}
	hasForks := len(forkLanes) > 0
	// Include commitLane in the range for determining horizontal bridging.
	if hasForks {
		if commitLane < forkMin {
			forkMin = commitLane
		}
		if commitLane > forkMax {
			forkMax = commitLane
		}
	}

	for lane := 0; lane < numLanes; lane++ {
		postActive := lane < len(postSnap.lanes) && postSnap.lanes[lane] != -1
		laneColor := g.colors[lane%len(g.colors)]
		style := lipgloss.NewStyle().Foreground(laneColor).Background(bg)

		if lane == commitLane {
			// The commit's own lane.
			if firstParentContinues {
				// Straight continuation from commit to its first parent.
				graphParts[lane] = style.Render(LineVertical)
			} else if hasForks && postActive {
				// First parent doesn't continue here, but lane is active and
				// forks are branching — show vertical (lane continues for
				// another commit below).
				graphParts[lane] = style.Render(LineVertical)
			} else if hasForks && !postActive {
				// Lane ends but forks are branching off from here.
				// Show a down-branch character to connect to horizontal fork lines.
				graphParts[lane] = style.Render(LineVertical)
			} else if postActive {
				// No forks, but lane continues (first parent in same lane).
				graphParts[lane] = style.Render(LineVertical)
			} else {
				// Lane ends entirely — no continuation, no forks.
				graphParts[lane] = lipgloss.NewStyle().Background(bg).Render(" ")
			}
		} else if forkLanes[lane] {
			// This lane is a new fork destination — draw a corner.
			if lane > commitLane {
				// Fork goes right: ╭ curves from left-horizontal to down-vertical.
				graphParts[lane] = style.Render(LineCornerTR)
			} else {
				// Fork goes left: ╮ curves from right-horizontal to down-vertical.
				graphParts[lane] = style.Render(LineCornerTL)
			}
		} else if postActive {
			// Regular continuation of an existing lane.
			graphParts[lane] = style.Render(LineVertical)
		} else if hasForks && lane > forkMin && lane < forkMax {
			// This lane is in the horizontal span between commit lane and
			// a fork target. Draw horizontal line unless the lane is also
			// being used as a continuing lane (handled above as postActive).
			// Use the color of the fork direction this line bridges toward.
			var bridgeColor lipgloss.Color
			if lane > commitLane {
				bridgeColor = g.colors[forkMax%len(g.colors)]
			} else {
				bridgeColor = g.colors[forkMin%len(g.colors)]
			}
			graphParts[lane] = lipgloss.NewStyle().Foreground(bridgeColor).Background(bg).Render(LineHorizontal)
		} else {
			graphParts[lane] = lipgloss.NewStyle().Background(bg).Render(" ")
		}
	}

	return strings.Join(graphParts, "")
}

// ---------------------------------------------------------------------------
// Side-by-side diff rendering
// ---------------------------------------------------------------------------

// diffLine represents one line from a unified diff with its type.
type diffLine struct {
	kind    byte // ' ' context, '+' add, '-' remove, '@' hunk header
	content string
	oldNum  int // 0 means blank
	newNum  int // 0 means blank
}

// parseDiffLines parses raw unified diff text into structured diffLines,
// skipping file-level headers (diff --git, index, ---, +++).
func parseDiffLines(raw string) []diffLine {
	lines := strings.Split(raw, "\n")
	var result []diffLine
	var oldLine, newLine int

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "deleted file") {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			oldLine, newLine = parseHunkHeader(line)
			result = append(result, diffLine{kind: '@', content: line})
			continue
		}

		if strings.HasPrefix(line, "-") {
			result = append(result, diffLine{kind: '-', content: line[1:], oldNum: oldLine})
			oldLine++
		} else if strings.HasPrefix(line, "+") {
			result = append(result, diffLine{kind: '+', content: line[1:], newNum: newLine})
			newLine++
		} else if strings.HasPrefix(line, "\\") {
			result = append(result, diffLine{kind: '\\', content: line})
		} else {
			result = append(result, diffLine{kind: ' ', content: strings.TrimPrefix(line, " "), oldNum: oldLine, newNum: newLine})
			oldLine++
			newLine++
		}
	}
	return result
}

func parseHunkHeader(line string) (oldStart, newStart int) {
	var oldCount, newCount int
	fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount)
	if oldStart == 0 && newStart == 0 {
		fmt.Sscanf(line, "@@ -%d +%d @@", &oldStart, &newStart)
	}
	if oldStart == 0 && newStart == 0 {
		fmt.Sscanf(line, "@@ -%d,%d +%d @@", &oldStart, &oldCount, &newStart)
	}
	if oldStart == 0 && newStart == 0 {
		fmt.Sscanf(line, "@@ -%d +%d,%d @@", &oldStart, &newStart, &newCount)
	}
	return
}

// sideBySidePair represents one rendered row of the side-by-side view.
type sideBySidePair struct {
	leftNum   int    // 0 = blank
	leftText  string // raw text (no prefix)
	leftKind  byte   // ' ', '-', or '@'
	rightNum  int
	rightText string
	rightKind byte // ' ', '+', or '@'
}

// buildSideBySidePairs converts parsed diff lines into paired left/right rows.
// Adjacent remove/add blocks are zipped together; context appears on both sides.
func buildSideBySidePairs(dlines []diffLine) []sideBySidePair {
	var pairs []sideBySidePair
	i := 0
	for i < len(dlines) {
		dl := dlines[i]

		switch dl.kind {
		case '@':
			pairs = append(pairs, sideBySidePair{
				leftKind:  '@',
				leftText:  dl.content,
				rightKind: '@',
				rightText: dl.content,
			})
			i++

		case ' ':
			pairs = append(pairs, sideBySidePair{
				leftNum:   dl.oldNum,
				leftText:  dl.content,
				leftKind:  ' ',
				rightNum:  dl.newNum,
				rightText: dl.content,
				rightKind: ' ',
			})
			i++

		case '-':
			// Collect consecutive removes.
			var removes []diffLine
			for i < len(dlines) && dlines[i].kind == '-' {
				removes = append(removes, dlines[i])
				i++
			}
			// Collect immediately following adds.
			var adds []diffLine
			for i < len(dlines) && dlines[i].kind == '+' {
				adds = append(adds, dlines[i])
				i++
			}
			// Zip them together.
			maxLen := len(removes)
			if len(adds) > maxLen {
				maxLen = len(adds)
			}
			for j := 0; j < maxLen; j++ {
				p := sideBySidePair{}
				if j < len(removes) {
					p.leftNum = removes[j].oldNum
					p.leftText = removes[j].content
					p.leftKind = '-'
				}
				if j < len(adds) {
					p.rightNum = adds[j].newNum
					p.rightText = adds[j].content
					p.rightKind = '+'
				}
				pairs = append(pairs, p)
			}

		case '+':
			// Orphan add (no preceding remove).
			pairs = append(pairs, sideBySidePair{
				rightNum:  dl.newNum,
				rightText: dl.content,
				rightKind: '+',
			})
			i++

		case '\\':
			// "\ No newline at end of file" — show on both sides.
			pairs = append(pairs, sideBySidePair{
				leftText:  dl.content,
				leftKind:  '\\',
				rightText: dl.content,
				rightKind: '\\',
			})
			i++

		default:
			i++
		}
	}
	return pairs
}

// FormatDiffLines takes a raw diff string and returns styled side-by-side lines.
// maxWidth is the total available character width for the diff area.
func (g *GraphRenderer) FormatDiffLines(diff string, maxWidth int) []string {
	if diff == "" {
		return nil
	}

	parsed := parseDiffLines(diff)
	pairs := buildSideBySidePairs(parsed)

	// Layout: [left half] [separator 1ch "│"] [right half]
	// Each half: [lineNum 5ch] [content]
	// We use lipgloss.Width on each half block to guarantee fixed column alignment.
	const sepWidth = 1 // "│"
	const numWidth = 5 // e.g. " 142 "
	halfWidth := (maxWidth - sepWidth) / 2
	if halfWidth < 10 {
		halfWidth = 10
	}
	contentWidth := halfWidth - numWidth
	if contentWidth < 4 {
		contentWidth = 4
	}

	removeBg := g.theme.DiffRemoveBg
	addBg := g.theme.DiffAddBg

	// Styles for the line number column — fixed width via lipgloss.
	numStyleOld := lipgloss.NewStyle().
		Foreground(g.theme.DiffRemove).
		Background(removeBg).
		Width(numWidth).
		Align(lipgloss.Right)
	numStyleNew := lipgloss.NewStyle().
		Foreground(g.theme.DiffAdd).
		Background(addBg).
		Width(numWidth).
		Align(lipgloss.Right)
	numStyleCtx := lipgloss.NewStyle().
		Foreground(g.theme.DiffContext).
		Background(g.theme.Background).
		Width(numWidth).
		Align(lipgloss.Right)
	numStyleBlank := lipgloss.NewStyle().
		Background(g.theme.Background).
		Width(numWidth)

	removeContentStyle := lipgloss.NewStyle().
		Foreground(g.theme.DiffRemove).
		Background(removeBg).
		Width(contentWidth)
	addContentStyle := lipgloss.NewStyle().
		Foreground(g.theme.DiffAdd).
		Background(addBg).
		Width(contentWidth)
	contextContentStyle := lipgloss.NewStyle().
		Foreground(g.theme.Foreground).
		Background(g.theme.Background).
		Width(contentWidth)
	blankContentStyle := lipgloss.NewStyle().
		Background(g.theme.Background).
		Width(contentWidth)

	hunkStyle := lipgloss.NewStyle().
		Foreground(g.theme.BranchFeature).
		Background(g.theme.BackgroundPanel).
		Width(maxWidth)
	sepStyle := lipgloss.NewStyle().
		Foreground(g.theme.DiffContext).
		Background(g.theme.Background)
	headerStyle := lipgloss.NewStyle().
		Foreground(g.theme.Subtext).
		Background(g.theme.Background).
		Italic(true).
		Width(maxWidth)

	sep := sepStyle.Render("│")

	var result []string

	for _, p := range pairs {
		if p.leftKind == '@' {
			result = append(result, hunkStyle.Render(truncate(p.leftText, maxWidth)))
			continue
		}

		if p.leftKind == '\\' || p.rightKind == '\\' {
			result = append(result, headerStyle.Render(truncate(p.leftText, maxWidth)))
			continue
		}

		// Build left half.
		var leftNum, leftContent string
		switch p.leftKind {
		case '-':
			leftNum = numStyleOld.Render(fmt.Sprintf("%d", p.leftNum))
			leftContent = removeContentStyle.Render(truncate(p.leftText, contentWidth))
		case ' ':
			leftNum = numStyleCtx.Render(fmt.Sprintf("%d", p.leftNum))
			leftContent = contextContentStyle.Render(truncate(p.leftText, contentWidth))
		default:
			leftNum = numStyleBlank.Render("")
			leftContent = blankContentStyle.Render("")
		}

		// Build right half.
		var rightNum, rightContent string
		switch p.rightKind {
		case '+':
			rightNum = numStyleNew.Render(fmt.Sprintf("%d", p.rightNum))
			rightContent = addContentStyle.Render(truncate(p.rightText, contentWidth))
		case ' ':
			rightNum = numStyleCtx.Render(fmt.Sprintf("%d", p.rightNum))
			rightContent = contextContentStyle.Render(truncate(p.rightText, contentWidth))
		default:
			rightNum = numStyleBlank.Render("")
			rightContent = blankContentStyle.Render("")
		}

		line := leftNum + leftContent + sep + rightNum + rightContent
		result = append(result, line)
	}

	// Limit to a reasonable number of lines for inline display.
	const maxDiffLines = 300
	if len(result) > maxDiffLines {
		result = result[:maxDiffLines]
		result = append(result, headerStyle.Render(
			fmt.Sprintf("  ... %d more lines (truncated)", len(pairs)-maxDiffLines)))
	}

	return result
}

func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) > maxWidth {
		return string(runes[:maxWidth])
	}
	return s
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
