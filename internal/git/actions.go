package git

import (
	"os/exec"
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
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = r.path
	return cmd.Run()
}

func (r *Repository) GetDiff(hash string) (string, error) {
	cmd := exec.Command("git", "show", "--color=always", hash)
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (r *Repository) GetStatus() (string, error) {
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
