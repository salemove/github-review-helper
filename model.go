package main

import (
	"fmt"
	"github.com/google/go-github/github"
)

type (
	Issue struct {
		Number     int
		Repository Repository
	}

	Issueable interface {
		Issue() Issue
	}

	IssueComment struct {
		IssueNumber   int
		Comment       string
		IsPullRequest bool
		Repository    Repository
	}

	PullRequestEvent struct {
		IssueNumber int
		Action      string
		Repository  Repository
	}

	Repository struct {
		Owner string
		Name  string
		URL   string
	}
)

func (i IssueComment) Issue() Issue {
	return Issue{
		Number:     i.IssueNumber,
		Repository: i.Repository,
	}
}

func (p PullRequestEvent) Issue() Issue {
	return Issue{
		Number:     p.IssueNumber,
		Repository: p.Repository,
	}
}

func (i Issue) FullName() string {
	return fmt.Sprintf("%s/%s#%d", i.Repository.Owner, i.Repository.Name, i.Number)
}

func internalRepositoryRepresentation(githubRepository *github.Repository) Repository {
	return Repository{
		Owner: *githubRepository.Owner.Login,
		Name:  *githubRepository.Name,
		URL:   *githubRepository.SSHURL,
	}
}
