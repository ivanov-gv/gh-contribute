package config

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the runtime configuration.
type Config struct {
	Token string // GitHub App user access token
	Owner string // Repository owner
	Repo  string // Repository name
}

// Load reads configuration from the environment and git context.
// Token priority: GH_CONTRIBUTE_TOKEN env var → ~/.config/gh-contribute/token file.
func Load() (*Config, error) {
	// load .env if present (ignore error — file is optional)
	_ = godotenv.Load()

	// load GitHub App user access token
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("LoadToken: %w", err)
	}

	// detect owner/repo from git remote
	owner, repo, err := detectRepo()
	if err != nil {
		return nil, fmt.Errorf("detectRepo: %w", err)
	}

	return &Config{
		Token: token,
		Owner: owner,
		Repo:  repo,
	}, nil
}

// detectRepo extracts owner/repo from the git remote "origin".
func detectRepo() (string, string, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("git remote get-url origin: %w", err)
	}

	remote := strings.TrimSpace(string(out))
	return parseRemoteURL(remote)
}

// parseRemoteURL extracts owner/repo from SSH or HTTPS remote URLs.
func parseRemoteURL(remote string) (string, string, error) {
	// SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("unexpected SSH remote format: %s", remote)
		}
		return parseOwnerRepo(parts[1])
	}

	// HTTPS: https://github.com/owner/repo.git
	remote = strings.TrimPrefix(remote, "https://")
	remote = strings.TrimPrefix(remote, "http://")
	// remove host part
	parts := strings.SplitN(remote, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected HTTPS remote format: %s", remote)
	}
	return parseOwnerRepo(parts[1])
}

// parseOwnerRepo extracts "owner/repo" from the last two slash-separated path segments.
// Handles standard paths ("owner/repo.git") and proxy paths ("/git/owner/repo").
func parseOwnerRepo(path string) (string, string, error) {
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	// filter out empty segments (leading slash, etc.)
	var segments []string
	for _, p := range parts {
		if p != "" {
			segments = append(segments, p)
		}
	}
	if len(segments) < 2 {
		return "", "", fmt.Errorf("cannot parse owner/repo from: %s", path)
	}
	owner := segments[len(segments)-2]
	repo := segments[len(segments)-1]
	return owner, repo, nil
}
