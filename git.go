package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Git interface {
	// GetUpdateRepo either clones the specified repository if it hasn't been cloned yet or simply
	// fetches the latest changes for it. Returns the Repo in any case.
	GetUpdatedRepo(url, repoOwner, repoName string) (Repo, error)
}

type Repo interface {
	Fetch() error
	// Runs `git rebase --interactive --autosquash` for the given refs and automatically saves and closes
	// the editor for interactive rebase.
	RebaseAutosquash(upstreamRef, branchRef string) error
	ForcePushHeadTo(remoteRef string) error
	GetHeadSHA() (string, error)
}

type git struct {
	sync.Mutex
	basePath string
	repos    map[string]repo
}

// NewGit creates a new Git implementation which will hold all its repos in the specified base path
func NewGit(basePath string) Git {
	return git{
		basePath: basePath,
		repos:    make(map[string]repo),
	}
}

func (g git) repo(path string) Repo {
	existingRepo, exists := g.repos[path]
	if !exists {
		newRepo := repo{path: path}
		g.repos[path] = newRepo
		return newRepo
	}
	return existingRepo
}

func (g git) clone(url, localPath string) (Repo, error) {
	if err := exec.Command("git", "clone", url, localPath).Run(); err != nil {
		return repo{}, fmt.Errorf("failed to clone: %v", err)
	}
	return g.repo(localPath), nil
}

func (g git) GetUpdatedRepo(url, repoOwner, repoName string) (Repo, error) {
	g.Lock()
	defer g.Unlock()

	localPath := filepath.Join(g.basePath, repoOwner, repoName)
	exists, err := exists(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if the repo exists locally: %v", err)
	}
	if !exists {
		log.Printf("Cloning %s into %s\n", url, localPath)
		return g.clone(url, localPath)
	}

	log.Printf("Fetching latest changes for %s\n", url)
	repo := g.repo(localPath)
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
	sync.Mutex
	path string
}

func (r repo) RebaseAutosquash(upstreamRef, branchRef string) error {
	r.Lock()
	defer r.Unlock()

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
	r.Lock()
	defer r.Unlock()

	if err := exec.Command("git", "-C", r.path, "fetch").Run(); err != nil {
		return fmt.Errorf("failed to fetch: %v", err)
	}
	return nil
}

func (r repo) ForcePushHeadTo(remoteRef string) error {
	r.Lock()
	defer r.Unlock()

	if err := exec.Command("git", "-C", r.path, "push", "--force", "origin", "@:"+remoteRef).Run(); err != nil {
		return fmt.Errorf("failed to force push to remote: %v", err)
	}
	return nil
}

func (r repo) GetHeadSHA() (string, error) {
	r.Lock()
	defer r.Unlock()

	output, err := exec.Command("git", "-C", r.path, "rev-parse", "@").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get the current HEAD's SHA: %v", err)
	}
	headSHA := strings.TrimSpace(string(output[:]))
	return headSHA, nil
}
