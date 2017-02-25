package main_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/salemove/github-review-helper/mocks"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func ForCollaborator(context WebhookTestContext, repoOwner, repoName, user string, test func()) {
	var (
		handle = context.Handle

		responseRecorder *httptest.ResponseRecorder
		repositories     *mocks.Repositories
		issues           *mocks.Issues
	)
	BeforeEach(func() {
		responseRecorder = *context.ResponseRecorder
		repositories = *context.Repositories
		issues = *context.Issues
	})

	Context("with collaborator status check failing", func() {
		BeforeEach(func() {
			repositories.
				On("IsCollaborator", anyContext, repoOwner, repoName, user).
				Return(false, emptyResponse, errArbitrary)
		})

		It("fails with a gateway error", func() {
			handle()
			Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
		})
	})

	Context("with user not being a collaborator", func() {
		BeforeEach(func() {
			repositories.
				On("IsCollaborator", anyContext, repoOwner, repoName, user).
				Return(false, emptyResponse, noError)
		})

		Context("with sending a comment failing", func() {
			BeforeEach(func() {
				issues.
					On("CreateComment", anyContext, repoOwner, repoName,
						issueNumber, mock.MatchedBy(commentMentioning(user))).
					Return(emptyResult, emptyResponse, errArbitrary)
			})

			It("fails with a gateway error", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusBadGateway))
			})
		})

		Context("with sending a comment succeeding", func() {
			BeforeEach(func() {
				issues.
					On("CreateComment", anyContext, repoOwner, repoName,
						issueNumber, mock.MatchedBy(commentMentioning(user))).
					Return(emptyResult, emptyResponse, noError)
			})

			It("returns 200 OK, ignoring the command", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Context("with user being a collaborator", func() {
		BeforeEach(func() {
			repositories.
				On("IsCollaborator", anyContext, repositoryOwner, repositoryName, user).
				Return(true, emptyResponse, noError)
		})

		test()
	})
}
