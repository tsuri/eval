package git

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func GetRepo(workspace string) (*git.Repository, error) {
	repo, err := git.PlainOpen(workspace)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func GetHead(workspace string) (branch string, commit string, err error) {
	repo, err := GetRepo(workspace)
	if err != nil {
		return "", "", err
	}
	ref, err := repo.Head()
	if err != nil {
		return "", "", err
	}
	return ref.Name().String(), ref.Name().String(), nil
}

func GetWorktree(workspace string) (*git.Worktree, error) {
	repo, err := GetRepo(workspace)
	if err != nil {
		return nil, err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	return worktree, nil
}

func WorkspaceStatus(workspace string) (git.Status, error) {
	worktree, err := GetWorktree(workspace)
	if err != nil {
		return nil, err
	}
	return worktree.Status()
}

type Commit struct {
	Branch string
	Hash   string
}

func GetTags(workspace string) (map[string]Commit, error) {
	tags := make(map[string]Commit)

	repo, err := GetRepo(workspace)
	if err != nil {
		return nil, err
	}
	tagIter, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	tagIter.ForEach(func(t *plumbing.Reference) error {
		tag := strings.Split(t.Name().String(), "/")
		tags[tag[len(tag)-1]] = Commit{
			// TODO I don't know how to do `git branch -a --contains commit`
			// for demo we assume tags are on 'main'
			Branch: "main",
			Hash:   t.Hash().String(),
		}
		fmt.Println(t)
		return nil
	})
	return tags, nil
}
