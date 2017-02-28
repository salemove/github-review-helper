package mocks

import "github.com/stretchr/testify/mock"

import "context"

import "github.com/google/go-github/github"

type Repositories struct {
	mock.Mock
}

func (_m *Repositories) CreateStatus(ctx context.Context, owner string, repo string, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error) {
	ret := _m.Called(ctx, owner, repo, ref, status)

	var r0 *github.RepoStatus
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, *github.RepoStatus) *github.RepoStatus); ok {
		r0 = rf(ctx, owner, repo, ref, status)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.RepoStatus)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, *github.RepoStatus) *github.Response); ok {
		r1 = rf(ctx, owner, repo, ref, status)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, string, string, *github.RepoStatus) error); ok {
		r2 = rf(ctx, owner, repo, ref, status)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *Repositories) GetCombinedStatus(ctx context.Context, owner string, repo string, ref string, opt *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
	ret := _m.Called(ctx, owner, repo, ref, opt)

	var r0 *github.CombinedStatus
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, *github.ListOptions) *github.CombinedStatus); ok {
		r0 = rf(ctx, owner, repo, ref, opt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.CombinedStatus)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, *github.ListOptions) *github.Response); ok {
		r1 = rf(ctx, owner, repo, ref, opt)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, string, string, *github.ListOptions) error); ok {
		r2 = rf(ctx, owner, repo, ref, opt)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *Repositories) IsCollaborator(ctx context.Context, owner string, repo string, user string) (bool, *github.Response, error) {
	ret := _m.Called(ctx, owner, repo, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) bool); ok {
		r0 = rf(ctx, owner, repo, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string) *github.Response); ok {
		r1 = rf(ctx, owner, repo, user)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, string, string) error); ok {
		r2 = rf(ctx, owner, repo, user)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
