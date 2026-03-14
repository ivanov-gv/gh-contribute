package cmd

import (
	"fmt"
	"os"

	ghrest "github.com/google/go-github/v69/github"
	"github.com/spf13/cobra"

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

// init loads config and initializes all services.
// Called by the root PersistentPreRunE before any authenticated command runs.
func (a *app) init() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}

	gql := ghclient.NewGraphQLClient(cfg.Token)
	rest := ghrest.NewClient(nil).WithAuthToken(cfg.Token)

	a.cfg = cfg
	a.prService = pr.NewService(gql, cfg.Owner, cfg.Repo)
	a.commentService = comment.NewService(gql, rest, cfg.Owner, cfg.Repo)
	a.reactionService = reaction.NewService(rest, cfg.Owner, cfg.Repo)
	a.reviewService = review.NewService(gql, cfg.Owner, cfg.Repo)

	return nil
}

// Execute wires and runs the root command.
func Execute() {
	_app := &app{}

	rootCmd := &cobra.Command{
		Use:          "gh-contribute",
		Short:        "A gh extension for simplifying agents interaction with PRs on GitHub",
		SilenceUsage: true,
		// initialize app before any authenticated command runs
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return _app.init()
		},
	}

	rootCmd.AddCommand(
		// auth commands override PersistentPreRunE with a no-op — no token required
		newAuthCmd(),
		// authenticated commands — app is initialized via PersistentPreRunE
		_app.newPRCmd(),
		_app.newCommentsCmd(),
		_app.newCommentCmd(),
		_app.newReactCmd(),
		_app.newReviewCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
