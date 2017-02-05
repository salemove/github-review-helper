package main_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	grh "github.com/salemove/github-review-helper"
	"github.com/salemove/github-review-helper/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	repositoryOwner      = "salemove"
	repositoryName       = "github-review-helper"
	sshURL               = "git@github.com:salemove/github-review-helper.git"
	issueNumber          = 7
	arbitraryIssueAuthor = "author"
)

var (
	repository = &github.Repository{
		Owner: &github.User{
			Login: github.String(repositoryOwner),
		},
		Name:   github.String(repositoryName),
		SSHURL: github.String(sshURL),
	}
	emptyResult   = (interface{})(nil)
	emptyResponse = &github.Response{Response: &http.Response{}}
	noError       = (error)(nil)
	errArbitrary  = errors.New("GitHub is down. Or something.")
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
	GitRepos         **mocks.Repos
	PullRequests     **mocks.PullRequests
	Repositories     **mocks.Repositories
	Issues           **mocks.Issues
	Search           **mocks.Search
}

type WebhookTest func(WebhookTestContext)

var TestWebhookHandler = func(test WebhookTest) bool {
	Describe("webhook handler", func() {
		var (
			conf grh.Config

			requestJSON = NewStringMemoizer(func() string {
				return ""
			})
			headers = NewStringMapMemoizer(func() map[string]string {
				return nil // nil is safe to read from, unsafe to write to
			})

			handler          = new(grh.Handler)
			request          = new(*http.Request)
			responseRecorder = new(*httptest.ResponseRecorder)
			gitRepos         = new(*mocks.Repos)
			pullRequests     = new(*mocks.PullRequests)
			repositories     = new(*mocks.Repositories)
			issues           = new(*mocks.Issues)
			search           = new(*mocks.Search)
		)

		BeforeEach(func() {
			*gitRepos = new(mocks.Repos)
			*pullRequests = new(mocks.PullRequests)
			*repositories = new(mocks.Repositories)
			*issues = new(mocks.Issues)
			*search = new(mocks.Search)

			*responseRecorder = httptest.NewRecorder()

			conf = grh.Config{
				Secret: "a-secret",
			}
			*handler = grh.CreateHandler(conf, *gitRepos, *pullRequests, *repositories, *issues, *search)
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
			(*gitRepos).AssertExpectations(GinkgoT())
			(*pullRequests).AssertExpectations(GinkgoT())
			(*repositories).AssertExpectations(GinkgoT())
			(*issues).AssertExpectations(GinkgoT())
			(*search).AssertExpectations(GinkgoT())
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
			GitRepos:         gitRepos,
			PullRequests:     pullRequests,
			Repositories:     repositories,
			Issues:           issues,
			Search:           search,
		})
	})

	// Return something, so that the function could be called in top level
	// scope with a `var _ =` assignment
	return true
}

var IssueCommentEvent = func(comment, issueAuthor string) string {
	return `{
  "issue": {
    "number": ` + strconv.Itoa(issueNumber) + `,
    "pull_request": {
      "url": "https://api.github.com/repos/` + repositoryOwner + `/` + repositoryName + `/pulls/` + strconv.Itoa(issueNumber) + `"
    },
    "user": {
      "login": "` + issueAuthor + `"
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

var PullRequestEvent = func(action, headSHA string, headRepository grh.Repository) string {
	return `{
  "action": "` + action + `",
  "number": ` + strconv.Itoa(issueNumber) + `,
  "pull_request": {
    "url": "https://api.github.com/repos/` + repositoryOwner + `/` + repositoryName + `/pulls/` + strconv.Itoa(issueNumber) + `",
    "head": {
      "sha": "` + headSHA + `",
      "repo": {
        "name": "` + headRepository.Name + `",
        "owner": {
          "login": "` + headRepository.Owner + `"
        },
        "ssh_url": "` + headRepository.URL + `"
      }
    },
    "user": {
      "login": "` + arbitraryIssueAuthor + `"
    }
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

var createStatusEvent = func(sha, state string, branches []grh.Branch) string {
	branchSHAs := make([]string, len(branches))
	for i, branch := range branches {
		branchSHAs[i] = branch.SHA
	}
	return `{
  "sha": "` + sha + `",
  "state": "` + state + `",
  "branches": [
    {
      "commit": {
        "sha": "` + strings.Join(branchSHAs, `"
      }
    },
    {
      "commit": {
        "sha": "`) + `"
      }
    }
  ],
  "repository": {
    "name": "` + repositoryName + `",
    "owner": {
      "login": "` + repositoryOwner + `"
    },
    "ssh_url": "` + sshURL + `"
  }
}`
}
