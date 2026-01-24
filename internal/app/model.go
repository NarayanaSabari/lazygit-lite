package app

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/lazygit-lite/internal/config"
	"github.com/yourusername/lazygit-lite/internal/git"
	"github.com/yourusername/lazygit-lite/internal/ui/components/actionbar"
	"github.com/yourusername/lazygit-lite/internal/ui/components/commitinfo"
	"github.com/yourusername/lazygit-lite/internal/ui/components/details"
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

	graphPanel      graph.Model
	commitInfoPanel commitinfo.Model
	detailsPanel    details.Model
	actionBar       actionbar.Model

	commitModal modals.CommitModal
	helpModal   modals.HelpModal

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

		if m.helpModal.IsVisible() {
			if keys.MatchesKey(msg, m.keyMap.Help) || msg.String() == "esc" {
				m.helpModal.Toggle()
				return m, nil
			}
			return m, nil
		}

		return m.handleKey(msg)

	case commitsLoadedMsg:
		return m.handleCommitsLoaded(msg)

	case diffLoadedMsg:
		m.commitInfoPanel.SetCommit(msg.commit)
		m.detailsPanel.SetDiff(msg.diff)
		return m, nil
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

	if m.helpModal.IsVisible() {
		return m.helpModal.View()
	}

	if m.commitModal.IsVisible() {
		return m.commitModal.View()
	}

	leftPanel := m.styles.Panel.Render(m.graphPanel.View())
	topRightPanel := m.styles.Panel.Render(m.commitInfoPanel.View())
	bottomRightPanel := m.styles.Panel.Render(m.detailsPanel.View())
	actionBarView := m.actionBar.View()

	return m.layout.Render(leftPanel, topRightPanel, bottomRightPanel, actionBarView)
}

func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	if !m.ready {
		m.layout = layout.New(m.width, m.height, m.config.Layout.SplitRatio)
		leftW, leftH, rightW, rightTopH, rightBottomH := m.layout.Calculate()

		commits, _ := m.repo.GetCommits(m.config.Performance.MaxCommits)
		m.graphPanel = graph.New(commits, m.styles.Theme, leftW, leftH)
		m.commitInfoPanel = commitinfo.New(m.styles, rightW, rightTopH)
		m.detailsPanel = details.New(m.styles, rightW, rightBottomH)
		m.actionBar = actionbar.New(m.styles, m.width)

		m.ready = true

		if len(commits) > 0 {
			return m, m.loadDiffCmd(commits[0])
		}
	} else {
		m.layout.SetSize(m.width, m.height)
		leftW, leftH, rightW, rightTopH, rightBottomH := m.layout.Calculate()

		m.graphPanel.SetSize(leftW, leftH)
		m.commitInfoPanel.SetSize(rightW, rightTopH)
		m.detailsPanel.SetSize(rightW, rightBottomH)
		m.actionBar.SetWidth(m.width)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if keys.MatchesKey(msg, m.keyMap.Quit) {
		return m, tea.Quit
	}

	if keys.MatchesKey(msg, m.keyMap.Help) {
		m.helpModal.Toggle()
		return m, nil
	}

	if keys.MatchesKey(msg, m.keyMap.Commit) {
		m.commitModal.Show()
		return m, nil
	}

	if keys.MatchesKey(msg, m.keyMap.Push) {
		return m, m.pushCmd()
	}

	if keys.MatchesKey(msg, m.keyMap.Pull) {
		return m, m.pullCmd()
	}

	if keys.MatchesKey(msg, m.keyMap.Fetch) {
		return m, m.fetchCmd()
	}

	if keys.MatchesKey(msg, m.keyMap.Enter) {
		return m.handleCommitSelection()
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

	var cmd tea.Cmd
	m.graphPanel, cmd = m.graphPanel.Update(msg)
	return m, cmd
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if !m.ready || m.commitModal.IsVisible() || m.helpModal.IsVisible() {
		return m, nil
	}

	var cmds []tea.Cmd

	leftW, _, rightW, _, _ := m.layout.Calculate()

	if msg.X < leftW {
		var cmd tea.Cmd
		m.graphPanel, cmd = m.graphPanel.Update(msg)
		cmds = append(cmds, cmd)

		if msg.Type == tea.MouseLeft && msg.Action == tea.MouseActionRelease {
			cmd = m.loadDiffForSelected()
			cmds = append(cmds, cmd)
		}
	} else if msg.X >= leftW+1 && msg.X < leftW+1+rightW {
		adjustedMsg := msg
		adjustedMsg.X = msg.X - leftW - 1

		var cmd tea.Cmd
		m.detailsPanel, cmd = m.detailsPanel.Update(adjustedMsg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) loadDiffForSelected() tea.Cmd {
	commit := m.graphPanel.SelectedCommit()
	if commit != nil {
		return m.loadDiffCmd(commit)
	}
	return nil
}

func (m Model) handleCommitModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.commitModal.Hide()
		return m, nil
	}

	if msg.String() == "ctrl+enter" {
		message := m.commitModal.Value()
		m.commitModal.Hide()
		return m, m.commitCmd(message)
	}

	var cmd tea.Cmd
	m.commitModal, cmd = m.commitModal.Update(msg)
	return m, cmd
}

func (m Model) handleCommitSelection() (tea.Model, tea.Cmd) {
	commit := m.graphPanel.SelectedCommit()
	if commit != nil {
		return m, m.loadDiffCmd(commit)
	}
	return m, nil
}

func (m Model) handleCopyHash() (tea.Model, tea.Cmd) {
	commit := m.graphPanel.SelectedCommit()
	if commit != nil {
		clipboard.WriteAll(commit.Hash)
	}
	return m, nil
}

func (m Model) handleCopyMessage() (tea.Model, tea.Cmd) {
	commit := m.graphPanel.SelectedCommit()
	if commit != nil {
		clipboard.WriteAll(commit.Message)
	}
	return m, nil
}

func (m Model) handleCopyDiff() (tea.Model, tea.Cmd) {
	commit := m.graphPanel.SelectedCommit()
	if commit != nil {
		diff, _ := m.repo.GetDiff(commit.Hash)
		clipboard.WriteAll(diff)
	}
	return m, nil
}

type commitsLoadedMsg struct {
	commits []*git.Commit
}

type diffLoadedMsg struct {
	commit *git.Commit
	diff   string
}

func (m Model) loadCommitsCmd() tea.Cmd {
	return func() tea.Msg {
		commits, _ := m.repo.GetCommits(m.config.Performance.MaxCommits)
		return commitsLoadedMsg{commits: commits}
	}
}

func (m Model) loadDiffCmd(commit *git.Commit) tea.Cmd {
	return func() tea.Msg {
		diff, _ := m.repo.GetDiff(commit.Hash)
		return diffLoadedMsg{commit: commit, diff: diff}
	}
}

func (m Model) handleCommitsLoaded(msg commitsLoadedMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) pushCmd() tea.Cmd {
	return func() tea.Msg {
		m.repo.Push()
		return nil
	}
}

func (m Model) pullCmd() tea.Cmd {
	return func() tea.Msg {
		m.repo.Pull(m.config.Git.PullRebase)
		return nil
	}
}

func (m Model) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		m.repo.Fetch()
		return nil
	}
}

func (m Model) commitCmd(message string) tea.Cmd {
	return func() tea.Msg {
		m.repo.Commit(message)
		return nil
	}
}
