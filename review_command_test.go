package main_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-github/github"
	. "github.com/salemove/github-review-helper"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = TestWebhookHandler(func(requestJSON StringMemoizer, headers StringMapMemoizer, handle func(),
	_responseRecorder **httptest.ResponseRecorder, _git **MockGit, _pullRequests **MockPullRequests,
	_repositories **MockRepositories, _issues **MockIssues) {
	Describe("+1 comment", func() {
		var (
			responseRecorder *httptest.ResponseRecorder
			pullRequests     *MockPullRequests
			repositories     *MockRepositories
		)
		BeforeEach(func() {
			responseRecorder = *_responseRecorder
			pullRequests = *_pullRequests
			repositories = *_repositories
		})

		var itMarksCommitPeerReviewed = func() {
			var commitRevision = "1235"
			Context("with GitHub request failing", func() {
				BeforeEach(func() {
					pullRequests.
						On("Get", repositoryOwner, repositoryName, issueNumber).
						Return(nil, nil, errors.New("an error"))
				})

				It("fails with a gateway error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
				})
			})

			Context("with GitHub request succeeding", func() {
				BeforeEach(func() {
					pullRequests.
						On("Get", repositoryOwner, repositoryName, issueNumber).
						Return(&github.PullRequest{
							Head: &github.PullRequestBranch{
								SHA:  github.String(commitRevision),
								Repo: repository,
							},
						}, nil, nil)
				})

				It("reports the status", func() {
					repositories.
						On("CreateStatus", repositoryOwner, repositoryName, commitRevision, mock.AnythingOfType("*github.RepoStatus")).
						Return(nil, nil, nil)

					handle()

					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					status := repositories.Calls[0].Arguments.Get(3).(*github.RepoStatus)
					Expect(*status.State).To(Equal("success"))
					Expect(*status.Context).To(Equal("review/peer"))
				})
			})
		}

		headers.Is(func() map[string][]string {
			return map[string][]string{
				"X-Github-Event": []string{"issue_comment"},
			}
		})

		Context("with +1 at the beginning of the comment", func() {
			requestJSON.Is(func() string {
				return IssueCommentEvent("+1, awesome job!")
			})

			itMarksCommitPeerReviewed()
		})

		Context("with +1 at the end of the comment", func() {
			requestJSON.Is(func() string {
				return IssueCommentEvent("Looking good! +1")
			})

			itMarksCommitPeerReviewed()
		})
	})
})
