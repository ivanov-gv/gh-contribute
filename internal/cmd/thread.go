package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func (a *app) newThreadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thread <thread-id>",
		Short: "Show all comments in a thread across all reviews",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid thread ID '%s': %w", args[0], err)
			}

			prNumber, _ := cmd.Flags().GetInt("pr")
			number, err := a.resolvePR(prNumber)
			if err != nil {
				return err
			}

			t, err := a.threadService.Get(number, threadID)
			if err != nil {
				return fmt.Errorf("threadService.Get [pr=%d, thread=%d]: %w", number, threadID, err)
			}

			fmt.Print(t.Format())
			return nil
		},
	}

	cmd.Flags().Int("pr", 0, "PR number (auto-detected from current branch if not set)")
	return cmd
}
