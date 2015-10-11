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
	"time"

	"gopkg.in/tylerb/graceful.v1"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type IssueComment struct {
	IssueNumber   int
	Comment       string
	IsPullRequest bool
	Repository    Repository
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
		if eventType != "issue_comment" {
			return SuccessResponse{"Not an event I understand. Ignoring."}
		}
		issueComment, err := parseIssueComment(body)
		if err != nil {
			return ErrorResponse{err, http.StatusInternalServerError, "Failed to parse the requests body"}
		}
		if !issueComment.IsPullRequest {
			return SuccessResponse{"Not a PR. Ignoring."}
		}
		if issueComment.Comment != "!squash" {
			return SuccessResponse{"Not a command I understand. Ignoring."}
		}
		pr, _, err := githubClient.PullRequests.Get(issueComment.Repository.Owner, issueComment.Repository.Name, issueComment.IssueNumber)
		if err != nil {
			message := fmt.Sprintf("Getting PR %s/%s#%d failed", issueComment.Repository.Owner, issueComment.Repository.Name, issueComment.IssueNumber)
			return ErrorResponse{err, http.StatusInternalServerError, message}
		}
		log.Printf("Squashing %s that's going to be merged into %s\n", *pr.Head.Ref, *pr.Base.Ref)
		repo, err := git.GetUpdatedRepo(issueComment.Repository.URL, issueComment.Repository.Owner, issueComment.Repository.Name)
		if err != nil {
			return ErrorResponse{err, http.StatusInternalServerError, "Failed to update the local repo"}
		}
		if err = repo.RebaseAutosquash(*pr.Base.SHA, *pr.Head.SHA); err != nil {
			return ErrorResponse{err, http.StatusInternalServerError, "Failed to autosquash the commits with an interactive rebase"}
		}
		if err = repo.ForcePushHeadTo(*pr.Head.Ref); err != nil {
			return ErrorResponse{err, http.StatusInternalServerError, "Failed to push the squashed version"}
		}
		return SuccessResponse{}
	}))

	graceful.Run(fmt.Sprintf(":%d", conf.Port), 10*time.Second, mux)
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
