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
		Head        PullRequestBranch
		Repository  Repository
	}

	Repository struct {
		Owner string
		Name  string
		URL   string
	}

	PullRequestBranch struct {
		SHA        string
		Repository Repository
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

func HeadRepository(pr *github.PullRequest) Repository {
	return Repository{
		Owner: *pr.Head.Repo.Owner.Login,
		Name:  *pr.Head.Repo.Name,
		URL:   *pr.Head.Repo.SSHURL,
	}
}
