package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGithubReviewHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GithubReviewHelper Suite")
}
