package layout

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

type Layout struct {
	width      int
	height     int
	background lipgloss.Color
	border     lipgloss.Color
	title      lipgloss.Color
}

func New(width, height int, _ float64, background, border, title lipgloss.Color) *Layout {
	return &Layout{
		width:      width,
		height:     height,
		background: background,
		border:     border,
		title:      title,
	}
}

// Calculate returns the usable inner dimensions for the single main panel.
// Returns contentWidth and contentHeight (inside borders).
func (l *Layout) Calculate() (contentWidth, contentHeight int) {
	return l.CalculateWithExtra(0)
}

// CalculateWithExtra is like Calculate but reserves additional rows for inline
// panels (commit input, help panel) between the main panel and the action bar.
func (l *Layout) CalculateWithExtra(extraHeight int) (contentWidth, contentHeight int) {
	// Reserve 1 row for the action bar at the bottom.
	actionBarHeight := 1
	contentHeight = l.height - actionBarHeight - 2 - extraHeight // 2 border rows (top+bottom)

	// 2 border columns (left+right).
	contentWidth = l.width - 2

	// Ensure minimum viable sizes. The graph panel needs at least 3 rows to be
	// usable (1 commit row + some breathing room).
	if contentHeight < 3 {
		contentHeight = 3
	}
	if contentWidth < 10 {
		contentWidth = 10
	}

	return
}

func (l *Layout) Render(mainPanel, actionBar string) string {
	return l.RenderWithExtra(mainPanel, "", actionBar)
}

// RenderWithExtra renders the layout with an optional extra panel between the
// main panel and the action bar (used for inline commit input, help, etc.).
// The entire output is placed into a full-screen area with a dark background
// using lipgloss.Place + WithWhitespaceBackground to fill every cell.
func (l *Layout) RenderWithExtra(mainPanel, extraPanel, actionBar string) string {
	extraHeight := 0
	if extraPanel != "" {
		extraHeight = lipgloss.Height(extraPanel)
	}
	contentW, contentH := l.CalculateWithExtra(extraHeight)

	titleStyle := lipgloss.NewStyle().
		Foreground(l.title).
		Bold(true)

	mainBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(l.border).
		BorderBackground(l.background).
		Background(l.background).
		Width(contentW).
		Height(contentH).
		Render(mainPanel)
	mainBox = l.renderWithTitle(mainBox, titleStyle.Render(" Commits "))

	var combined string
	if extraPanel != "" {
		combined = lipgloss.JoinVertical(lipgloss.Left, mainBox, extraPanel, actionBar)
	} else {
		combined = lipgloss.JoinVertical(lipgloss.Left, mainBox, actionBar)
	}

	// Place the combined UI into a full-terminal-sized area. WithWhitespaceBackground
	// ensures every cell in the terminal — including padding and gaps — is filled
	// with the base background color.
	return lipgloss.Place(
		l.width, l.height,
		lipgloss.Left, lipgloss.Top,
		combined,
		lipgloss.WithWhitespaceBackground(l.background),
	)
}

func (l *Layout) renderWithTitle(box, title string) string {
	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}

	// The first line is the top border rendered by lipgloss, e.g. "╭──────...──╮".
	// It contains ANSI escape codes for the border color. We need to splice the
	// styled title into it without corrupting escape sequences.
	//
	// Strategy: extract the visible (plain) runes and their byte positions from
	// the first line, then reconstruct it by slicing at byte boundaries and
	// inserting the title string in between.
	firstLine := lines[0]
	titleStart := 2 // visual position to insert title
	titleVisWidth := lipgloss.Width(title)

	// Build a map of visible-character index → byte offset.
	type charPos struct {
		byteStart int
		byteEnd   int
	}
	var visChars []charPos
	i := 0
	raw := []byte(firstLine)
	for i < len(raw) {
		if raw[i] == '\x1b' {
			// Skip ANSI escape sequence.
			for i < len(raw) && raw[i] != 'm' {
				i++
			}
			if i < len(raw) {
				i++ // skip 'm'
			}
			continue
		}
		// Decode one UTF-8 rune.
		r, size := utf8.DecodeRune(raw[i:])
		_ = r
		visChars = append(visChars, charPos{byteStart: i, byteEnd: i + size})
		i += size
	}

	if titleStart+titleVisWidth >= len(visChars) {
		// Title doesn't fit — just return unchanged.
		return box
	}

	// Byte offset where the title region starts (just before the titleStart-th visible char).
	spliceStart := visChars[titleStart].byteStart
	// Byte offset where the title region ends (just before the char after the title).
	spliceEnd := visChars[titleStart+titleVisWidth].byteStart

	// Find the ANSI reset/style code active at spliceEnd so the border color
	// resumes correctly. Walk backward from spliceEnd to find the last escape
	// sequence start before the splice region.
	// Simpler approach: just grab the border style's ANSI prefix from the
	// beginning of the line (everything before the first visible char).
	borderPrefix := firstLine[:visChars[0].byteStart]

	newFirst := firstLine[:spliceStart] + title + borderPrefix + firstLine[spliceEnd:]
	lines[0] = newFirst
	return strings.Join(lines, "\n")
}

func (l *Layout) SetSize(width, height int) {
	l.width = width
	l.height = height
}
