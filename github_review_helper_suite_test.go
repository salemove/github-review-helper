package main_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/google/go-github/github"
	. "github.com/salemove/github-review-helper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	repositoryOwner = "salemove"
	repositoryName  = "github-review-helper"
	sshURL          = "git@github.com:salemove/github-review-helper.git"
	issueNumber     = 7
)

var (
	repository = &github.Repository{
		Owner: &github.User{
			Login: github.String(repositoryOwner),
		},
		Name:   github.String(repositoryName),
		SSHURL: github.String(sshURL),
	}
)

func TestGithubReviewHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	log.SetOutput(ioutil.Discard)
	RunSpecs(t, "GithubReviewHelper Suite")
}

type WebhookTestContext struct {
	RequestJSON      StringMemoizer
	Headers          StringMapMemoizer
	Handle           func()
	ResponseRecorder **httptest.ResponseRecorder
	Git              **MockGit
	PullRequests     **MockPullRequests
	Repositories     **MockRepositories
	Issues           **MockIssues
}

type WebhookTest func(WebhookTestContext)

var TestWebhookHandler = func(test WebhookTest) bool {
	Describe("webhook handler", func() {
		var (
			conf Config

			requestJSON = NewStringMemoizer(func() string {
				return ""
			})
			headers = NewStringMapMemoizer(func() map[string]string {
				return nil // nil is safe to read from, unsafe to write to
			})

			handler          = new(Handler)
			request          = new(*http.Request)
			responseRecorder = new(*httptest.ResponseRecorder)
			git              = new(*MockGit)
			pullRequests     = new(*MockPullRequests)
			repositories     = new(*MockRepositories)
			issues           = new(*MockIssues)
		)

		BeforeEach(func() {
			*git = new(MockGit)
			*pullRequests = new(MockPullRequests)
			*repositories = new(MockRepositories)
			*issues = new(MockIssues)

			*responseRecorder = httptest.NewRecorder()

			conf = Config{
				Secret: "a-secret",
			}
			*handler = CreateHandler(conf, *git, *pullRequests, *repositories, *issues)
		})

		JustBeforeEach(func() {
			data := []byte(requestJSON.Get())
			var err error
			*request, err = http.NewRequest("GET", "http://localhost/whatever", bytes.NewBuffer(data))
			Expect(err).NotTo(HaveOccurred())
			(*request).Header.Add("Content-Type", "application/json")
			(*request).Header.Add("Content-Length", strconv.Itoa(len(data)))

			mac := hmac.New(sha1.New, []byte(conf.Secret))
			mac.Write([]byte(requestJSON.Get()))
			sig := hex.EncodeToString(mac.Sum(nil))
			(*request).Header.Add("X-Hub-Signature", "sha1="+sig)

			for key, val := range headers.Get() {
				(*request).Header.Set(key, val)
			}

		})

		AfterEach(func() {
			(*git).AssertExpectations(GinkgoT())
			(*pullRequests).AssertExpectations(GinkgoT())
			(*repositories).AssertExpectations(GinkgoT())
			(*issues).AssertExpectations(GinkgoT())
		})

		var handle = func() {
			response := (*handler)(*responseRecorder, *request)
			response.WriteResponse(*responseRecorder)
		}

		test(WebhookTestContext{
			RequestJSON:      requestJSON,
			Headers:          headers,
			Handle:           handle,
			ResponseRecorder: responseRecorder,
			Git:              git,
			PullRequests:     pullRequests,
			Repositories:     repositories,
			Issues:           issues,
		})
	})

	// Return something, so that the function could be called in top level
	// scope with a `var _ =` assignment
	return true
}

var IssueCommentEvent = func(comment string) string {
	return `{
  "issue": {
    "number": ` + strconv.Itoa(issueNumber) + `,
    "pull_request": {
      "url": "https://api.github.com/repos/` + repositoryOwner + `/` + repositoryName + `/pulls/` + strconv.Itoa(issueNumber) + `"
    }
  },
  "comment": {
    "body": "` + comment + `"
  },
  "repository": {
    "name": "` + repositoryName + `",
    "owner": {
      "login": "` + repositoryOwner + `"
    },
    "ssh_url": "` + sshURL + `"
  }
}`
}

var PullRequestsEvent = func(action string) string {
	return `{
  "action": "` + action + `",
  "number": ` + strconv.Itoa(issueNumber) + `,
  "pull_request": {
    "url": "https://api.github.com/repos/` + repositoryOwner + `/` + repositoryName + `/pulls/` + strconv.Itoa(issueNumber) + `"
  },
  "repository": {
    "name": "` + repositoryName + `",
    "owner": {
      "login": "` + repositoryOwner + `"
    },
    "ssh_url": "` + sshURL + `"
  }
}`
}
