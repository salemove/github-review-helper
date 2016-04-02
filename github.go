package main

import (
	"fmt"
	"net/http"

	"github.com/google/go-github/github"
)

type PullRequests interface {
	Get(owner, repo string, number int) (*github.PullRequest, *github.Response, error)
	ListCommits(owner, repo string, number int, opt *github.ListOptions) ([]github.RepositoryCommit, *github.Response, error)
}

type Repositories interface {
	CreateStatus(owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error)
}

type Issues interface {
	AddLabelsToIssue(owner, repo string, number int, labels []string) ([]github.Label, *github.Response, error)
}

func setPRHeadStatus(issueable Issueable, status *github.RepoStatus, pullRequests PullRequests, repositories Repositories) *ErrorResponse {
	pr, errResp := getPR(issueable, pullRequests)
	if errResp != nil {
		return errResp
	}
	repository := issueable.Issue().Repository
	return setStatus(repository, *pr.Head.SHA, status, repositories)
}

func setStatus(repository Repository, commitRef string, status *github.RepoStatus, repositories Repositories) *ErrorResponse {
	_, _, err := repositories.CreateStatus(repository.Owner, repository.Name, commitRef, status)
	if err != nil {
		message := fmt.Sprintf("Failed to create a %s status for commit %s", *status.State, commitRef)
		return &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return nil
}

func getPR(issueable Issueable, pullRequests PullRequests) (*github.PullRequest, *ErrorResponse) {
	issue := issueable.Issue()
	pr, _, err := pullRequests.Get(issue.Repository.Owner, issue.Repository.Name, issue.Number)
	if err != nil {
		message := fmt.Sprintf("Getting PR %s failed", issue.FullName())
		return nil, &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return pr, nil
}

func getCommits(issueable Issueable, pullRequests PullRequests) ([]github.RepositoryCommit, *ErrorResponse) {
	issue := issueable.Issue()
	commits, _, err := pullRequests.ListCommits(issue.Repository.Owner, issue.Repository.Name, issue.Number, nil)
	if err != nil {
		message := fmt.Sprintf("Getting commits for PR %s failed", issue.FullName())
		return nil, &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return commits, nil
}

func addLabel(repository Repository, issueNumber int, label string, issues Issues) *ErrorResponse {
	_, _, err := issues.AddLabelsToIssue(repository.Owner, repository.Name, issueNumber, []string{label})
	if err != nil {
		message := fmt.Sprintf("Failed to set the label %s for issue nr %d", label, issueNumber)
		return &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return nil
}
