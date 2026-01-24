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

		commits = append(commits, &Commit{
			Hash:      c.Hash.String(),
			ShortHash: c.Hash.String()[:7],
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Date:      c.Author.When,
			Message:   c.Message,
			Subject:   subject,
			Parents:   parents,
		})

		count++
		return nil
	})

	if err != nil {
		return nil, err
	}

	return commits, nil
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

func (r *Repository) GetCommitDetails(hash string) (*Commit, error) {
	h := plumbing.NewHash(hash)
	c, err := r.repo.CommitObject(h)
	if err != nil {
		return nil, err
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

	return &Commit{
		Hash:      c.Hash.String(),
		ShortHash: c.Hash.String()[:7],
		Author:    c.Author.Name,
		Email:     c.Author.Email,
		Date:      c.Author.When,
		Message:   c.Message,
		Subject:   subject,
		Parents:   parents,
	}, nil
}
