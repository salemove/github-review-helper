package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const MergingLabel = "merging"

func isMergeCommand(comment string) bool {
	return strings.TrimSpace(comment) == "!merge"
}

func handleMergeCommand(issueComment IssueComment, issues Issues, pullRequests PullRequests) Response {
	errResp := addLabel(issueComment.Repository, issueComment.IssueNumber, MergingLabel, issues)
	if errResp != nil {
		return errResp
	}
	return mergeWithRetry(3, issueComment, issues, pullRequests)
}

func mergeWithRetry(nrOfRetries int, issueComment IssueComment, issues Issues, pullRequests PullRequests) Response {
	pr, errResp := getPR(issueComment, pullRequests)
	if errResp != nil {
		return errResp
	} else if *pr.Merged {
		log.Printf("PR #%d already merged. Removing the '%s' label.\n", issueComment.IssueNumber, MergingLabel)
		errResp = removeLabel(issueComment.Repository, issueComment.IssueNumber, MergingLabel, issues)
		if errResp != nil {
			return errResp
		}
		return SuccessResponse{}
	} else if !*pr.Mergeable {
		return SuccessResponse{}
	}
	err := merge(issueComment.Repository, issueComment.IssueNumber, pullRequests)
	if err == OutdatedMergeRefError && nrOfRetries > 0 {
		return mergeWithRetry(nrOfRetries-1, issueComment, issues, pullRequests)
	} else if err != nil {
		message := fmt.Sprintf("Failed to merge PR #%d", issueComment.IssueNumber)
		return ErrorResponse{err, http.StatusBadGateway, message}
	}
	log.Printf("PR #%d successfully merged. Removing the '%s' label.\n", issueComment.IssueNumber, MergingLabel)
	errResp = removeLabel(issueComment.Repository, issueComment.IssueNumber, MergingLabel, issues)
	if errResp != nil {
		return errResp
	}
	return SuccessResponse{}
}
