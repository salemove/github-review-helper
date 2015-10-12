package main

import "github.com/stretchr/testify/mock"

type MockRepo struct {
	mock.Mock
}

func (_m *MockRepo) Fetch() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *MockRepo) RebaseAutosquash(upstreamRef string, branchRef string) error {
	ret := _m.Called(upstreamRef, branchRef)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(upstreamRef, branchRef)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *MockRepo) ForcePushHeadTo(remoteRef string) error {
	ret := _m.Called(remoteRef)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(remoteRef)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *MockRepo) GetHeadSHA() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
