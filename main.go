package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"gopkg.in/tylerb/graceful.v1"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/salemove/github-review-helper/git"
	"golang.org/x/oauth2"
)

const (
	githubStatusSquashContext     = "review/squash"
	githubStatusPeerReviewContext = "review/peer"
)

type retryGithubOperation func(func() asyncResponse) MaybeSyncResponse

func main() {
	conf := NewConfig()
	githubClient := initGithubClient(conf.AccessToken)
	reposDir, err := ioutil.TempDir("", "github-review-helper")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(reposDir)

	gitRepos := git.NewRepos(reposDir)
	var asyncOperationWg sync.WaitGroup

	mux := http.NewServeMux()
	mux.Handle("/", CreateHandler(
		conf,
		gitRepos,
		&asyncOperationWg,
		githubClient.PullRequests,
		githubClient.Repositories,
		githubClient.Issues,
		githubClient.Search,
	))

	graceful.Run(fmt.Sprintf(":%d", conf.Port), 10*time.Second, mux)
	asyncOperationWg.Wait()
}

func CreateHandler(conf Config, gitRepos git.Repos, asyncOperationWg *sync.WaitGroup,
	pullRequests PullRequests, repositories Repositories, issues Issues, search Search) Handler {

	retry := func(operation func() asyncResponse) MaybeSyncResponse {
		return delayWithRetries(conf.GithubAPITryDeltas, operation, asyncOperationWg)
	}

	return func(w http.ResponseWriter, r *http.Request) Response {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return ErrorResponse{err, http.StatusInternalServerError, "Failed to read the request's body"}
		}
		if errResp := checkAuthentication(body, r, conf.Secret); errResp != nil {
			return errResp
		}
		eventType := r.Header.Get("X-Github-Event")
		switch eventType {
		case "issue_comment":
			return handleIssueComment(body, retry, gitRepos, pullRequests, repositories, issues)
		case "pull_request":
			return handlePullRequestEvent(body, retry, pullRequests, repositories)
		case "status":
			return handleStatusEvent(body, retry, gitRepos, search, issues, pullRequests)
		}
		return SuccessResponse{"Not an event I understand. Ignoring."}
	}
}

func handleIssueComment(body []byte, retry retryGithubOperation, gitRepos git.Repos,
	pullRequests PullRequests, repositories Repositories, issues Issues) Response {

	issueComment, err := parseIssueComment(body)
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to parse the request's body"}
	}
	if !issueComment.IsPullRequest {
		return SuccessResponse{"Not a PR. Ignoring."}
	}
	commentCategory := parseComment(issueComment.Comment)
	if commentCategory == regularComment {
		return SuccessResponse{"Not a command I understand. Ignoring."}
	}
	if successResp, errResp := checkUserAuthorization(issueComment, issues, repositories); errResp != nil {
		return errResp
	} else if successResp != nil {
		return successResp
	}
	switch commentCategory {
	case squashCommand:
		return handleSquashCommand(issueComment, gitRepos, pullRequests, repositories)
	case mergeCommand:
		return handleMergeCommand(issueComment, issues, pullRequests, repositories, gitRepos)
	case checkCommand:
		return checkForFixupCommitsOnIssueComment(issueComment, pullRequests, repositories, retry)
	}
	return ErrorResponse{
		Code:         http.StatusInternalServerError,
		ErrorMessage: fmt.Sprintf("Unhandled comment type: %v", commentCategory),
	}
}

func handlePullRequestEvent(body []byte, retry retryGithubOperation, pullRequests PullRequests,
	repositories Repositories) Response {

	pullRequestEvent, err := parsePullRequestEvent(body)
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to parse the request's body"}
	} else if !(pullRequestEvent.Action == "opened" || pullRequestEvent.Action == "synchronize") {
		return SuccessResponse{"PR not opened or synchronized. Ignoring."}
	}
	return checkForFixupCommitsOnPREvent(pullRequestEvent, pullRequests, repositories, retry)
}

func handleStatusEvent(body []byte, retry retryGithubOperation, gitRepos git.Repos, search Search,
	issues Issues, pullRequests PullRequests) Response {

	statusEvent, err := parseStatusEvent(body)
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to parse the request's body"}
	} else if newPullRequestsPossiblyReadyForMerging(statusEvent) {
		maybeSyncResponse := retry(func() asyncResponse {
			return mergePullRequestsReadyForMerging(statusEvent, gitRepos, search, issues, pullRequests)
		})
		if maybeSyncResponse.OperationFinishedSynchronously {
			return maybeSyncResponse.Response
		}
		return SuccessResponse{"Status update might have caused a PR to become mergeable. Will check for " +
			"mergeable PRs asynchronously"}
	}
	return SuccessResponse{"Status update does not affect any PRs mergeability. Ignoring."}
}

func initGithubClient(accessToken string) *github.Client {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	oauthTransport := &oauth2.Transport{
		Source: tokenSource,
	}

	memoryCacheTransport := &httpcache.Transport{
		Transport:           oauthTransport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}

	httpClient := &http.Client{
		Transport: memoryCacheTransport,
		Timeout:   30 * time.Second,
	}
	return github.NewClient(httpClient)
}

type commentType int

const (
	squashCommand commentType = iota
	mergeCommand
	checkCommand
	regularComment
)

func parseComment(comment string) commentType {
	switch {
	case isSquashCommand(comment):
		return squashCommand
	case isMergeCommand(comment):
		return mergeCommand
	case isCheckCommand(comment):
		return checkCommand
	}
	return regularComment
}

func checkUserAuthorization(issueComment IssueComment, issues Issues, repositories Repositories) (*SuccessResponse, *ErrorResponse) {
	if isAuthorized, err := isCollaborator(issueComment.Repository, issueComment.User, repositories); err != nil {
		return nil, &ErrorResponse{err, http.StatusBadGateway, "Failed to check if the user is authorized to issue the command"}
	} else if !isAuthorized {
		err = comment(
			fmt.Sprintf("I'm sorry, @%s. I'm afraid I can't do that.", issueComment.User.Login),
			issueComment.Repository,
			issueComment.IssueNumber,
			issues,
		)
		if err != nil {
			return nil, &ErrorResponse{err, http.StatusBadGateway, "Failed to respond to unauthorized command"}
		}
		return &SuccessResponse{"Command issued by a someone who's not a collaborator." +
			" Responded with a comment. Ignoring the command."}, nil
	}
	return nil, nil
}
