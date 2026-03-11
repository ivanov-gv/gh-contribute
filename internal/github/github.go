package github

import (
	"context"

	gh "github.com/google/go-github/v69/github"

	"github.com/ivanov-gv/gh-contribute/internal/config"
)

// NewClient creates an authenticated GitHub API client
func NewClient(cfg *config.Config) *gh.Client {
	return gh.NewClient(nil).WithAuthToken(cfg.Token)
}

// Context returns a background context for API calls
func Context() context.Context {
	return context.Background()
}
