package main_test

import (
	"net/http"
	"net/http/httptest"

	grh "github.com/salemove/github-review-helper"
	"github.com/salemove/github-review-helper/mocks"

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
			repositories     *mocks.Repositories
			issues           *mocks.Issues
		)
		BeforeEach(func() {
			responseRecorder = *context.ResponseRecorder
			pullRequests = *context.PullRequests
			repositories = *context.Repositories
			issues = *context.Issues
		})

		headers.Is(func() map[string]string {
			return map[string]string{
				"X-Github-Event": "status",
			}
		})

		for _, badStatus := range []string{"pending", "failure", "error"} {
			Context("with "+badStatus+" status", func() {
				branches := []grh.Branch{grh.Branch{
					SHA: mockSHA,
				}}

				requestJSON.Is(func() string {
					return createStatusEvent(mockSHA, badStatus, branches)
				})

				It("fails with internal error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
				})
			})
		}

		Context("with success status", func() {
			status := "success"

			Context("when updating a commit that is not a branch's head", func() {
				otherSHA := "4eaf26faa8819ab5aee991461b8c4fff41778f41"
				branches := []grh.Branch{grh.Branch{
					SHA: otherSHA,
				}}

				requestJSON.Is(func() string {
					return createStatusEvent(mockSHA, status, branches)
				})

				It("fails with internal error", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
				})
			})

			Context("when updating a commit that is a branch's head", func() {
				branches := []grh.Branch{grh.Branch{
					SHA: mockSHA,
				}}

				requestJSON.Is(func() string {
					return createStatusEvent(mockSHA, status, branches)
				})

				It("returns 200 OK", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				})
			})
		})
	})
})
