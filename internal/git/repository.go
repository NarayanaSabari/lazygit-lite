package git

import (
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	Status string // "A" added, "M" modified, "D" deleted, "R" renamed, "?" untracked
	Path   string
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

func (r *Repository) GetCommits(limit int) ([]*Commit, error) {
	ref, err := r.repo.Head()
	if err != nil {
		return nil, err
	}

	refMap := r.buildRefMap()

	iter, err := r.repo.Log(&git.LogOptions{
		From: ref.Hash(),
		All:  true,
	})
	if err != nil {
		return nil, err
	}

	commits := make([]*Commit, 0, limit)
	count := 0

	err = iter.ForEach(func(c *object.Commit) error {
		if count >= limit {
			return nil
		}

		parents := make([]string, len(c.ParentHashes))
		for i, p := range c.ParentHashes {
			parents[i] = p.String()
		}

		subject := c.Message
		if idx := len(c.Message); idx > 0 {
			for i, ch := range c.Message {
				if ch == '\n' {
					subject = c.Message[:i]
					break
				}
			}
		}

		refs := refMap[c.Hash.String()]

		commits = append(commits, &Commit{
			Hash:      c.Hash.String(),
			ShortHash: c.Hash.String()[:7],
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Date:      c.Author.When,
			Message:   c.Message,
			Subject:   subject,
			Parents:   parents,
			Refs:      refs,
		})

		count++
		return nil
	})

	if err != nil {
		return nil, err
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
