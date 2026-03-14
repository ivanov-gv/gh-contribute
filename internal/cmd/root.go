package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	ghrest "github.com/google/go-github/v69/github"
	"github.com/spf13/cobra"

	"github.com/ivanov-gv/gh-contribute/internal/auth"
	"github.com/ivanov-gv/gh-contribute/internal/config"
	ghclient "github.com/ivanov-gv/gh-contribute/internal/github"
	"github.com/ivanov-gv/gh-contribute/internal/service/comment"
	"github.com/ivanov-gv/gh-contribute/internal/service/pr"
	"github.com/ivanov-gv/gh-contribute/internal/service/reaction"
	"github.com/ivanov-gv/gh-contribute/internal/service/review"
)

// app holds shared dependencies for all authenticated commands.
type app struct {
	cfg             *config.Config
	prService       *pr.Service
	commentService  *comment.Service
	reactionService *reaction.Service
	reviewService   *review.Service
}

// lazyApp initializes the app on first use, so auth commands can run without a token.
type lazyApp struct {
	_app *app
}

// get initializes and returns the app, loading config (and the token) on first call.
func (l *lazyApp) get() (*app, error) {
	if l._app != nil {
		return l._app, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// GraphQL client for read operations, REST client for write operations
	gql := ghclient.NewGraphQLClient(cfg.Token)
	rest := ghrest.NewClient(nil).WithAuthToken(cfg.Token)

	l._app = &app{
		cfg:             cfg,
		prService:       pr.NewService(gql, cfg.Owner, cfg.Repo),
		commentService:  comment.NewService(gql, rest, cfg.Owner, cfg.Repo),
		reactionService: reaction.NewService(rest, cfg.Owner, cfg.Repo),
		reviewService:   review.NewService(gql, cfg.Owner, cfg.Repo),
	}

	return l._app, nil
}

// Execute runs the root command.
func Execute() {
	lazy := &lazyApp{}

	rootCmd := &cobra.Command{
		Use:          "gh-contribute",
		Short:        "A gh extension for simplifying agents interaction with PRs on GitHub",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		// auth commands — do not require a token
		newAuthCmd(),
		// all other commands — token is loaded lazily on first use
		lazy.newPRCmd(),
		lazy.newCommentsCmd(),
		lazy.newCommentCmd(),
		lazy.newReactCmd(),
		lazy.newReviewCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// handleErr prints an actionable error message for known error types and exits.
func handleErr(err error) {
	switch {
	case errors.Is(err, auth.ErrNotAuthenticated):
		fmt.Fprintln(os.Stderr, "Not authenticated. Run 'gh myext auth login' first.")
	case isUnauthorized(err):
		fmt.Fprintln(os.Stderr, "Token invalid or expired. Run 'gh myext auth login' to reauthenticate.")
	default:
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(1)
}

// isUnauthorized reports whether err represents a 401 Unauthorized response
// from either the REST API (go-github ErrorResponse) or the GraphQL API (HTTP 401).
func isUnauthorized(err error) bool {
	var ghErr *ghrest.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		return ghErr.Response.StatusCode == 401
	}
	// GraphQL client formats non-200 as "GraphQL HTTP <code>: ..."
	return strings.Contains(err.Error(), "GraphQL HTTP 401")
}
