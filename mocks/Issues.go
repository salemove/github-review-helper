package mocks

import "github.com/stretchr/testify/mock"

import "context"

import "github.com/google/go-github/github"

type Issues struct {
	mock.Mock
}

func (_m *Issues) AddLabelsToIssue(ctx context.Context, owner string, repo string, number int, labels []string) ([]*github.Label, *github.Response, error) {
	ret := _m.Called(ctx, owner, repo, number, labels)

	var r0 []*github.Label
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int, []string) []*github.Label); ok {
		r0 = rf(ctx, owner, repo, number, labels)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.Label)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, string, int, []string) *github.Response); ok {
		r1 = rf(ctx, owner, repo, number, labels)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, string, int, []string) error); ok {
		r2 = rf(ctx, owner, repo, number, labels)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *Issues) RemoveLabelForIssue(ctx context.Context, owner string, repo string, number int, label string) (*github.Response, error) {
	ret := _m.Called(ctx, owner, repo, number, label)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int, string) *github.Response); ok {
		r0 = rf(ctx, owner, repo, number, label)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, int, string) error); ok {
		r1 = rf(ctx, owner, repo, number, label)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Issues) CreateComment(ctx context.Context, owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	ret := _m.Called(ctx, owner, repo, number, comment)

	var r0 *github.IssueComment
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int, *github.IssueComment) *github.IssueComment); ok {
		r0 = rf(ctx, owner, repo, number, comment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.IssueComment)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, string, int, *github.IssueComment) *github.Response); ok {
		r1 = rf(ctx, owner, repo, number, comment)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, string, int, *github.IssueComment) error); ok {
		r2 = rf(ctx, owner, repo, number, comment)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
