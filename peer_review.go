package main

import (
	"log"

	"github.com/google/go-github/github"
)

func handlePlusOne(issueComment IssueComment, pullRequests PullRequests, repositories Repositories) Response {
	log.Printf("Marking PR %s as peer reviewed\n", issueComment.Issue().FullName())
	status := createPeerReviewStatus("success", "This PR has been peer reviewed")
	if errResp := setPRHeadStatus(issueComment, status, pullRequests, repositories); errResp != nil {
		return errResp
	}
	return SuccessResponse{}
}

func createPeerReviewStatus(state, description string) *github.RepoStatus {
	return &github.RepoStatus{
		State:       github.String(state),
		Description: github.String(description),
		Context:     github.String(githubStatusPeerReviewContext),
	}
}
