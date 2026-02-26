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
	LineCornerTR   = "┌"
	LineCornerBR   = "└"
	LineCornerTL   = "┐"
	LineCornerBL   = "┘"
	LineMergeDown  = "┬"
	LineMergeUp    = "┴"

	// LaneSpacing is the number of padding characters after each lane glyph.
	// This controls the horizontal gap between branch lines.
	LaneSpacing = 1
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
	lanes      []int
	laneColors []int // color index for each lane (branch-aware, not position-based)
	maxLanes   int
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

	// Pre-compute the set of vertices that appear as a secondary parent
	// of any merge commit. These are "branch-off" commits that should get
	// their own visual lane rather than sharing their first-parent's lane.
	isMergeTarget := make(map[int]bool)
	for _, v := range gb.vertices {
		for j := 1; j < len(v.parents); j++ {
			isMergeTarget[v.parents[j]] = true
		}
	}

	lanes := make([]int, 0)
	laneColors := make([]int, 0) // color index per lane (branch-aware)
	nextColor := 0               // rotating counter for new branches

	for i := 0; i < len(gb.vertices); i++ {
		v := gb.vertices[i]

		assignedLane := -1
		inheritedColor := -1

		// Step 1: Try to inherit a lane from a child whose first parent is
		// this vertex (first-parent chain continuation). Pick the leftmost.
		// Skip children that are merge targets (secondary parents of some
		// merge commit) — they represent branch-off points and should keep
		// their own lane so the fork is visible in the graph.
		for _, childIdx := range v.children {
			child := gb.vertices[childIdx]
			if child.x >= 0 {
				isFirstParent := len(child.parents) > 0 && child.parents[0] == i
				if isFirstParent && !isMergeTarget[childIdx] {
					if assignedLane == -1 || child.x < assignedLane {
						assignedLane = child.x
						inheritedColor = child.color // inherit the branch color
					}
				}
			}
		}

		// Step 2: If no child donated a lane, also check if a merge-target
		// lane was already reserved for this vertex by a child merge commit.
		// However, skip reservations that came from a merge-target child
		// (a child that is itself a secondary parent of some merge). Those
		// children represent branch-off points and should keep their own
		// lane — the current vertex should get a fresh lane instead.
		if assignedLane == -1 {
			for laneIdx, occupant := range lanes {
				if occupant == i {
					// Check if the reservation came from a merge-target child.
					reservedByMergeTarget := false
					for _, childIdx := range v.children {
						child := gb.vertices[childIdx]
						if child.x == laneIdx && len(child.parents) > 0 && child.parents[0] == i && isMergeTarget[childIdx] {
							reservedByMergeTarget = true
							break
						}
					}
					if !reservedByMergeTarget {
						assignedLane = laneIdx
						inheritedColor = laneColors[laneIdx]
					}
					break
				}
			}
		}

		// Step 3: Fallback — find a free lane.
		if assignedLane == -1 {
			assignedLane = findAvailableLane(lanes)
		}

		for len(lanes) <= assignedLane {
			lanes = append(lanes, -1)
			laneColors = append(laneColors, -1)
		}

		// Assign color: inherit from child/reservation, or allocate new.
		if inheritedColor >= 0 {
			v.color = inheritedColor
		} else {
			v.color = nextColor
			nextColor = (nextColor + 1) % 5
		}

		v.x = assignedLane
		lanes[assignedLane] = i
		laneColors[assignedLane] = v.color

		// Step 4: Clear any OTHER lanes that were also reserved for this
		// vertex (e.g., merge-target lanes from multiple children). Now
		// that the vertex has been placed, those extra reservations are
		// redundant.
		for laneIdx := range lanes {
			if lanes[laneIdx] == i && laneIdx != assignedLane {
				lanes[laneIdx] = -1
				laneColors[laneIdx] = -1
			}
		}

		// Capture pre-snapshot: lane state at this commit row, before
		// convergence freeing and parent reservation.
		lanesCopy := make([]int, len(lanes))
		copy(lanesCopy, lanes)
		colorsCopy := make([]int, len(laneColors))
		copy(colorsCopy, laneColors)
		gb.laneSnapshots[i] = LaneState{lanes: lanesCopy, laneColors: colorsCopy, maxLanes: len(lanes)}

		// Step 5: Free convergence lanes — children whose first parent is
		// this commit but who live in a different lane. Their lane was
		// carrying this commit's index down from above; free it now.
		for _, childIdx := range v.children {
			child := gb.vertices[childIdx]
			childIsFirstParent := len(child.parents) > 0 && child.parents[0] == i
			if childIsFirstParent && child.x != assignedLane {
				if child.x >= 0 && child.x < len(lanes) && lanes[child.x] == i {
					lanes[child.x] = -1
					laneColors[child.x] = -1
				}
			}
		}

		// Step 6: Hand off the current lane to the first parent.
		if len(v.parents) > 0 {
			firstParent := v.parents[0]
			lanes[assignedLane] = firstParent
			laneColors[assignedLane] = v.color // parent inherits this branch's color
		} else {
			lanes[assignedLane] = -1
			laneColors[assignedLane] = -1
		}

		// Step 7: Reserve lanes for secondary parents (merge edges).
		// If the parent already occupies a lane (from another branch's
		// first-parent chain), don't allocate a duplicate — reuse it.
		for j := 1; j < len(v.parents); j++ {
			parentIdx := v.parents[j]

			// Check if this parent is already in a lane.
			alreadyPlaced := false
			for laneIdx := range lanes {
				if lanes[laneIdx] == parentIdx {
					alreadyPlaced = true
					break
				}
			}
			if alreadyPlaced {
				continue
			}

			parentLane := findAvailableLane(lanes)
			for len(lanes) <= parentLane {
				lanes = append(lanes, -1)
				laneColors = append(laneColors, -1)
			}
			// New merge branch gets a new color.
			mergeColor := nextColor
			nextColor = (nextColor + 1) % 5
			lanes[parentLane] = parentIdx
			laneColors[parentLane] = mergeColor
		}

		lanes, laneColors = trimEmptyTrailingLanesWithColors(lanes, laneColors)

		// Capture the post-snapshot: lane state after this commit's parents
		// have been assigned. This is used for lane gutters on expanded content.
		postCopy := make([]int, len(lanes))
		copy(postCopy, lanes)
		postColors := make([]int, len(laneColors))
		copy(postColors, laneColors)
		gb.postLaneSnapshots[i] = LaneState{lanes: postCopy, laneColors: postColors, maxLanes: len(lanes)}

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

// trimEmptyTrailingLanesWithColors trims both the lanes and laneColors slices
// in lockstep, removing trailing empty (-1) entries.
func trimEmptyTrailingLanesWithColors(lanes []int, laneColors []int) ([]int, []int) {
	lastActive := -1
	for i := len(lanes) - 1; i >= 0; i-- {
		if lanes[i] != -1 {
			lastActive = i
			break
		}
	}
	if lastActive == -1 {
		return lanes[:0], laneColors[:0]
	}
	return lanes[:lastActive+1], laneColors[:lastActive+1]
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

	// mergeTargetLanes: lanes where this commit's 2nd+ parents were reserved.
	// We look up the parent's lane from the postLaneSnapshot (which captures
	// the state right after parent lanes were reserved in computeLayout),
	// rather than parentVertex.x (which may not be set yet for parents
	// that appear later in the topological order).
	mergeTargetLanes := make(map[int]bool)
	if isMerge {
		postSnap := g.graph.postLaneSnapshots[index]
		for j := 1; j < len(v.parents); j++ {
			parentIdx := v.parents[j]
			if parentIdx < len(g.graph.vertices) {
				// Find the lane reserved for this parent in the post-snapshot.
				for lane := 0; lane < len(postSnap.lanes); lane++ {
					if postSnap.lanes[lane] == parentIdx && lane != v.x {
						mergeTargetLanes[lane] = true
						break
					}
				}
			}
		}
	}

	// convergeLanes: lanes where children have this commit as their first
	// parent but live in a different lane. Lines converge FROM those child
	// lanes (above) INTO this commit's lane. This is the visual fork point
	// where a branch was created from this commit.
	convergeLanes := make(map[int]bool)
	for _, childIdx := range v.children {
		child := g.graph.vertices[childIdx]
		if len(child.parents) > 0 && child.parents[0] == index && child.x != v.x {
			convergeLanes[child.x] = true
		}
	}
	hasConverge := len(convergeLanes) > 0

	// Compute the horizontal span for convergence bridging.
	convergeMin, convergeMax := numLanes, -1
	for lane := range convergeLanes {
		if lane < convergeMin {
			convergeMin = lane
		}
		if lane > convergeMax {
			convergeMax = lane
		}
	}
	if hasConverge {
		if v.x < convergeMin {
			convergeMin = v.x
		}
		if v.x > convergeMax {
			convergeMax = v.x
		}
	}

	// Also compute merge bridging span for merge commits.
	hasMergeTarget := len(mergeTargetLanes) > 0
	mergeMin, mergeMax := numLanes, -1
	for lane := range mergeTargetLanes {
		if lane < mergeMin {
			mergeMin = lane
		}
		if lane > mergeMax {
			mergeMax = lane
		}
	}
	if hasMergeTarget {
		if v.x < mergeMin {
			mergeMin = v.x
		}
		if v.x > mergeMax {
			mergeMax = v.x
		}
	}

	for lane := 0; lane < numLanes; lane++ {
		// Use the branch-aware color from the snapshot, falling back to
		// lane-position-based color if no snapshot color is available.
		laneColorIdx := lane % len(g.colors)
		if lane < len(snapshot.laneColors) && snapshot.laneColors[lane] >= 0 {
			laneColorIdx = snapshot.laneColors[lane] % len(g.colors)
		}
		laneColor := g.colors[laneColorIdx]

		// Determine if this lane's padding should be a horizontal bridge.
		// A lane is "bridging" if it sits within the convergence or merge
		// span (between the commit lane and the outermost target lane).
		isBridging := false
		var bridgeFg lipgloss.Color
		if hasConverge && lane >= convergeMin && lane < convergeMax {
			isBridging = true
			if lane >= v.x {
				bridgeColorIdx := convergeMax % len(g.colors)
				if convergeMax < len(snapshot.laneColors) && snapshot.laneColors[convergeMax] >= 0 {
					bridgeColorIdx = snapshot.laneColors[convergeMax] % len(g.colors)
				}
				bridgeFg = g.colors[bridgeColorIdx]
			} else {
				bridgeColorIdx := convergeMin % len(g.colors)
				if convergeMin < len(snapshot.laneColors) && snapshot.laneColors[convergeMin] >= 0 {
					bridgeColorIdx = snapshot.laneColors[convergeMin] % len(g.colors)
				}
				bridgeFg = g.colors[bridgeColorIdx]
			}
		} else if hasMergeTarget && lane >= mergeMin && lane < mergeMax {
			isBridging = true
			if lane >= v.x {
				bridgeColorIdx := mergeMax % len(g.colors)
				if mergeMax < len(snapshot.laneColors) && snapshot.laneColors[mergeMax] >= 0 {
					bridgeColorIdx = snapshot.laneColors[mergeMax] % len(g.colors)
				}
				bridgeFg = g.colors[bridgeColorIdx]
			} else {
				bridgeColorIdx := mergeMin % len(g.colors)
				if mergeMin < len(snapshot.laneColors) && snapshot.laneColors[mergeMin] >= 0 {
					bridgeColorIdx = snapshot.laneColors[mergeMin] % len(g.colors)
				}
				bridgeFg = g.colors[bridgeColorIdx]
			}
		}

		if lane == v.x {
			if isUncommitted {
				// Distinct symbol for uncommitted changes.
				uncommittedColor := g.theme.CommitHash // Peach/orange from theme
				graphParts[lane] = laneCell("◌", bg, uncommittedColor, isBridging)
			} else {
				color := g.colors[v.color%len(g.colors)]
				graphParts[lane] = laneCell(CommitSymbol, bg, color, isBridging)
			}
		} else if convergeLanes[lane] {
			// A child branch lived in this lane and converges into this
			// commit. Draw a corner: line comes down from above and curves
			// toward the commit's lane.
			if lane > v.x {
				// Child lane is to the right → ╯ (down-left curve)
				// Last convergence lane — no bridge after it.
				graphParts[lane] = laneCell(LineCornerBL, bg, laneColor, false)
			} else {
				// Child lane is to the left → ╰ (down-right curve)
				graphParts[lane] = laneCell(LineCornerBR, bg, laneColor, isBridging)
			}
		} else if mergeTargetLanes[lane] {
			// A secondary parent lives in this lane. Draw a corner: line
			// goes down from here to the parent below, and horizontally
			// toward the merge commit.
			if lane > v.x {
				// Merge target is to the RIGHT of the commit → turn left-and-down ┐
				graphParts[lane] = laneCell(LineCornerTL, bg, laneColor, false)
			} else {
				// Merge target is to the LEFT of the commit → turn right-and-down ┌
				graphParts[lane] = laneCell(LineCornerTR, bg, laneColor, isBridging)
			}
		} else if lane < len(snapshot.lanes) && snapshot.lanes[lane] != -1 {
			// Vertical continuation — if a bridge crosses through, draw the
			// bridge padding in the bridge color (not the lane color).
			if isBridging {
				graphParts[lane] = laneCellBridge(LineVertical, bg, laneColor, bridgeFg, true)
			} else {
				graphParts[lane] = laneCell(LineVertical, bg, laneColor, false)
			}
		} else if isBridging {
			// Horizontal bridge — the glyph itself is ─ and the padding is also ─.
			graphParts[lane] = laneCell(LineHorizontal, bg, bridgeFg, true)
		} else {
			graphParts[lane] = blankCell(bg)
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
		return 1 + LaneSpacing
	}
	n := g.graph.maxLanes
	if n == 0 {
		return 1 + LaneSpacing
	}
	// Each lane occupies 1 glyph + LaneSpacing padding characters.
	return n * (1 + LaneSpacing)
}

// laneCell renders a single lane cell: glyph followed by LaneSpacing spaces,
// all styled with the given background. For horizontal bridging, the padding
// also uses the horizontal line character. bridgeFg sets the color for the
// bridge padding (if different from fg, e.g. when a vertical line has a
// bridge crossing through its padding).
func laneCell(glyph string, bg lipgloss.Color, fg lipgloss.Color, bridge bool) string {
	return laneCellBridge(glyph, bg, fg, fg, bridge)
}

func laneCellBridge(glyph string, bg lipgloss.Color, fg lipgloss.Color, bridgeFg lipgloss.Color, bridge bool) string {
	style := lipgloss.NewStyle().Foreground(fg).Background(bg)
	pad := strings.Repeat(" ", LaneSpacing)
	if bridge {
		pad = strings.Repeat(LineHorizontal, LaneSpacing)
	}
	padStyle := lipgloss.NewStyle().Foreground(bridgeFg).Background(bg)
	return style.Render(glyph) + padStyle.Render(pad)
}

// blankCell renders an empty lane cell (spaces only) with the given background.
func blankCell(bg lipgloss.Color) string {
	return lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", 1+LaneSpacing))
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
			laneColorIdx := lane % len(g.colors)
			if lane < len(postSnap.laneColors) && postSnap.laneColors[lane] >= 0 {
				laneColorIdx = postSnap.laneColors[lane] % len(g.colors)
			}
			laneColor := g.colors[laneColorIdx]
			parts[lane] = laneCell(LineVertical, bg, laneColor, false)
		} else {
			parts[lane] = blankCell(bg)
		}
	}
	return strings.Join(parts, "")
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
