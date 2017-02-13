package main

import (
	"fmt"
	"github.com/google/go-github/github"
)

type (
	Issue struct {
		Number     int
		Repository Repository
		User       User
	}

	Issueable interface {
		Issue() Issue
	}

	IssueComment struct {
		IssueNumber   int
		Comment       string
		IsPullRequest bool
		Repository    Repository
		User          User
	}

	PullRequestEvent struct {
		IssueNumber int
		Action      string
		Head        PullRequestBranch
		Repository  Repository
		User        User
	}

	StatusEvent struct {
		SHA        string
		State      string
		Branches   []Branch
		Repository Repository
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

	Branch struct {
		SHA string // The SHA of the head commit of the branch
	}

	User struct {
		Login string
	}
)

func (i IssueComment) Issue() Issue {
	return Issue{
		Number:     i.IssueNumber,
		Repository: i.Repository,
		User:       i.User,
	}
}

func (p PullRequestEvent) Issue() Issue {
	return Issue{
		Number:     p.IssueNumber,
		Repository: p.Repository,
		User:       p.User,
	}
}

func (i Issue) Issue() Issue {
	return i
}

func (i Issue) FullName() string {
	return fmt.Sprintf("%s/%s#%d", i.Repository.Owner, i.Repository.Name, i.Number)
}

func prFullName(pr *github.PullRequest) string {
	baseRepository := pr.Base.Repo
	return fmt.Sprintf("%s/%s#%d", *baseRepository.Owner.Login, *baseRepository.Name, *pr.Number)
}

func prIssue(pr *github.PullRequest) Issue {
	return Issue{
		Number:     *pr.Number,
		Repository: baseRepository(pr),
		User: User{
			Login: *pr.User.Login,
		},
	}
}

func baseRepository(pr *github.PullRequest) Repository {
	return repositoryInternalRepresentation(pr.Base.Repo)
}

func headRepository(pr *github.PullRequest) Repository {
	return repositoryInternalRepresentation(pr.Head.Repo)
}

func repositoryInternalRepresentation(repo *github.Repository) Repository {
	return Repository{
		Owner: *repo.Owner.Login,
		Name:  *repo.Name,
		URL:   *repo.SSHURL,
	}
}
