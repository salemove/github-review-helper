package main_test

import (
	"io/ioutil"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGithubReviewHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	log.SetOutput(ioutil.Discard)
	RunSpecs(t, "GithubReviewHelper Suite")
}
