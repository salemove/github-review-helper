package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/salemove/github-review-helper/git"
)

var ErrSquashConflict = errors.New("Rebase failed due to a squash conflict")

func isSquashCommand(comment string) bool {
	return strings.TrimSpace(comment) == "!squash"
}

func isCheckCommand(comment string) bool {
	return strings.TrimSpace(comment) == "!check"
}

func handleSquashCommand(issueComment IssueComment, gitRepos git.Repos, pullRequests PullRequests, repositories Repositories) Response {
	pr, errResp := getPR(issueComment, pullRequests)
	if errResp != nil {
		return errResp
	}
	return squashAndReportFailure(pr, gitRepos, repositories)
}

func checkForFixupCommitsOnPREvent(pullRequestEvent PullRequestEvent, pullRequests PullRequests,
	repositories Repositories, retry retryGithubOperation) Response {

	isExpectedHead := func(head string) bool {
		return head == pullRequestEvent.Head.SHA
	}
	setStatus := func(status *github.RepoStatus) *ErrorResponse {
		return setStatusForPREvent(pullRequestEvent, status, repositories)
	}
	return checkForFixupCommits(pullRequestEvent, isExpectedHead, setStatus, pullRequests, retry)
}

func checkForFixupCommitsOnIssueComment(issueComment IssueComment, pullRequests PullRequests,
	repositories Repositories, retry retryGithubOperation) Response {

	isExpectedHead := func(string) bool { return true }
	setStatus := func(status *github.RepoStatus) *ErrorResponse {
		pr, errResp := getPR(issueComment, pullRequests)
		if errResp != nil {
			return errResp
		}
		return setStatusForPR(pr, status, repositories)
	}
	return checkForFixupCommits(issueComment, isExpectedHead, setStatus, pullRequests, retry)
}

func checkForFixupCommits(issueable Issueable, isExpectedHead func(string) bool,
	setStatus func(*github.RepoStatus) *ErrorResponse, pullRequests PullRequests,
	retry retryGithubOperation) Response {

	log.Printf("Checking for fixup commits for PR %s.\n", issueable.Issue().FullName())
	maybeSyncResponse := retry(func() asyncResponse {
		commits, asyncErrResp := getCommits(issueable, isExpectedHead, pullRequests)
		if asyncErrResp != nil {
			return asyncErrResp.toAsyncResponse()
		}
		if !includesFixupCommits(commits) {
			status := createSquashStatus("success", "No fixup! or squash! commits to be squashed")
			if errResp := setStatus(status); errResp != nil {
				return nonRetriable(errResp)
			}
			return nonRetriable(SuccessResponse{})
		}
		status := createSquashStatus("pending", "This PR needs to be squashed with !squash before merging")
		if errResp := setStatus(status); errResp != nil {
			return nonRetriable(errResp)
		}
		return nonRetriable(SuccessResponse{})
	})
	if maybeSyncResponse.OperationFinishedSynchronously {
		return maybeSyncResponse.Response
	}
	return SuccessResponse{fmt.Sprintf(
		"Continuing checking for fixup commits for PR %s asynchronously.",
		issueable.Issue().FullName(),
	)}
}

func includesFixupCommits(commits []*github.RepositoryCommit) bool {
	for _, commit := range commits {
		if strings.HasPrefix(*commit.Commit.Message, "fixup! ") || strings.HasPrefix(*commit.Commit.Message, "squash! ") {
			return true
		}
	}
	return false
}

func createSquashStatus(state, description string) *github.RepoStatus {
	return &github.RepoStatus{
		State:       github.String(state),
		Description: github.String(description),
		Context:     github.String(githubStatusSquashContext),
	}
}

func squashAndReportFailure(pr *github.PullRequest, gitRepos git.Repos, repositories Repositories) Response {
	log.Printf("Squashing %s that's going to be merged into %s\n", *pr.Head.Ref, *pr.Base.Ref)
	err := squash(pr, gitRepos, repositories)
	if err == ErrSquashConflict {
		log.Printf("Failed to autosquash the commits with an interactive rebase: %s. Setting a failure status.\n", err)
		status := createSquashStatus("failure", "Automatic squash failed. Please squash manually")
		if errResp := setStatusForPR(pr, status, repositories); errResp != nil {
			return errResp
		}
		return SuccessResponse{}
	} else if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to squash the commits in the PR"}
	}
	return SuccessResponse{}
}

func squash(pr *github.PullRequest, gitRepos git.Repos, repositories Repositories) error {
	headRepository := headRepository(pr)
	gitRepo, err := gitRepos.GetUpdatedRepo(headRepository.URL, headRepository.Owner, headRepository.Name)
	if err != nil {
		log.Println(err)
		return errors.New("Failed to update the local repo")
	}
	if err = gitRepo.AutosquashAndPush("origin/"+*pr.Base.Ref, *pr.Head.SHA, *pr.Head.Ref); err != nil {
		log.Println(err)
		if _, ok := err.(*git.ErrSquashConflict); ok {
			return ErrSquashConflict
		}
		return err
	}
	return nil
}
