package main

import (
	"fmt"
	"net/http"
	"strings"
)

const mergingLabel = "merging"

func isMergeCommand(comment string) bool {
	return strings.TrimSpace(comment) == "!merge"
}

func handleMergeCommand(issueComment IssueComment, issues Issues, pullRequests PullRequests) Response {
	errResp := addLabel(issueComment.Repository, issueComment.IssueNumber, mergingLabel, issues)
	if errResp != nil {
		return errResp
	}
	pr, errResp := getPR(issueComment, pullRequests)
	if errResp != nil {
		return errResp
	} else if *pr.Merged {
		return SuccessResponse{}
	} else if !*pr.Mergeable {
		return SuccessResponse{}
	}
	err := merge(issueComment.Repository, issueComment.IssueNumber, pullRequests)
	if err != nil {
		message := fmt.Sprintf("Failed to merge PR #%d", issueComment.IssueNumber)
		return ErrorResponse{err, http.StatusBadGateway, message}
	}
	return SuccessResponse{}
}
