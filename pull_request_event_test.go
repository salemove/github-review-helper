package main_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-github/github"
	"github.com/salemove/github-review-helper/mocks"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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

		headers.Is(func() map[string]string {
			return map[string]string{
				"X-Github-Event": "pull_request",
			}
		})

		Context("with the PR being closed", func() {
			requestJSON.Is(func() string {
				return PullRequestsEvent("closed")
			})

			It("succeeds with 'ignored' response", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
			})
		})

		Context("with the PR being synchronized", func() {
			var commitRevision = "1235"

			requestJSON.Is(func() string {
				return PullRequestsEvent("synchronize")
			})

			Context("with GitHub request to list commits failing", func() {
				BeforeEach(func() {
					pullRequests.
						On("ListCommits", repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
						Return(nil, nil, errors.New("an error"))
				})

				It("fails with a gateway error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
				})
			})

			Context("with list of commits from GitHub NOT including fixup commits", func() {
				BeforeEach(func() {
					pullRequests.
						On("ListCommits", repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
						Return([]github.RepositoryCommit{
							github.RepositoryCommit{
								Commit: &github.Commit{
									Message: github.String("Changing things"),
								},
							},
							github.RepositoryCommit{
								Commit: &github.Commit{
									Message: github.String("Another casual commit"),
								},
							},
						}, &github.Response{}, nil)
					pullRequests.
						On("Get", repositoryOwner, repositoryName, issueNumber).
						Return(&github.PullRequest{
							Head: &github.PullRequestBranch{
								SHA:  github.String(commitRevision),
								Repo: repository,
							},
						}, nil, nil)
				})

				It("reports success status to GitHub", func() {
					repositories.
						On("CreateStatus", repositoryOwner, repositoryName, commitRevision,
							mock.MatchedBy(func(status *github.RepoStatus) bool {
								return *status.State == "success" && *status.Context == "review/squash"
							}),
						).
						Return(nil, nil, nil)

					handle()

					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})

			Context("with paged list of commits from GitHub including fixup commits", func() {
				BeforeEach(func() {
					pullRequests.
						On("ListCommits", repositoryOwner, repositoryName, issueNumber, &github.ListOptions{
							Page:    1,
							PerPage: 30,
						}).
						Return([]github.RepositoryCommit{
							github.RepositoryCommit{
								Commit: &github.Commit{
									Message: github.String("Changing things"),
								},
							},
						}, &github.Response{
							NextPage: 2,
						}, nil)
					pullRequests.
						On("ListCommits", repositoryOwner, repositoryName, issueNumber, &github.ListOptions{
							Page:    2,
							PerPage: 30,
						}).
						Return([]github.RepositoryCommit{
							github.RepositoryCommit{
								Commit: &github.Commit{
									Message: github.String("fixup! Changing things\n\nOopsie. Forgot a thing"),
								},
							},
						}, &github.Response{}, nil)
					pullRequests.
						On("Get", repositoryOwner, repositoryName, issueNumber).
						Return(&github.PullRequest{
							Head: &github.PullRequestBranch{
								SHA:  github.String(commitRevision),
								Repo: repository,
							},
						}, nil, nil)
				})

				It("reports pending squash status to GitHub", func() {
					repositories.
						On("CreateStatus", repositoryOwner, repositoryName, commitRevision,
							mock.MatchedBy(func(status *github.RepoStatus) bool {
								return *status.State == "pending" && *status.Context == "review/squash"
							}),
						).
						Return(nil, nil, nil)

					handle()

					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})
		})
	})
})
