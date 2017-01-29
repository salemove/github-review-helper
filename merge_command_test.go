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

var noSquashOpts = &github.PullRequestOptions{MergeMethod: "merge"}

var _ = TestWebhookHandler(func(context WebhookTestContext) {
	Describe("!merge comment", func() {
		var (
			handle      = context.Handle
			headers     = context.Headers
			requestJSON = context.RequestJSON

			responseRecorder *httptest.ResponseRecorder
			pullRequests     *mocks.PullRequests
			repositories     *mocks.Repositories
			issues           *mocks.Issues

			issueAuthor = "procoder"
		)
		BeforeEach(func() {
			responseRecorder = *context.ResponseRecorder
			pullRequests = *context.PullRequests
			repositories = *context.Repositories
			issues = *context.Issues
		})

		headers.Is(func() map[string]string {
			return map[string]string{
				"X-Github-Event": "issue_comment",
			}
		})
		requestJSON.Is(func() string {
			return IssueCommentEvent("!merge", issueAuthor)
		})

		Context("with github request to add the label failing", func() {
			BeforeEach(func() {
				issues.
					On("AddLabelsToIssue", repositoryOwner, repositoryName, issueNumber, []string{grh.MergingLabel}).
					Return(nil, nil, errors.New("an error"))
			})

			It("fails with a gateway error", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
			})
		})

		Context("with github request to add the label succeeding", func() {
			BeforeEach(func() {
				issues.
					On("AddLabelsToIssue", repositoryOwner, repositoryName, issueNumber, []string{grh.MergingLabel}).
					Return(nil, nil, nil)
			})

			Context("with fetching the PR failing", func() {
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

			Context("with the PR being already merged", func() {
				BeforeEach(func() {
					pullRequests.
						On("Get", repositoryOwner, repositoryName, issueNumber).
						Return(&github.PullRequest{
							Merged: github.Bool(true),
						}, nil, nil)
				})

				It("removes the 'merging' label from the PR", func() {
					issues.
						On("RemoveLabelForIssue", repositoryOwner, repositoryName, issueNumber, grh.MergingLabel).
						Return(nil, nil)

					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})

			Context("with the PR not being mergeable", func() {
				BeforeEach(func() {
					pullRequests.
						On("Get", repositoryOwner, repositoryName, issueNumber).
						Return(&github.PullRequest{
							Merged:    github.Bool(false),
							Mergeable: github.Bool(false),
						}, nil, nil)
				})

				It("succeeds", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})

			Context("with the PR being mergeable", func() {
				headSHA := "1235"
				pr := &github.PullRequest{
					Number:    github.Int(issueNumber),
					Merged:    github.Bool(false),
					Mergeable: github.Bool(true),
					Base: &github.PullRequestBranch{
						SHA:  github.String("1234"),
						Ref:  github.String("master"),
						Repo: repository,
					},
					Head: &github.PullRequestBranch{
						SHA:  github.String(headSHA),
						Ref:  github.String("feature"),
						Repo: repository,
					},
				}

				BeforeEach(func() {
					pullRequests.
						On("Get", repositoryOwner, repositoryName, issueNumber).
						Return(pr, nil, nil)
				})

				Context("with combined state being failing", func() {
					BeforeEach(func() {
						repositories.
							On("GetCombinedStatus", repositoryOwner, repositoryName, headSHA, mock.AnythingOfType("*github.ListOptions")).
							Return(&github.CombinedStatus{
								State: github.String("failing"),
							}, &github.Response{}, nil)
					})

					It("succeeds", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					})
				})

				Context("with a pending squash status in paged combined status request", func() {
					BeforeEach(func() {
						repositories.
							On("GetCombinedStatus", repositoryOwner, repositoryName, headSHA, &github.ListOptions{
								Page:    1,
								PerPage: 100,
							}).
							Return(&github.CombinedStatus{
								State: github.String("pending"),
								Statuses: []github.RepoStatus{
									github.RepoStatus{
										Context: github.String("jenkins/pr"),
										State:   github.String("success"),
									},
								},
							}, &github.Response{
								NextPage: 2,
							}, nil)
						repositories.
							On("GetCombinedStatus", repositoryOwner, repositoryName, headSHA, &github.ListOptions{
								Page:    2,
								PerPage: 100,
							}).
							Return(&github.CombinedStatus{
								State: github.String("pending"),
								Statuses: []github.RepoStatus{
									github.RepoStatus{
										Context: github.String("review/squash"),
										State:   github.String("pending"),
									},
								},
							}, &github.Response{}, nil)
					})

					ItSquashesPR(context, pr)
				})

				Context("with combined state being success", func() {
					BeforeEach(func() {
						repositories.
							On("GetCombinedStatus", repositoryOwner, repositoryName, headSHA, mock.AnythingOfType("*github.ListOptions")).
							Return(&github.CombinedStatus{
								State: github.String("success"),
							}, &github.Response{}, nil)
					})

					ItMergesPR(context, issueAuthor, issueNumber)
				})
			})
		})
	})
})

var ItMergesPR = func(context WebhookTestContext, issueAuthor string, issueNumber int) {
	var (
		handle = context.Handle

		responseRecorder *httptest.ResponseRecorder
		pullRequests     *mocks.PullRequests
		issues           *mocks.Issues
	)
	BeforeEach(func() {
		responseRecorder = *context.ResponseRecorder
		pullRequests = *context.PullRequests
		issues = *context.Issues
	})

	Context("with merge failing with an unknown error", func() {
		BeforeEach(func() {
			additionalCommitMessage := ""
			pullRequests.
				On(
					"Merge",
					repositoryOwner,
					repositoryName,
					issueNumber,
					additionalCommitMessage,
					noSquashOpts,
				).
				Return(nil, nil, errors.New("an error")).
				Once()
		})

		It("fails with a gateway error", func() {
			handle()
			Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
		})
	})

	Context("with merge failing due to a merge conflict", func() {
		mockMergeFailWithConflict := func() *mock.Call {
			additionalCommitMessage := ""
			resp := &http.Response{
				StatusCode: http.StatusConflict,
			}
			return pullRequests.
				On(
					"Merge",
					repositoryOwner,
					repositoryName,
					issueNumber,
					additionalCommitMessage,
					noSquashOpts,
				).
				Return(nil, &github.Response{
					Response: resp,
				}, &github.ErrorResponse{
					Response: resp,
					Message:  "Merge conflict",
				})
		}

		matchIssueCommentContainingAuthorMention := mock.MatchedBy(func(issueComment *github.IssueComment) bool {
			return strings.Contains(*issueComment.Body, "@"+issueAuthor)
		})

		BeforeEach(func() {
			mockMergeFailWithConflict()
		})

		Context("with removing the label failing", func() {
			BeforeEach(func() {
				issues.
					On("RemoveLabelForIssue", repositoryOwner, repositoryName,
						issueNumber, grh.MergingLabel).
					Return(nil, errors.New("arbitrary error"))
			})

			It("notifies PR author and fails with a gateway error", func() {
				issues.
					On("CreateComment", repositoryOwner, repositoryName,
						issueNumber, matchIssueCommentContainingAuthorMention).
					Return(nil, nil, nil)

				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
			})

			Context("with author notification failing", func() {
				BeforeEach(func() {
					issues.
						On("CreateComment", repositoryOwner, repositoryName,
							issueNumber, matchIssueCommentContainingAuthorMention).
						Return(nil, nil, errors.New("arbitrary error"))
				})

				It("fails with a gateway error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
				})
			})
		})

		It("removes the 'merging' label and notifies the author", func() {
			issues.
				On("RemoveLabelForIssue", repositoryOwner, repositoryName,
					issueNumber, grh.MergingLabel).
				Return(nil, nil)
			issues.
				On("CreateComment", repositoryOwner, repositoryName,
					issueNumber, matchIssueCommentContainingAuthorMention).
				Return(nil, nil, nil)

			handle()

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		})
	})

	Context("with merge failing, because PR not mergeable", func() {
		BeforeEach(func() {
			additionalCommitMessage := ""
			resp := &http.Response{
				StatusCode: http.StatusMethodNotAllowed,
			}
			pullRequests.
				On(
					"Merge",
					repositoryOwner,
					repositoryName,
					issueNumber,
					additionalCommitMessage,
					noSquashOpts,
				).
				Return(nil, &github.Response{
					Response: resp,
				}, &github.ErrorResponse{
					Response: resp,
					Message:  "Pull Request is not mergeable",
				}).
				Once()
		})

		It("fails with a gateway error", func() {
			handle()
			Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
		})
	})

	Context("with merge succeeding", func() {
		BeforeEach(func() {
			additionalCommitMessage := ""
			pullRequests.
				On(
					"Merge",
					repositoryOwner,
					repositoryName,
					issueNumber,
					additionalCommitMessage,
					noSquashOpts,
				).
				Return(&github.PullRequestMergeResult{
					Merged: github.Bool(true),
				}, nil, nil).
				Once()
		})

		It("removes the 'merging' label from the PR after the merge", func() {
			issues.
				On("RemoveLabelForIssue", repositoryOwner, repositoryName, issueNumber, grh.MergingLabel).
				Return(nil, nil)

			handle()
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		})
	})
}
