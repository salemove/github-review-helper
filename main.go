package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/tylerb/graceful.v1"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	githubStatusSquashContext     = "review/squash"
	githubStatusPeerReviewContext = "review/peer"
)

type IssueComment struct {
	IssueNumber   int
	Comment       string
	IsPullRequest bool
	Repository    Repository
}

type PullRequestEvent struct {
	IssueNumber int
	Action      string
	Repository  Repository
}

type Repository struct {
	Owner string
	Name  string
	URL   string
}

func main() {
	conf := NewConfig()
	githubClient := initGithubClient(conf.AccessToken)
	reposDir, err := ioutil.TempDir("", "github-review-helper")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(reposDir)

	git := NewGit(reposDir)

	mux := http.NewServeMux()
	mux.Handle("/", Handler(func(w http.ResponseWriter, r *http.Request) Response {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return ErrorResponse{err, http.StatusInternalServerError, "Failed to read the request's body"}
		}
		signature := r.Header.Get("X-Hub-Signature")
		hasSecret, err := hasSecret(body, signature, conf.Secret)
		if err != nil {
			return ErrorResponse{err, http.StatusInternalServerError, "Failed to check the signature"}
		}
		if !hasSecret {
			return ErrorResponse{nil, http.StatusBadRequest, "Bad X-Hub-Signature"}
		}
		eventType := r.Header.Get("X-Github-Event")
		switch eventType {
		case "issue_comment":
			return handleIssueComment(w, body, git, githubClient)
		case "pull_request":
			return handlePullRequest(w, body, git, githubClient)
		}
		return SuccessResponse{"Not an event I understand. Ignoring."}
	}))

	graceful.Run(fmt.Sprintf(":%d", conf.Port), 10*time.Second, mux)
}

// startsWithPlusOne matches strings that start with either a +1 (not followed by other digits) or a :+1: emoji
var startsWithPlusOne = regexp.MustCompile("^(:\\+1:|\\+1($|\\D))")

func handleIssueComment(w http.ResponseWriter, body []byte, git Git, githubClient *github.Client) Response {
	issueComment, err := parseIssueComment(body)
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to parse the request's body"}
	}
	if !issueComment.IsPullRequest {
		return SuccessResponse{"Not a PR. Ignoring."}
	}
	switch {
	case issueComment.Comment == "!squash":
		return handleSquash(w, issueComment, git, githubClient)
	case startsWithPlusOne.MatchString(issueComment.Comment):
		return handlePlusOne(w, issueComment, git, githubClient)
	}
	return SuccessResponse{"Not a command I understand. Ignoring."}
}

func handleSquash(w http.ResponseWriter, issueComment IssueComment, git Git, githubClient *github.Client) Response {
	pr, _, err := githubClient.PullRequests.Get(issueComment.Repository.Owner, issueComment.Repository.Name, issueComment.IssueNumber)
	if err != nil {
		message := fmt.Sprintf("Getting PR %s failed", issueComment.PullRequestName())
		return ErrorResponse{err, http.StatusBadGateway, message}
	}
	log.Printf("Squashing %s that's going to be merged into %s\n", *pr.Head.Ref, *pr.Base.Ref)
	repo, err := git.GetUpdatedRepo(issueComment.Repository.URL, issueComment.Repository.Owner, issueComment.Repository.Name)
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to update the local repo"}
	}
	if err = repo.RebaseAutosquash(*pr.Base.SHA, *pr.Head.SHA); err != nil {
		log.Printf("Failed to autosquash the commits with an interactive rebase: %s. Setting a failure status.\n", err)
		_, _, err = githubClient.Repositories.CreateStatus(issueComment.Repository.Owner, issueComment.Repository.Name, *pr.Head.SHA, &github.RepoStatus{
			State:       github.String("failure"),
			Description: github.String("Failed to automatically squash the fixup! and squash! commits. Please squash manually"),
			Context:     github.String(githubStatusSquashContext),
		})
		if err != nil {
			message := fmt.Sprintf("Failed to create a failure status for commit %s", *pr.Head.SHA)
			return ErrorResponse{err, http.StatusBadGateway, message}
		}
		return SuccessResponse{"Failed to autosquash the commits with an interactive rebase. Reported the failure."}
	}
	if err = repo.ForcePushHeadTo(*pr.Head.Ref); err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to push the squashed version"}
	}
	headSHA, err := repo.GetHeadSHA()
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to get the squashed branch's HEAD's SHA"}
	}
	_, _, err = githubClient.Repositories.CreateStatus(issueComment.Repository.Owner, issueComment.Repository.Name, headSHA, &github.RepoStatus{
		State:       github.String("success"),
		Description: github.String("All fixup! and squash! commits successfully squashed"),
		Context:     github.String(githubStatusSquashContext),
	})
	if err != nil {
		message := fmt.Sprintf("Failed to create a success status for commit %s", headSHA)
		return ErrorResponse{err, http.StatusBadGateway, message}
	}
	return SuccessResponse{}
}

func handlePlusOne(w http.ResponseWriter, issueComment IssueComment, git Git, githubClient *github.Client) Response {
	pr, _, err := githubClient.PullRequests.Get(issueComment.Repository.Owner, issueComment.Repository.Name, issueComment.IssueNumber)
	if err != nil {
		message := fmt.Sprintf("Getting PR %s failed", issueComment.PullRequestName())
		return ErrorResponse{err, http.StatusBadGateway, message}
	}
	log.Printf("Marking PR %s as peer reviewed\n", issueComment.PullRequestName())
	_, _, err = githubClient.Repositories.CreateStatus(issueComment.Repository.Owner, issueComment.Repository.Name, *pr.Head.SHA, &github.RepoStatus{
		State:       github.String("success"),
		Description: github.String("This PR has been peer reviewed"),
		Context:     github.String(githubStatusPeerReviewContext),
	})
	if err != nil {
		message := fmt.Sprintf("Failed to create a success status for commit %s", *pr.Head.SHA)
		return ErrorResponse{err, http.StatusBadGateway, message}
	}
	return SuccessResponse{}
}

func handlePullRequest(w http.ResponseWriter, body []byte, git Git, githubClient *github.Client) Response {
	pullRequestEvent, err := parsePullRequestEvent(body)
	if err != nil {
		return ErrorResponse{err, http.StatusInternalServerError, "Failed to parse the request's body"}
	}
	if !(pullRequestEvent.Action == "opened" || pullRequestEvent.Action == "synchronize") {
		return SuccessResponse{"PR not opened or synchronized. Ignoring."}
	}
	log.Printf("Checking for fixup commits for PR %s.\n", pullRequestEvent.PullRequestName())
	commits, _, err := githubClient.PullRequests.ListCommits(pullRequestEvent.Repository.Owner, pullRequestEvent.Repository.Name, pullRequestEvent.IssueNumber, nil)
	if err != nil {
		message := fmt.Sprintf("Getting commits for PR %s failed", pullRequestEvent.PullRequestName())
		return ErrorResponse{err, http.StatusBadGateway, message}
	}
	if includesFixupCommits(commits) {
		pr, _, err := githubClient.PullRequests.Get(pullRequestEvent.Repository.Owner, pullRequestEvent.Repository.Name, pullRequestEvent.IssueNumber)
		if err != nil {
			message := fmt.Sprintf("Getting PR %s failed", pullRequestEvent.PullRequestName())
			return ErrorResponse{err, http.StatusBadGateway, message}
		}
		_, _, err = githubClient.Repositories.CreateStatus(pullRequestEvent.Repository.Owner, pullRequestEvent.Repository.Name, *pr.Head.SHA, &github.RepoStatus{
			State:       github.String("pending"),
			Description: github.String("This PR needs to be squashed with !squash before merging"),
			Context:     github.String(githubStatusSquashContext),
		})
		if err != nil {
			message := fmt.Sprintf("Failed to create a pending status for commit %s", *pr.Head.SHA)
			return ErrorResponse{err, http.StatusBadGateway, message}
		}
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

func initGithubClient(accessToken string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc)
}

func parseIssueComment(body []byte) (IssueComment, error) {
	var message struct {
		Issue struct {
			Number      int `json:"Number"`
			PullRequest struct {
				URL string `json:"url"`
			} `json:"pull_request"`
		} `json:"issue"`
		Repository struct {
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
			SSHURL string `json:"ssh_url"`
		} `json:"repository"`
		Comment struct {
			Body string `json:"body"`
		} `json:"comment"`
	}
	err := json.Unmarshal(body, &message)
	if err != nil {
		return IssueComment{}, err
	}
	return IssueComment{
		IssueNumber:   message.Issue.Number,
		Comment:       message.Comment.Body,
		IsPullRequest: message.Issue.PullRequest.URL != "",
		Repository: Repository{
			Owner: message.Repository.Owner.Login,
			Name:  message.Repository.Name,
			URL:   message.Repository.SSHURL,
		},
	}, nil
}

func parsePullRequestEvent(body []byte) (PullRequestEvent, error) {
	var message struct {
		Action      string `json:"action"`
		Number      int    `json:"number"`
		PullRequest struct {
			URL string `json:"url"`
		} `json:"pull_request"`
		Repository struct {
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
			SSHURL string `json:"ssh_url"`
		} `json:"repository"`
	}
	err := json.Unmarshal(body, &message)
	if err != nil {
		return PullRequestEvent{}, err
	}
	return PullRequestEvent{
		IssueNumber: message.Number,
		Action:      message.Action,
		Repository: Repository{
			Owner: message.Repository.Owner.Login,
			Name:  message.Repository.Name,
			URL:   message.Repository.SSHURL,
		},
	}, nil
}

func hasSecret(message []byte, signature, key string) (bool, error) {
	var messageMACString string
	fmt.Sscanf(signature, "sha1=%s", &messageMACString)
	messageMAC, err := hex.DecodeString(messageMACString)
	if err != nil {
		return false, err
	}

	mac := hmac.New(sha1.New, []byte(key))
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC), nil
}

func (i IssueComment) PullRequestName() string {
	return fmt.Sprintf("%s/%s#%d", i.Repository.Owner, i.Repository.Name, i.IssueNumber)
}

func (p PullRequestEvent) PullRequestName() string {
	return fmt.Sprintf("%s/%s#%d", p.Repository.Owner, p.Repository.Name, p.IssueNumber)
}
