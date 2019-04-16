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

		ForCollaborator(context, repositoryOwner, repositoryName, issueAuthor, func() {
			Context("with github request to add the label failing", func() {
				BeforeEach(func() {
					issues.
						On("AddLabelsToIssue", anyContext, repositoryOwner, repositoryName, issueNumber, []string{grh.MergingLabel}).
						Return(emptyResult, emptyResponse, errors.New("an error"))
				})

				It("fails with a gateway error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
				})
			})

			Context("with github request to add the label succeeding", func() {
				BeforeEach(func() {
					issues.
						On("AddLabelsToIssue", anyContext, repositoryOwner, repositoryName, issueNumber, []string{grh.MergingLabel}).
						Return(emptyResult, emptyResponse, noError)
				})

				Context("with fetching the PR failing", func() {
					BeforeEach(func() {
						pullRequests.
							On("Get", anyContext, repositoryOwner, repositoryName, issueNumber).
							Return(emptyResult, emptyResponse, errors.New("an error"))
					})

					It("fails with a gateway error", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
					})
				})

				Context("with the PR being already merged", func() {
					BeforeEach(func() {
						pullRequests.
							On("Get", anyContext, repositoryOwner, repositoryName, issueNumber).
							Return(&github.PullRequest{
								Merged: github.Bool(true),
							}, emptyResponse, noError)
					})

					It("removes the 'merging' label from the PR", func() {
						issues.
							On("RemoveLabelForIssue", anyContext, repositoryOwner, repositoryName, issueNumber, grh.MergingLabel).
							Return(emptyResponse, noError)

						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					})
				})

				Context("with the PR not being mergeable", func() {
					BeforeEach(func() {
						pullRequests.
							On("Get", anyContext, repositoryOwner, repositoryName, issueNumber).
							Return(&github.PullRequest{
								Merged:    github.Bool(false),
								Mergeable: github.Bool(false),
							}, emptyResponse, noError)
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
						User: &github.User{
							Login: github.String(issueAuthor),
						},
					}

					BeforeEach(func() {
						pullRequests.
							On("Get", anyContext, repositoryOwner, repositoryName, issueNumber).
							Return(pr, emptyResponse, noError)
					})

					Context("with combined state being failing", func() {
						BeforeEach(func() {
							repositories.
								On("GetCombinedStatus", anyContext, repositoryOwner, repositoryName, headSHA, mock.AnythingOfType("*github.ListOptions")).
								Return(&github.CombinedStatus{
									State: github.String("failing"),
								}, emptyResponse, noError)
						})

						It("succeeds", func() {
							handle()
							Expect(responseRecorder.Code).To(Equal(http.StatusOK))
						})
					})

					Context("with a pending squash status in paged combined status request", func() {
						BeforeEach(func() {
							repositories.
								On("GetCombinedStatus", anyContext, repositoryOwner, repositoryName, headSHA, &github.ListOptions{
									Page:    1,
									PerPage: 100,
								}).
								Return(&github.CombinedStatus{
									State: github.String("pending"),
									Statuses: []github.RepoStatus{
										{
											Context: github.String("jenkins/pr"),
											State:   github.String("success"),
										},
									},
								}, &github.Response{
									NextPage: 2,
								}, noError)
							repositories.
								On("GetCombinedStatus", anyContext, repositoryOwner, repositoryName, headSHA, &github.ListOptions{
									Page:    2,
									PerPage: 100,
								}).
								Return(&github.CombinedStatus{
									State: github.String("pending"),
									Statuses: []github.RepoStatus{
										{
											Context: github.String("review/squash"),
											State:   github.String("pending"),
										},
									},
								}, &github.Response{}, noError)
						})

						ItSquashesPR(context, pr)
					})

					Context("with combined state being success", func() {
						BeforeEach(func() {
							repositories.
								On("GetCombinedStatus", anyContext, repositoryOwner, repositoryName, headSHA, mock.AnythingOfType("*github.ListOptions")).
								Return(&github.CombinedStatus{
									State: github.String("success"),
								}, emptyResponse, noError)
						})

						ItMergesPR(context, pr)

						Context("with a '!merge this' comment", func() {
							requestJSON.Is(func() string {
								return IssueCommentEvent("!merge this", issueAuthor)
							})

							ItMergesPR(context, pr)
						})

						Context("with a '!merge this' comment with newlines", func() {
							requestJSON.Is(func() string {
								return IssueCommentEvent("!merge\n\nthis", issueAuthor)
							})

							ItMergesPR(context, pr)
						})
					})
				})
			})
		})
	})
})

func commentMentioning(user string) func(issueComment *github.IssueComment) bool {
	return func(issueComment *github.IssueComment) bool {
		return strings.Contains(*issueComment.Body, "@"+user)
	}
}

var ItMergesPR = func(context WebhookTestContext, pr *github.PullRequest) {
	var (
		handle = context.Handle

		responseRecorder *httptest.ResponseRecorder
		pullRequests     *mocks.PullRequests
		issues           *mocks.Issues
		gitRepos         *mocks.Repos

		issueAuthor string
		issueNumber int
		headRef     string
	)
	BeforeEach(func() {
		responseRecorder = *context.ResponseRecorder
		pullRequests = *context.PullRequests
		issues = *context.Issues
		gitRepos = *context.GitRepos

		issueAuthor = *pr.User.Login
		issueNumber = *pr.Number
		headRef = *pr.Head.Ref
	})

	Context("with merge failing with an unknown error", func() {
		BeforeEach(func() {
			additionalCommitMessage := ""
			pullRequests.
				On(
					"Merge",
					anyContext,
					repositoryOwner,
					repositoryName,
					issueNumber,
					additionalCommitMessage,
					noSquashOpts,
				).
				Return(emptyResult, emptyResponse, errors.New("an error")).
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
					anyContext,
					repositoryOwner,
					repositoryName,
					issueNumber,
					additionalCommitMessage,
					noSquashOpts,
				).
				Return(emptyResult, &github.Response{
					Response: resp,
				}, &github.ErrorResponse{
					Response: resp,
					Message:  "Merge conflict",
				})
		}

		BeforeEach(func() {
			mockMergeFailWithConflict()
		})

		Context("with removing the label failing", func() {
			BeforeEach(func() {
				issues.
					On("RemoveLabelForIssue", anyContext, repositoryOwner, repositoryName,
						issueNumber, grh.MergingLabel).
					Return(emptyResponse, errors.New("arbitrary error"))
			})

			It("notifies PR author and fails with a gateway error", func() {
				issues.
					On("CreateComment", anyContext, repositoryOwner, repositoryName,
						issueNumber, mock.MatchedBy(commentMentioning(issueAuthor))).
					Return(emptyResult, emptyResponse, noError)

				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
			})

			Context("with author notification failing", func() {
				BeforeEach(func() {
					issues.
						On("CreateComment", anyContext, repositoryOwner, repositoryName,
							issueNumber, mock.MatchedBy(commentMentioning(issueAuthor))).
						Return(emptyResult, emptyResponse, errors.New("arbitrary error"))
				})

				It("fails with a gateway error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
				})
			})
		})

		It("removes the 'merging' label and notifies the author", func() {
			issues.
				On("RemoveLabelForIssue", anyContext, repositoryOwner, repositoryName,
					issueNumber, grh.MergingLabel).
				Return(emptyResponse, noError)
			issues.
				On("CreateComment", anyContext, repositoryOwner, repositoryName,
					issueNumber, mock.MatchedBy(commentMentioning(issueAuthor))).
				Return(emptyResult, emptyResponse, noError)

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
					anyContext,
					repositoryOwner,
					repositoryName,
					issueNumber,
					additionalCommitMessage,
					noSquashOpts,
				).
				Return(emptyResult, &github.Response{
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
					anyContext,
					repositoryOwner,
					repositoryName,
					issueNumber,
					additionalCommitMessage,
					noSquashOpts,
				).
				Return(&github.PullRequestMergeResult{
					Merged: github.Bool(true),
				}, emptyResponse, noError).
				Once()
		})

		Context("with removing the 'merging' label failing", func() {
			BeforeEach(func() {
				issues.
					On("RemoveLabelForIssue", anyContext, repositoryOwner, repositoryName, issueNumber, grh.MergingLabel).
					Return(emptyResponse, errArbitrary)
			})

			It("fails with a gateway error", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
			})
		})

		Context("with removing the 'merging' label succeeding", func() {
			BeforeEach(func() {
				issues.
					On("RemoveLabelForIssue", anyContext, repositoryOwner, repositoryName, issueNumber, grh.MergingLabel).
					Return(emptyResponse, noError)
			})

			Context("with getting an updated git repository failing", func() {
				BeforeEach(func() {
					gitRepo := new(mocks.Repo)
					gitRepos.
						On("GetUpdatedRepo", sshURL, repositoryOwner, repositoryName).
						Return(gitRepo, errArbitrary)
				})

				It("fails with an internal error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
				})
			})

			Context("with getting an updated git repository succeeding", func() {
				var gitRepo *mocks.Repo

				BeforeEach(func() {
					gitRepo = new(mocks.Repo)
					gitRepos.
						On("GetUpdatedRepo", sshURL, repositoryOwner, repositoryName).
						Return(gitRepo, noError)
				})

				Context("with deleting the remote branch failing", func() {
					BeforeEach(func() {
						gitRepo.On("DeleteRemoteBranch", headRef).Return(errArbitrary)
					})

					It("fails with an internal error", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
					})
				})

				Context("with deleting the remote branch succeeding", func() {
					BeforeEach(func() {
						gitRepo.On("DeleteRemoteBranch", headRef).Return(noError)
					})

					It("returns 200 OK", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					})
				})
			})
		})
	})
}
