package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type IssueComment struct {
	IssueNumber     int
	Comment         string
	IsPullRequest   bool
	RepositoryOwner string
	RepositoryName  string
}

func main() {
	conf := NewConfig()
	githubClient := initGithubClient(conf.AccessToken)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Failed to read the request's body")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		bodyMAC := r.Header.Get("X-Hub-Signature")
		if !areMACsEqual(body, []byte(bodyMAC), []byte(conf.Secret)) {
			log.Println("Failed to read the request's body")
			http.Error(w, "Bad X-Hub-Signature", http.StatusBadRequest)
			return
		}
		eventType := r.Header.Get("X-Github-Event")
		if eventType != "issue_comment" {
			log.Println("Unexpected event type: " + eventType)
			http.Error(w, "Unexpected event type", http.StatusBadRequest)
			return
		}
		issueComment, err := parseIssueComment(body)
		if err != nil {
			log.Println("Failed to parse the requests body")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		if !issueComment.IsPullRequest {
			w.Write([]byte{})
			return
		}
		if issueComment.Comment != "!squash" {
			w.Write([]byte{})
			return
		}
		pr, _, err := githubClient.PullRequests.Get(issueComment.RepositoryOwner, issueComment.RepositoryName, issueComment.IssueNumber)
		if err != nil {
			log.Printf("Getting PR %s/%s#%d failed: %s\n", issueComment.RepositoryOwner, issueComment.RepositoryName, issueComment.IssueNumber, err.Error())
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		log.Printf("Wants to merge branch %s to %s", pr.Head.Ref, pr.Base.Ref)
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil))
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
			Number int `json:"Number"`
		} `json:"issue"`
		Repository struct {
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
		Comment struct {
			Body string `json:"body"`
		} `json:"comment"`
		PullRequest struct {
			URL string `json:"url"`
		} `json:"pull_request"`
	}
	err := json.Unmarshal(body, &message)
	if err != nil {
		return IssueComment{}, err
	}
	return IssueComment{
		IssueNumber:     message.Issue.Number,
		Comment:         message.Comment.Body,
		IsPullRequest:   message.PullRequest.URL != "",
		RepositoryOwner: message.Repository.Owner.Login,
		RepositoryName:  message.Repository.Name,
	}, nil
}

func areMACsEqual(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}
