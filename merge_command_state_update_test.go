package main_test

import (
	"errors"
	"fmt"
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
	mockSHA := "c9b5e1096a18765a14f6fb295c585efd40487a24"

	Describe("status update", func() {
		var (
			handle      = context.Handle
			headers     = context.Headers
			requestJSON = context.RequestJSON

			responseRecorder *httptest.ResponseRecorder
			pullRequests     *mocks.PullRequests
			issues           *mocks.Issues
			search           *mocks.Search
			gitRepos         *mocks.Repos
		)
		BeforeEach(func() {
			responseRecorder = *context.ResponseRecorder
			pullRequests = *context.PullRequests
			issues = *context.Issues
			search = *context.Search
			gitRepos = *context.GitRepos
		})

		headers.Is(func() map[string]string {
			return map[string]string{
				"X-Github-Event": "status",
			}
		})

		for _, badStatus := range []string{"pending", "failure", "error"} {
			Context("with "+badStatus+" status", func() {
				branches := []grh.Branch{{
					SHA: mockSHA,
				}}

				requestJSON.Is(func() string {
					return createStatusEvent(mockSHA, badStatus, branches)
				})

				It("returns 200 OK", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})
		}

		Context("with success status", func() {
			status := "success"

			Context("when updating a commit that is not a branch's head", func() {
				otherSHA := "4eaf26faa8819ab5aee991461b8c4fff41778f41"
				branches := []grh.Branch{{
					SHA: otherSHA,
				}}

				requestJSON.Is(func() string {
					return createStatusEvent(mockSHA, status, branches)
				})

				It("returns 200 OK", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})

			Context("when updating a commit that is a branch's head", func() {
				branches := []grh.Branch{{
					SHA: mockSHA,
				}}

				requestJSON.Is(func() string {
					return createStatusEvent(mockSHA, status, branches)
				})

				mockSearchQuery := func(pageNr int) *mock.Call {
					searchQuery := fmt.Sprintf("%s label:\"%s\" is:open repo:%s/%s status:success",
						mockSHA, grh.MergingLabel, repositoryOwner, repositoryName)
					return search.
						On("Issues", anyContext, searchQuery, mock.MatchedBy(func(searchOptions *github.SearchOptions) bool {
							return searchOptions.Page == pageNr
						}))
				}

				Context("with issue search failing", func() {
					BeforeEach(func() {
						mockSearchQuery(1).Return(emptyResult, emptyResponse, errors.New("arbitrary error"))
					})

					It("fails with a gateway error", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
					})

					It("tries once", func() {
						handle()
						search.AssertNumberOfCalls(GinkgoT(), "Issues", 1)
					})
				})

				Context("with issue search return 0 PRs", func() {
					BeforeEach(func() {
						searchResult := &github.IssuesSearchResult{
							Total:  github.Int(0),
							Issues: []github.Issue{},
						}
						mockSearchQuery(1).Return(searchResult, &github.Response{}, noError)
					})

					It("returns 200 OK", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					})

					It("tries the configured amount of times", func() {
						handle()
						search.AssertNumberOfCalls(GinkgoT(), "Issues", numberOfGithubTries)
					})
				})

				Context("with issue search returning a PR", func() {
					userName := "bestcoder"
					issueNumber := 7331

					BeforeEach(func() {
						searchResult := &github.IssuesSearchResult{
							Total: github.Int(1),
							Issues: []github.Issue{{
								Number: github.Int(issueNumber),
								User: &github.User{
									Login: github.String(userName),
								},
							}},
						}
						mockSearchQuery(1).Return(searchResult, &github.Response{}, noError)
					})

					Context("with GitHub API request for that PR failing", func() {
						BeforeEach(func() {
							pullRequests.
								On("Get", anyContext, repositoryOwner, repositoryName, issueNumber).
								Return(emptyResult, emptyResponse, errArbitrary)
						})

						It("fails with a gateway error", func() {
							handle()
							Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
						})

						It("tries once", func() {
							handle()
							search.AssertNumberOfCalls(GinkgoT(), "Issues", 1)
						})
					})

					Context("with GitHub API request for that PR succeeding", func() {
						pr := &github.PullRequest{
							Number: github.Int(issueNumber),
							Base: &github.PullRequestBranch{
								Ref:  github.String("master"),
								Repo: repository,
							},
							Head: &github.PullRequestBranch{
								Ref:  github.String("feature"),
								Repo: repository,
							},
							User: &github.User{
								Login: github.String(userName),
							},
						}

						BeforeEach(func() {
							pullRequests.
								On("Get", anyContext, repositoryOwner, repositoryName, issueNumber).
								Return(pr, emptyResponse, noError)
						})

						ItMergesPR(context, pr)
					})
				})

				Context("with issue search returning 2 PRs", func() {
					firstIssueNumber := 561
					secondIssueNumber := 562
					firstAuthor := "me"
					secondAuthor := "you"

					expectMerge := func(number int, author string) {
						// Fetch PR
						headRef := "feature"
						pr := &github.PullRequest{
							Number: github.Int(number),
							Base: &github.PullRequestBranch{
								Ref:  github.String("master"),
								Repo: repository,
							},
							Head: &github.PullRequestBranch{
								Ref:  github.String(headRef),
								Repo: repository,
							},
							User: &github.User{
								Login: github.String(author),
							},
						}
						pullRequests.
							On("Get", anyContext, repositoryOwner, repositoryName, number).
							Return(pr, emptyResponse, noError).
							Once()
						// Merge
						additionalCommitMessage := ""
						pullRequests.
							On(
								"Merge",
								anyContext,
								repositoryOwner,
								repositoryName,
								number,
								additionalCommitMessage,
								noSquashOpts,
							).
							Return(&github.PullRequestMergeResult{
								Merged: github.Bool(true),
							}, emptyResponse, noError).
							Once()
						// Remove label
						issues.
							On("RemoveLabelForIssue", anyContext, repositoryOwner, repositoryName, number, grh.MergingLabel).
							Return(emptyResponse, noError).
							Once()
						// Delete branch
						gitRepo := new(mocks.Repo)
						gitRepos.
							On("GetUpdatedRepo", sshURL, repositoryOwner, repositoryName).
							Return(gitRepo, noError).
							Once()
						gitRepo.On("DeleteRemoteBranch", headRef).Return(noError).Once()
					}

					BeforeEach(func() {
						firstPageSearchResult := &github.IssuesSearchResult{
							Total: github.Int(1),
							Issues: []github.Issue{{
								Number: github.Int(firstIssueNumber),
								User: &github.User{
									Login: github.String(firstAuthor),
								},
							}},
						}
						secondPageSearchResult := &github.IssuesSearchResult{
							Total: github.Int(1),
							Issues: []github.Issue{{
								Number: github.Int(secondIssueNumber),
								User: &github.User{
									Login: github.String(secondAuthor),
								},
							}},
						}
						mockSearchQuery(1).Return(firstPageSearchResult, &github.Response{NextPage: 2}, noError)
						mockSearchQuery(2).Return(secondPageSearchResult, &github.Response{}, noError)
					})

					It("it merges both PRs and removes the 'merging' label from both PRs after the merge", func() {
						expectMerge(firstIssueNumber, firstAuthor)
						expectMerge(secondIssueNumber, secondAuthor)

						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					})
				})
			})
		})
	})
})
