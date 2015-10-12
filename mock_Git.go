package main

import "github.com/stretchr/testify/mock"

type MockGit struct {
	mock.Mock
}

func (_m *MockGit) GetUpdatedRepo(url string, repoOwner string, repoName string) (Repo, error) {
	ret := _m.Called(url, repoOwner, repoName)

	var r0 Repo
	if rf, ok := ret.Get(0).(func(string, string, string) Repo); ok {
		r0 = rf(url, repoOwner, repoName)
	} else {
		r0 = ret.Get(0).(Repo)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string) error); ok {
		r1 = rf(url, repoOwner, repoName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
