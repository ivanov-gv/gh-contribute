package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func (a *app) newCommentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comments [comment-id]",
		Short: "List comments on a PR, or show a single comment by ID",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// resolve PR number
			prNumber, _ := cmd.Flags().GetInt("pr")
			number, err := a.resolvePR(prNumber)
			if err != nil {
				return err
			}

			result, err := a.commentService.List(number)
			if err != nil {
				return fmt.Errorf("commentService.List [pr=%d]: %w", number, err)
			}

			// filter by comment ID if provided
			if len(args) > 0 {
				commentID, err := strconv.ParseInt(args[0], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid comment ID '%s': %w", args[0], err)
				}
				filtered := result.FilterByID(commentID)
				if filtered == nil {
					return fmt.Errorf("comment #%d not found", commentID)
				}
				fmt.Print(filtered.Format())
				return nil
			}

			fmt.Print(result.Format())
			return nil
		},
	}

	cmd.Flags().Int("pr", 0, "PR number (auto-detected from current branch if not set)")
	return cmd
}
