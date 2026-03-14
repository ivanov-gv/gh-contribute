package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ivanov-gv/gh-contribute/internal/auth"
)

// newAuthCmd returns the "auth" parent command with login and status subcommands.
func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage GitHub App authentication",
	}
	authCmd.AddCommand(newAuthLoginCmd(), newAuthStatusCmd())
	return authCmd
}

// newAuthLoginCmd initiates the Device Authorization Flow and stores the resulting token.
func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate via GitHub App Device Authorization Flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := auth.RunDeviceFlow()
			if err != nil {
				return fmt.Errorf("auth.RunDeviceFlow: %w", err)
			}

			if err := auth.SaveToken(token); err != nil {
				return fmt.Errorf("auth.SaveToken: %w", err)
			}

			// confirm success to stderr — never print the token to stdout
			fmt.Fprintln(os.Stderr, "Authentication successful.")
			return nil
		},
	}
}

// newAuthStatusCmd prints the GitHub username associated with the stored token.
func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := auth.LoadToken()
			if err != nil {
				if errors.Is(err, auth.ErrNotAuthenticated) {
					fmt.Fprintln(os.Stderr, "Not authenticated. Run 'gh myext auth login' first.")
					os.Exit(1)
				}
				return fmt.Errorf("auth.LoadToken: %w", err)
			}

			username, err := auth.GetUsername(token)
			if err != nil {
				if errors.Is(err, auth.ErrTokenInvalid) {
					fmt.Fprintln(os.Stderr, "Token invalid or expired. Run 'gh myext auth login' to reauthenticate.")
					os.Exit(1)
				}
				return fmt.Errorf("auth.GetUsername: %w", err)
			}

			fmt.Printf("Logged in as: %s\n", username)
			return nil
		},
	}
}
