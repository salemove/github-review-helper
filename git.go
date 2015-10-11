package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

type Git interface {
	Repo(path string) Repo
}

type Repo interface {
	RebaseAutosquash(upstreamRef, branchRef string) error
}

type git struct{}

func NewGit() Git {
	return git{}
}

func (g git) Repo(path string) Repo {
	return repo{path}
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
