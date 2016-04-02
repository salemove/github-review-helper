package main

import "strings"

const mergingLabel = "merging"

func isMergeCommand(comment string) bool {
	return strings.TrimSpace(comment) == "!merge"
}

func handleMergeCommand(issueComment IssueComment, issues Issues) Response {
	err := addLabel(issueComment.Repository, issueComment.IssueNumber, mergingLabel, issues)
	if err != nil {
		return err
	}
	return SuccessResponse{}
}
