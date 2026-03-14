package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func (l *lazyApp) newReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review <review-id>",
		Short: "Show review details with inline comments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := l.get()
			if err != nil {
				handleErr(err)
				return nil
			}

			reviewID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid review ID '%s': %w", args[0], err)
			}

			// resolve PR number
			prNumber, _ := cmd.Flags().GetInt("pr")
			number, err := resolvePR(a, prNumber)
			if err != nil {
				return err
			}

			detail, err := a.reviewService.Get(number, reviewID)
			if err != nil {
				return fmt.Errorf("reviewService.Get [pr=%d, review=%d]: %w", number, reviewID, err)
			}

			fmt.Print(detail.Format())
			return nil
		},
	}

	cmd.Flags().Int("pr", 0, "PR number (auto-detected from current branch if not set)")
	return cmd
}
