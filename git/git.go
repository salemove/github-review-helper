package git

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type Repos interface {
	// GetUpdatedRepo either clones the specified repository if it hasn't been cloned yet or simply
	// fetches the latest changes for it. Returns the Repo in any case.
	GetUpdatedRepo(url, repoOwner, repoName string) (Repo, error)
}

type Repo interface {
	Fetch() error
	// Runs `git rebase --interactive --autosquash` for the given refs and automatically saves and closes
	// the editor for interactive rebase. Then force pushes the current HEAD to destinationRef on origin.
	AutosquashAndPush(upstreamRef, branchRef, destinationRef string) error
	DeleteRemoteBranch(remoteRef string) error
}

type ErrSquashConflict struct {
	Err error
}

func (e *ErrSquashConflict) Error() string {
	return fmt.Sprintf("failed to rebase with autosquash: %v", e.Err)
}

type repos struct {
	sync.Mutex
	basePath string
	repos    map[string]*repo
}

// NewRepos creates a new Repos instance which will hold all its repos in the specified base path
func NewRepos(basePath string) Repos {
	return &repos{
		basePath: basePath,
		repos:    make(map[string]*repo),
	}
}

func (g *repos) repo(path string) *repo {
	existingRepo, exists := g.repos[path]
	if !exists {
		newRepo := &repo{path: path}
		g.repos[path] = newRepo
		return newRepo
	}
	return existingRepo
}

func (g *repos) clone(url, localPath string) (Repo, error) {
	if err := runWithLogging("git", "clone", url, localPath); err != nil {
		return nil, fmt.Errorf("failed to clone: %v", err)
	}
	newRepo := g.repo(localPath)
	if err := newRepo.configureNameEmail(); err != nil {
		return nil, fmt.Errorf("failed to configure name and email: %v", err)
	}
	return newRepo, nil
}

func (g *repos) GetUpdatedRepo(url, repoOwner, repoName string) (Repo, error) {
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

func (r *repo) AutosquashAndPush(upstreamRef, branchRef, destinationRef string) error {
	r.Lock()
	defer r.Unlock()

	if err := r.rebaseAutosquash(upstreamRef, branchRef); err != nil {
		return err
	}
	return r.forcePushHeadTo(destinationRef)
}

func (r *repo) Fetch() error {
	r.Lock()
	defer r.Unlock()

	if err := r.git("fetch"); err != nil {
		return fmt.Errorf("failed to fetch: %v", err)
	}
	return nil
}

func (r *repo) rebaseAutosquash(upstreamRef, branchRef string) error {
	// This makes the --interactive rebase not actually interactive
	if err := os.Setenv("GIT_SEQUENCE_EDITOR", "true"); err != nil {
		return fmt.Errorf("failed to change the env variable: %v", err)
	}
	defer os.Unsetenv("GIT_SEQUENCE_EDITOR")

	if err := r.git("rebase", "--interactive", "--autosquash", upstreamRef, branchRef); err != nil {
		err = &ErrSquashConflict{err}
		log.Println(err, " Trying to clean up.")
		if cleanupErr := r.git("rebase", "--abort"); cleanupErr != nil {
			log.Println("Also failed to clean up after the failed rebase: ", cleanupErr)
		}
		return err
	}
	return nil
}

func (r *repo) forcePushHeadTo(destinationRef string) error {
	if err := r.git("push", "--force", "origin", "@:"+destinationRef); err != nil {
		return fmt.Errorf("failed to force push to remote: %v", err)
	}
	return nil
}

func (r *repo) configureNameEmail() error {
	if err := r.git("config", "user.name", "github-review-helper"); err != nil {
		return err
	}
	return r.git("config", "user.email", "<>")
}

func (r *repo) git(args ...string) error {
	allArgs := append([]string{"-C", r.path}, args...)
	return runWithLogging("git", allArgs...)
}

func (r *repo) DeleteRemoteBranch(remoteRef string) error {
	r.Lock()
	defer r.Unlock()

	if err := runWithLogging("git", "-C", r.path, "push", "origin", "--delete", remoteRef); err != nil {
		return fmt.Errorf("failed to remove remote branch %s: %v", remoteRef, err)
	}
	return nil
}

func runWithLogging(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for scanner.Scan() {
		log.Printf("%s: %s\n", name, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading %s's stdout/stderr: %s\n", name, err)
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
