package main_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-github/github"
	grh "github.com/salemove/github-review-helper"
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
			return IssueCommentEvent("!check")
		})

		Context("with GitHub request to list commits failing", func() {
			Context("with a 404", func() {
				BeforeEach(func() {
					resp, err := createGithubErrorResponse(http.StatusNotFound)
					pullRequests.
						On("ListCommits", repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
						Return(nil, resp, err)
				})

				It("fails with a gateway error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
				})

				It("tries multiple times", func() {
					handle()
					// +1 because of the initial attempt
					pullRequests.AssertNumberOfCalls(GinkgoT(), "ListCommits", grh.GetCommitsRetryLimit+1)
				})
			})

			Context("with a different error", func() {
				BeforeEach(func() {
					pullRequests.
						On("ListCommits", repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
						Return(nil, nil, errors.New("an error"))
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
					On("ListCommits", repositoryOwner, repositoryName, issueNumber, mock.AnythingOfType("*github.ListOptions")).
					Return([]*github.RepositoryCommit{
						&github.RepositoryCommit{
							Commit: &github.Commit{
								Message: github.String("Changing things"),
							},
						},
						&github.RepositoryCommit{
							Commit: &github.Commit{
								Message: github.String("Another casual commit"),
							},
						},
					}, &github.Response{}, nil)
				pullRequests.
					On("Get", repositoryOwner, repositoryName, issueNumber).
					Return(&github.PullRequest{
						Number: github.Int(issueNumber),
						Head: &github.PullRequestBranch{
							SHA:  github.String(commitRevision),
							Repo: headRepository,
						},
						Base: &github.PullRequestBranch{
							Repo: repository,
						},
					}, nil, nil)
			})

			It("reports success status to GitHub", func() {
				repositories.
					On("CreateStatus", *headRepository.Owner.Login, *headRepository.Name, commitRevision,
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
					Return([]*github.RepositoryCommit{
						&github.RepositoryCommit{
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
					Return([]*github.RepositoryCommit{
						&github.RepositoryCommit{
							Commit: &github.Commit{
								Message: github.String("fixup! Changing things\n\nOopsie. Forgot a thing"),
							},
						},
					}, &github.Response{}, nil)
				pullRequests.
					On("Get", repositoryOwner, repositoryName, issueNumber).
					Return(&github.PullRequest{
						Number: github.Int(issueNumber),
						Head: &github.PullRequestBranch{
							SHA:  github.String(commitRevision),
							Repo: headRepository,
						},
						Base: &github.PullRequestBranch{
							Repo: repository,
						},
					}, nil, nil)
			})

			It("reports pending squash status to GitHub", func() {
				repositories.
					On("CreateStatus", *headRepository.Owner.Login, *headRepository.Name, commitRevision,
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
