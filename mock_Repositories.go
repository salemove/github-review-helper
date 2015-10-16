package main

import "github.com/stretchr/testify/mock"

import "github.com/google/go-github/github"

type MockRepositories struct {
	mock.Mock
}

func (_m *MockRepositories) CreateStatus(owner string, repo string, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error) {
	ret := _m.Called(owner, repo, ref, status)

	var r0 *github.RepoStatus
	if rf, ok := ret.Get(0).(func(string, string, string, *github.RepoStatus) *github.RepoStatus); ok {
		r0 = rf(owner, repo, ref, status)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.RepoStatus)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(string, string, string, *github.RepoStatus) *github.Response); ok {
		r1 = rf(owner, repo, ref, status)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, string, string, *github.RepoStatus) error); ok {
		r2 = rf(owner, repo, ref, status)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
