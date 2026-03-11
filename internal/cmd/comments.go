package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *app) newCommentsCmd() *cobra.Command {
	var prNumber int

	cmd := &cobra.Command{
		Use:   "comments",
		Short: "List comments on a PR",
		RunE: func(cmd *cobra.Command, args []string) error {
			// resolve PR number
			number, err := a.resolvePR(prNumber)
			if err != nil {
				return err
			}

			result, err := a.commentService.List(number)
			if err != nil {
				return fmt.Errorf("commentService.List [pr=%d]: %w", number, err)
			}

			fmt.Print(result.Format())
			return nil
		},
	}

	cmd.Flags().IntVar(&prNumber, "pr", 0, "PR number (auto-detected from current branch if not set)")
	return cmd
}
