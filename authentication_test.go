package main_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = TestWebhookHandler(func(context WebhookTestContext) {
	Describe("authentication", func() {
		var (
			handle      = context.Handle
			headers     = context.Headers
			requestJSON = context.RequestJSON

			responseRecorder *httptest.ResponseRecorder
		)
		BeforeEach(func() {
			responseRecorder = *context.ResponseRecorder
		})

		Context("with an empty X-Hub-Signature header", func() {
			headers.Is(func() map[string]string {
				return map[string]string{
					"X-Hub-Signature": "",
				}
			})
			It("fails with StatusUnauthorized", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("with an invalid X-Hub-Signature header", func() {
			requestJSON.Is(func() string {
				return "{}"
			})
			headers.Is(func() map[string]string {
				return map[string]string{
					"X-Hub-Signature": "sha1=2f539a59127d552f4565b1a114ec8f4fa2d55f55",
				}
			})

			It("fails with StatusForbidden", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusForbidden))
			})
		})

		Context("with an empty request with a proper signature", func() {
			var validSignature = "sha1=33c829a9c355e7722cb74d25dfa54c6c623cde63"
			requestJSON.Is(func() string {
				return "{}"
			})
			headers.Is(func() map[string]string {
				return map[string]string{
					"X-Hub-Signature": validSignature,
				}
			})

			It("succeeds with 'ignored' response", func() {
				handle()
				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
			})

			Context("with a gibberish event", func() {
				headers.Is(func() map[string]string {
					return map[string]string{
						"X-Hub-Signature": validSignature,
						"X-Github-Event":  "gibberish",
					}
				})

				It("succeeds with 'ignored' response", func() {
					handle()
					Expect(responseRecorder.Code).To(Equal(http.StatusOK))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
				})
			})
		})

		Context("with a valid signature", func() {
			Describe("issue_comment event", func() {
				headers.Is(func() map[string]string {
					return map[string]string{
						"X-Github-Event": "issue_comment",
					}
				})

				Context("with an arbitrary comment", func() {
					requestJSON.Is(func() string {
						return IssueCommentEvent("just a simple comment", arbitraryIssueAuthor)
					})

					It("succeeds with 'ignored' response", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
						Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
					})
				})

				Context("with a '!mergethis' comment (without space after '!merge')", func() {
					requestJSON.Is(func() string {
						return IssueCommentEvent("!mergethis", arbitraryIssueAuthor)
					})

					It("succeeds with 'ignored' response", func() {
						handle()
						Expect(responseRecorder.Code).To(Equal(http.StatusOK))
						Expect(responseRecorder.Body.String()).To(ContainSubstring("Ignoring"))
					})
				})
			})
		})
	})
})
