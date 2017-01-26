package mocks

import "github.com/stretchr/testify/mock"

type Repo struct {
	mock.Mock
}

func (_m *Repo) Fetch() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Repo) RebaseAutosquash(upstreamRef string, branchRef string) error {
	ret := _m.Called(upstreamRef, branchRef)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(upstreamRef, branchRef)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Repo) ForcePushHeadTo(remoteRef string) error {
	ret := _m.Called(remoteRef)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(remoteRef)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
