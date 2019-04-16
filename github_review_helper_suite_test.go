package main_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/github"
	grh "github.com/salemove/github-review-helper"
	"github.com/salemove/github-review-helper/mocks"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	repositoryID         = 456
	repositoryOwner      = "salemove"
	repositoryName       = "github-review-helper"
	sshURL               = "git@github.com:salemove/github-review-helper.git"
	issueNumber          = 7
	arbitraryIssueAuthor = "author"
	arbitrarySHA         = "1afdea0acb09ff392fcdb89acfa9d7e9feac4bc1"
	numberOfGithubTries  = 4
)

var (
	repository = &github.Repository{
		ID: github.Int64(repositoryID),
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
	anyContext    = mock.MatchedBy(func(ctx context.Context) bool { return true })
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
			conf             grh.Config
			asyncOperationWg *sync.WaitGroup

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

			githubAPITryDeltas := make([]time.Duration, numberOfGithubTries)
			for i := range githubAPITryDeltas {
				if i == 0 {
					// Allow the first try to be synchronous
					githubAPITryDeltas[i] = 0
				} else {
					// Try every 1ms
					githubAPITryDeltas[i] = time.Millisecond
				}
			}
			conf = grh.Config{
				Secret:             "a-secret",
				GithubAPITryDeltas: githubAPITryDeltas,
			}

			asyncOperationWg = &sync.WaitGroup{}
			*handler = grh.CreateHandler(conf, *gitRepos, asyncOperationWg, *pullRequests,
				*repositories, *issues, *search)
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
			// The delay is set to 0 for tests. Wait for all of the operations
			// to finish to simplify test code.
			asyncOperationWg.Wait()
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
	commentMsg, err := json.Marshal(comment)
	if err != nil {
		panic(err)
	}
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
    "body": ` + string(commentMsg) + `
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

type commit struct {
	SHA, Message string
}

// arbitraryParentSHA is a SHA different from arbitrarySHA, to avoid cycles
// and other issues in the commit list.
const arbitraryParentSHA = "43c3c0c406518f3f326474f9e378027f86f27caf"

// githubCommits returns an ordered list of github commits, where the HEAD
// commit is the last one in the list.
var githubCommits = func(commitList ...commit) []*github.RepositoryCommit {
	githubCommitList := make([]*github.RepositoryCommit, len(commitList))
	for i, commitData := range commitList {
		githubCommitList[i] = &github.RepositoryCommit{
			SHA: github.String(commitData.SHA),
			Commit: &github.Commit{
				Message: github.String(commitData.Message),
			},
		}
		if i > 0 {
			githubCommitList[i].Parents = []github.Commit{
				{SHA: githubCommitList[i-1].SHA},
			}
		} else {
			githubCommitList[i].Parents = []github.Commit{
				{SHA: github.String(arbitraryParentSHA)},
			}
		}
	}
	return githubCommitList
}

var githubCommitsInMixedOrder = func(commitList ...commit) []*github.RepositoryCommit {
	githubCommitList := githubCommits(commitList...)
	if len(githubCommitList) > 1 {
		// swap 2 last elements
		lastIndex := len(githubCommitList) - 1
		githubCommitList[lastIndex], githubCommitList[lastIndex-1] = githubCommitList[lastIndex-1], githubCommitList[lastIndex]
	}
	return githubCommitList
}

var mockListCommits = func(commits []*github.RepositoryCommit, perPage int, repositoryOwner,
	repositoryName string, issueNumber int, pullRequests *mocks.PullRequests) {

	pageNumber := 1
	for {
		pageStartIndex := (pageNumber - 1) * perPage
		if len(commits) <= pageNumber*perPage {
			commitsOnThisPage := commits[pageStartIndex:]
			pullRequests.
				On("ListCommits", anyContext, repositoryOwner, repositoryName, issueNumber, &github.ListOptions{
					Page:    pageNumber,
					PerPage: 30,
				}).
				Return(commitsOnThisPage, &github.Response{}, noError)
			break
		}

		commitsOnThisPage := commits[pageStartIndex : pageNumber*perPage]
		pullRequests.
			On("ListCommits", anyContext, repositoryOwner, repositoryName, issueNumber, &github.ListOptions{
				Page:    pageNumber,
				PerPage: 30,
			}).
			Return(commitsOnThisPage, &github.Response{NextPage: pageNumber + 1}, noError)
		pageNumber++
	}
}
