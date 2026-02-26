package graph

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/lazygit-lite/internal/git"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// SelectionChangedMsg is sent when the user moves the cursor to a different commit.
type SelectionChangedMsg struct {
	Commit *git.Commit
}

// FilesLoadedMsg is sent asynchronously after the file list for a commit is loaded.
type FilesLoadedMsg struct {
	Hash  string
	Files []git.ChangedFile
	Err   error
}

// FileDiffLoadedMsg is sent after a per-file diff is loaded.
type FileDiffLoadedMsg struct {
	Hash     string
	FilePath string
	Diff     string
	Err      error
}

// ---------------------------------------------------------------------------
// ExpandState tracks the inline-expand state for a single commit.
// ---------------------------------------------------------------------------

type ExpandState struct {
	// Files loaded for this commit.
	Files []git.ChangedFile

	// Index of the cursor inside the file list (-1 = on metadata header).
	FileIndex int

	// Which file path (if any) has its diff expanded.
	ExpandedFile string

	// The formatted diff content for ExpandedFile, split into lines.
	DiffLines []string
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type Model struct {
	commits  []*git.Commit
	renderer *GraphRenderer
	theme    styles.Theme
	width    int
	height   int

	// Cursor points at a commit index in m.commits.
	cursor int

	// Scroll offset: the first *visual* line shown in the viewport.
	scrollOffset int

	// Which commit index is expanded (-1 = none).
	expandedIdx int

	// Expand state for the currently expanded commit.
	expandState *ExpandState

	// Track last cursor for selection-changed detection.
	lastCursor int
}

func New(commits []*git.Commit, theme styles.Theme, width, height int) Model {
	renderer := NewGraphRenderer(theme)
	renderer.InitGraph(commits)

	return Model{
		commits:      commits,
		renderer:     renderer,
		theme:        theme,
		width:        width,
		height:       height,
		cursor:       0,
		scrollOffset: 0,
		expandedIdx:  -1,
		expandState:  nil,
		lastCursor:   0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case FilesLoadedMsg:
		return m.handleFilesLoaded(msg)

	case FileDiffLoadedMsg:
		return m.handleFileDiffLoaded(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "j", "down":
		return m.moveCursorDown()
	case "k", "up":
		return m.moveCursorUp()
	case "g", "home":
		return m.goToTop()
	case "G", "end":
		return m.goToBottom()
	case "ctrl+d":
		return m.pageDown()
	case "ctrl+u":
		return m.pageUp()
	}

	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	switch {
	case msg.Button == tea.MouseButtonWheelUp:
		m.collapseExpanded()
		m.scrollOffset -= 3
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
		// Move cursor to stay within the visible viewport.
		if m.cursor > m.scrollOffset+m.height-1 {
			m.cursor = m.scrollOffset + m.height - 1
		}
		if m.cursor >= len(m.commits) {
			m.cursor = len(m.commits) - 1
		}
		if m.cursor < m.scrollOffset {
			m.cursor = m.scrollOffset
		}
		return m.emitSelectionChanged()
	case msg.Button == tea.MouseButtonWheelDown:
		m.collapseExpanded()
		m.scrollOffset += 3
		m.clampScroll()
		// Move cursor to stay within the visible viewport.
		if m.cursor < m.scrollOffset {
			m.cursor = m.scrollOffset
		}
		if m.cursor >= len(m.commits) {
			m.cursor = len(m.commits) - 1
		}
		if m.cursor > m.scrollOffset+m.height-1 {
			m.cursor = m.scrollOffset + m.height - 1
		}
		return m.emitSelectionChanged()
	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease:
		return m.handleClick(msg.Y)
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Navigation helpers
// ---------------------------------------------------------------------------

func (m Model) moveCursorDown() (Model, tea.Cmd) {
	if m.isExpanded() {
		es := m.expandState

		// If a file diff is open, scroll the viewport through the diff first.
		if es.ExpandedFile != "" && len(es.DiffLines) > 0 {
			// Calculate the visual line of the last diff line.
			lastDiffVisLine := m.expandedFileDiffEndVisLine()
			// If there's still diff content below the viewport, scroll down.
			if lastDiffVisLine >= m.scrollOffset+m.height {
				m.scrollOffset++
				m.clampScroll()
				return m, nil
			}
			// Past the end of the diff — collapse it and move to next file.
			es.ExpandedFile = ""
			es.DiffLines = nil
			if es.FileIndex < len(es.Files)-1 {
				es.FileIndex++
				m.ensureCursorVisible()
				return m, nil
			}
			// Was the last file — collapse and move to next commit.
			m.collapseExpanded()
			if m.cursor < len(m.commits)-1 {
				m.cursor++
				m.ensureCursorVisible()
				return m.emitSelectionChanged()
			}
			return m, nil
		}

		// Navigate within the expanded commit's file list.
		if es.FileIndex < len(es.Files)-1 {
			es.FileIndex++
			m.ensureCursorVisible()
			return m, nil
		}
		// At the bottom of file list — collapse and move to next commit.
		m.collapseExpanded()
	}

	if m.cursor < len(m.commits)-1 {
		m.cursor++
		m.ensureCursorVisible()
		return m.emitSelectionChanged()
	}
	return m, nil
}

func (m Model) moveCursorUp() (Model, tea.Cmd) {
	if m.isExpanded() {
		es := m.expandState

		// If a file diff is open, scroll the viewport through the diff first.
		if es.ExpandedFile != "" && len(es.DiffLines) > 0 {
			// Calculate the visual line of the file entry (which owns the diff).
			fileEntryVisLine := m.cursorVisualLine()
			// If the file entry is above the viewport, scroll up.
			if fileEntryVisLine < m.scrollOffset {
				m.scrollOffset--
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
				return m, nil
			}
			// At the top of the diff — collapse it and stay on this file.
			es.ExpandedFile = ""
			es.DiffLines = nil
			m.ensureCursorVisible()
			return m, nil
		}

		if es.FileIndex > -1 {
			es.FileIndex--
			m.ensureCursorVisible()
			return m, nil
		}
		// At the top of expanded area — collapse and stay on same commit.
		m.collapseExpanded()
		m.ensureCursorVisible()
		return m, nil
	}

	if m.cursor > 0 {
		m.cursor--
		m.ensureCursorVisible()
		return m.emitSelectionChanged()
	}
	return m, nil
}

func (m Model) goToTop() (Model, tea.Cmd) {
	m.collapseExpanded()
	m.cursor = 0
	m.scrollOffset = 0
	return m.emitSelectionChanged()
}

func (m Model) goToBottom() (Model, tea.Cmd) {
	m.collapseExpanded()
	if len(m.commits) > 0 {
		m.cursor = len(m.commits) - 1
	}
	m.ensureCursorVisible()
	return m.emitSelectionChanged()
}

func (m Model) pageDown() (Model, tea.Cmd) {
	m.collapseExpanded()
	m.cursor += m.height / 2
	if m.cursor >= len(m.commits) {
		m.cursor = len(m.commits) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureCursorVisible()
	return m.emitSelectionChanged()
}

func (m Model) pageUp() (Model, tea.Cmd) {
	m.collapseExpanded()
	m.cursor -= m.height / 2
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureCursorVisible()
	return m.emitSelectionChanged()
}

func (m Model) handleClick(y int) (Model, tea.Cmd) {
	// Map visual y position (relative to viewport) to a commit or file row.
	targetVisLine := m.scrollOffset + y
	visLine := 0

	for i := 0; i < len(m.commits); i++ {
		if visLine == targetVisLine {
			// Clicked on a commit row.
			if m.cursor != i {
				m.collapseExpanded()
				m.cursor = i
				return m.emitSelectionChanged()
			}
			return m, nil
		}
		visLine++ // commit line itself

		if i == m.expandedIdx && m.expandState != nil {
			expandLines := m.expandedLineCount()
			if targetVisLine > visLine-1 && targetVisLine < visLine+expandLines {
				// Clicked inside the expanded area. Determine file index.
				localLine := targetVisLine - visLine
				// Line 0..metadataLines-1 = metadata header, then file entries.
				metaLines := m.metadataLineCount()
				if localLine < metaLines {
					// Clicked on metadata — select file index -1.
					m.expandState.FileIndex = -1
					return m, nil
				}
				fileClickLine := localLine - metaLines
				// Each file is 1 line, plus optional diff lines below the expanded file.
				fileLine := 0
				for fi := 0; fi < len(m.expandState.Files); fi++ {
					if fileLine == fileClickLine {
						m.expandState.FileIndex = fi
						return m, nil
					}
					fileLine++
					if m.expandState.Files[fi].Path == m.expandState.ExpandedFile && len(m.expandState.DiffLines) > 0 {
						fileLine += len(m.expandState.DiffLines)
					}
				}
				return m, nil
			}
			visLine += expandLines
		}
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Expand / Collapse
// ---------------------------------------------------------------------------

// ToggleExpand is called by the parent model when Enter is pressed.
// Returns a command to load files if expanding.
func (m *Model) ToggleExpand(repo *git.Repository) tea.Cmd {
	if m.isExpanded() {
		if m.expandedIdx == m.cursor {
			// Already expanded on this commit.
			es := m.expandState
			if es.FileIndex >= 0 && es.FileIndex < len(es.Files) {
				// A file is selected — toggle its diff.
				file := es.Files[es.FileIndex]
				if es.ExpandedFile == file.Path {
					// Collapse the file diff.
					es.ExpandedFile = ""
					es.DiffLines = nil
					return nil
				}
				// Expand a different file diff.
				es.ExpandedFile = file.Path
				es.DiffLines = nil
				hash := m.commits[m.cursor].Hash
				filePath := file.Path
				if hash == git.UncommittedHash {
					return func() tea.Msg {
						diff, err := repo.GetWorkingTreeFileDiff(filePath)
						return FileDiffLoadedMsg{Hash: hash, FilePath: filePath, Diff: diff, Err: err}
					}
				}
				return func() tea.Msg {
					diff, err := repo.GetFileDiff(hash, filePath)
					return FileDiffLoadedMsg{Hash: hash, FilePath: filePath, Diff: diff, Err: err}
				}
			}
			// FileIndex == -1 (on metadata) — collapse the whole commit.
			m.collapseExpanded()
			return nil
		}
		// Different commit — collapse old, expand new.
		m.collapseExpanded()
	}

	// Expand current commit.
	m.expandedIdx = m.cursor
	m.expandState = &ExpandState{
		FileIndex: -1,
	}
	hash := m.commits[m.cursor].Hash
	if hash == git.UncommittedHash {
		return func() tea.Msg {
			files, err := repo.GetWorkingTreeFiles()
			return FilesLoadedMsg{Hash: hash, Files: files, Err: err}
		}
	}
	return func() tea.Msg {
		files, err := repo.GetChangedFiles(hash)
		return FilesLoadedMsg{Hash: hash, Files: files, Err: err}
	}
}

func (m *Model) collapseExpanded() {
	m.expandedIdx = -1
	m.expandState = nil
}

// Collapse unconditionally closes any expanded commit.
func (m *Model) Collapse() {
	m.collapseExpanded()
}

func (m Model) isExpanded() bool {
	return m.expandedIdx >= 0 && m.expandState != nil
}

// ---------------------------------------------------------------------------
// Message handlers for async loads
// ---------------------------------------------------------------------------

func (m Model) handleFilesLoaded(msg FilesLoadedMsg) (Model, tea.Cmd) {
	if m.expandedIdx < 0 || m.expandedIdx >= len(m.commits) {
		return m, nil
	}
	if m.commits[m.expandedIdx].Hash != msg.Hash {
		return m, nil
	}
	m.expandState.Files = msg.Files
	if len(msg.Files) > 0 {
		m.expandState.FileIndex = 0
	}
	// The expanded content just grew (metadata + file list appeared). Make sure
	// the cursor is still visible, but only scroll forward — never snap back.
	m.ensureExpandedVisible()
	return m, nil
}

func (m Model) handleFileDiffLoaded(msg FileDiffLoadedMsg) (Model, tea.Cmd) {
	if m.expandState == nil {
		return m, nil
	}
	if m.expandedIdx < 0 || m.expandedIdx >= len(m.commits) {
		return m, nil
	}
	if m.commits[m.expandedIdx].Hash != msg.Hash || m.expandState.ExpandedFile != msg.FilePath {
		return m, nil
	}
	// Subtract the lane gutter width so diff lines fit alongside the gutter.
	gutterWidth := m.renderer.MaxLanes()
	if gutterWidth < 1 {
		gutterWidth = 1
	}
	diffWidth := m.width - gutterWidth
	if diffWidth < 20 {
		diffWidth = 20
	}
	m.expandState.DiffLines = m.renderer.FormatDiffLines(msg.Diff, diffWidth)
	// Don't call ensureCursorVisible here — the cursor (file entry) is already
	// visible since the user just pressed Enter on it. Calling it would snap
	// the viewport back to the cursor line, fighting any scroll the user has
	// already done. Just clamp to valid range.
	m.clampScroll()
	return m, nil
}

// ---------------------------------------------------------------------------
// Emit selection changed
// ---------------------------------------------------------------------------

func (m Model) emitSelectionChanged() (Model, tea.Cmd) {
	if m.cursor == m.lastCursor {
		return m, nil
	}
	m.lastCursor = m.cursor
	commit := m.SelectedCommit()
	if commit == nil {
		return m, nil
	}
	return m, func() tea.Msg {
		return SelectionChangedMsg{Commit: commit}
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m Model) View() string {
	if len(m.commits) == 0 {
		return "No commits"
	}

	m.clampScroll()

	var lines []string
	visLine := 0

	for i := 0; i < len(m.commits); i++ {
		commitLine := m.renderCommitRow(i)
		if visLine >= m.scrollOffset && visLine < m.scrollOffset+m.height {
			lines = append(lines, commitLine)
		}
		visLine++

		// Render expanded content below the expanded commit.
		if i == m.expandedIdx && m.expandState != nil {
			expandLines := m.renderExpandedContent(i)
			for _, el := range expandLines {
				if visLine >= m.scrollOffset && visLine < m.scrollOffset+m.height {
					lines = append(lines, el)
				}
				visLine++
			}
		}

		if len(lines) >= m.height {
			break
		}
	}

	// Pad remaining lines if we don't fill the viewport.
	emptyLine := lipgloss.NewStyle().
		Background(m.theme.Background).
		Width(m.width).
		Render("")
	for len(lines) < m.height {
		lines = append(lines, emptyLine)
	}

	return strings.Join(lines[:m.height], "\n")
}

// ---------------------------------------------------------------------------
// Render helpers
// ---------------------------------------------------------------------------

func (m Model) renderCommitRow(idx int) string {
	commit := m.commits[idx]

	isSelected := idx == m.cursor && (!m.isExpanded() || m.expandState.FileIndex == -1)
	isSelectedFinal := isSelected && (!m.isExpanded() || (m.isExpanded() && m.expandedIdx == m.cursor && m.expandState.FileIndex == -1))
	isExpandedHeader := idx == m.expandedIdx && m.isExpanded()

	// Determine the background color for this row so every character is rendered with it.
	var rowBg lipgloss.Color
	if isSelectedFinal {
		rowBg = m.theme.Selection
	} else if isExpandedHeader {
		rowBg = m.theme.BackgroundPanel
	} else {
		rowBg = m.theme.Background
	}

	line := m.renderer.RenderCommitLine(commit, idx, m.width, rowBg)

	// Pad to full width with the same background.
	visWidth := lipgloss.Width(line)
	if visWidth < m.width {
		line = line + lipgloss.NewStyle().Background(rowBg).Width(m.width-visWidth).Render("")
	}

	// Wrap the line to enforce full-width background coverage (fills any remaining gaps).
	if isSelectedFinal {
		line = lipgloss.NewStyle().
			Background(m.theme.Selection).
			Bold(true).
			Width(m.width).
			Render(line)
	} else if isExpandedHeader {
		line = lipgloss.NewStyle().
			Background(m.theme.BackgroundPanel).
			Width(m.width).
			Render(line)
	} else {
		line = lipgloss.NewStyle().
			Background(m.theme.Background).
			Width(m.width).
			Render(line)
	}

	return line
}

func (m Model) renderExpandedContent(commitIdx int) []string {
	if m.expandState == nil {
		return nil
	}

	// Compute the lane gutter that prefixes every expanded content line.
	// This uses the post-snapshot (lane state after parent assignment) so
	// the flow lines continue through the expanded section.
	panelBg := m.theme.BackgroundPanel
	gutter := m.renderer.RenderLaneGutter(commitIdx, panelBg)
	gutterWidth := lipgloss.Width(gutter)

	// Reduce effective width to leave room for the gutter.
	m.width = m.width - gutterWidth
	if m.width < 20 {
		m.width = 20
	}

	var lines []string

	commit := m.commits[commitIdx]

	// Metadata lines.
	metaLines := m.renderMetadata(commit)
	for _, ml := range metaLines {
		lines = append(lines, gutter+ml)
	}

	// File list.
	for fi, file := range m.expandState.Files {
		fileLine := m.renderFileEntry(fi, file)
		lines = append(lines, gutter+fileLine)

		// If this file has its diff expanded, render diff lines below it.
		if file.Path == m.expandState.ExpandedFile && len(m.expandState.DiffLines) > 0 {
			for _, dl := range m.expandState.DiffLines {
				lines = append(lines, gutter+dl)
			}
		}
	}

	return lines
}

func (m Model) renderMetadata(commit *git.Commit) []string {
	indent := "    "
	panelBg := m.theme.BackgroundPanel
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext).Background(panelBg)
	valueStyle := lipgloss.NewStyle().Foreground(m.theme.Foreground).Background(panelBg)
	hashStyle := lipgloss.NewStyle().Foreground(m.theme.CommitHash).Background(panelBg).Bold(true)
	authorStyle := lipgloss.NewStyle().Foreground(m.theme.BranchMain).Background(panelBg).Bold(true)
	emailStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext).Background(panelBg)
	dateStyle := lipgloss.NewStyle().Foreground(m.theme.Foreground).Background(panelBg)
	bgStyle := lipgloss.NewStyle().Background(panelBg)
	indentStr := bgStyle.Render(indent)
	spacer := bgStyle.Render("  ")

	isUncommitted := commit.Hash == git.UncommittedHash

	// maxContent is the usable character width inside the indent.
	maxContent := m.width - len(indent)
	if maxContent < 10 {
		maxContent = 10
	}

	padToWidth := func(line string) string {
		w := lipgloss.Width(line)
		if w < m.width {
			return line + bgStyle.Width(m.width-w).Render("")
		}
		return line
	}

	// truncStr truncates a plain string to fit within n runes, appending "…" if cut.
	truncStr := func(s string, n int) string {
		runes := []rune(s)
		if len(runes) > n && n > 1 {
			return string(runes[:n-1]) + "…"
		}
		return s
	}

	var lines []string

	if isUncommitted {
		// Simplified header for uncommitted changes.
		uncommittedColor := m.theme.CommitHash
		uncommittedStyle := lipgloss.NewStyle().Foreground(uncommittedColor).Background(panelBg).Bold(true)
		line1 := indentStr + uncommittedStyle.Render("Uncommitted changes") + spacer + labelStyle.Render("(working tree)")
		lines = append(lines, bgStyle.Width(m.width).Render(padToWidth(line1)))
	} else {
		// Hash + Author line. Truncate email if it would overflow.
		authorDisplay := truncStr(commit.Author, maxContent/3)
		emailDisplay := truncStr("<"+commit.Email+">", maxContent/3)
		line1 := indentStr +
			labelStyle.Render("Commit:") + bgStyle.Render(" ") +
			hashStyle.Render(commit.Hash[:10]) + spacer +
			labelStyle.Render("Author:") + bgStyle.Render(" ") +
			authorStyle.Render(authorDisplay) + bgStyle.Render(" ") +
			emailStyle.Render(emailDisplay)
		lines = append(lines, bgStyle.Width(m.width).Render(padToWidth(line1)))

		// Date + Message line. Truncate subject to remaining space.
		dateStr := commit.Date.Format("2006-01-02 15:04:05")
		// "    Date: YYYY-MM-DD HH:MM:SS  Msg: " is about 42 chars
		subjectAvail := maxContent - 42
		if subjectAvail < 8 {
			subjectAvail = 8
		}
		subjectDisplay := truncStr(commit.Subject, subjectAvail)
		line2 := indentStr +
			labelStyle.Render("Date:") + bgStyle.Render(" ") +
			dateStyle.Render(dateStr) + spacer +
			labelStyle.Render("Msg:") + bgStyle.Render(" ") +
			valueStyle.Render(subjectDisplay)
		lines = append(lines, bgStyle.Width(m.width).Render(padToWidth(line2)))

		// Refs line if any.
		if len(commit.Refs) > 0 {
			var refParts []string
			for _, ref := range commit.Refs {
				switch ref.RefType {
				case git.RefTypeBranch:
					if ref.IsHead {
						refParts = append(refParts, lipgloss.NewStyle().
							Foreground(m.theme.Head).Background(panelBg).Bold(true).Render("HEAD -> "+ref.Name))
					} else {
						refParts = append(refParts, lipgloss.NewStyle().
							Foreground(m.theme.BranchMain).Background(panelBg).Render(ref.Name))
					}
				case git.RefTypeTag:
					refParts = append(refParts, lipgloss.NewStyle().
						Foreground(m.theme.Tag).Background(panelBg).Render("tag: "+ref.Name))
				}
			}
			if len(refParts) > 0 {
				commaStyle := bgStyle
				line3 := indentStr + labelStyle.Render("Refs:") + bgStyle.Render(" ") + strings.Join(refParts, commaStyle.Render(", "))
				lines = append(lines, bgStyle.Width(m.width).Render(padToWidth(line3)))
			}
		}
	}

	// Separator / file list header.
	filesHeader := indentStr + labelStyle.Render(fmt.Sprintf("Changed files (%d):", len(m.expandState.Files)))
	lines = append(lines, bgStyle.Width(m.width).Render(padToWidth(filesHeader)))

	return lines
}

func (m Model) renderFileEntry(fileIdx int, file git.ChangedFile) string {
	indent := "      "

	// Determine background: selected file gets Selection, others get base Background.
	isFileSelected := m.expandState != nil && m.expandState.FileIndex == fileIdx && m.expandedIdx == m.cursor
	var bg lipgloss.Color
	if isFileSelected {
		bg = m.theme.Selection
	} else {
		bg = m.theme.Background
	}

	// Status icon and color.
	var statusIcon string
	var statusColor lipgloss.Color
	switch file.Status {
	case "A":
		statusIcon = "+"
		statusColor = m.theme.DiffAdd
	case "D":
		statusIcon = "-"
		statusColor = m.theme.DiffRemove
	case "M":
		statusIcon = "~"
		statusColor = m.theme.CommitHash
	case "?":
		statusIcon = "?"
		statusColor = m.theme.DiffAdd // Untracked files shown as green (new)
	default:
		statusIcon = "?"
		statusColor = m.theme.Subtext
	}

	bgStyle := lipgloss.NewStyle().Background(bg)
	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Background(bg).Bold(true)
	fileStyle := lipgloss.NewStyle().Foreground(m.theme.Foreground).Background(bg)

	isFileExpanded := m.expandState != nil && m.expandState.ExpandedFile == file.Path
	expandIndicator := " "
	if isFileExpanded {
		expandIndicator = "▼"
	} else if m.expandState != nil && m.expandState.FileIndex == fileIdx {
		expandIndicator = "▸"
	}

	indicatorStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext).Background(bg)

	// Build the line stats string: "+N -M" (colored green/red).
	addStyle := lipgloss.NewStyle().Foreground(m.theme.DiffAdd).Background(bg)
	delStyle := lipgloss.NewStyle().Foreground(m.theme.DiffRemove).Background(bg)
	var statsStr string
	statsWidth := 0
	if file.Additions > 0 || file.Deletions > 0 {
		addText := fmt.Sprintf("+%d", file.Additions)
		delText := fmt.Sprintf("-%d", file.Deletions)
		statsStr = bgStyle.Render(" ") + addStyle.Render(addText) + bgStyle.Render(" ") + delStyle.Render(delText)
		// Visual width: space(1) + addText + space(1) + delText
		statsWidth = 1 + len(addText) + 1 + len(delText)
	}

	// Truncate the file path to prevent overflow.
	// Prefix consumes: indent(6) + indicator(1) + space(1) + status(1) + space(1) = 10 chars
	// Stats are right-aligned and consume statsWidth chars.
	pathAvail := m.width - 10 - statsWidth
	if pathAvail < 8 {
		pathAvail = 8
	}
	displayPath := file.Path
	pathRunes := []rune(displayPath)
	if len(pathRunes) > pathAvail {
		displayPath = "…" + string(pathRunes[len(pathRunes)-pathAvail+1:])
	}

	line := bgStyle.Render(indent) +
		indicatorStyle.Render(expandIndicator) + bgStyle.Render(" ") +
		statusStyle.Render(statusIcon) + bgStyle.Render(" ") +
		fileStyle.Render(displayPath) +
		statsStr

	// Pad to full width with themed background.
	visWidth := lipgloss.Width(line)
	if visWidth < m.width {
		line = line + bgStyle.Width(m.width-visWidth).Render("")
	}

	// Outer wrap for full-width coverage.
	if isFileSelected {
		line = lipgloss.NewStyle().
			Background(m.theme.Selection).
			Bold(true).
			Width(m.width).
			Render(line)
	} else {
		line = lipgloss.NewStyle().
			Background(bg).
			Width(m.width).
			Render(line)
	}

	return line
}

// ---------------------------------------------------------------------------
// Scroll management
// ---------------------------------------------------------------------------

func (m *Model) ensureCursorVisible() {
	// Find the visual line of the cursor position.
	cursorVisLine := m.cursorVisualLine()

	if cursorVisLine < m.scrollOffset {
		m.scrollOffset = cursorVisLine
	}
	if cursorVisLine >= m.scrollOffset+m.height {
		m.scrollOffset = cursorVisLine - m.height + 1
	}
	m.clampScroll()
}

// ensureExpandedVisible scrolls only if the cursor has moved below the visible
// area (e.g. after new expanded content pushed it down). It never scrolls
// backwards, so it won't fight the user's current scroll position.
func (m *Model) ensureExpandedVisible() {
	cursorVisLine := m.cursorVisualLine()
	if cursorVisLine >= m.scrollOffset+m.height {
		m.scrollOffset = cursorVisLine - m.height + 1
	}
	m.clampScroll()
}

func (m *Model) clampScroll() {
	totalLines := m.totalVisualLines()
	maxScroll := totalLines - m.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m Model) cursorVisualLine() int {
	visLine := 0
	for i := 0; i < len(m.commits); i++ {
		if i == m.cursor {
			if m.isExpanded() && m.expandedIdx == m.cursor && m.expandState.FileIndex >= 0 {
				// Cursor is inside the expanded area.
				visLine++ // skip commit row
				visLine += m.metadataLineCount()
				// Add file lines up to the selected file.
				for fi := 0; fi < m.expandState.FileIndex; fi++ {
					visLine++ // file entry
					if m.expandState.Files[fi].Path == m.expandState.ExpandedFile && len(m.expandState.DiffLines) > 0 {
						visLine += len(m.expandState.DiffLines)
					}
				}
				return visLine
			}
			return visLine
		}
		visLine++ // commit line
		if i == m.expandedIdx && m.expandState != nil {
			visLine += m.expandedLineCount()
		}
	}
	return visLine
}

// expandedFileDiffEndVisLine returns the visual line number of the last diff
// line for the currently expanded file. Returns 0 if no diff is expanded.
func (m Model) expandedFileDiffEndVisLine() int {
	if !m.isExpanded() || m.expandState == nil || m.expandState.ExpandedFile == "" || len(m.expandState.DiffLines) == 0 {
		return 0
	}
	// Start from the cursor's visual line (which is on the file entry).
	visLine := m.cursorVisualLine()
	// The diff lines appear directly below the file entry.
	visLine += len(m.expandState.DiffLines)
	return visLine
}

func (m Model) totalVisualLines() int {
	// Each commit takes 1 line, plus expanded content if any.
	total := len(m.commits)
	if m.isExpanded() {
		total += m.expandedLineCount()
	}
	return total
}

func (m Model) expandedLineCount() int {
	if m.expandState == nil {
		return 0
	}
	count := m.metadataLineCount()
	for _, file := range m.expandState.Files {
		count++ // file entry line
		if file.Path == m.expandState.ExpandedFile && len(m.expandState.DiffLines) > 0 {
			count += len(m.expandState.DiffLines)
		}
	}
	return count
}

func (m Model) metadataLineCount() int {
	if m.expandState == nil || m.expandedIdx < 0 || m.expandedIdx >= len(m.commits) {
		return 0
	}
	commit := m.commits[m.expandedIdx]

	if commit.Hash == git.UncommittedHash {
		return 2 // header line + files header
	}

	count := 3 // hash+author, date+msg, files header
	if len(commit.Refs) > 0 {
		count++ // refs line
	}
	return count
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

func (m Model) SelectedCommit() *git.Commit {
	if m.cursor >= 0 && m.cursor < len(m.commits) {
		return m.commits[m.cursor]
	}
	return nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetCommits replaces the commit list and rebuilds the graph, while trying
// to preserve the cursor position and expanded state. If the previously
// selected commit still exists in the new list, the cursor is placed on it.
// If a commit was expanded and still exists, it stays expanded.
func (m *Model) SetCommits(commits []*git.Commit) {
	// Remember the currently selected commit hash so we can restore position.
	var prevHash string
	if m.cursor >= 0 && m.cursor < len(m.commits) {
		prevHash = m.commits[m.cursor].Hash
	}

	// Remember the expanded commit hash (if any) so we can preserve it.
	var expandedHash string
	if m.expandedIdx >= 0 && m.expandedIdx < len(m.commits) {
		expandedHash = m.commits[m.expandedIdx].Hash
	}

	// Preserve scroll offset — we'll only call ensureCursorVisible if
	// cursor/expand state actually changed position.
	prevScroll := m.scrollOffset

	m.commits = commits
	m.renderer.InitGraph(commits)

	// Try to find the previously selected commit in the new list.
	cursorPreserved := false
	newCursor := -1
	if prevHash != "" {
		for i, c := range commits {
			if c.Hash == prevHash {
				newCursor = i
				break
			}
		}
	}
	if newCursor >= 0 {
		cursorPreserved = (newCursor == m.cursor)
		m.cursor = newCursor
	} else if m.cursor >= len(commits) {
		m.cursor = len(commits) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
	m.lastCursor = m.cursor

	// Try to preserve the expanded commit.
	expandPreserved := false
	if expandedHash != "" {
		newExpandedIdx := -1
		for i, c := range commits {
			if c.Hash == expandedHash {
				newExpandedIdx = i
				break
			}
		}
		if newExpandedIdx >= 0 {
			expandPreserved = (newExpandedIdx == m.expandedIdx)
			m.expandedIdx = newExpandedIdx
			// expandState stays as-is (files/diff still valid for same hash)
		} else {
			// Expanded commit no longer exists — collapse.
			m.expandedIdx = -1
			m.expandState = nil
		}
	}

	// If both cursor and expand state are in the same positions, preserve the
	// user's scroll offset instead of snapping. This prevents the file watcher
	// reload from fighting the user's scroll position while viewing a diff.
	if cursorPreserved && expandPreserved {
		m.scrollOffset = prevScroll
		m.clampScroll()
	} else {
		m.ensureCursorVisible()
	}
}

func (m Model) MaxLanes() int {
	return m.renderer.MaxLanes()
}

func (m Model) Index() int {
	return m.cursor
}

func (m Model) IsExpanded() bool {
	return m.isExpanded()
}

func (m Model) ExpandedIdx() int {
	return m.expandedIdx
}

func (m *Model) ExpandState() *ExpandState {
	return m.expandState
}
