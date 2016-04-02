package main

import "github.com/stretchr/testify/mock"

import "github.com/google/go-github/github"

type MockIssues struct {
	mock.Mock
}

func (_m *MockIssues) AddLabelsToIssue(owner string, repo string, number int, labels []string) ([]github.Label, *github.Response, error) {
	ret := _m.Called(owner, repo, number, labels)

	var r0 []github.Label
	if rf, ok := ret.Get(0).(func(string, string, int, []string) []github.Label); ok {
		r0 = rf(owner, repo, number, labels)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]github.Label)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(string, string, int, []string) *github.Response); ok {
		r1 = rf(owner, repo, number, labels)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, string, int, []string) error); ok {
		r2 = rf(owner, repo, number, labels)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
