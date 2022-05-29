package git

import (
	"github.com/go-git/go-git/v5"
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
