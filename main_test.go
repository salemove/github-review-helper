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

		It("fails with StatusBadRequest if no headers set", func() {
			handle()
			Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
		})
	})
})
