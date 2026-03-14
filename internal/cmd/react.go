package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ivanov-gv/gh-contribute/internal/service/reaction"
)

func (a *app) newReactCmd() *cobra.Command {
	var commentType string

	cmd := &cobra.Command{
		Use:   "react <comment-id> <reaction>",
		Short: "Add a reaction to a comment",
		Long: fmt.Sprintf(
			"Add a reaction to a PR comment.\nValid reactions: %v\nComment types: review (default), issue",
			reaction.ValidReactions,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			commentID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid comment ID '%s': %w", args[0], err)
			}
			reactionContent := args[1]

			switch commentType {
			case "issue":
				err = a.reactionService.AddToIssueComment(commentID, reactionContent)
			default:
				err = a.reactionService.AddToReviewComment(commentID, reactionContent)
			}

			if err != nil {
				return fmt.Errorf("reactionService.Add [comment=%d, reaction='%s']: %w", commentID, reactionContent, err)
			}

			fmt.Printf("added '%s' reaction to comment %d\n", reactionContent, commentID)
			return nil
		},
	}

	cmd.Flags().StringVar(&commentType, "type", "review", "Comment type: review (default) or issue")
	return cmd
}
