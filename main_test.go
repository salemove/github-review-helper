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
		)

		BeforeEach(func() {
			git = new(MockGit)
			pullRequests = new(MockPullRequests)
			repositories = new(MockRepositories)
			headers = make(map[string][]string)
			conf = Config{
				Secret: "a-secret",
			}

			responseRecorder = httptest.NewRecorder()
		})

		JustBeforeEach(func() {
			handler := CreateHandler(conf, git, pullRequests, repositories)

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
						requestJSON = `{
  "action": "created",
  "issue": {
    "number": 7,
    "pull_request": {
      "url": "https://api.github.com/repos/salemove/github-review-helper/pulls/7"
    }
  },
  "comment": {
    "body": "just a simple comment"
  },
  "repository": {
    "name": "github-review-helper",
    "owner": {
      "login": "salemove"
    },
    "ssh_url": "git@github.com:salemove/github-review-helper.git"
  }
}`
						mockSignature()
					})

					It("succeeds with a message that says the request is ignored", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
						Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
					})
				})

				Context("with a !squash comment", func() {
					BeforeEach(func() {
						requestJSON = `{
  "action": "created",
  "issue": {
    "number": 7,
    "pull_request": {
      "url": "https://api.github.com/repos/salemove/github-review-helper/pulls/7"
    }
  },
  "comment": {
    "body": "!squash"
  },
  "repository": {
    "name": "github-review-helper",
    "owner": {
      "login": "salemove"
    },
    "ssh_url": "git@github.com:salemove/github-review-helper.git"
  }
}`
						mockSignature()
					})

					Context("with GitHub request failing", func() {
						BeforeEach(func() {
							pullRequests.On("Get", "salemove", "github-review-helper", 7).Return(nil, nil, errors.New("an error"))
						})

						It("fails with a gateway error", func() {
							handle()
							Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
						})
					})

					Context("with GitHub request succeeding", func() {
						var repo *MockRepo

						BeforeEach(func() {
							pullRequests.On("Get", "salemove", "github-review-helper", 7).Return(&github.PullRequest{
								Base: &github.PullRequestBranch{
									SHA: github.String("1234"),
									Ref: github.String("master"),
								},
								Head: &github.PullRequestBranch{
									SHA: github.String("1235"),
									Ref: github.String("feature"),
								},
							}, nil, nil)
							repo = new(MockRepo)
							git.On("GetUpdatedRepo", "git@github.com:salemove/github-review-helper.git", "salemove", "github-review-helper").Return(repo, nil)
						})

						Context("with autosquash failing", func() {
							BeforeEach(func() {
								repo.On("RebaseAutosquash", "1234", "1235").Return(errors.New("merge conflict"))
							})

							It("reports the status and succeeds", func() {
								repositories.On("CreateStatus", "salemove", "github-review-helper", "1235", mock.AnythingOfType("*github.RepoStatus")).Return(nil, nil, nil)

								handle()

								Expect(responseRecorder.Code).To(Equal(http.StatusOK))
								status := repositories.Calls[0].Arguments.Get(3).(*github.RepoStatus)
								Expect(*status.State).To(Equal("failure"))
								Expect(*status.Context).To(Equal("review/squash"))
							})
						})

						Context("with autosquash succeeding", func() {
							BeforeEach(func() {
								repo.On("RebaseAutosquash", "1234", "1235").Return(nil)
							})

							It("pushes the squashed changes, reports status and succeeds", func() {
								repo.On("ForcePushHeadTo", "feature").Return(nil)
								repo.On("GetHeadSHA").Return("1236", nil)
								repositories.On("CreateStatus", "salemove", "github-review-helper", "1236", mock.AnythingOfType("*github.RepoStatus")).Return(nil, nil, nil)

								handle()

								Expect(responseRecorder.Code).To(Equal(http.StatusOK))
								status := repositories.Calls[0].Arguments.Get(3).(*github.RepoStatus)
								Expect(*status.State).To(Equal("success"))
								Expect(*status.Context).To(Equal("review/squash"))
							})
						})
					})
				})

				Context("with a +1 comment", func() {
					BeforeEach(func() {
						requestJSON = `{
  "action": "created",
  "issue": {
    "number": 7,
    "pull_request": {
      "url": "https://api.github.com/repos/salemove/github-review-helper/pulls/7"
    }
  },
  "comment": {
    "body": "+1, awesome job!"
  },
  "repository": {
    "name": "github-review-helper",
    "owner": {
      "login": "salemove"
    },
    "ssh_url": "git@github.com:salemove/github-review-helper.git"
  }
}`
						mockSignature()
					})

					Context("with GitHub request failing", func() {
						BeforeEach(func() {
							pullRequests.On("Get", "salemove", "github-review-helper", 7).Return(nil, nil, errors.New("an error"))
						})

						It("fails with a gateway error", func() {
							handle()
							Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
						})
					})

					Context("with GitHub request succeeding", func() {
						BeforeEach(func() {
							pullRequests.On("Get", "salemove", "github-review-helper", 7).Return(&github.PullRequest{
								Head: &github.PullRequestBranch{
									SHA: github.String("1235"),
									Ref: github.String("feature"),
								},
							}, nil, nil)
						})

						It("reports the status and succeeds", func() {
							repositories.On("CreateStatus", "salemove", "github-review-helper", "1235", mock.AnythingOfType("*github.RepoStatus")).Return(nil, nil, nil)

							handle()

							Expect(responseRecorder.Code).To(Equal(http.StatusOK))
							status := repositories.Calls[0].Arguments.Get(3).(*github.RepoStatus)
							Expect(*status.State).To(Equal("success"))
							Expect(*status.Context).To(Equal("review/peer"))
						})
					})
				})
			})
		})
	})
})
