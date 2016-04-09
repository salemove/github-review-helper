package main_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/google/go-github/github"
	. "github.com/salemove/github-review-helper"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	repositoryOwner = "salemove"
	repositoryName  = "github-review-helper"
	sshURL          = "git@github.com:salemove/github-review-helper.git"
	issueNumber     = 7
)

var _ = Describe("github-review-helper", func() {
	Describe("/ handler", func() {
		var (
			responseRecorder *httptest.ResponseRecorder
			requestJSON      string
			headers          map[string][]string

			handle func()

			conf         Config
			git          *MockGit
			pullRequests *MockPullRequests
			repositories *MockRepositories
			issues       *MockIssues
		)

		BeforeEach(func() {
			git = new(MockGit)
			pullRequests = new(MockPullRequests)
			repositories = new(MockRepositories)
			issues = new(MockIssues)
			headers = make(map[string][]string)
			conf = Config{
				Secret: "a-secret",
			}

			responseRecorder = httptest.NewRecorder()
		})

		AfterEach(func() {
			git.AssertExpectations(GinkgoT())
			pullRequests.AssertExpectations(GinkgoT())
			repositories.AssertExpectations(GinkgoT())
			issues.AssertExpectations(GinkgoT())
		})

		JustBeforeEach(func() {
			handler := CreateHandler(conf, git, pullRequests, repositories, issues)

			data := []byte(requestJSON)
			request, err := http.NewRequest("GET", "http://localhost/whatever", bytes.NewBuffer(data))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Add("Content-Type", "application/json")
			request.Header.Add("Content-Length", strconv.Itoa(len(data)))
			for key, vals := range headers {
				for _, val := range vals {
					request.Header.Add(key, val)
				}
			}

			handle = func() {
				response := handler(responseRecorder, request)
				response.WriteResponse(responseRecorder)
			}
		})

		It("fails with StatusUnauthorized if no headers set", func() {
			handle()
			Expect(responseRecorder.Code).To(Equal(http.StatusUnauthorized))
		})

		Context("with an invalid X-Hub-Signature header", func() {
			BeforeEach(func() {
				requestJSON = "{}"
				headers["X-Hub-Signature"] = []string{"sha1=2f539a59127d552f4565b1a114ec8f4fa2d55f55"}
			})

			It("fails with StatusForbidden", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusForbidden))
			})
		})

		Context("with an empty request with a proper signature", func() {
			BeforeEach(func() {
				requestJSON = "{}"
				headers["X-Hub-Signature"] = []string{"sha1=33c829a9c355e7722cb74d25dfa54c6c623cde63"}
			})

			It("succeeds with a message that says the request is ignored", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
			})

			Context("with a gibberish event", func() {
				BeforeEach(func() {
					headers["X-Github-Event"] = []string{"gibberish"}
				})

				It("succeeds with a message that says the request is ignored", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
				})
			})
		})

		var issueCommentEvent = func(comment string) string {
			return `{
  "issue": {
    "number": ` + strconv.Itoa(issueNumber) + `,
    "pull_request": {
      "url": "https://api.github.com/repos/` + repositoryOwner + `/` + repositoryName + `/pulls/` + strconv.Itoa(issueNumber) + `"
    }
  },
  "comment": {
    "body": "` + comment + `"
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

		Context("with a valid signature", func() {
			var mockSignature func()

			BeforeEach(func() {
				mockSignature = func() {
					mac := hmac.New(sha1.New, []byte(conf.Secret))
					mac.Write([]byte(requestJSON))
					sig := hex.EncodeToString(mac.Sum(nil))
					headers["X-Hub-Signature"] = []string{"sha1=" + sig}
				}
			})

			Describe("issue_comment event", func() {
				BeforeEach(func() {
					headers["X-Github-Event"] = []string{"issue_comment"}
				})

				Context("with an arbitrary comment", func() {
					BeforeEach(func() {
						requestJSON = issueCommentEvent("just a simple comment")
						mockSignature()
					})

					It("succeeds with a message that says the request is ignored", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
						Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
					})
				})

				Describe("!squash comment", func() {
					BeforeEach(func() {
						requestJSON = issueCommentEvent("!squash")
						mockSignature()
					})

					Context("with GitHub request failing", func() {
						BeforeEach(func() {
							pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(nil, nil, errors.New("an error"))
						})

						It("fails with a gateway error", func() {
							handle()
							Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
						})
					})

					Context("with GitHub request succeeding", func() {
						var (
							repo *MockRepo

							baseRef = "master"
							baseSHA = "1234"
							headRef = "feature"
							headSHA = "1235"
						)

						BeforeEach(func() {
							pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(&github.PullRequest{
								Base: &github.PullRequestBranch{
									SHA: github.String(baseSHA),
									Ref: github.String(baseRef),
								},
								Head: &github.PullRequestBranch{
									SHA: github.String(headSHA),
									Ref: github.String(headRef),
									Repo: &github.Repository{
										Owner: &github.User{
											Login: github.String(repositoryOwner),
										},
										Name:   github.String(repositoryName),
										SSHURL: github.String(sshURL),
									},
								},
							}, nil, nil)
							repo = new(MockRepo)
							git.On("GetUpdatedRepo", sshURL, repositoryOwner, repositoryName).Return(repo, nil)
						})

						AfterEach(func() {
							repo.AssertExpectations(GinkgoT())
						})

						Context("with autosquash failing", func() {
							BeforeEach(func() {
								repo.On("RebaseAutosquash", baseRef, headSHA).Return(errors.New("merge conflict"))
							})

							It("reports the failure", func() {
								repositories.On("CreateStatus", repositoryOwner, repositoryName, headSHA, mock.AnythingOfType("*github.RepoStatus")).Return(nil, nil, nil)

								handle()

								Expect(responseRecorder.Code).To(Equal(http.StatusOK))
								status := repositories.Calls[0].Arguments.Get(3).(*github.RepoStatus)
								Expect(*status.State).To(Equal("failure"))
								Expect(*status.Context).To(Equal("review/squash"))
							})
						})

						Context("with autosquash succeeding", func() {
							BeforeEach(func() {
								repo.On("RebaseAutosquash", baseRef, headSHA).Return(nil)
							})

							It("pushes the squashed changes, reports status", func() {
								repo.On("ForcePushHeadTo", headRef).Return(nil)

								handle()
							})
						})
					})
				})

				Describe("!merge comment", func() {
					BeforeEach(func() {
						requestJSON = issueCommentEvent("!merge")
						mockSignature()
					})

					Context("with github request to add the label failing", func() {
						BeforeEach(func() {
							issues.
								On("AddLabelsToIssue", repositoryOwner, repositoryName, issueNumber, []string{MergingLabel}).
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
								On("AddLabelsToIssue", repositoryOwner, repositoryName, issueNumber, []string{MergingLabel}).
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
								pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(&github.PullRequest{
									Merged: github.Bool(true),
								}, nil, nil)
							})

							It("removes the 'merging' label from the PR", func() {
								issues.
									On("RemoveLabelForIssue", repositoryOwner, repositoryName, issueNumber, MergingLabel).
									Return(nil, nil, nil)

								handle()
								Expect(responseRecorder.Code).To(Equal(http.StatusOK))
							})
						})

						Context("with the PR not being mergeable", func() {
							BeforeEach(func() {
								pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(&github.PullRequest{
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
							BeforeEach(func() {
								pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(&github.PullRequest{
									Merged:    github.Bool(false),
									Mergeable: github.Bool(true),
								}, nil, nil)
							})

							Context("with merge failing with an unknown error", func() {
								BeforeEach(func() {
									additionalCommitMessage := ""
									pullRequests.
										On("Merge", repositoryOwner, repositoryName, issueNumber, additionalCommitMessage).
										Return(nil, nil, errors.New("an error")).
										Once()
								})

								It("fails with a gateway error", func() {
									handle()
									Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
								})
							})

							Context("with head branch having changed", func() {
								mockMergeFailWithConflict := func() *mock.Call {
									additionalCommitMessage := ""
									resp := &http.Response{
										StatusCode: http.StatusConflict,
									}
									return pullRequests.
										On("Merge", repositoryOwner, repositoryName, issueNumber, additionalCommitMessage).
										Return(nil, &github.Response{
											Response: resp,
										}, &github.ErrorResponse{
											Response: resp,
											Message:  "Head branch was modified. Review and try the merge again.",
										})
								}

								Context("every time", func() {
									BeforeEach(func() {
										mockMergeFailWithConflict()
									})

									It("retries 3 times and fails with a gateway error", func() {
										handle()
										pullRequests.AssertNumberOfCalls(GinkgoT(), "Get", 4)
										pullRequests.AssertNumberOfCalls(GinkgoT(), "Merge", 4)
										Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
									})
								})

								Context("with merge succeeding with first retry", func() {
									BeforeEach(func() {
										mockMergeFailWithConflict().Once()

										additionalCommitMessage := ""
										pullRequests.
											On("Merge", repositoryOwner, repositoryName, issueNumber, additionalCommitMessage).
											Return(&github.PullRequestMergeResult{
												Merged: github.Bool(true),
											}, nil, nil).
											Once()
									})

									It("removes the 'merging' label from the PR after the merge", func() {
										issues.
											On("RemoveLabelForIssue", repositoryOwner, repositoryName, issueNumber, MergingLabel).
											Return(nil, nil, nil)

										handle()
										pullRequests.AssertNumberOfCalls(GinkgoT(), "Get", 2)
										pullRequests.AssertNumberOfCalls(GinkgoT(), "Merge", 2)
										Expect(responseRecorder.Code).To(Equal(http.StatusOK))
									})
								})
							})

							Context("with PR not being mergeable", func() {
								BeforeEach(func() {
									additionalCommitMessage := ""
									resp := &http.Response{
										StatusCode: http.StatusMethodNotAllowed,
									}
									pullRequests.
										On("Merge", repositoryOwner, repositoryName, issueNumber, additionalCommitMessage).
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
										On("Merge", repositoryOwner, repositoryName, issueNumber, additionalCommitMessage).
										Return(&github.PullRequestMergeResult{
											Merged: github.Bool(true),
										}, nil, nil).
										Once()
								})

								It("removes the 'merging' label from the PR after the merge", func() {
									issues.
										On("RemoveLabelForIssue", repositoryOwner, repositoryName, issueNumber, MergingLabel).
										Return(nil, nil, nil)

									handle()
									Expect(responseRecorder.Code).To(Equal(http.StatusOK))
								})
							})
						})
					})
				})

				Describe("+1 comment", func() {
					var commitRevision = "1235"
					var itMarksCommitPeerReviewed = func() {
						Context("with GitHub request failing", func() {
							BeforeEach(func() {
								pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(nil, nil, errors.New("an error"))
							})

							It("fails with a gateway error", func() {
								handle()
								Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
							})
						})

						Context("with GitHub request succeeding", func() {
							BeforeEach(func() {
								pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(&github.PullRequest{
									Head: &github.PullRequestBranch{
										SHA: github.String(commitRevision),
									},
								}, nil, nil)
							})

							It("reports the status", func() {
								repositories.On("CreateStatus", repositoryOwner, repositoryName, commitRevision, mock.AnythingOfType("*github.RepoStatus")).Return(nil, nil, nil)

								handle()

								Expect(responseRecorder.Code).To(Equal(http.StatusOK))
								status := repositories.Calls[0].Arguments.Get(3).(*github.RepoStatus)
								Expect(*status.State).To(Equal("success"))
								Expect(*status.Context).To(Equal("review/peer"))
							})
						})
					}

					Context("with +1 at the beginning of the comment", func() {
						BeforeEach(func() {
							requestJSON = issueCommentEvent("+1, awesome job!")
							mockSignature()
						})

						itMarksCommitPeerReviewed()
					})

					Context("with +1 at the end of the comment", func() {
						BeforeEach(func() {
							requestJSON = issueCommentEvent("Looking good! +1")
							mockSignature()
						})

						itMarksCommitPeerReviewed()
					})
				})
			})

			var pullRequestsEvent = func(action string) string {
				return `{
  "action": "` + action + `",
  "number": ` + strconv.Itoa(issueNumber) + `,
  "pull_request": {
    "url": "https://api.github.com/repos/` + repositoryOwner + `/` + repositoryName + `/pulls/` + strconv.Itoa(issueNumber) + `"
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

			Describe("pull_request event", func() {
				BeforeEach(func() {
					headers["X-Github-Event"] = []string{"pull_request"}
				})

				Context("with the PR being closed", func() {
					BeforeEach(func() {
						requestJSON = pullRequestsEvent("closed")
						mockSignature()
					})

					It("succeeds with a message that says the request is ignored as PR not opened/synchronized", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
						Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
					})
				})

				Context("with the PR being synchronized", func() {
					var commitRevision = "1235"

					BeforeEach(func() {
						requestJSON = pullRequestsEvent("synchronize")
						mockSignature()
					})

					Context("with GitHub request to list commits failing", func() {
						BeforeEach(func() {
							var listOptions *github.ListOptions
							pullRequests.On("ListCommits", repositoryOwner, repositoryName, issueNumber, listOptions).Return(nil, nil, errors.New("an error"))
						})

						It("fails with a gateway error", func() {
							handle()
							Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
						})
					})

					Context("with list of commits from GitHub NOT including fixup commits", func() {
						BeforeEach(func() {
							var listOptions *github.ListOptions
							pullRequests.On("ListCommits", repositoryOwner, repositoryName, issueNumber, listOptions).Return([]github.RepositoryCommit{
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
							}, nil, nil)
							pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(&github.PullRequest{
								Head: &github.PullRequestBranch{
									SHA: github.String(commitRevision),
								},
							}, nil, nil)
						})

						It("reports success status to GitHub", func() {
							repositories.On("CreateStatus", repositoryOwner, repositoryName, commitRevision, mock.AnythingOfType("*github.RepoStatus")).Return(nil, nil, nil)

							handle()

							Expect(responseRecorder.Code).To(Equal(http.StatusOK))
							status := repositories.Calls[0].Arguments.Get(3).(*github.RepoStatus)
							Expect(*status.State).To(Equal("success"))
							Expect(*status.Context).To(Equal("review/squash"))
						})
					})

					Context("with list of commits from GitHub including fixup commits", func() {
						BeforeEach(func() {
							var listOptions *github.ListOptions
							pullRequests.On("ListCommits", repositoryOwner, repositoryName, issueNumber, listOptions).Return([]github.RepositoryCommit{
								github.RepositoryCommit{
									Commit: &github.Commit{
										Message: github.String("Changing things"),
									},
								},
								github.RepositoryCommit{
									Commit: &github.Commit{
										Message: github.String("fixup! Changing things\n\nOopsie. Forgot a thing"),
									},
								},
							}, nil, nil)
							pullRequests.On("Get", repositoryOwner, repositoryName, issueNumber).Return(&github.PullRequest{
								Head: &github.PullRequestBranch{
									SHA: github.String(commitRevision),
								},
							}, nil, nil)
						})

						It("reports pending squash status to GitHub", func() {
							repositories.On("CreateStatus", repositoryOwner, repositoryName, commitRevision, mock.AnythingOfType("*github.RepoStatus")).Return(nil, nil, nil)

							handle()

							Expect(responseRecorder.Code).To(Equal(http.StatusOK))
							status := repositories.Calls[0].Arguments.Get(3).(*github.RepoStatus)
							Expect(*status.State).To(Equal("pending"))
							Expect(*status.Context).To(Equal("review/squash"))
						})
					})
				})
			})
		})
	})
})
