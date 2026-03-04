package sync

import (
	"github.com/stretchr/testify/mock"
)

type mockGitClient struct {
	mock.Mock
}

// UpdateBranch mocks updating the given branch to point to the specified ref.
func (g *mockGitClient) UpdateBranch(b, r string) error {
	args := g.Called(b, r)
	return args.Error(0)
}

// CreateBranch mocks creating a new branch at the specified ref.
func (g *mockGitClient) CreateBranch(b, r, u string) error {
	args := g.Called(b, r, u)
	return args.Error(0)
}

// CurrentBranch mocks returning the currently checked-out branch name.
func (g *mockGitClient) CurrentBranch() (string, error) {
	args := g.Called()
	return args.String(0), args.Error(1)
}

// Fetch mocks fetching a ref from a remote.
func (g *mockGitClient) Fetch(a, b string) error {
	args := g.Called(a, b)
	return args.Error(0)
}

// HasLocalBranch mocks checking whether a local branch exists.
func (g *mockGitClient) HasLocalBranch(a string) bool {
	args := g.Called(a)
	return args.Bool(0)
}

// IsAncestor mocks checking whether one commit is an ancestor of another.
func (g *mockGitClient) IsAncestor(a, b string) (bool, error) {
	args := g.Called(a, b)
	return args.Bool(0), args.Error(1)
}

// IsDirty mocks checking whether the working tree has uncommitted changes.
func (g *mockGitClient) IsDirty() (bool, error) {
	args := g.Called()
	return args.Bool(0), args.Error(1)
}

// MergeFastForward mocks performing a fast-forward merge.
func (g *mockGitClient) MergeFastForward(a string) error {
	args := g.Called(a)
	return args.Error(0)
}

// ResetHard mocks performing a hard reset to the given ref.
func (g *mockGitClient) ResetHard(a string) error {
	args := g.Called(a)
	return args.Error(0)
}
