package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type Repository struct {
	repo *git.Repository
	path string
}

type Commit struct {
	Hash      string
	ShortHash string
	Author    string
	Email     string
	Date      time.Time
	Message   string
	Subject   string
	Parents   []string
	Refs      []Ref
}

type Ref struct {
	Name     string
	RefType  RefType
	IsHead   bool
	IsRemote bool
}

type RefType int

const (
	RefTypeBranch RefType = iota
	RefTypeTag
)

// UncommittedHash is a sentinel hash used for the synthetic "Uncommitted changes"
// entry at the top of the commit list.
const UncommittedHash = "0000000000000000000000000000000000000000"

// UncommittedShortHash is the short hash displayed for uncommitted changes.
const UncommittedShortHash = "·······"

type ChangedFile struct {
	Status    string // "A" added, "M" modified, "D" deleted, "R" renamed, "?" untracked
	Path      string
	Additions int // lines added (0 for binary files)
	Deletions int // lines removed (0 for binary files)
}

type Branch struct {
	Name      string
	IsHead    bool
	IsCurrent bool
	Hash      string
}

func OpenRepository(path string) (*Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}

	return &Repository{
		repo: repo,
		path: path,
	}, nil
}

// Path returns the filesystem path of the repository root.
func (r *Repository) Path() string {
	return r.path
}

func (r *Repository) GetCommits(limit int) ([]*Commit, error) {
	refMap := r.buildRefMap()

	// Use git log shell command instead of go-git's Log, which fails to
	// return commits from all branches in proper topological order.
	// Delimiter \x00 (NUL) is safe — it cannot appear in commit metadata.
	format := "%H%x00%P%x00%an%x00%ae%x00%at%x00%s"
	args := []string{
		"-C", r.path,
		"log", "--all", "--topo-order",
		fmt.Sprintf("--format=%s", format),
		fmt.Sprintf("-%d", limit),
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	commits := make([]*Commit, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\x00", 6)
		if len(parts) < 6 {
			continue // malformed line
		}

		hash := parts[0]
		parentStr := parts[1]
		author := parts[2]
		email := parts[3]
		tsStr := parts[4]
		subject := parts[5]

		var parents []string
		if parentStr != "" {
			parents = strings.Split(parentStr, " ")
		}

		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			ts = 0
		}

		refs := refMap[hash]
		shortHash := hash
		if len(hash) >= 7 {
			shortHash = hash[:7]
		}

		commits = append(commits, &Commit{
			Hash:      hash,
			ShortHash: shortHash,
			Author:    author,
			Email:     email,
			Date:      time.Unix(ts, 0),
			Message:   subject,
			Subject:   subject,
			Parents:   parents,
			Refs:      refs,
		})
	}

	return commits, nil
}

func (r *Repository) buildRefMap() map[string][]Ref {
	refMap := make(map[string][]Ref)

	head, _ := r.repo.Head()
	headName := ""
	if head != nil {
		headName = head.Name().String()
	}

	refs, err := r.repo.References()
	if err != nil {
		return refMap
	}

	refs.ForEach(func(ref *plumbing.Reference) error {
		hash := ref.Hash().String()
		name := ref.Name()

		if name.IsBranch() {
			refMap[hash] = append(refMap[hash], Ref{
				Name:     name.Short(),
				RefType:  RefTypeBranch,
				IsHead:   name.String() == headName,
				IsRemote: false,
			})
		} else if name.IsRemote() {
			refMap[hash] = append(refMap[hash], Ref{
				Name:     name.Short(),
				RefType:  RefTypeBranch,
				IsHead:   false,
				IsRemote: true,
			})
		} else if name.IsTag() {
			refMap[hash] = append(refMap[hash], Ref{
				Name:     name.Short(),
				RefType:  RefTypeTag,
				IsHead:   false,
				IsRemote: false,
			})
		}
		return nil
	})

	return refMap
}

func (r *Repository) GetBranches() ([]*Branch, error) {
	branches := []*Branch{}

	head, err := r.repo.Head()
	if err != nil {
		return nil, err
	}

	refs, err := r.repo.References()
	if err != nil {
		return nil, err
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branchName := ref.Name().Short()
			isHead := ref.Name() == head.Name()

			branches = append(branches, &Branch{
				Name:      branchName,
				IsHead:    isHead,
				IsCurrent: isHead,
				Hash:      ref.Hash().String(),
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return branches, nil
}
