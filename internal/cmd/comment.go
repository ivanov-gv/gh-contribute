package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (l *lazyApp) newCommentCmd() *cobra.Command {
	var prNumber int

	cmd := &cobra.Command{
		Use:   "comment <body>",
		Short: "Post a comment on a PR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := l.get()
			if err != nil {
				handleErr(err)
				return nil
			}

			body := args[0]

			// resolve PR number
			number, err := resolvePR(a, prNumber)
			if err != nil {
				return err
			}

			created, err := a.commentService.Post(number, body)
			if err != nil {
				return fmt.Errorf("commentService.Post [pr=%d]: %w", number, err)
			}

			fmt.Printf("posted comment #%d on PR #%d\n", created.DatabaseID, number)
			return nil
		},
	}

	cmd.Flags().IntVar(&prNumber, "pr", 0, "PR number (auto-detected from current branch if not set)")
	return cmd
}
