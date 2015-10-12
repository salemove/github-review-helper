package main

import "github.com/google/go-github/github"

type PullRequests interface {
	Get(owner string, repo string, number int) (*github.PullRequest, *github.Response, error)
	ListCommits(owner string, repo string, number int, opt *github.ListOptions) ([]github.RepositoryCommit, *github.Response, error)
}

type Repositories interface {
	CreateStatus(owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error)
}
