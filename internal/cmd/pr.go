package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ivanov-gv/gh-contribute/internal/git"
)

func (a *app) newPRCmd() *cobra.Command {
	var prNumber int

	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Show PR details",
		RunE: func(cmd *cobra.Command, args []string) error {
			// resolve PR number
			number, err := a.resolvePR(prNumber)
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

	cmd.Flags().IntVar(&prNumber, "pr", 0, "PR number (auto-detected from current branch if not set)")
	return cmd
}

// resolvePR determines the PR number — from flag or by looking up current branch
func (a *app) resolvePR(prNumber int) (int, error) {
	if prNumber > 0 {
		return prNumber, nil
	}

	// auto-detect from current branch
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
