package mocks

import "github.com/stretchr/testify/mock"

import "github.com/google/go-github/github"

type PullRequests struct {
	mock.Mock
}

func (_m *PullRequests) Get(owner string, repo string, number int) (*github.PullRequest, *github.Response, error) {
	ret := _m.Called(owner, repo, number)

	var r0 *github.PullRequest
	if rf, ok := ret.Get(0).(func(string, string, int) *github.PullRequest); ok {
		r0 = rf(owner, repo, number)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.PullRequest)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(string, string, int) *github.Response); ok {
		r1 = rf(owner, repo, number)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, string, int) error); ok {
		r2 = rf(owner, repo, number)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *PullRequests) ListCommits(owner string, repo string, number int, opt *github.ListOptions) ([]github.RepositoryCommit, *github.Response, error) {
	ret := _m.Called(owner, repo, number, opt)

	var r0 []github.RepositoryCommit
	if rf, ok := ret.Get(0).(func(string, string, int, *github.ListOptions) []github.RepositoryCommit); ok {
		r0 = rf(owner, repo, number, opt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]github.RepositoryCommit)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(string, string, int, *github.ListOptions) *github.Response); ok {
		r1 = rf(owner, repo, number, opt)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, string, int, *github.ListOptions) error); ok {
		r2 = rf(owner, repo, number, opt)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *PullRequests) Merge(owner string, repo string, number int, commitMessage string) (*github.PullRequestMergeResult, *github.Response, error) {
	ret := _m.Called(owner, repo, number, commitMessage)

	var r0 *github.PullRequestMergeResult
	if rf, ok := ret.Get(0).(func(string, string, int, string) *github.PullRequestMergeResult); ok {
		r0 = rf(owner, repo, number, commitMessage)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.PullRequestMergeResult)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(string, string, int, string) *github.Response); ok {
		r1 = rf(owner, repo, number, commitMessage)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, string, int, string) error); ok {
		r2 = rf(owner, repo, number, commitMessage)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
