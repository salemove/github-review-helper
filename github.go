package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/github"
)

var NotMergeableError = errors.New("PullRequests is not mergeable.")
var OutdatedMergeRefError = errors.New("Merge failed because head branch has been modified.")

type PullRequests interface {
	Get(owner, repo string, number int) (*github.PullRequest, *github.Response, error)
	ListCommits(owner, repo string, number int, opt *github.ListOptions) ([]github.RepositoryCommit, *github.Response, error)
	Merge(owner, repo string, number int, commitMessage string) (*github.PullRequestMergeResult, *github.Response, error)
}

type Repositories interface {
	CreateStatus(owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error)
}

type Issues interface {
	AddLabelsToIssue(owner, repo string, number int, labels []string) ([]github.Label, *github.Response, error)
	RemoveLabelForIssue(owner, repo string, number int, label string) (*github.Response, error)
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
	pageNr := 1
	commits := []github.RepositoryCommit{}
	for {
		listOptions := &github.ListOptions{
			Page:    pageNr,
			PerPage: 30,
		}
		pageCommits, resp, err := pullRequests.ListCommits(issue.Repository.Owner, issue.Repository.Name, issue.Number, listOptions)
		if err != nil {
			message := fmt.Sprintf("Getting commits for PR %s failed", issue.FullName())
			return nil, &ErrorResponse{err, http.StatusBadGateway, message}
		}
		commits = append(commits, pageCommits...)
		if resp.NextPage == 0 {
			break
		}
		pageNr = resp.NextPage
	}
	return commits, nil
}

func addLabel(repository Repository, issueNumber int, label string, issues Issues) *ErrorResponse {
	_, _, err := issues.AddLabelsToIssue(repository.Owner, repository.Name, issueNumber, []string{label})
	if err != nil {
		message := fmt.Sprintf("Failed to set the label %s for issue #%d", label, issueNumber)
		return &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return nil
}

func removeLabel(repository Repository, issueNumber int, label string, issues Issues) *ErrorResponse {
	_, err := issues.RemoveLabelForIssue(repository.Owner, repository.Name, issueNumber, label)
	if err != nil {
		message := fmt.Sprintf("Failed to remove the label %s for issue #%d", label, issueNumber)
		return &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return nil
}

func merge(repository Repository, issueNumber int, pullRequests PullRequests) error {
	additionalCommitMessage := ""
	result, resp, err := pullRequests.Merge(repository.Owner, repository.Name, issueNumber, additionalCommitMessage)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
			return NotMergeableError
		} else if resp != nil && resp.StatusCode == http.StatusConflict {
			return OutdatedMergeRefError
		}
		return err
	} else if result.Merged == nil || !*result.Merged {
		return errors.New("Request successful, but PR not merged.")
	}
	return nil
}
