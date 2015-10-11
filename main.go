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
	"os/exec"
	"path/filepath"
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

	git := NewGit()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Failed to read the request's body")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		signature := r.Header.Get("X-Hub-Signature")
		hasSecret, err := hasSecret(body, signature, conf.Secret)
		if err != nil {
			log.Println("Failed to check the signature")
			http.Error(w, "Failed to check the signature", http.StatusInternalServerError)
			return
		}
		if !hasSecret {
			log.Println("Bad X-Hub-Signature")
			http.Error(w, "Bad X-Hub-Signature", http.StatusBadRequest)
			return
		}
		eventType := r.Header.Get("X-Github-Event")
		if eventType != "issue_comment" {
			w.Write([]byte("Not an event I understand. Ignoring."))
			return
		}
		issueComment, err := parseIssueComment(body)
		if err != nil {
			log.Println("Failed to parse the requests body")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		if !issueComment.IsPullRequest {
			w.Write([]byte("Not a PR. Ignoring."))
			return
		}
		if issueComment.Comment != "!squash" {
			w.Write([]byte("Not a command I understand. Ignoring."))
			return
		}
		pr, _, err := githubClient.PullRequests.Get(issueComment.Repository.Owner, issueComment.Repository.Name, issueComment.IssueNumber)
		if err != nil {
			log.Printf("Getting PR %s/%s#%d failed: %s\n", issueComment.Repository.Owner, issueComment.Repository.Name, issueComment.IssueNumber, err.Error())
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		log.Printf("Squashing %s that's going to be merged into %s", *pr.Head.Ref, *pr.Base.Ref)
		localRepoPath := filepath.Join(reposDir, issueComment.Repository.Owner, issueComment.Repository.Name)
		exists, err := exists(localRepoPath)
		if err != nil {
			log.Println("Failed to check if dir " + localRepoPath + " exists")
			http.Error(w, "Failed check if the repo is already checked out", http.StatusInternalServerError)
			return
		}
		if !exists {
			log.Printf("Cloning %s/%s into %s\n", issueComment.Repository.Owner, issueComment.Repository.Name, localRepoPath)
			if err = exec.Command("git", "clone", issueComment.Repository.URL, localRepoPath).Run(); err != nil {
				log.Println("The clone failed: ", err)
				http.Error(w, "Failed to clone the repo", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Fetching latest changes for %s/%s\n", issueComment.Repository.Owner, issueComment.Repository.Name)
			if err = exec.Command("git", "-C", localRepoPath, "fetch").Run(); err != nil {
				log.Println("The fetch failed: ", err)
				http.Error(w, "Failed to fetch the latest changes for the repo", http.StatusInternalServerError)
				return
			}
		}
		repo := git.Repo(localRepoPath)
		if err = repo.RebaseAutosquash(*pr.Base.SHA, *pr.Head.SHA); err != nil {
			log.Println("Failed to autosquash the commits with an interactive rebase: ", err)
			http.Error(w, "Failed to autosquash the commits with an interactive rebase", http.StatusInternalServerError)
			return
		}
		if err = exec.Command("git", "-C", localRepoPath, "push", "--force", "origin", "@:"+*pr.Head.Ref).Run(); err != nil {
			log.Println("Failed to push the squashed version: ", err)
			http.Error(w, "Failed to push the squashed version", http.StatusInternalServerError)
			return
		}
		w.Write([]byte{})
	})

	graceful.Run(fmt.Sprintf(":%d", conf.Port), 10*time.Second, mux)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
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
