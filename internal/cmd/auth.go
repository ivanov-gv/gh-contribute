package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/ivanov-gv/gh-contribute/internal/auth"
	"github.com/ivanov-gv/gh-contribute/internal/config"
)

// newAuthCmd returns the "auth" parent command with login and status subcommands.
func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage GitHub App authentication",
		// skip app initialization — auth commands do not require a stored token
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
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
			const logfmt = "auth login: "

			token, err := auth.RunDeviceFlow(func(userCode, verificationURI string) {
				// print URL and code to stdout so the user can act on them
				fmt.Printf("Open: %s\nEnter code: %s\n", verificationURI, userCode)

				// best-effort: try to open the browser
				openBrowser(verificationURI)

				log.Info().Msg(logfmt + "waiting for authorization...")
			})
			if err != nil {
				return fmt.Errorf(logfmt+"auth.RunDeviceFlow: %w", err)
			}

			if err := config.SaveToken(token); err != nil {
				return fmt.Errorf(logfmt+"config.SaveToken: %w", err)
			}

			log.Info().Msg(logfmt + "authentication successful")
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
			const logfmt = "auth status: "

			token, err := config.LoadToken()
			if err != nil {
				return fmt.Errorf(logfmt+"config.LoadToken: %w", err)
			}

			username, err := auth.GetUsername(context.Background(), token)
			if err != nil {
				return fmt.Errorf(logfmt+"auth.GetUsername: %w", err)
			}

			fmt.Printf("Logged in as: %s\n", username)
			return nil
		},
	}
}

// openBrowser attempts to open uri in the user's default browser.
// Failures are logged at debug level — the user can always open the URL manually.
func openBrowser(uri string) {
	var cmd string
	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "darwin":
		cmd = "open"
	default:
		log.Debug().Str("os", runtime.GOOS).Msg("openBrowser: unsupported OS, please open URL manually")
		return
	}
	if err := exec.Command(cmd, uri).Start(); err != nil {
		log.Debug().Err(err).Msg("openBrowser: could not open browser, please open URL manually")
	}
}
