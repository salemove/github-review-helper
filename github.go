package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-github/github"
)

const (
	GetCommitsRetryLimit = 3
)

var ErrNotMergeable = errors.New("PullRequests is not mergeable.")
var ErrMergeConflict = errors.New("Merge failed because of a merge conflict.")

type PullRequests interface {
	Get(ctx context.Context, owner, repo string, number int) (*github.PullRequest, *github.Response, error)
	ListCommits(ctx context.Context, owner, repo string, number int, opt *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error)
	Merge(ctx context.Context, owner, repo string, number int, commitMessage string, opt *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error)
}

type Repositories interface {
	CreateStatus(ctx context.Context, owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error)
	GetCombinedStatus(ctx context.Context, owner, repo, ref string, opt *github.ListOptions) (*github.CombinedStatus, *github.Response, error)
	IsCollaborator(ctx context.Context, owner, repo, user string) (bool, *github.Response, error)
}

type Issues interface {
	AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
	RemoveLabelForIssue(ctx context.Context, owner, repo string, number int, label string) (*github.Response, error)
	CreateComment(ctx context.Context, owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
}

type Search interface {
	Issues(ctx context.Context, query string, opt *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error)
}

func setStatusForPREvent(pullRequestEvent PullRequestEvent, status *github.RepoStatus, repositories Repositories) *ErrorResponse {
	// see comment in setStatusForPR for why Head is used instead of Base here
	repository := pullRequestEvent.Head.Repository
	revision := pullRequestEvent.Head.SHA
	log.Printf(
		"Setting %s status to %s for PR %s (revision %s).\n",
		*status.Context,
		*status.State,
		pullRequestEvent.Issue().FullName(),
		revision,
	)
	return setStatus(revision, repository, status, repositories)
}

func setStatusForPR(pr *github.PullRequest, status *github.RepoStatus, repositories Repositories) *ErrorResponse {
	// I'm assuming (because the documentation on this is unclear) that the
	// status has to be reported for the Head repository. It might seem
	// weird, because why should a bot configured for the Base repository
	// have access to the Head repository, but AFAIK all forks must be
	// public and reporting statuses on public repos is always allowed.
	repository := headRepository(pr)
	revision := *pr.Head.SHA
	log.Printf(
		"Setting %s status to %s for PR %s (revision %s).\n",
		*status.Context,
		*status.State,
		prFullName(pr),
		revision,
	)
	return setStatus(revision, repository, status, repositories)
}

func setStatus(revision string, repository Repository, status *github.RepoStatus, repositories Repositories) *ErrorResponse {
	_, _, err := repositories.CreateStatus(context.TODO(), repository.Owner, repository.Name, revision, status)
	if err != nil {
		message := fmt.Sprintf("Failed to create a %s status for commit %s", *status.State, revision)
		return &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return nil
}

func getStatuses(pr *github.PullRequest, repositories Repositories) (string, []github.RepoStatus, *ErrorResponse) {
	headRepository := headRepository(pr)
	pageNr := 1
	statuses := []github.RepoStatus{}
	var state string
	for {
		listOptions := &github.ListOptions{
			Page: pageNr,
			// The maximum for this endpoint is higher than the default 30:
			// https://developer.github.com/v3/repos/statuses/#get-the-combined-status-for-a-specific-ref
			PerPage: 100,
		}
		combinedStatus, resp, err := repositories.GetCombinedStatus(context.TODO(), headRepository.Owner,
			headRepository.Name, *pr.Head.SHA, listOptions)
		if err != nil {
			message := fmt.Sprintf("Failed to get combined status for ref %s", *pr.Head.SHA)
			return "", nil, &ErrorResponse{err, http.StatusBadGateway, message}
		}
		// Although the combined state should be the same for all pages, use
		// the combined state from the latest request, because that's always
		// the most up to date one
		state = *combinedStatus.State
		statuses = append(statuses, combinedStatus.Statuses...)
		if resp.NextPage == 0 {
			break
		}
		pageNr = resp.NextPage
	}
	return state, statuses, nil
}

func searchIssues(query string, search Search) ([]github.Issue, error) {
	pageNr := 1
	issues := []github.Issue{}
	for {
		listOptions := github.ListOptions{
			Page: pageNr,
			// Max is 100: https://developer.github.com/v3/#pagination
			PerPage: 100,
		}
		searchOptions := &github.SearchOptions{ListOptions: listOptions}

		searchResult, resp, err := search.Issues(context.TODO(), query, searchOptions)
		if err != nil {
			return nil, err
		}

		issues = append(issues, searchResult.Issues...)
		if resp.NextPage == 0 {
			break
		}
		pageNr = resp.NextPage
	}
	return issues, nil
}

func getPR(issueable Issueable, pullRequests PullRequests) (*github.PullRequest, *ErrorResponse) {
	issue := issueable.Issue()
	pr, _, err := pullRequests.Get(context.TODO(), issue.Repository.Owner, issue.Repository.Name, issue.Number)
	if err != nil {
		message := fmt.Sprintf("Getting PR %s failed", issue.FullName())
		return nil, &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return pr, nil
}

func getCommits(issueable Issueable, isExpectedHead func(string) bool, pullRequests PullRequests) ([]*github.RepositoryCommit, *ErrorResponse) {
	issue := issueable.Issue()
	pageNr := 1
	nrOfRetriesLeft := GetCommitsRetryLimit
	commits := []*github.RepositoryCommit{}
	for {
		listOptions := &github.ListOptions{
			Page:    pageNr,
			PerPage: 30,
		}
		pageCommits, resp, err := pullRequests.ListCommits(context.TODO(), issue.Repository.Owner,
			issue.Repository.Name, issue.Number, listOptions)
		if err != nil {
			if is404Error(resp) && nrOfRetriesLeft > 0 {
				log.Printf("Getting commits for PR %s failed with a 404: \"%s\". Trying again.\n", issue.FullName(), err.Error())
				nrOfRetriesLeft--
				continue
			}
			message := fmt.Sprintf("Getting commits for PR %s failed", issue.FullName())
			return nil, &ErrorResponse{err, http.StatusBadGateway, message}
		}
		isLastPage := resp.NextPage == 0
		// Check if commit list is outdated by comparing the SHA of the
		// received HEAD (last commit of the last page) with that of the
		// expected HEAD.
		if isLastPage && !isExpectedHead(*lastCommit(pageCommits).SHA) {
			message := fmt.Sprintf(
				"Getting commits for PR %s failed. Received an unexpected head with SHA of %s.",
				issue.FullName(),
				*lastCommit(pageCommits).SHA,
			)
			if nrOfRetriesLeft <= 0 {
				return nil, &ErrorResponse{nil, http.StatusBadGateway, message}
			}
			log.Printf("%s. Trying again.\n", message)
			// Retry from page 1. This means clearing all the existing commits
			// and restoring the page counter to 1.
			commits = []*github.RepositoryCommit{}
			pageNr = 1
			nrOfRetriesLeft--
			continue
		}
		commits = append(commits, pageCommits...)
		if isLastPage {
			break
		}
		pageNr = resp.NextPage
	}
	return commits, nil
}

func lastCommit(commits []*github.RepositoryCommit) *github.RepositoryCommit {
	return commits[len(commits)-1]
}

func addLabel(repository Repository, issueNumber int, label string, issues Issues) *ErrorResponse {
	_, _, err := issues.AddLabelsToIssue(context.TODO(), repository.Owner, repository.Name, issueNumber, []string{label})
	if err != nil {
		message := fmt.Sprintf("Failed to set the label %s for issue #%d", label, issueNumber)
		return &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return nil
}

func removeLabel(repository Repository, issueNumber int, label string, issues Issues) *ErrorResponse {
	_, err := issues.RemoveLabelForIssue(context.TODO(), repository.Owner, repository.Name, issueNumber, label)
	if err != nil {
		message := fmt.Sprintf("Failed to remove the label %s for issue #%d", label, issueNumber)
		return &ErrorResponse{err, http.StatusBadGateway, message}
	}
	return nil
}

func merge(repository Repository, issueNumber int, pullRequests PullRequests) error {
	additionalCommitMessage := ""
	opt := &github.PullRequestOptions{MergeMethod: "merge"}
	result, resp, err := pullRequests.Merge(context.TODO(), repository.Owner, repository.Name,
		issueNumber, additionalCommitMessage, opt)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
			return ErrNotMergeable
		} else if resp != nil && resp.StatusCode == http.StatusConflict {
			return ErrMergeConflict
		}
		return err
	} else if result.Merged == nil || !*result.Merged {
		return errors.New("Request successful, but PR not merged.")
	}
	return nil
}

func comment(message string, repository Repository, issueNumber int, issues Issues) error {
	issueComment := &github.IssueComment{
		Body: github.String(message),
	}
	_, _, err := issues.CreateComment(context.TODO(), repository.Owner, repository.Name, issueNumber, issueComment)
	return err
}

func isCollaborator(repository Repository, user User, repositories Repositories) (bool, error) {
	isCollab, _, err := repositories.IsCollaborator(context.TODO(), repository.Owner, repository.Name, user.Login)
	return isCollab, err
}

func is404Error(resp *github.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}

func isAcrossForks(pr *github.PullRequest) bool {
	return *pr.Base.Repo.ID != *pr.Head.Repo.ID
}
