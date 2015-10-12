package main_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"

	. "github.com/salemove/github-review-helper"

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
				conf.Secret = "a-secret"
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
				conf.Secret = "a-secret"
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
	})
})
