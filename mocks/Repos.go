package mocks

import "github.com/salemove/github-review-helper/git"
import "github.com/stretchr/testify/mock"

type Repos struct {
	mock.Mock
}

func (_m *Repos) GetUpdatedRepo(url string, repoOwner string, repoName string) (git.Repo, error) {
	ret := _m.Called(url, repoOwner, repoName)

	var r0 git.Repo
	if rf, ok := ret.Get(0).(func(string, string, string) git.Repo); ok {
		r0 = rf(url, repoOwner, repoName)
	} else {
		r0 = ret.Get(0).(git.Repo)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string) error); ok {
		r1 = rf(url, repoOwner, repoName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
