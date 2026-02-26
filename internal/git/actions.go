package git

import (
	"os/exec"
	"strconv"
	"strings"
)

func (r *Repository) Push() error {
	cmd := exec.Command("git", "push")
	cmd.Dir = r.path
	return cmd.Run()
}

func (r *Repository) Pull(rebase bool) error {
	args := []string{"pull"}
	if rebase {
		args = append(args, "--rebase")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = r.path
	return cmd.Run()
}

func (r *Repository) Fetch() error {
	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = r.path
	return cmd.Run()
}

func (r *Repository) Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = r.path
	return cmd.Run()
}

func (r *Repository) Commit(message string) error {
	// Stage all changes (tracked + untracked) before committing, since
	// there is no staging UI yet.
	stageCmd := exec.Command("git", "add", "-A")
	stageCmd.Dir = r.path
	if err := stageCmd.Run(); err != nil {
		return err
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = r.path
	return cmd.Run()
}

func (r *Repository) GetDiff(hash string) (string, error) {
	cmd := exec.Command("git", "show", "--no-color", "--format=", hash)
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (r *Repository) GetChangedFiles(hash string) ([]ChangedFile, error) {
	// Get file status (A/M/D/R) via --name-status.
	statusCmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-status", "-r", hash)
	statusCmd.Dir = r.path
	statusOut, err := statusCmd.Output()
	if err != nil {
		return nil, err
	}

	// Get per-file line additions/deletions via --numstat.
	numstatCmd := exec.Command("git", "diff-tree", "--no-commit-id", "--numstat", "-r", hash)
	numstatCmd.Dir = r.path
	numstatOut, _ := numstatCmd.Output() // best-effort; ignore errors

	// Build a map of path -> (additions, deletions) from numstat output.
	stats := make(map[string][2]int)
	for _, line := range strings.Split(strings.TrimSpace(string(numstatOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) == 3 {
			adds, _ := strconv.Atoi(parts[0]) // "-" for binary -> 0
			dels, _ := strconv.Atoi(parts[1])
			stats[parts[2]] = [2]int{adds, dels}
		}
	}

	var files []ChangedFile
	for _, line := range strings.Split(strings.TrimSpace(string(statusOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			s := stats[parts[1]]
			files = append(files, ChangedFile{
				Status:    parts[0],
				Path:      parts[1],
				Additions: s[0],
				Deletions: s[1],
			})
		}
	}
	return files, nil
}

func (r *Repository) GetFileDiff(hash, filePath string) (string, error) {
	cmd := exec.Command("git", "show", "--no-color", "--format=", hash, "--", filePath)
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetWorkingTreeFiles returns all staged and unstaged changed files in the
// working tree using `git status --porcelain`, with per-file line stats
// from `git diff --numstat HEAD`.
func (r *Repository) GetWorkingTreeFiles() ([]ChangedFile, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Get line stats for all working tree changes vs HEAD.
	numstatCmd := exec.Command("git", "diff", "--numstat", "HEAD")
	numstatCmd.Dir = r.path
	numstatOut, _ := numstatCmd.Output() // best-effort

	stats := make(map[string][2]int)
	for _, line := range strings.Split(strings.TrimSpace(string(numstatOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) == 3 {
			adds, _ := strconv.Atoi(parts[0])
			dels, _ := strconv.Atoi(parts[1])
			stats[parts[2]] = [2]int{adds, dels}
		}
	}

	var files []ChangedFile
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) < 4 {
			continue
		}
		// Porcelain format: XY <path>
		// X = index status, Y = worktree status
		xy := line[:2]
		path := line[3:]

		status := "M" // default
		switch {
		case xy[0] == '?' || xy[1] == '?':
			status = "?"
		case xy[0] == 'A' || xy[1] == 'A':
			status = "A"
		case xy[0] == 'D' || xy[1] == 'D':
			status = "D"
		case xy[0] == 'R' || xy[1] == 'R':
			status = "R"
		case xy[0] == 'M' || xy[1] == 'M':
			status = "M"
		}

		s := stats[path]
		files = append(files, ChangedFile{
			Status:    status,
			Path:      path,
			Additions: s[0],
			Deletions: s[1],
		})
	}
	return files, nil
}

// GetWorkingTreeFileDiff returns the combined (staged + unstaged) diff for a
// single file in the working tree.
func (r *Repository) GetWorkingTreeFileDiff(filePath string) (string, error) {
	// Get unstaged changes.
	cmd := exec.Command("git", "diff", "--no-color", "--", filePath)
	cmd.Dir = r.path
	unstaged, _ := cmd.Output()

	// Get staged changes.
	cmd2 := exec.Command("git", "diff", "--cached", "--no-color", "--", filePath)
	cmd2.Dir = r.path
	staged, _ := cmd2.Output()

	// For untracked files, show the whole file as an add.
	if len(unstaged) == 0 && len(staged) == 0 {
		cmd3 := exec.Command("git", "diff", "--no-color", "--no-index", "/dev/null", filePath)
		cmd3.Dir = r.path
		untracked, _ := cmd3.Output()
		return string(untracked), nil
	}

	// Prefer staged if present, otherwise unstaged.
	if len(staged) > 0 {
		return string(staged), nil
	}
	return string(unstaged), nil
}

// HasWorkingTreeChanges returns true if there are any uncommitted changes.
func (r *Repository) HasWorkingTreeChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.path
	output, _ := cmd.Output()
	return len(strings.TrimSpace(string(output))) > 0
}
