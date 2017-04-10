package main_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/google/go-github/github"
	"github.com/salemove/github-review-helper/mocks"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = TestWebhookHandler(func(context WebhookTestContext) {
	Describe("!check comment", func() {
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

		var commitRevision = "1235"
		var headRepository = &github.Repository{
			Owner: &github.User{
				Login: github.String("other"),
			},
			Name:   github.String("github-review-helper-fork"),
			SSHURL: github.String("git@github.com:other/github-review-helper-fork.git"),
		}

		headers.Is(func() map[string]string {
			return map[string]string{
				"X-Github-Event": "issue_comment",
			}
		})
		requestJSON.Is(func() string {
			return IssueCommentEvent("!check", arbitraryIssueAuthor)
		})

		ForCollaborator(context, repositoryOwner, repositoryName, arbitraryIssueAuthor, func() {
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
						resp, err := createGithubErrorResponse(http.StatusInternalServerError)
						pullRequests.
							On("ListCommits", anyContext, repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
							Return(emptyResult, resp, err)
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

			Context("with list of commits from GitHub NOT including fixup commits", func() {
				BeforeEach(func() {
					pullRequests.
						On("ListCommits", anyContext, repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
						Return(githubCommits(
							commit{arbitrarySHA, "Changing things"},
							commit{commitRevision, "Another casual commit"},
						), &github.Response{}, noError)
					pullRequests.
						On("Get", anyContext, repositoryOwner, repositoryName, issueNumber).
						Return(&github.PullRequest{
							Number: github.Int(issueNumber),
							Head: &github.PullRequestBranch{
								SHA:  github.String(commitRevision),
								Repo: headRepository,
							},
							Base: &github.PullRequestBranch{
								Repo: repository,
							},
						}, emptyResponse, noError)
				})

				It("reports success status to GitHub", func() {
					repositories.
						On("CreateStatus", anyContext, *headRepository.Owner.Login, *headRepository.Name, commitRevision,
							mock.MatchedBy(func(status *github.RepoStatus) bool {
								return *status.State == "success" && *status.Context == "review/squash"
							}),
						).
						Return(emptyResult, emptyResult, noError)

					handle()

					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})

			Context("with paged list of commits from GitHub including fixup commits", func() {
				BeforeEach(func() {
					perPage := 1
					commits := githubCommits(
						commit{arbitrarySHA, "Changing things"},
						commit{commitRevision, "fixup! Changing things\n\nOopsie. Forgot a thing"},
					)
					mockListCommits(commits, perPage, repositoryOwner, repositoryName, issueNumber, pullRequests)
					pullRequests.
						On("Get", anyContext, repositoryOwner, repositoryName, issueNumber).
						Return(&github.PullRequest{
							Number: github.Int(issueNumber),
							Head: &github.PullRequestBranch{
								SHA:  github.String(commitRevision),
								Repo: headRepository,
							},
							Base: &github.PullRequestBranch{
								Repo: repository,
							},
						}, emptyResponse, noError)
				})

				It("reports pending squash status to GitHub", func() {
					repositories.
						On("CreateStatus", anyContext, *headRepository.Owner.Login, *headRepository.Name, commitRevision,
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
