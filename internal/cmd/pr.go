package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ivanov-gv/gh-contribute/internal/git"
)

func (l *lazyApp) newPRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr [number]",
		Short: "Show PR details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := l.get()
			if err != nil {
				handleErr(err)
				return nil
			}

			var prNumber int
			if len(args) > 0 {
				n, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid PR number '%s': %w", args[0], err)
				}
				prNumber = n
			}

			number, err := resolvePR(a, prNumber)
			if err != nil {
				return err
			}

			info, err := a.prService.Get(number)
			if err != nil {
				return fmt.Errorf("prService.Get [number=%d]: %w", number, err)
			}

			fmt.Print(info.Format())
			return nil
		},
	}

	return cmd
}

// resolvePR determines the PR number from a positional arg or by looking up the current branch.
func resolvePR(a *app, prNumber int) (int, error) {
	if prNumber > 0 {
		return prNumber, nil
	}

	branch, err := git.CurrentBranch()
	if err != nil {
		return 0, fmt.Errorf("git.CurrentBranch: %w", err)
	}

	number, err := a.prService.FindByBranch(branch)
	if err != nil {
		return 0, fmt.Errorf("prService.FindByBranch [branch='%s']: %w", branch, err)
	}

	return number, nil
}
