package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type IssueComment struct {
	IssueID       string
	Comment       string
	IsPullRequest bool
}

func main() {
	conf := NewConfig()

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
		// TODO get the PR stuff and squash+push
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil))
}

func parseIssueComment(body []byte) (IssueComment, error) {
	var message struct {
		Issue struct {
			ID string `json:"id"`
		} `json:"issue"`
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
		IssueID:       message.Issue.ID,
		Comment:       message.Comment.Body,
		IsPullRequest: message.PullRequest.URL != "",
	}, nil
}

func areMACsEqual(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}
