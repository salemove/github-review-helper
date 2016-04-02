package main

import "strings"

func isMergeCommand(comment string) bool {
	return strings.TrimSpace(comment) == "!merge"
}

func handleMergeCommand(issueComment IssueComment) Response {
	return SuccessResponse{}
}
