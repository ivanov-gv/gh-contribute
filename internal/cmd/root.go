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

// app holds shared dependencies for all commands
type app struct {
	cfg             *config.Config
	prService       *pr.Service
	commentService  *comment.Service
	reactionService *reaction.Service
	reviewService   *review.Service
}

// Execute runs the root command
func Execute() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config.Load: %v\n", err)
		os.Exit(1)
	}

	// GraphQL client for read operations
	gql := ghclient.NewGraphQLClient(cfg.Token)
	// REST client for write operations (post comment, add reaction)
	rest := ghrest.NewClient(nil).WithAuthToken(cfg.Token)

	a := &app{
		cfg:             cfg,
		prService:       pr.NewService(gql, cfg.Owner, cfg.Repo),
		commentService:  comment.NewService(gql, rest, cfg.Owner, cfg.Repo),
		reactionService: reaction.NewService(rest, cfg.Owner, cfg.Repo),
		reviewService:   review.NewService(gql, cfg.Owner, cfg.Repo),
	}

	rootCmd := &cobra.Command{
		Use:   "gh-contribute",
		Short: "A gh extension for simplifying agents interaction with PRs on GitHub",
	}

	rootCmd.AddCommand(
		a.newPRCmd(),
		a.newCommentsCmd(),
		a.newCommentCmd(),
		a.newReactCmd(),
		a.newReviewCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
