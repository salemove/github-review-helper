package main_test

import . "github.com/onsi/ginkgo"

// Memoization method strongly influenced by https://github.com/d11wtq/node-memo-is

type StringMemoizer interface {
	Get() string
	Is(func() string)
}

type StringMapMemoizer interface {
	Get() map[string]string
	Is(func() map[string]string)
}

type stringMemoizer struct {
	value   string
	stack   []func() string
	invoked bool
}

type stringMapMemoizer struct {
	value   map[string]string
	stack   []func() map[string]string
	invoked bool
}

func NewStringMemoizer(cb func() string) StringMemoizer {
	memo := &stringMemoizer{
		stack:   []func() string{},
		invoked: false,
	}
	memo.Is(cb)
	return memo
}

func NewStringMapMemoizer(cb func() map[string]string) StringMapMemoizer {
	memo := &stringMapMemoizer{
		stack:   []func() map[string]string{},
		invoked: false,
	}
	memo.Is(cb)
	return memo
}

func (s *stringMemoizer) Is(cb func() string) {
	BeforeEach(func() {
		s.stack = append(s.stack, cb)
	})

	AfterEach(func() {
		s.invoked = false
		s.value = ""
		s.stack = s.stack[:len(s.stack)-1]
	})
}

func (s *stringMapMemoizer) Is(cb func() map[string]string) {
	BeforeEach(func() {
		s.stack = append(s.stack, cb)
	})

	AfterEach(func() {
		s.invoked = false
		s.value = nil
		s.stack = s.stack[:len(s.stack)-1]
	})
}

func (s *stringMemoizer) Get() string {
	if len(s.stack) == 0 {
		Fail("Memoized function called outside test example scope")
	}

	if !s.invoked {
		s.value = s.stack[len(s.stack)-1]()
		s.invoked = true
	}
	return s.value
}

func (s *stringMapMemoizer) Get() map[string]string {
	if len(s.stack) == 0 {
		Fail("Memoized function called outside test example scope")
	}

	if !s.invoked {
		s.value = s.stack[len(s.stack)-1]()
		s.invoked = true
	}
	return s.value
}
