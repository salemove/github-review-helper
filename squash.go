package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
)

func isSquashCommand(comment string) bool {
	return strings.TrimSpace(comment) == "!squash"
}

func handleSquashCommand(issueComment IssueComment, git Git, pullRequests PullRequests, repositories Repositories) Response {
	pr, errResp := getPR(issueComment, pullRequests)
	if errResp != nil {
		return errResp
	}
	log.Printf("Squashing %s that's going to be merged into %s\n", *pr.Head.Ref, *pr.Base.Ref)
	repo, err := git.GetUpdatedRepo(issueComment.Repository.URL, issueComment.Repository.Owner, issueComment.Repository.Name)
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to update the local repo"}
	}
	if err = repo.RebaseAutosquash(*pr.Base.SHA, *pr.Head.SHA); err != nil {
		log.Printf("Failed to autosquash the commits with an interactive rebase: %s. Setting a failure status.\n", err)
		status := createSquashStatus("failure", "Automatic squash failed. Please squash manually")
		if errResp := setStatus(issueComment.Repository, *pr.Head.SHA, status, repositories); errResp != nil {
			return errResp
		}
		return SuccessResponse{"Failed to autosquash the commits with an interactive rebase. Reported the failure."}
	}
	if err = repo.ForcePushHeadTo(*pr.Head.Ref); err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to push the squashed version"}
	}
	squashedHeadSHA, err := repo.GetHeadSHA()
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to get the squashed branch's HEAD's SHA"}
	}
	status := createSquashStatus("success", "All fixup! and squash! commits successfully squashed")
	if errResp := setStatus(issueComment.Repository, squashedHeadSHA, status, repositories); errResp != nil {
		return errResp
	}
	return SuccessResponse{}
}

func checkForFixupCommits(pullRequestEvent PullRequestEvent, pullRequests PullRequests, repositories Repositories) Response {
	if !(pullRequestEvent.Action == "opened" || pullRequestEvent.Action == "synchronize") {
		return SuccessResponse{"PR not opened or synchronized. Ignoring."}
	}
	log.Printf("Checking for fixup commits for PR %s.\n", pullRequestEvent.Issue().FullName())
	commits, errResp := getCommits(pullRequestEvent, pullRequests)
	if errResp != nil {
		return errResp
	}
	if !includesFixupCommits(commits) {
		return SuccessResponse{}
	}
	status := createSquashStatus("pending", "This PR needs to be squashed with !squash before merging")
	if errResp := setPRHeadStatus(pullRequestEvent, status, pullRequests, repositories); errResp != nil {
		return errResp
	}
	return SuccessResponse{}
}

func includesFixupCommits(commits []github.RepositoryCommit) bool {
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
