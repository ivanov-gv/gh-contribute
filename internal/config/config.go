package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the runtime configuration
type Config struct {
	Token string // GitHub API token
	Owner string // Repository owner
	Repo  string // Repository name
}

// Load reads configuration from environment and git context
func Load() (*Config, error) {
	// load .env if present (ignore error — file is optional)
	_ = godotenv.Load()

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		// try gh auth token as fallback
		out, err := exec.Command("gh", "auth", "token").Output()
		if err != nil {
			return nil, fmt.Errorf("GITHUB_TOKEN not set and gh auth token failed: %w", err)
		}
		token = strings.TrimSpace(string(out))
	}
	if token == "" {
		return nil, fmt.Errorf("no GitHub token found")
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

// detectRepo extracts owner/repo from the git remote "origin"
func detectRepo() (string, string, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("git remote get-url origin: %w", err)
	}

	remote := strings.TrimSpace(string(out))
	return parseRemoteURL(remote)
}

// parseRemoteURL extracts owner/repo from SSH or HTTPS remote URLs
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

// parseOwnerRepo splits "owner/repo.git" into owner and repo
func parseOwnerRepo(path string) (string, string, error) {
	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cannot parse owner/repo from: %s", path)
	}
	return parts[0], parts[1], nil
}
