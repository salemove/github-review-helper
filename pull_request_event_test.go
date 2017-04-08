package main_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/go-github/github"
	grh "github.com/salemove/github-review-helper"
	"github.com/salemove/github-review-helper/mocks"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var createGithubErrorResponse = func(statusCode int) (*github.Response, *github.ErrorResponse) {
	httpResponse := &http.Response{
		StatusCode: statusCode,
		Request:    &http.Request{},
	}
	return &github.Response{Response: httpResponse}, &github.ErrorResponse{Response: httpResponse}
}

var _ = TestWebhookHandler(func(context WebhookTestContext) {
	Describe("pull_request event", func() {
		var (
			handle      = context.Handle
			headers     = context.Headers
			requestJSON = context.RequestJSON

			responseRecorder *httptest.ResponseRecorder
			pullRequests     *mocks.PullRequests
			repositories     *mocks.Repositories
		)
		BeforeEach(func() {
			responseRecorder = *context.ResponseRecorder
			pullRequests = *context.PullRequests
			repositories = *context.Repositories
		})

		var pullRequestHeadSHA = "1235"
		var headRepository = grh.Repository{
			Owner: "other",
			Name:  "github-review-helper-fork",
			URL:   "git@github.com:other/github-review-helper-fork.git",
		}

		headers.Is(func() map[string]string {
			return map[string]string{
				"X-Github-Event": "pull_request",
			}
		})

		Context("with the PR being closed", func() {
			requestJSON.Is(func() string {
				return PullRequestEvent("closed", pullRequestHeadSHA, headRepository)
			})

			It("succeeds with 'ignored' response", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
			})
		})

		Context("with the PR being synchronized", func() {
			requestJSON.Is(func() string {
				return PullRequestEvent("synchronize", pullRequestHeadSHA, headRepository)
			})

			Context("with GitHub request to list commits failing", func() {
				Context("with a 404", func() {
					BeforeEach(func() {
						resp, err := createGithubErrorResponse(http.StatusNotFound)
						pullRequests.
							On("ListCommits", anyContext, repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
							Return(emptyResult, resp, err)
					})

					// Responds with 200, because the operation will be retries asynchronously
					It("responds with 200 OK", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					})

					It("tries the configured amount of times", func() {
						handle()
						pullRequests.AssertNumberOfCalls(GinkgoT(), "ListCommits", numberOfGithubTries)
					})
				})

				Context("with a different error", func() {
					BeforeEach(func() {
						pullRequests.
							On("ListCommits", anyContext, repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
							Return(emptyResult, emptyResponse, errors.New("an error"))
					})

					It("fails with a gateway error", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
					})

					It("tries once", func() {
						handle()
						pullRequests.AssertNumberOfCalls(GinkgoT(), "ListCommits", 1)
					})
				})
			})

			Context("with the head commit differing from the head SHA in the event", func() {
				headCommitRevision := strings.Replace(pullRequestHeadSHA, "123", "223", 1)

				BeforeEach(func() {
					pullRequests.
						On("ListCommits", anyContext, repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
						Return(githubCommits(
							commit{arbitrarySHA, "Changing things"},
							commit{headCommitRevision, "Another casual commit"},
						), emptyResponse, noError)
				})

				// Responds with 200, because the operation will be retries asynchronously
				It("responds with 200 OK", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})

				It("tries the configured amount of times", func() {
					handle()
					pullRequests.AssertNumberOfCalls(GinkgoT(), "ListCommits", numberOfGithubTries)
				})
			})

			Context("with list of commits from GitHub NOT including fixup commits", func() {
				BeforeEach(func() {
					pullRequests.
						On("ListCommits", anyContext, repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
						Return(githubCommits(
							commit{arbitrarySHA, "Changing things"},
							commit{pullRequestHeadSHA, "Another casual commit"},
						), emptyResponse, noError)
				})

				It("reports success status to GitHub", func() {
					repositories.
						On("CreateStatus", anyContext, headRepository.Owner, headRepository.Name, pullRequestHeadSHA,
							mock.MatchedBy(func(status *github.RepoStatus) bool {
								return *status.State == "success" && *status.Context == "review/squash"
							}),
						).
						Return(emptyResult, emptyResponse, noError)

					handle()

					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})

			Context("with paged list of commits from GitHub including fixup commits", func() {
				BeforeEach(func() {
					perPage := 1
					commits := githubCommits(
						commit{arbitrarySHA, "Changing things"},
						commit{pullRequestHeadSHA, "fixup! Changing things\n\nOopsie. Forgot a thing"},
					)
					mockListCommits(commits, perPage, repositoryOwner, repositoryName, issueNumber, pullRequests)
				})

				It("reports pending squash status to GitHub", func() {
					repositories.
						On("CreateStatus", anyContext, headRepository.Owner, headRepository.Name, pullRequestHeadSHA,
							mock.MatchedBy(func(status *github.RepoStatus) bool {
								return *status.State == "pending" && *status.Context == "review/squash"
							}),
						).
						Return(emptyResult, emptyResponse, noError)

					handle()

					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})

			Context("with paged list of commits in mixed order from GitHub including fixup commits", func() {
				BeforeEach(func() {
					perPage := 1
					commits := githubCommitsInMixedOrder(
						commit{arbitrarySHA, "Changing things"},
						commit{pullRequestHeadSHA, "fixup! Changing things\n\nOopsie. Forgot a thing"},
					)
					mockListCommits(commits, perPage, repositoryOwner, repositoryName, issueNumber, pullRequests)
				})

				It("reports pending squash status to GitHub", func() {
					repositories.
						On("CreateStatus", anyContext, headRepository.Owner, headRepository.Name, pullRequestHeadSHA,
							mock.MatchedBy(func(status *github.RepoStatus) bool {
								return *status.State == "pending" && *status.Context == "review/squash"
							}),
						).
						Return(emptyResult, emptyResponse, noError)

					handle()

					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})
		})
	})
})
