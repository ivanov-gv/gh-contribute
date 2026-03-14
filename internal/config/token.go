package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotAuthenticated is returned when no token is found.
var ErrNotAuthenticated = errors.New(
	"not authenticated — run 'gh contribute auth login' first",
)

const (
	// TokenEnv is the environment variable checked first, suitable for CI / non-interactive use.
	TokenEnv = "GH_CONTRIBUTE_TOKEN"

	// tokenConfigPath is the token file path relative to the user's home directory.
	tokenConfigPath = ".config/gh-contribute/token"
)

// LoadToken returns the GitHub App user access token.
// Priority: GH_CONTRIBUTE_TOKEN env var → ~/.config/gh-contribute/token file.
func LoadToken() (string, error) {
	// check env var first — CI / non-interactive environments
	if t := os.Getenv(TokenEnv); t != "" {
		return t, nil
	}

	// load from config file
	path, err := tokenFilePath()
	if err != nil {
		return "", fmt.Errorf("tokenFilePath: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotAuthenticated
		}
		return "", fmt.Errorf("os.ReadFile [path='%s']: %w", path, err)
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", ErrNotAuthenticated
	}

	return token, nil
}

// SaveToken persists the token to ~/.config/gh-contribute/token with 0600 permissions.
func SaveToken(token string) error {
	path, err := tokenFilePath()
	if err != nil {
		return fmt.Errorf("tokenFilePath: %w", err)
	}

	// create parent directories with restricted permissions
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("os.MkdirAll [dir='%s']: %w", filepath.Dir(path), err)
	}

	// write with owner-only permissions
	if err := os.WriteFile(path, []byte(token), 0600); err != nil {
		return fmt.Errorf("os.WriteFile [path='%s']: %w", path, err)
	}

	return nil
}

// tokenFilePath returns the absolute path to the token config file.
func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("os.UserHomeDir: %w", err)
	}
	return filepath.Join(home, tokenConfigPath), nil
}
