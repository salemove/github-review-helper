package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type Git interface {
	Repo(path string) Repo
	Clone(url, localPath string) (Repo, error)
	GetUpdatedRepo(url, repoOwner, repoName string) (Repo, error)
}

type Repo interface {
	Fetch() error
	RebaseAutosquash(upstreamRef, branchRef string) error
	ForcePushHeadTo(remoteRef string) error
}

type git struct {
	basePath string
}

func NewGit(basePath string) Git {
	return git{basePath}
}

func (g git) Repo(path string) Repo {
	return repo{path}
}

func (g git) Clone(url, localPath string) (Repo, error) {
	if err := exec.Command("git", "clone", url, localPath).Run(); err != nil {
		return repo{}, fmt.Errorf("failed to clone: %v", err)
	}
	return g.Repo(localPath), nil
}

func (g git) GetUpdatedRepo(url, repoOwner, repoName string) (Repo, error) {
	localPath := filepath.Join(g.basePath, repoOwner, repoName)
	exists, err := exists(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if the repo exists locally: %v", err)
	}
	if !exists {
		log.Printf("Cloning %s into %s\n", url, localPath)
		return g.Clone(url, localPath)
	}

	log.Printf("Fetching latest changes for %s\n", url)
	repo := g.Repo(localPath)
	err = repo.Fetch()
	return repo, err
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

type repo struct {
	path string
}

func (r repo) RebaseAutosquash(upstreamRef, branchRef string) error {
	// This makes the --interactive rebase not actually interactive
	if err := os.Setenv("GIT_SEQUENCE_EDITOR", "true"); err != nil {
		return fmt.Errorf("failed to change the env variable: %v", err)
	}
	defer os.Unsetenv("GIT_SEQUENCE_EDITOR")

	if err := exec.Command("git", "-C", r.path, "rebase", "--interactive", "--autosquash", upstreamRef, branchRef).Run(); err != nil {
		err = fmt.Errorf("failed to rebase: %v", err)
		log.Println(err, " Trying to clean up.")
		if cleanupErr := exec.Command("git", "-C", r.path, "rebase", "--abort").Run(); cleanupErr != nil {
			log.Println("Also failed to clean up after the failed rebase: ", cleanupErr)
		}
		return err
	}
	return nil
}

func (r repo) Fetch() error {
	if err := exec.Command("git", "-C", r.path, "fetch").Run(); err != nil {
		return fmt.Errorf("failed to fetch: %v", err)
	}
	return nil
}

func (r repo) ForcePushHeadTo(remoteRef string) error {
	if err := exec.Command("git", "-C", r.path, "push", "--force", "origin", "@:"+remoteRef).Run(); err != nil {
		return fmt.Errorf("failed to force push to remote: %v", err)
	}
	return nil
}
