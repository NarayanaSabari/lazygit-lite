package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/lazygit-lite/internal/config"
	"github.com/yourusername/lazygit-lite/internal/git"
	"github.com/yourusername/lazygit-lite/internal/ui/components/actionbar"
	"github.com/yourusername/lazygit-lite/internal/ui/components/graph"
	"github.com/yourusername/lazygit-lite/internal/ui/components/modals"
	"github.com/yourusername/lazygit-lite/internal/ui/keys"
	"github.com/yourusername/lazygit-lite/internal/ui/layout"
	"github.com/yourusername/lazygit-lite/internal/ui/styles"
)

type Model struct {
	config *config.Config
	repo   *git.Repository
	styles *styles.Styles
	layout *layout.Layout
	keyMap keys.KeyMap

	graphPanel graph.Model
	actionBar  actionbar.Model

	commitModal modals.CommitModal
	helpModal   modals.HelpModal
	branchModal modals.BranchModal

	width  int
	height int
	ready  bool
}

func New(cfg *config.Config, repoPath string) (*Model, error) {
	repo, err := git.OpenRepository(repoPath)
	if err != nil {
		return nil, err
	}

	theme := styles.GetTheme(cfg.UI.Theme)
	st := styles.NewStyles(theme)

	return &Model{
		config:      cfg,
		repo:        repo,
		styles:      st,
		keyMap:      keys.DefaultKeyMap(),
		commitModal: modals.NewCommitModal(st),
		helpModal:   modals.NewHelpModal(st),
		branchModal: modals.NewBranchModal(st),
	}, nil
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.commitModal.Init(),
		m.loadCommitsCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		if m.commitModal.IsVisible() {
			return m.handleCommitModal(msg)
		}

		if m.branchModal.IsVisible() {
			return m.handleBranchModal(msg)
		}

		if m.helpModal.IsVisible() {
			if keys.MatchesKey(msg, m.keyMap.Help) || msg.String() == "esc" {
				m.helpModal.Toggle()
				m.recalcGraphSize()
				return m, nil
			}
			return m, nil
		}

		return m.handleKey(msg)

	case commitsLoadedMsg:
		return m.handleCommitsLoaded(msg)

	case operationResultMsg:
		return m.handleOperationResult(msg)

	case clearMessageMsg:
		m.actionBar.ClearMessage()
		return m, nil

	case branchesLoadedMsg:
		return m.handleBranchesLoaded(msg)

	case graph.SelectionChangedMsg:
		// No auto-load needed — diffs are shown inline on expand.
		return m, nil

	case graph.FilesLoadedMsg, graph.FileDiffLoadedMsg:
		// Check for errors and display in action bar.
		switch typedMsg := msg.(type) {
		case graph.FilesLoadedMsg:
			if typedMsg.Err != nil {
				m.actionBar.SetMessage("Failed to load files: " + typedMsg.Err.Error())
				return m, m.clearMessageAfter(3 * time.Second)
			}
		case graph.FileDiffLoadedMsg:
			if typedMsg.Err != nil {
				m.actionBar.SetMessage("Failed to load diff: " + typedMsg.Err.Error())
				return m, m.clearMessageAfter(3 * time.Second)
			}
		}
		// Forward to graph panel.
		var cmd tea.Cmd
		m.graphPanel, cmd = m.graphPanel.Update(msg)
		return m, cmd
	}

	if m.ready {
		var cmd tea.Cmd
		m.graphPanel, cmd = m.graphPanel.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	mainPanel := m.graphPanel.View()
	actionBarView := m.actionBar.View()

	// Determine if any inline bottom panel is active.
	var extraPanel string
	if m.commitModal.IsVisible() {
		extraPanel = m.commitModal.View()
	} else if m.branchModal.IsVisible() {
		extraPanel = m.branchModal.View()
	} else if m.helpModal.IsVisible() {
		extraPanel = m.helpModal.View()
	}

	return m.layout.RenderWithExtra(mainPanel, extraPanel, actionBarView)
}

func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	if !m.ready {
		m.layout = layout.New(m.width, m.height, m.config.Layout.SplitRatio,
			m.styles.Theme.Background, m.styles.Theme.Border, m.styles.Theme.Foreground)
		contentW, contentH := m.layout.Calculate()

		commits, err := m.repo.GetCommits(m.config.Performance.MaxCommits)
		if err == nil {
			commits = m.prependUncommitted(commits)
		}
		m.graphPanel = graph.New(commits, m.styles.Theme, contentW, contentH)
		m.actionBar = actionbar.New(m.styles, m.width)

		// Set current branch on the action bar.
		m.updateBranchInfo()

		// Size the modals to terminal dimensions.
		m.commitModal.SetSize(m.width, m.height)
		m.helpModal.SetSize(m.width, m.height)
		m.branchModal.SetSize(m.width, m.height)

		m.ready = true
	} else {
		m.layout.SetSize(m.width, m.height)
		contentW, contentH := m.layout.Calculate()

		m.graphPanel.SetSize(contentW, contentH)
		m.actionBar.SetWidth(m.width)
		m.commitModal.SetSize(m.width, m.height)
		m.helpModal.SetSize(m.width, m.height)
		m.branchModal.SetSize(m.width, m.height)
	}

	return m, nil
}

// recalcGraphSize recalculates the graph panel dimensions based on the current
// visibility of inline panels (commit input, help). Call this whenever a modal
// is toggled so the graph's scroll and rendering use the correct height.
func (m *Model) recalcGraphSize() {
	if m.layout == nil {
		return
	}
	extra := m.commitModal.Height() + m.helpModal.Height() + m.branchModal.Height()

	// If the modal(s) would leave the graph panel with fewer than 3 rows,
	// auto-close the help modal (the largest one) to reclaim space.
	_, testH := m.layout.CalculateWithExtra(extra)
	if testH <= 3 && m.helpModal.IsVisible() {
		m.helpModal.Toggle()
		extra = m.commitModal.Height() + m.helpModal.Height() + m.branchModal.Height()
	}

	contentW, contentH := m.layout.CalculateWithExtra(extra)
	m.graphPanel.SetSize(contentW, contentH)
}

func (m *Model) updateBranchInfo() {
	branches, err := m.repo.GetBranches()
	if err != nil {
		return
	}
	for _, b := range branches {
		if b.IsHead {
			m.actionBar.SetBranch(b.Name)
			return
		}
	}
}

// prependUncommitted checks for working tree changes and prepends a synthetic
// "Uncommitted changes" entry to the commit list so it appears at the top.
func (m Model) prependUncommitted(commits []*git.Commit) []*git.Commit {
	if !m.repo.HasWorkingTreeChanges() {
		return commits
	}

	parentHash := ""
	if len(commits) > 0 {
		parentHash = commits[0].Hash
	}

	uncommitted := &git.Commit{
		Hash:      git.UncommittedHash,
		ShortHash: git.UncommittedShortHash,
		Author:    "You",
		Email:     "",
		Date:      time.Now(),
		Message:   "Uncommitted changes",
		Subject:   "Uncommitted changes",
		Parents:   []string{parentHash},
		Refs:      nil,
	}

	return append([]*git.Commit{uncommitted}, commits...)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if keys.MatchesKey(msg, m.keyMap.Quit) {
		return m, tea.Quit
	}

	if keys.MatchesKey(msg, m.keyMap.Help) {
		m.helpModal.Toggle()
		m.recalcGraphSize()
		return m, nil
	}

	if keys.MatchesKey(msg, m.keyMap.Commit) {
		m.commitModal.Show()
		m.recalcGraphSize()
		return m, nil
	}

	if keys.MatchesKey(msg, m.keyMap.Push) {
		m.actionBar.SetMessage("Pushing...")
		return m, m.pushCmd()
	}

	if keys.MatchesKey(msg, m.keyMap.Pull) {
		m.actionBar.SetMessage("Pulling...")
		return m, m.pullCmd()
	}

	if keys.MatchesKey(msg, m.keyMap.Fetch) {
		m.actionBar.SetMessage("Fetching...")
		return m, m.fetchCmd()
	}

	if keys.MatchesKey(msg, m.keyMap.Branch) {
		return m, m.showBranchPickerCmd()
	}

	// Enter toggles expand on the selected commit / file.
	if keys.MatchesKey(msg, m.keyMap.Enter) {
		cmd := m.graphPanel.ToggleExpand(m.repo)
		return m, cmd
	}

	// Esc collapses any expanded commit.
	if msg.String() == "esc" {
		if m.graphPanel.IsExpanded() {
			m.graphPanel.ToggleExpand(m.repo) // will collapse since cursor is on expanded
			return m, nil
		}
		return m, nil
	}

	if keys.MatchesKey(msg, m.keyMap.CopyHash) {
		return m.handleCopyHash()
	}

	if keys.MatchesKey(msg, m.keyMap.CopyMessage) {
		return m.handleCopyMessage()
	}

	if keys.MatchesKey(msg, m.keyMap.CopyDiff) {
		return m.handleCopyDiff()
	}

	// All other keys (j/k/g/G/ctrl+d/ctrl+u) go to the graph panel.
	var cmd tea.Cmd
	m.graphPanel, cmd = m.graphPanel.Update(msg)
	return m, cmd
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if !m.ready || m.commitModal.IsVisible() || m.helpModal.IsVisible() {
		return m, nil
	}

	var cmd tea.Cmd
	m.graphPanel, cmd = m.graphPanel.Update(msg)
	return m, cmd
}

func (m Model) handleCommitModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.commitModal.Hide()
		m.recalcGraphSize()
		return m, nil
	}

	if msg.String() == "enter" {
		message := m.commitModal.Value()
		if strings.TrimSpace(message) == "" {
			// Don't commit with an empty message.
			return m, nil
		}
		m.commitModal.Hide()
		m.recalcGraphSize()
		m.actionBar.SetMessage("Committing...")
		return m, m.commitCmd(message)
	}

	var cmd tea.Cmd
	m.commitModal, cmd = m.commitModal.Update(msg)
	return m, cmd
}

func (m Model) handleCopyHash() (tea.Model, tea.Cmd) {
	commit := m.graphPanel.SelectedCommit()
	if commit == nil {
		return m, nil
	}
	if commit.Hash == git.UncommittedHash {
		m.actionBar.SetMessage("Cannot copy hash for uncommitted changes")
		return m, m.clearMessageAfter(3 * time.Second)
	}
	clipboard.WriteAll(commit.Hash)
	m.actionBar.SetMessage("Copied hash: " + commit.ShortHash)
	return m, m.clearMessageAfter(3 * time.Second)
}

func (m Model) handleCopyMessage() (tea.Model, tea.Cmd) {
	commit := m.graphPanel.SelectedCommit()
	if commit == nil {
		return m, nil
	}
	if commit.Hash == git.UncommittedHash {
		m.actionBar.SetMessage("Cannot copy message for uncommitted changes")
		return m, m.clearMessageAfter(3 * time.Second)
	}
	clipboard.WriteAll(commit.Message)
	m.actionBar.SetMessage("Copied commit message")
	return m, m.clearMessageAfter(3 * time.Second)
}

func (m Model) handleCopyDiff() (tea.Model, tea.Cmd) {
	commit := m.graphPanel.SelectedCommit()
	if commit == nil {
		return m, nil
	}
	if commit.Hash == git.UncommittedHash {
		m.actionBar.SetMessage("Cannot copy full diff for uncommitted changes")
		return m, m.clearMessageAfter(3 * time.Second)
	}
	diff, err := m.repo.GetDiff(commit.Hash)
	if err != nil {
		m.actionBar.SetMessage("Failed to get diff: " + err.Error())
		return m, m.clearMessageAfter(3 * time.Second)
	}
	clipboard.WriteAll(diff)
	m.actionBar.SetMessage("Copied diff")
	return m, m.clearMessageAfter(3 * time.Second)
}

func (m Model) handleBranchModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b":
		m.branchModal.Hide()
		m.recalcGraphSize()
		return m, nil
	case "j", "down":
		m.branchModal.MoveDown()
		return m, nil
	case "k", "up":
		m.branchModal.MoveUp()
		return m, nil
	case "enter":
		branch := m.branchModal.SelectedBranch()
		if branch == nil || branch.IsCurrent {
			m.branchModal.Hide()
			m.recalcGraphSize()
			return m, nil
		}
		branchName := branch.Name
		m.branchModal.Hide()
		m.recalcGraphSize()
		m.actionBar.SetMessage("Checking out " + branchName + "...")
		return m, m.checkoutCmd(branchName)
	}
	return m, nil
}

func (m Model) handleBranchesLoaded(msg branchesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.branches == nil || len(msg.branches) == 0 {
		m.actionBar.SetMessage("No branches found")
		return m, m.clearMessageAfter(3 * time.Second)
	}
	m.branchModal.Show(msg.branches)
	m.recalcGraphSize()
	return m, nil
}

func (m Model) showBranchPickerCmd() tea.Cmd {
	return func() tea.Msg {
		branches, err := m.repo.GetBranches()
		if err != nil {
			return operationResultMsg{operation: "branch list", err: err}
		}
		return branchesLoadedMsg{branches: branches}
	}
}

func (m Model) checkoutCmd(branch string) tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Checkout(branch)
		return operationResultMsg{operation: "checkout", err: err}
	}
}

type commitsLoadedMsg struct {
	commits []*git.Commit
	err     error
}

// operationResultMsg is sent when a git operation (push/pull/fetch/commit) completes.
type operationResultMsg struct {
	operation string // "push", "pull", "fetch", "commit"
	err       error
}

// clearMessageMsg is sent after a delay to clear the action bar message.
type clearMessageMsg struct{}

// branchesLoadedMsg is sent when the branch list has been loaded for the picker.
type branchesLoadedMsg struct {
	branches []*git.Branch
}

func (m Model) loadCommitsCmd() tea.Cmd {
	return func() tea.Msg {
		commits, err := m.repo.GetCommits(m.config.Performance.MaxCommits)
		if err != nil {
			return commitsLoadedMsg{err: err}
		}
		commits = m.prependUncommitted(commits)
		return commitsLoadedMsg{commits: commits}
	}
}

func (m Model) handleCommitsLoaded(msg commitsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.actionBar.SetMessage("Failed to load commits: " + msg.err.Error())
		return m, m.clearMessageAfter(3 * time.Second)
	}
	if m.ready && msg.commits != nil {
		contentW, contentH := m.layout.Calculate()
		m.graphPanel = graph.New(msg.commits, m.styles.Theme, contentW, contentH)
	}
	return m, nil
}

func (m Model) handleOperationResult(msg operationResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.actionBar.SetMessage(fmt.Sprintf("%s failed: %s", msg.operation, msg.err.Error()))
	} else {
		switch msg.operation {
		case "push":
			m.actionBar.SetMessage("Changes pushed successfully")
		case "pull":
			m.actionBar.SetMessage("Changes pulled successfully")
		case "fetch":
			m.actionBar.SetMessage("Fetch completed successfully")
		case "commit":
			m.actionBar.SetMessage("Commit created successfully")
		case "checkout":
			m.actionBar.SetMessage("Checked out successfully")
			m.updateBranchInfo()
		default:
			m.actionBar.SetMessage(msg.operation + " completed")
		}
	}

	// Reload commits after any git operation (they may have changed).
	return m, tea.Batch(
		m.clearMessageAfter(3*time.Second),
		m.loadCommitsCmd(),
	)
}

func (m Model) clearMessageAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}

func (m Model) pushCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Push()
		return operationResultMsg{operation: "push", err: err}
	}
}

func (m Model) pullCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Pull(m.config.Git.PullRebase)
		return operationResultMsg{operation: "pull", err: err}
	}
}

func (m Model) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Fetch()
		return operationResultMsg{operation: "fetch", err: err}
	}
}

func (m Model) commitCmd(message string) tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Commit(message)
		return operationResultMsg{operation: "commit", err: err}
	}
}
