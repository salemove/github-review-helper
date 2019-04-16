package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/google/go-github/github"
	"github.com/salemove/github-review-helper/git"
)

const (
	MergingLabel = "merging"
)

var (
	mergeCommandPattern = regexp.MustCompile(`(?s)^\s*!merge(\s.*)?$`)
)

func isMergeCommand(comment string) bool {
	return mergeCommandPattern.MatchString(comment)
}

func newPullRequestsPossiblyReadyForMerging(statusEvent StatusEvent) bool {
	// We only care about success events, because only these events have the
	// possibility of changing a PR's combined status into "success" and so
	// enabling us to merge the PR.
	// In a similar manner we also only care about status updates to commits
	// that are the head commit of a branch, because only they have the ability
	// to change a PR's combined status.
	return statusEvent.State == "success" && isStatusForBranchHead(statusEvent)
}

func handleMergeCommand(issueComment IssueComment, issues Issues, pullRequests PullRequests,
	repositories Repositories, gitRepos git.Repos) Response {
	errResp := addLabel(issueComment.Repository, issueComment.IssueNumber, MergingLabel, issues)
	if errResp != nil {
		return errResp
	}
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
	state, statuses, errResp := getStatuses(pr, repositories)
	if errResp != nil {
		return errResp
	} else if state == "pending" && containsPendingSquashStatus(statuses) {
		return squashAndReportFailure(pr, gitRepos, repositories)
	} else if state != "success" {
		log.Printf("PR #%d has pending and/or failed statuses. Not merging.\n", issueComment.IssueNumber)
		return SuccessResponse{}
	}
	if errResp = mergeReadyPR(pr, gitRepos, issues, pullRequests); errResp != nil {
		return errResp
	}
	return SuccessResponse{fmt.Sprintf("Successfully merged PR %s", issueComment.Issue().FullName())}
}

func mergeReadyPR(pr *github.PullRequest, gitRepos git.Repos, issues Issues,
	pullRequests PullRequests) *ErrorResponse {
	issue := prIssue(pr)
	err := merge(issue.Repository, issue.Number, pullRequests)
	if err == ErrMergeConflict {
		return handleMergeConflict(issue, issues)
	} else if err != nil {
		message := fmt.Sprintf("Failed to merge PR %s", issue.FullName())
		return &ErrorResponse{err, http.StatusBadGateway, message}
	}
	log.Printf(
		"PR %s successfully merged. Removing the '%s' label.\n",
		issue.FullName(),
		MergingLabel,
	)
	errResp := removeLabel(issue.Repository, issue.Number, MergingLabel, issues)
	if errResp != nil {
		return errResp
	}
	if isAcrossForks(pr) {
		log.Printf("PR %s is across forks. Not removing the head branch.\n", issue.FullName())
	} else {
		errResp = deleteRemoteBranch(pr, gitRepos)
		if errResp != nil {
			return errResp
		}
	}
	return nil
}

func mergePullRequestsReadyForMerging(statusEvent StatusEvent, gitRepos git.Repos, search Search,
	issues Issues, pullRequests PullRequests) asyncResponse {
	// Not sure if applying the additional repo:owner/name filter to the query
	// works for cross-fork PRs, but nothing else has been tested with
	// cross-fork PRs either so this is left in for now.
	//
	// Also, specifying the SHA for the search query doesn't guarantee that the
	// SHA is the HEAD of the returned PRs. This means that, if the commit is
	// in 2 different PRs, both of which have the "merging" label and have
	// "success" status then it can happen that it will try to merge both.
	// Which might not be intended, but is still okay, because both PRs do
	// match all the criteria required for merging.
	query := fmt.Sprintf(
		"%s label:\"%s\" is:open repo:%s/%s status:success",
		statusEvent.SHA,
		MergingLabel,
		statusEvent.Repository.Owner,
		statusEvent.Repository.Name,
	)
	issuesToMerge, err := searchIssues(query, search)
	if err != nil {
		message := fmt.Sprintf("Searching for issues with query '%s' failed", query)
		return nonRetriable(ErrorResponse{err, http.StatusBadGateway, message})
	} else if len(issuesToMerge) == 0 {
		return retriable(SuccessResponse{"Found no PRs to merge"})
	}

	var finalErrResp *ErrorResponse
	handleErrResp := func(errResp *ErrorResponse) {
		if finalErrResp == nil {
			finalErrResp = errResp
		} else {
			log.Printf("Multiple PR merge errors have occured. Marking the latest error to be "+
				"returned as a response, replacing the previous error. Logging the previous "+
				"error:\n%s: %v\n", finalErrResp.ErrorMessage, finalErrResp.Error)
			finalErrResp = errResp
		}
	}

	for _, issueToMerge := range issuesToMerge {
		issue := Issue{
			Number:     *issueToMerge.Number,
			Repository: statusEvent.Repository,
			User: User{
				Login: *issueToMerge.User.Login,
			},
		}
		pr, errResp := getPR(issue, pullRequests)
		if errResp != nil {
			handleErrResp(errResp)
			continue
		}
		if errResp := mergeReadyPR(pr, gitRepos, issues, pullRequests); errResp != nil {
			handleErrResp(errResp)
		}
	}
	if finalErrResp != nil {
		return nonRetriable(finalErrResp)
	}
	return nonRetriable(
		SuccessResponse{fmt.Sprintf("Successfully merged %d PRs", len(issuesToMerge))},
	)
}

func containsPendingSquashStatus(statuses []github.RepoStatus) bool {
	for _, status := range statuses {
		if *status.Context == githubStatusSquashContext && *status.State == "pending" {
			return true
		}
	}
	return false
}

func isStatusForBranchHead(statusEvent StatusEvent) bool {
	for _, branch := range statusEvent.Branches {
		if statusEvent.SHA == branch.SHA {
			return true
		}
	}
	return false
}

func handleMergeConflict(issue Issue, issues Issues) *ErrorResponse {
	log.Printf(
		"Merging PR %s failed due to a merge conflict. Removing the '%s' label and notifying the author.\n",
		issue.FullName(),
		MergingLabel,
	)
	removeLabelErrResp := removeLabel(issue.Repository, issue.Number, MergingLabel, issues)
	if removeLabelErrResp != nil {
		log.Printf(
			"Failed to remove the '%s' label. Still notifying the author of the merge conflict. %v\n",
			MergingLabel,
			removeLabelErrResp.Error,
		)
	}
	message := fmt.Sprintf("I'm unable to merge this PR because of a merge conflict."+
		" @%s, can you please take a look?", issue.User.Login)
	err := comment(message, issue.Repository, issue.Number, issues)
	if err != nil {
		errorMessage := fmt.Sprintf(
			"Failed to notify the author of PR %s about the merge conflict",
			issue.FullName(),
		)
		return &ErrorResponse{err, http.StatusBadGateway, errorMessage}
	} else if removeLabelErrResp != nil {
		// Still mark the request as failed, because we were unable to
		// remove the label properly.
		return removeLabelErrResp
	}
	return nil
}

func deleteRemoteBranch(pr *github.PullRequest, gitRepos git.Repos) *ErrorResponse {
	log.Printf("Deleting head branch %s for PR %s.\n", *pr.Head.Ref, prFullName(pr))

	repository := baseRepository(pr)
	gitRepo, err := gitRepos.GetUpdatedRepo(repository.URL, repository.Owner, repository.Name)
	if err != nil {
		message := fmt.Sprintf("Failed to get an updated repo for PR %s", prFullName(pr))
		return &ErrorResponse{err, http.StatusInternalServerError, message}
	}
	err = gitRepo.DeleteRemoteBranch(*pr.Head.Ref)
	if err != nil {
		message := fmt.Sprintf(
			"Failed to delete branch %s for PR %s",
			*pr.Head.Ref,
			prFullName(pr),
		)
		return &ErrorResponse{err, http.StatusInternalServerError, message}
	}
	return nil
}
