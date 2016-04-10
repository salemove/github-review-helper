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

var _ = TestWebhookHandler(func(context WebhookTestContext) {
	Describe("+1 comment", func() {
		var (
			handle      = context.Handle
			headers     = context.Headers
			requestJSON = context.RequestJSON

			responseRecorder *httptest.ResponseRecorder
			pullRequests     *MockPullRequests
			repositories     *MockRepositories
		)
		BeforeEach(func() {
			responseRecorder = *context.ResponseRecorder
			pullRequests = *context.PullRequests
			repositories = *context.Repositories
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
						On("CreateStatus", repositoryOwner, repositoryName, commitRevision,
							mock.MatchedBy(func(status *github.RepoStatus) bool {
								return *status.State == "success" && *status.Context == "review/peer"
							}),
						).
						Return(nil, nil, nil)

					handle()

					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})
		}

		headers.Is(func() map[string][]string {
			return map[string][]string{
				"X-Github-Event": []string{"issue_comment"},
			}
		})

		positiveTests := map[string]string{
			"with +1 at the beginning of the comment": "+1, awesome job!",
			"with +1 at the end of the comment":       "Looking good! +1",
			"with +1 in the middle of the comment":    "Good job! +1 PS: Don't forget to update that other thing.",
			"with +1 smiley":                          "Good job! :+1:",
		}
		for desc, comment := range positiveTests {
			Context(desc, func() {
				requestJSON.Is(func() string {
					return IssueCommentEvent(comment)
				})

				itMarksCommitPeerReviewed()
			})
		}

		Context("with +1 as a part of a number", func() {
			requestJSON.Is(func() string {
				return IssueCommentEvent("Wow +1832 -534 changes. Slow down!")
			})

			It("ignores the comment", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
			})
		})
	})
})
